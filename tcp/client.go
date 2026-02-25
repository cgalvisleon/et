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
	mg "github.com/cgalvisleon/et/msg"
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
	isDebug      bool                      `json:"-"`
	isNode       bool                      `json:"-"`
	alive        atomic.Bool               `json:"-"`
}

/**
* NewClient
* @param addr string
* @return *Client, error
**/
func NewClient(addr string) *Client {
	isDebug := envar.GetBool("IS_DEBUG", false)
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
		inbound:      make(chan []byte),
		messages:     make(map[string]chan *Message),
		timeout:      timeout,
		mu:           sync.Mutex{},
		Ctx:          et.Json{},
		onConnect:    make([]func(*Client), 0),
		onDisconnect: make([]func(*Client), 0),
		onError:      make([]func(*Client, error), 0),
		onOutbound:   make([]func(*Client, []byte), 0),
		onInbound:    make([]func(*Client, *Message), 0),
		isDebug:      isDebug,
	}
	return result
}

/**
* ToJson
* @return et.Json
**/
func (s *Client) toJson() et.Json {
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
	s.mu.Lock()
	status := s.Status
	s.mu.Unlock()

	if status == Disconnected {
		return
	}

	s.mu.Lock()
	s.Status = Disconnected
	s.mu.Unlock()
	logs.Logf(packageName, msg.MSG_TCP_DISCONNECTED, s.Addr)

	if s.conn != nil {
		s.conn.Close()
	}

	for _, fn := range s.onDisconnect {
		fn(s)
	}

	close(s.inbound)
}

/**
* readLoop
**/
func (s *Client) readLoop() {
	reader := bufio.NewReader(s.conn)

	for {
		// Leer tamaño (4 bytes)
		lenBuf := make([]byte, 4)
		_, err := io.ReadFull(reader, lenBuf)
		if err != nil {
			if err == io.EOF {
				s.disconnect()
			} else {
				s.error(err)
			}
			return
		}

		// Leer tamaño payload
		length := binary.BigEndian.Uint32(lenBuf)
		limitReader := envar.GetInt("LIMIT_SIZE_MG", 10)
		if length > uint32(limitReader*1024*1024) {
			continue
		}

		// Leer payload completo
		data := make([]byte, length)
		_, err = io.ReadFull(reader, data)
		if err != nil {
			s.error(err)
			return
		}

		s.inbound <- data
	}
}

/**
* run
**/
func (s *Client) run() {
	for {
		select {
		case <-s.ctx.Done():
			logs.Logf(packageName, msg.MSG_TCP_DISCONNECTED, s.Addr)
			s.conn.Close()
			return
		default:
			msg := <-s.inbound
			s.inbox(msg)
		}
	}
}

/**
* send
* @param msg *Message
* @return error
**/
func (s *Client) send(msg *Message) error {
	bt, err := msg.serialize()
	if err != nil {
		return err
	}
	_, err = s.conn.Write(bt)
	if err != nil {
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
		ch <- msg
		return
	}

	for _, fn := range s.onInbound {
		fn(s, msg)
	}

	logs.Debugf("inbox :%s", msg.ID)
}

/**
* request
* @param m *Message
* @return *Message, error
**/
func (s *Client) request(m *Message) (*Message, error) {
	// Channel for response
	ch := make(chan *Message, 1)
	s.mu.Lock()
	s.messages[m.ID] = ch
	s.mu.Unlock()

	// Send
	err := s.send(m)
	if err != nil {
		return nil, s.error(err)
	}

	// Wait response or timeout
	select {
	case resp := <-ch:
		s.mu.Lock()
		delete(s.messages, m.ID)
		s.mu.Unlock()
		logs.Debugf("response: %s type:%d", resp.ID, resp.Type)
		return resp, nil

	case <-time.After(s.timeout):
		s.mu.Lock()
		delete(s.messages, m.ID)
		s.mu.Unlock()
		logs.Debugf("response timeout: %s", m.ID)
		return nil, fmt.Errorf(msg.MSG_TCP_TIMEOUT)
	}
}

/**
* Connect
**/
func (s *Client) Connect() error {
	err := s.connect()
	if err != nil {
		return s.error(err)
	}

	go s.readLoop()
	go s.run()

	logs.Logf(packageName, msg.MSG_TCP_CONNECTED_TO, s.Addr)
	if s.isDebug {
		logs.Debugf("connected: %s", s.toJson().ToString())

		res := s.Request(PingMehtod, s.ID, s.Ctx)
		if res.Error != nil {
			return s.error(res.Error)
		}

		logs.Debugf("respuesta: %s", res.Response)
	}

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
	s.cancel()
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

	if s.isDebug {
		logs.Debugf(mg.MSG_SEND_TO, msg.ToJson().ToString(), s.RemoteAddr)
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
