package tcp

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
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
	ACKMessage    int = 3
	CloseMessage  int = 8
	PingMessage   int = 9
	PongMessage   int = 10
)

type Outbound struct {
	Type    int    `json:"type"`
	Message []byte `json:"message"`
}

/**
* serialize
* @return []byte, error
**/
func (s *Outbound) serialize() ([]byte, error) {
	return json.Marshal(s)
}

/**
* ToJson
* @return et.Json
**/
func (s *Outbound) ToJson() (et.Json, error) {
	bt, err := s.serialize()
	if err != nil {
		return nil, err
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* ToOutbound
* @param bt []byte
* @return Outbound, error
**/
func ToOutbound(bt []byte) (Outbound, error) {
	var result Outbound
	err := json.Unmarshal(bt, &result)
	if err != nil {
		return Outbound{}, err
	}

	return result, nil
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

	go s.readLoop()
	go s.writeLoop()

	logs.Logf(packageName, msg.MSG_CLIENT_CONNECTED, s.Addr)
	s.SendHola()

	utility.AppWait()
	return nil
}

/**
* readLoop
**/
func (c *Client) readLoop() {
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
* writeLoop
**/
func (c *Client) writeLoop() {
	for out := range c.outbound {
		err := c.write(out)
		if err != nil {
			return
		}
	}
}

/**
* write
**/
func (c *Client) write(out Outbound) error {
	if c.Status != Connected {
		return nil
	}

	switch out.Type {
	case PingMessage:
		out.Message = []byte("PING\n")
	case PongMessage:
		out.Message = []byte("PONG\n")
	case ACKMessage:
		out.Message = []byte("ACK\n")
	case CloseMessage:
		c.close()
		return nil
	}

	payload, err := out.serialize()
	if err != nil {
		return err
	}

	_, err = c.conn.Write(payload)
	if err != nil {
		return err
	}

	return nil
}

/**
* listener
* @param data []byte
**/
func (s *Client) listener(data []byte) {
	out, err := ToOutbound(data)
	if err != nil {
		return
	}

	// msg := strings.TrimSpace(string(out.Message))
	// if s.isDebug {
	// 	logs.Debug("recv:", msg)
	// }

	switch out.Type {
	case PingMessage:
		s.Send(PongMessage, nil)

	case PongMessage:
		// heartbeat ok

	default:
		logs.Debug("listener message:", out.Message)
	}
}

/**
* Response
* @param tp int, value any
* @return error
**/
func (s *Client) Response(tp int, value any) error {
	return s.write(Outbound{
		Type:    tp,
		Message: value,
	})
}

/**
* Send
* @param tp int, bt []byte
* @return error
**/
func (s *Client) Send(tp int, value any) error {
	bt, ok := value.([]byte)
	if !ok {
		var err error
		bt, err = json.Marshal(value)
		if err != nil {
			return err
		}
	}

	s.outbound <- Outbound{
		Type:    tp,
		Message: bt,
	}
	return nil
}

/**
* SendMessage
* @param message interface{}
* @return error
**/
func (s *Client) SendMessage(message interface{}) error {
	return s.Send(TextMessage, message)
}

/**
* Error
* @param err error
**/
func (s *Client) SendError(err error) error {
	ms := et.Item{
		Ok: false,
		Result: et.Json{
			"message": err.Error(),
		},
	}
	return s.SendMessage(ms)
}

/**
* SendHola
**/
func (s *Client) SendHola() error {
	ms := et.Item{
		Ok: true,
		Result: et.Json{
			"message": msg.MSG_HOLA,
		},
	}
	return s.SendMessage(ms)
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

	if err != nil {
		logs.Errorm("desconectado:" + err.Error())
	}

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
