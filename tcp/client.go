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
func NewClient(addr string) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	now := timezone.Now()
	result := &Client{
		Created_at: now,
		Addr:       addr,
		Status:     Connected,
		conn:       conn,
		inbox:      make(chan string),
		done:       make(chan struct{}),
		mu:         sync.Mutex{},
		ctx:        context.Background(),
	}

	go result.read()
	go result.write()
	return result, nil
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
		select {
		case <-s.done:
			return
		default:
			msg, err := reader.ReadString('\n')
			if err != nil {
				close(s.done)
				return
			}
			s.inbox <- msg
		}
	}
}

/**
* write
**/
func (s *Client) write() {
	for msg := range s.inbox {
		logs.Debugf(packageName, "recv: %s", msg)
	}
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
		s.close()
		return nil
	}

	payload, err := m.serialize()
	if err != nil {
		return err
	}

	_, err = s.conn.Write(payload)
	if err != nil {
		return err
	}

	return nil
}

/**
* handleDisconnect
* @param err error
**/
func (s *Client) handleDisconnect(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Status == Disconnected {
		return
	}

	if err != nil {
		if err != io.EOF {
			logs.Info(msg.MSG_TCP_SERVER_CLOSED)
		} else {
			logs.Info(msg.MSG_TCP_CLIENT_CLOSED)
		}
		return
	}

	s.Status = Disconnected
	if s.conn != nil {
		s.conn.Close()
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
	close(s.inbox)

	if s.conn != nil {
		s.conn.Close()
	}
}
