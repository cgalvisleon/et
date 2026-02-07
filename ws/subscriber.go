package ws

import (
	"context"
	"encoding/json"
	"slices"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
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
	mutex      sync.RWMutex    `json:"-"`
	hub        *Hub            `json:"-"`
	ctx        context.Context `json:"-"`
}

/**
* newSubscriber
* @param name string, socket *websocket.Conn
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
		mutex:      sync.RWMutex{},
		hub:        hub,
		ctx:        ctx,
	}
}

/**
* ToJson
* @return et.Json
**/
func (s *Client) ToJson() et.Json {
	return et.Json{
		"created_at": s.Created_at,
		"name":       s.Name,
		"addr":       s.Addr,
		"status":     s.Status,
		"channels":   s.Channels,
	}
}

/**
* read
**/
func (s *Client) read() {
	for {
		_, message, err := s.socket.ReadMessage()
		if err != nil {
			s.hub.unregister <- s
			break
		}

		s.listener(message)
	}
}

/**
* write
**/
func (s *Client) write() {
	for message := range s.outbound {
		s.socket.WriteMessage(TextMessage, message.message)
	}

	s.socket.WriteMessage(CloseMessage, []byte{})
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
	s.outbound <- Outbound{
		messageType: tp,
		message:     bt,
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
	idx := slices.IndexFunc(s.Channels, func(c string) bool {
		return c == channel
	})
	if idx == -1 {
		return
	}

	s.Channels = append(s.Channels[:idx], s.Channels[idx+1:]...)
}
