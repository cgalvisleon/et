package jws

import (
	"context"
	"encoding/json"
	"slices"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/gorilla/websocket"
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
	messageType int
	message     []byte
}

type Client struct {
	Created_at time.Time       `json:"created_at"`
	Name       string          `json:"name"`
	Addr       string          `json:"addr"`
	Status     Status          `json:"status"`
	Channels   []string        `json:"channels"`
	socket     *websocket.Conn `json:"-"`
	outbound   chan Outbound   `json:"-"`
	done       chan struct{}   `json:"-"`
	closed     bool            `json:"-"`
	mu         sync.RWMutex    `json:"-"`
	hub        *Hub            `json:"-"`
	ctx        context.Context `json:"-"`
}

/**
* newSubscriber
* @param hub *Hub, ctx context.Context, username string, socket *websocket.Conn
* @return *Client
**/
func newSubscriber(hub *Hub, ctx context.Context, username string, socket *websocket.Conn) *Client {
	return &Client{
		Created_at: timezone.Now(),
		Status:     Pending,
		Name:       username,
		Addr:       socket.RemoteAddr().String(),
		Channels:   []string{},
		socket:     socket,
		outbound:   make(chan Outbound),
		done:       make(chan struct{}),
		hub:        hub,
		ctx:        ctx,
	}
}

/**
* ToJson
* @return et.Json
**/
func (s *Client) ToJson() et.Json {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return et.Json{
		"created_at": s.Created_at,
		"name":       s.Name,
		"addr":       s.Addr,
		"status":     s.Status,
		"channels":   append([]string{}, s.Channels...),
	}
}

/**
* rebind retires the client's current connection (if any) and installs a new
* socket, returning the fresh outbound/done pair for the caller to start
* read()/write() goroutines with. Used when a client reconnects under the
* same username so the previous connection's goroutines are cleanly retired
* instead of leaking.
* @param socket *websocket.Conn
* @return (chan Outbound, chan struct{})
**/
func (s *Client) rebind(socket *websocket.Conn) (chan Outbound, chan struct{}) {
	s.mu.Lock()
	oldSocket := s.socket
	oldDone := s.done
	wasClosed := s.closed

	newOutbound := make(chan Outbound)
	newDone := make(chan struct{})
	s.Addr = socket.RemoteAddr().String()
	s.socket = socket
	s.outbound = newOutbound
	s.done = newDone
	s.closed = false
	s.mu.Unlock()

	if !wasClosed {
		close(oldDone)
		oldSocket.Close()
	}

	return newOutbound, newDone
}

/**
* retire permanently shuts down the client's current connection: it closes
* the done channel (signals read()/write() to stop) and the socket. Safe to
* call more than once.
**/
func (s *Client) retire() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	done := s.done
	socket := s.socket
	s.mu.Unlock()

	close(done)
	socket.Close()
}

/**
* read
* @param socket *websocket.Conn, done chan struct{}
**/
func (s *Client) read(socket *websocket.Conn, done chan struct{}) {
	for {
		_, message, err := socket.ReadMessage()
		if err != nil {
			select {
			case <-done:
				// this connection was already retired (disconnect or reconnect); nothing to do.
			default:
				s.hub.unregister <- s
			}
			return
		}

		s.listener(message)
	}
}

/**
* write
* @param socket *websocket.Conn, outbound chan Outbound, done chan struct{}
**/
func (s *Client) write(socket *websocket.Conn, outbound chan Outbound, done chan struct{}) {
	for {
		select {
		case message := <-outbound:
			if err := socket.WriteMessage(message.messageType, message.message); err != nil {
				logs.Alertf("jws: write error to %s: %v", s.Name, err)
				select {
				case <-done:
				default:
					s.hub.unregister <- s
				}
				socket.Close()
				return
			}
		case <-done:
			socket.WriteMessage(CloseMessage, []byte{})
			socket.Close()
			return
		}
	}
}

/**
* listener
* @param message []byte
**/
func (s *Client) listener(message []byte) {
	for _, fn := range s.hub.onListener {
		fn(s, message)
	}
}

/**
* send
* @param tp int, bt []byte
**/
func (s *Client) Send(tp int, bt []byte) {
	s.mu.RLock()
	outbound := s.outbound
	done := s.done
	s.mu.RUnlock()

	select {
	case outbound <- Outbound{messageType: tp, message: bt}:
	case <-done:
		// client has been retired or is mid-reconnect; drop the message.
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
* addChannel
* @param channel string
**/
func (s *Client) addChannel(channel string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := slices.IndexFunc(s.Channels, func(c string) bool {
		return c == channel
	})
	if idx != -1 {
		return
	}
	s.Channels = append(s.Channels, channel)
}

/**
* removeChannel
* @param channel string
**/
func (s *Client) removeChannel(channel string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := slices.IndexFunc(s.Channels, func(c string) bool {
		return c == channel
	})
	if idx == -1 {
		return
	}

	s.Channels = append(s.Channels[:idx], s.Channels[idx+1:]...)
}
