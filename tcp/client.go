package tcp

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
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
	Created_at   time.Time                 `json:"created_at"`
	ID           string                    `json:"id"`
	Addr         string                    `json:"addr"`
	LocalAddr    string                    `json:"local_addr"`
	RemoteAddr   string                    `json:"remote_addr"`
	Status       Status                    `json:"status"`
	conn         net.Conn                  `json:"-"`
	inbound      chan []byte               `json:"-"`
	outbound     chan []byte               `json:"-"`
	pending      map[string]chan *Message  `json:"-"`
	done         chan struct{}             `json:"-"`
	timeout      time.Duration             `json:"-"`
	mu           sync.Mutex                `json:"-"`
	ctx          context.Context           `json:"-"`
	onConnect    []func(*Client)           `json:"-"`
	onDisconnect []func(*Client)           `json:"-"`
	onError      []func(*Client, error)    `json:"-"`
	onOutbound   []func(*Client, []byte)   `json:"-"`
	onInbound    []func(*Client, *Message) `json:"-"`
	isDebug      bool                      `json:"-"`
	isNode       bool                      `json:"-"`
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
	result := &Client{
		Created_at:   timezone.Now(),
		ID:           reg.ULID(),
		Addr:         addr,
		Status:       Pending,
		inbound:      make(chan []byte),
		outbound:     make(chan []byte),
		pending:      make(map[string]chan *Message),
		done:         make(chan struct{}),
		timeout:      timeout,
		mu:           sync.Mutex{},
		ctx:          context.Background(),
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
* NewNode
* @param addr string
* @return *Client
**/
func NewNode(addr string) *Client {
	result := NewClient(addr)
	result.isNode = true
	return result
}

/**
* ToJson
* @return et.Json
**/
func (s *Client) toJson() et.Json {
	return et.Json{
		"created_at": s.Created_at,
		"addr":       s.Addr,
		"status":     s.Status,
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

	return err
}

/**
* incoming
**/
func (s *Client) incoming() {
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
* inbound
**/
func (s *Client) inbox() {
	for bt := range s.inbound {
		msg, err := toMessage(bt)
		if err != nil {
			s.error(err)
			return
		}

		s.mu.Lock()
		ch, ok := s.pending[msg.ID]
		s.mu.Unlock()

		if ok {
			ch <- msg
			return
		}

		switch msg.Type {
		default:
			for _, fn := range s.onInbound {
				fn(s, msg)
			}
		}

		if !s.isNode {
			logs.Debugf("recv: %s", msg.ToJson().ToString())
		}
	}
}

/**
* send
**/
func (s *Client) send() {
	for msg := range s.outbound {
		_, err := s.conn.Write(msg)
		if err != nil {
			s.disconnect()
			s.error(err)
			return
		}

		for _, fn := range s.onOutbound {
			fn(s, msg)
		}
	}
}

/**
* disconnect
**/
func (s *Client) disconnect() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Status == Disconnected {
		return
	}

	s.Status = Disconnected
	logs.Logf(packageName, msg.MSG_CLIENT_DISCONNECTED, s.Addr)

	if s.conn != nil {
		s.conn.Close()
	}

	close(s.inbound)
	close(s.outbound)
	close(s.done)
	for _, fn := range s.onDisconnect {
		fn(s)
	}
}

/**
* connect
**/
func (s *Client) connect() (net.Conn, error) {
	dialer := net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	result, err := dialer.Dial("tcp", s.Addr)
	if err != nil {
		return nil, s.error(err)
	}

	for _, fn := range s.onConnect {
		fn(s)
	}

	return result, nil
}

/**
* Connect
**/
func (s *Client) Connect() error {
	conn, err := s.connect()
	if err != nil {
		return s.error(err)
	}

	s.LocalAddr = conn.LocalAddr().String()
	s.RemoteAddr = conn.RemoteAddr().String()
	s.Status = Connected
	s.conn = conn
	go s.incoming()
	go s.inbox()
	go s.send()
	if s.isNode {
		logs.Logf(packageName, msg.MSG_NODE_CONNECTED, s.Addr)
	} else {
		logs.Logf(packageName, msg.MSG_CLIENT_CONNECTED, s.Addr)
	}
	if !s.isNode && s.isDebug {
		// msg, err := s.Request(RequestVote, "", 10*time.Second)
		// if err != nil {
		// 	s.error(err)
		// }
		// if msg != nil {
		// 	logs.Debug("send:", msg.ToJson().ToString())
		// }
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
* Send
* @param tp int, message any
* @return error
**/
func (s *Client) Send(tp int, message any) error {
	msg, err := newMessage(tp, message)
	if err != nil {
		return s.error(err)
	}

	bt, err := msg.serialize()
	if err != nil {
		return s.error(err)
	}

	// Connect
	if s.Status != Connected {
		conn, err := s.connect()
		if err != nil {
			return s.error(err)
		}
		s.Status = Connected
		s.conn = conn
	}

	// Send
	s.outbound <- bt

	if tp == CloseMessage {
		s.disconnect()
	}

	if s.isDebug {
		logs.Debugf("send: %s", msg.ToJson().ToString())
	}

	return nil
}

/**
* Request
* @param tp int, payload any
* @return *Message, error
**/
func (s *Client) Request(tp int, payload any) (*Message, error) {
	m, err := newMessage(tp, payload)
	if err != nil {
		return nil, err
	}

	// Channel for response
	ch := make(chan *Message, 1)
	s.mu.Lock()
	s.pending[m.ID] = ch
	s.mu.Unlock()

	bt, err := m.serialize()
	if err != nil {
		return nil, s.error(err)
	}

	// Connect
	if s.Status != Connected {
		conn, err := s.connect()
		if err != nil {
			return nil, s.error(err)
		}
		s.Status = Connected
		s.conn = conn
	}

	// Send
	s.outbound <- bt

	if s.isDebug {
		logs.Debugf("Request: %s", m.ToJson().ToString())
	}

	// Wait response or timeout
	select {
	case resp := <-ch:
		s.mu.Lock()
		delete(s.pending, m.ID)
		s.mu.Unlock()
		return resp, nil

	case <-time.After(s.timeout):
		s.mu.Lock()
		delete(s.pending, m.ID)
		s.mu.Unlock()
		return nil, fmt.Errorf(msg.MSG_TCP_TIMEOUT)

	case <-s.done:
		return nil, fmt.Errorf(msg.MSG_CLIENT_DISCONNECTED, s.Addr)
	}
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
