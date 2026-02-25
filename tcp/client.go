package tcp

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

type Status string

const (
	Pending      Status = "pending"
	Connected    Status = "connected"
	Disconnected Status = "disconnected"
)

type Client struct {
	ctx          context.Context           `json:"-"`
	cancel       context.CancelFunc        `json:"-"`
	CreatedAt    time.Time                 `json:"created_at"`
	ID           string                    `json:"id"`
	Addr         string                    `json:"addr"`
	LocalAddr    string                    `json:"local_addr"`
	RemoteAddr   string                    `json:"remote_addr"`
	Status       Status                    `json:"status"`
	Ctx          et.Json                   `json:"-"`
	conn         net.Conn                  `json:"-"`
	inbound      chan []byte               `json:"-"`
	messages     map[string]chan *Message  `json:"-"`
	timeout      time.Duration             `json:"-"`
	mu           sync.Mutex                `json:"-"`
	onConnect    []func(*Client)           `json:"-"`
	onDisconnect []func(*Client)           `json:"-"`
	onError      []func(*Client, error)    `json:"-"`
	onOutbound   []func(*Client, []byte)   `json:"-"`
	onInbound    []func(*Client, *Message) `json:"-"`
	isNode       bool                      `json:"-"`
	alive        atomic.Bool               `json:"-"`
	closed       atomic.Bool               `json:"-"`
}

/**
* NewClient
* @param addr string
* @return *Client, error
**/
func NewClient(addr string) *Client {
	timeout, err := time.ParseDuration(envar.GetStr("TIMEOUT", "10s"))
	if err != nil {
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := &Client{
		ctx:          ctx,
		cancel:       cancel,
		CreatedAt:    timezone.Now(),
		ID:           reg.ULID(),
		Addr:         addr,
		Status:       Pending,
		inbound:      make(chan []byte, 128),
		messages:     make(map[string]chan *Message),
		timeout:      timeout,
		mu:           sync.Mutex{},
		Ctx:          et.Json{},
		onConnect:    make([]func(*Client), 0),
		onDisconnect: make([]func(*Client), 0),
		onError:      make([]func(*Client, error), 0),
		onOutbound:   make([]func(*Client, []byte), 0),
		onInbound:    make([]func(*Client, *Message), 0),
	}
	return result
}

/**
* ToJson
* @return et.Json
**/
func (s *Client) ToJson() et.Json {
	return et.Json{
		"created_at":  s.CreatedAt,
		"addr":        s.Addr,
		"local_addr":  s.LocalAddr,
		"remote_addr": s.RemoteAddr,
		"status":      s.Status,
	}
}

/**
* error
* @param err error
* @return error
**/
func (s *Client) error(err error) error {
	for _, fn := range s.onError {
		fn(s, err)
	}

	logs.Error(err)

	return err
}

/**
* connect
**/
func (s *Client) connect() error {
	dialer := net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	conn, err := dialer.Dial("tcp", s.Addr)
	if err != nil {
		return s.error(err)
	}

	s.mu.Lock()
	s.conn = conn
	s.Status = Connected
	s.LocalAddr = s.conn.LocalAddr().String()
	s.RemoteAddr = s.conn.RemoteAddr().String()
	s.mu.Unlock()

	for _, fn := range s.onConnect {
		fn(s)
	}

	return nil
}

/**
* disconnect
**/
func (s *Client) disconnect() {
	if !s.closed.CompareAndSwap(false, true) {
		return
	}

	s.mu.Lock()
	if s.Status == Disconnected {
		s.mu.Unlock()
		return
	}
	s.Status = Disconnected
	conn := s.conn
	s.mu.Unlock()

	logs.Logf(packageName, msg.MSG_TCP_DISCONNECTED, s.Addr)

	if conn != nil {
		_ = conn.Close()
	}

	s.cancel()

	for _, fn := range s.onDisconnect {
		fn(s)
	}
}

/**
* readLoop
**/
func (s *Client) readLoop() {
	reader := bufio.NewReader(s.conn)

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		lenBuf := make([]byte, 4)
		_, err := io.ReadFull(reader, lenBuf)
		if err != nil {
			s.disconnect()
			return
		}

		length := binary.BigEndian.Uint32(lenBuf)
		limitReader := envar.GetInt("LIMIT_SIZE_MG", 10)
		if length > uint32(limitReader*1024*1024) {
			continue
		}

		data := make([]byte, length)
		_, err = io.ReadFull(reader, data)
		if err != nil {
			s.disconnect()
			return
		}

		select {
		case s.inbound <- data:
		case <-s.ctx.Done():
			return
		default:
		}
	}
}

/**
* run
**/
func (s *Client) run() {
	for {
		select {
		case <-s.ctx.Done():
			return

		case bt := <-s.inbound:
			if bt != nil {
				s.inbox(bt)
			}
		}
	}
}

/**
* send
* @param msg *Message
* @return error
**/
func (s *Client) send(msg *Message) error {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("connection not established")
	}

	bt, err := msg.serialize()
	if err != nil {
		return err
	}

	conn.SetWriteDeadline(time.Now().Add(s.timeout))

	_, err = conn.Write(bt)
	if err != nil {
		s.disconnect()
		return err
	}

	for _, fn := range s.onOutbound {
		fn(s, bt)
	}

	return nil
}

/**
* inbox
**/
func (s *Client) inbox(bt []byte) {
	msg, err := ToMessage(bt)
	if err != nil {
		s.error(err)
		return
	}

	s.mu.Lock()
	ch, ok := s.messages[msg.ID]
	s.mu.Unlock()

	if ok {
		select {
		case ch <- msg:
		default:
		}
		return
	}

	for _, fn := range s.onInbound {
		fn(s, msg)
	}
}

/**
* request
* @param m *Message
* @return *Message, error
**/
func (s *Client) request(m *Message) (*Message, error) {
	ch := make(chan *Message, 1)

	s.mu.Lock()
	s.messages[m.ID] = ch
	s.mu.Unlock()

	err := s.send(m)
	if err != nil {
		s.mu.Lock()
		delete(s.messages, m.ID)
		s.mu.Unlock()
		return nil, s.error(err)
	}

	select {
	case resp := <-ch:
		s.mu.Lock()
		delete(s.messages, m.ID)
		s.mu.Unlock()
		return resp, nil

	case <-time.After(s.timeout):
		s.mu.Lock()
		delete(s.messages, m.ID)
		s.mu.Unlock()
		return nil, fmt.Errorf(msg.MSG_TCP_TIMEOUT)
	}
}

/**
* Connect
**/
func (s *Client) Connect() error {
	if s.closed.Load() {
		return fmt.Errorf("client already closed")
	}

	err := s.connect()
	if err != nil {
		return err
	}

	go s.readLoop()
	go s.run()

	logs.Logf(packageName, msg.MSG_TCP_CONNECTED_TO, s.Addr)

	return nil
}

/**
* Start
**/
func (s *Client) Start() error {
	err := s.Connect()
	if err != nil {
		return err
	}

	utility.AppWait()
	return nil
}

/**
* Close
* @return error
**/
func (s *Client) Close() error {
	s.disconnect()
	return nil
}

/**
* Send
* @param tp int, message any
* @return error
**/
func (s *Client) Send(msg *Message) error {
	// Send
	err := s.send(msg)
	if err != nil {
		return s.error(err)
	}

	if msg.Type == CloseMessage {
		s.disconnect()
	}

	return nil
}

/**
* Request
* @param method string, args ...any
* @return *Response
**/
func (s *Client) Request(method string, args ...any) *Response {
	m, err := NewMessage(Method, "")
	if err != nil {
		return TcpError(err)
	}
	m.Method = method
	for _, arg := range args {
		m.Args = append(m.Args, arg)
	}

	res, err := s.request(m)
	if err != nil {
		return TcpError(err)
	}

	if res.Error != "" {
		return TcpError(res.Error)
	}

	result, err := res.Response()
	if err != nil {
		return TcpError(err)
	}

	return result
}

/**
* OnConnect
* @param fn func(*Client)
**/
func (s *Client) OnConnect(fn func(*Client)) {
	s.onConnect = append(s.onConnect, fn)
}

/**
* OnDisconnect
* @param fn func(*Client)
**/
func (s *Client) OnDisconnect(fn func(*Client)) {
	s.onDisconnect = append(s.onDisconnect, fn)
}

/**
* OnError
* @param fn func(*Client, error)
**/
func (s *Client) OnError(fn func(*Client, error)) {
	s.onError = append(s.onError, fn)
}

/**
* OnOutbound
* @param fn func(*Client, []byte)
**/
func (s *Client) OnOutbound(fn func(*Client, []byte)) {
	s.onOutbound = append(s.onOutbound, fn)
}

/**
* OnInbound
* @param fn func(*Client, *Message)
**/
func (s *Client) OnInbound(fn func(*Client, *Message)) {
	s.onInbound = append(s.onInbound, fn)
}
