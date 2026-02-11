package tcp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
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

type Client struct {
	Created_at time.Time       `json:"created_at"`
	Addr       string          `json:"addr"`
	Status     Status          `json:"status"`
	conn       net.Conn        `json:"-"`
	inbox      chan string     `json:"-"`
	done       chan struct{}   `json:"-"`
	mu         sync.Mutex      `json:"-"`
	ctx        context.Context `json:"-"`
	isDebug    bool            `json:"-"`
}

/**
* NewClient
* @param addr string
* @return *Client, error
**/
func NewClient(addr string) *Client {
	now := timezone.Now()
	result := &Client{
		Created_at: now,
		Addr:       addr,
		Status:     Connected,
		inbox:      make(chan string),
		done:       make(chan struct{}),
		mu:         sync.Mutex{},
		ctx:        context.Background(),
	}
	return result
}

/**
* ToJson
* @return et.Json
**/
func (s *Client) ToJson() et.Json {
	return et.Json{
		"created_at": s.Created_at,
		"addr":       s.Addr,
		"status":     s.Status,
	}
}

/**
* read
**/
func (s *Client) read() {
	reader := bufio.NewReader(s.conn)

	for {
		s.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		read, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				logs.Log(packageName, msg.MSG_TCP_SERVER_CLOSED)
			} else {
				logs.Logf(packageName, msg.MSG_TCP_ERROR_READ, err)
			}

			s.handleDisconnect()
			return
		}
		s.inbox <- read
	}
}

/**
* write
**/
func (s *Client) write() {
	for msg := range s.inbox {
		logs.Debugf("recv: %s", msg)
	}
}

/**
* handleDisconnect
**/
func (s *Client) handleDisconnect() {
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

	close(s.inbox)
	close(s.done)
}

/**
* Start
**/
func (s *Client) Start() error {
	conn, err := net.Dial("tcp", s.Addr)
	if err != nil {
		return err
	}

	s.conn = conn
	go s.read()
	go s.write()
	logs.Logf(packageName, msg.MSG_CLIENT_CONNECTED, s.Addr)
	s.Send(TextMessage, "Hola")

	utility.AppWait()
	return nil
}

/**
* Send
**/
func (s *Client) Send(tp int, message any) error {
	if s.Status != Connected {
		return nil
	}

	bt, ok := message.([]byte)
	if !ok {
		var err error
		bt, err = json.Marshal(message)
		if err != nil {
			return err
		}
	}

	m := Message{
		Type:    tp,
		Message: bt,
	}

	switch tp {
	case PingMessage:
		m.Message = []byte("PING\n")
	case PongMessage:
		m.Message = []byte("PONG\n")
	case ACKMessage:
		m.Message = []byte("ACK\n")
	case CloseMessage:
		s.handleDisconnect()
		return nil
	}

	payload, err := m.serialize()
	if err != nil {
		return err
	}

	_, err = s.conn.Write(payload)
	if err != nil {
		s.handleDisconnect()
		return err
	}

	return nil
}
