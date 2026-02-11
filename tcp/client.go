package tcp

import (
	"bufio"
	"context"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"time"

	"github.com/cgalvisleon/et/envar"
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
	Created_at   time.Time               `json:"created_at"`
	Addr         string                  `json:"addr"`
	Status       Status                  `json:"status"`
	conn         net.Conn                `json:"-"`
	inbound      chan []byte             `json:"-"`
	outbound     chan []byte             `json:"-"`
	done         chan struct{}           `json:"-"`
	mu           sync.Mutex              `json:"-"`
	ctx          context.Context         `json:"-"`
	onConnect    []func(*Client)         `json:"-"`
	onDisconnect []func(*Client)         `json:"-"`
	onError      []func(*Client, error)  `json:"-"`
	onOutbound   []func(*Client, []byte) `json:"-"`
	onInbound    []func(*Client, []byte) `json:"-"`
	isDebug      bool                    `json:"-"`
}

/**
* NewClient
* @param addr string
* @return *Client, error
**/
func NewClient(addr string) *Client {
	isDebug := envar.GetBool("IS_DEBUG", false)
	now := timezone.Now()
	result := &Client{
		Created_at:   now,
		Addr:         addr,
		Status:       Connected,
		inbound:      make(chan []byte),
		outbound:     make(chan []byte),
		done:         make(chan struct{}),
		mu:           sync.Mutex{},
		ctx:          context.Background(),
		onConnect:    make([]func(*Client), 0),
		onDisconnect: make([]func(*Client), 0),
		onError:      make([]func(*Client, error), 0),
		onOutbound:   make([]func(*Client, []byte), 0),
		onInbound:    make([]func(*Client, []byte), 0),
		isDebug:      isDebug,
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
		// Leer tamaño (4 bytes)
		lenBuf := make([]byte, 4)
		_, err := io.ReadFull(reader, lenBuf)
		if err != nil {
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
			return
		}

		s.inbound <- data
	}
}

/**
* inbound
**/
func (s *Client) inbox() {
	for msg := range s.inbound {
		logs.Debugf("recv: %s", msg)
	}
}

/**
* send
**/
func (s *Client) send() {
	for msg := range s.outbound {
		_, err := s.conn.Write(msg)
		if err != nil {
			s.handleDisconnect()
			return
		}

		logs.Debugf("send: %s", msg)
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

	close(s.inbound)
	close(s.outbound)
	close(s.done)
}

/**
* connect
**/
func (s *Client) connect() (net.Conn, error) {
	result, err := net.Dial("tcp", s.Addr)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* Start
**/
func (s *Client) Start() error {
	conn, err := s.connect()
	if err != nil {
		return err
	}

	s.conn = conn
	go s.read()
	go s.inbox()
	go s.send()
	logs.Logf(packageName, msg.MSG_CLIENT_CONNECTED, s.Addr)
	s.Send(TextMessage, "Hola")

	utility.AppWait()
	return nil
}

/**
* Send
* @param tp int, message any
* @return error
**/
func (s *Client) Send(tp int, message any) error {
	if s.Status != Connected {
		conn, err := s.connect()
		if err != nil {
			return err
		}
		s.conn = conn
	}

	msg, err := newMessage(tp, message)
	if err != nil {
		return err
	}

	bt, err := msg.serialize()
	if err != nil {
		return err
	}

	s.outbound <- bt
	if tp == CloseMessage {
		s.handleDisconnect()
		return nil
	}

	return nil
}
