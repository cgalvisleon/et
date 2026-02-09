package tcp

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

type Status string

const (
	Pending      Status = "pending"
	Connected    Status = "connected"
	Disconnected Status = "disconnected"
)

const (
	TextMessage   int = 1
	BinaryMessage int = 2
	CloseMessage  int = 8
	PingMessage   int = 9
	PongMessage   int = 10
)

type Outbound struct {
	MessageType int    `json:"message_type"`
	Message     []byte `json:"message"`
}

/**
* serialize
* @return ([]byte, error)
**/
func (s *Outbound) serialize() ([]byte, error) {
	result, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return result, nil
}

/**
* ToOutbound
* @param data []byte
* @return Outbound
**/
func ToOutbound(data []byte) (Outbound, error) {
	var result Outbound
	err := json.Unmarshal(data, &result)
	return result, err
}

type Client struct {
	Created_at time.Time       `json:"created_at"`
	Name       string          `json:"name"`
	Addr       string          `json:"addr"`
	Status     Status          `json:"status"`
	conn       net.Conn        `json:"-"`
	outbound   chan Outbound   `json:"-"`
	mu         sync.Mutex      `json:"-"`
	ctx        context.Context `json:"-"`
	isDebug    bool            `json:"-"`
}

func NewClient(name, addr string) *Client {
	now := timezone.Now()
	return &Client{
		Created_at: now,
		Name:       name,
		Addr:       addr,
		Status:     Pending,
		outbound:   make(chan Outbound),
		mu:         sync.Mutex{},
		ctx:        context.Background(),
	}
}

/**
* ToJson
* @return et.Json
**/
func (c *Client) ToJson() et.Json {
	return et.Json{
		"created_at": c.Created_at,
		"name":       c.Name,
		"addr":       c.Addr,
		"status":     c.Status,
	}
}

/**
* Connect
* @return error
**/
func (s *Client) Connect() error {
	conn, err := net.Dial("tcp", s.Addr)
	if err != nil {
		s.Status = Disconnected
		return err
	}

	s.mu.Lock()
	s.conn = conn
	s.Status = Connected
	s.mu.Unlock()

	if tcp, ok := conn.(*net.TCPConn); ok {
		tcp.SetKeepAlive(true)
		tcp.SetKeepAlivePeriod(10 * time.Second)
	}

	go s.read()
	go s.write()

	logs.Logf("TCP", "Client connected: %s", s.Addr)

	s.SendHola()

	utility.AppWait()
	return nil
}

/**
* read
**/
func (c *Client) read() {
	reader := bufio.NewReader(c.conn)

	for {
		// c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		c.conn.SetReadDeadline(time.Time{})
		data, err := reader.ReadBytes('\n')
		if err != nil {
			c.handleDisconnect(err)
			return
		}

		c.listener(data)
	}
}

/**
* write
**/
func (c *Client) write() {
	for out := range c.outbound {
		if c.Status != Connected {
			return
		}

		c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		switch out.MessageType {
		case PingMessage:
			out.Message = []byte("PING\n")
		case PongMessage:
			out.Message = []byte("PONG\n")
		case CloseMessage:
			c.close()
			return
		}

		payload, err := out.serialize()
		if err != nil {
			return
		}

		_, err = c.conn.Write(payload)
		if err != nil {
			c.handleDisconnect(err)
			return
		}
	}
}

/**
* listener
* @param data []byte
**/
func (s *Client) listener(data []byte) {
	msg := strings.TrimSpace(string(data))

	if s.isDebug {
		logs.Debug("recv:", msg)
	}

	switch msg {
	case "PING":
		s.Send(PongMessage, nil)

	case "PONG":
		// heartbeat ok

	default:
		logs.Debug("listener message:", msg)
	}
}

/**
* send
* @param tp int, bt []byte
**/
func (c *Client) Send(tp int, bt []byte) {
	c.outbound <- Outbound{
		MessageType: tp,
		Message:     bt,
	}
}

/**
* SendMessage
* @param message interface{}
**/
func (s *Client) SendMessage(message interface{}) {
	bt, err := json.Marshal(message)
	if err != nil {
		return
	}
	s.Send(TextMessage, bt)
}

/**
* Error
* @param err error
**/
func (s *Client) SendError(err error) {
	ms := et.Item{
		Ok: false,
		Result: et.Json{
			"message": err.Error(),
		},
	}
	s.SendMessage(ms)
}

/**
* SendHola
**/
func (s *Client) SendHola() {
	ms := et.Item{
		Ok: true,
		Result: et.Json{
			"message": msg.MSG_HOLA,
		},
	}
	bt, err := ms.ToByte()
	if err != nil {
		return
	}

	s.Send(TextMessage, bt)
}

/**
* handleDisconnect
* @param err error
**/
func (c *Client) handleDisconnect(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Status == Disconnected {
		return
	}

	if c.isDebug {
		logs.Debug("desconectado:", err)
	}

	logs.Debug("desconectado:", err)

	c.Status = Disconnected
	if c.conn != nil {
		c.conn.Close()
	}
}

/**
* close
**/
func (s *Client) close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Status == Disconnected {
		return
	}

	s.Status = Disconnected
	close(s.outbound)

	if s.conn != nil {
		s.conn.Close()
	}
}
