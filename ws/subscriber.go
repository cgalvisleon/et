package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
	"github.com/gorilla/websocket"
)

type WsMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type Subscriber struct {
	hub        *Hub
	Created_at time.Time           `json:"created_at"`
	Id         string              `json:"id"`
	Name       string              `json:"name"`
	Addr       string              `json:"addr"`
	Channels   map[string]*Channel `json:"channels"`
	Queue      map[string]*Queue   `json:"queue"`
	socket     *websocket.Conn
	outbound   chan []byte
	mutex      sync.RWMutex
}

/**
* newSubscriber
* @param *Hub
* @param *websocket.Conn
* @param string
* @param string
* @return *Subscriber
* @return bool
**/
func newSubscriber(hub *Hub, socket *websocket.Conn, id, name string) (*Subscriber, bool) {
	id = utility.GenKey(id)
	return &Subscriber{
		hub:        hub,
		Created_at: timezone.NowTime(),
		Id:         id,
		Name:       name,
		Addr:       socket.RemoteAddr().String(),
		Channels:   make(map[string]*Channel),
		Queue:      make(map[string]*Queue),
		socket:     socket,
		outbound:   make(chan []byte),
	}, true
}

/**
* Describe
* @return et.Json
**/
func (c *Subscriber) describe() et.Json {
	channels := []et.Json{}
	for _, ch := range c.Channels {
		channels = append(channels, ch.describe(1))
	}

	for _, q := range c.Queue {
		channels = append(channels, q.describe(1))
	}

	return et.Json{
		"created_at": strs.FormatDateTime("02/01/2006 03:04:05 PM", c.Created_at),
		"id":         c.Id,
		"name":       c.Name,
		"addr":       c.Addr,
		"count":      len(channels),
		"channels":   channels,
	}
}

/**
* close
**/
func (c *Subscriber) close() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, channel := range c.Channels {
		channel.unsubscribe(c)
	}

	for _, queue := range c.Queue {
		queue.unsubscribe(c)
	}

	c.socket.Close()
	close(c.outbound)
}

/**
* From
* @return et.Json
**/
func (c *Subscriber) From() et.Json {
	return et.Json{
		"id":   c.Id,
		"name": c.Name,
	}
}

/**
* read
**/
func (c *Subscriber) read() {
	defer func() {
		if c.hub != nil {
			c.hub.unregister <- c
			c.socket.Close()
		}
	}()

	for {
		_, message, err := c.socket.ReadMessage()
		if err != nil {
			break
		}

		c.listener(message)
	}
}

/**
* write
**/
func (c *Subscriber) write() {
	for message := range c.outbound {
		c.socket.WriteMessage(websocket.TextMessage, message)
	}

	c.socket.WriteMessage(websocket.CloseMessage, []byte{})
}

/**
* send
* @param message Message
* @return error
**/
func (c *Subscriber) send(message Message) error {
	message.To = c.Id
	msg, err := message.Encode()
	if err != nil {
		return err
	}

	if c.socket == nil {
		return logs.Alertm(ERR_NOT_WS_SERVICE)
	}

	if c.outbound == nil {
		return logs.Alertm(ERR_NOT_WS_SERVICE)
	}

	c.outbound <- msg

	return nil
}

/**
* listener
* @param message []byte
**/
func (c *Subscriber) listener(message []byte) {
	response := func(ok bool, message string) {
		msg := NewMessage(c.hub.From(), et.Json{
			"ok":      ok,
			"message": message,
		}, TpDirect)

		c.send(msg)
	}

	msg, err := DecodeMessage(message)
	if err != nil {
		response(false, err.Error())
		return
	}

	msg.From = c.From()
	switch msg.Tp {
	case TpPing:
		response(true, "pong")
	case TpSetFrom:
		src, err := json.Marshal(msg.Data)
		if err != nil {
			response(false, err.Error())
			return
		}

		data, err := et.Object(string(src))
		if err != nil {
			response(false, err.Error())
			return
		}

		name := data.ValStr("", "name")
		if name == "" {
			c.Name = utility.GetOTP(6)
		}

		response(true, PARAMS_UPDATED)
	case TpSubscribe:
		if msg.Channel == "" {
			response(false, ERR_CHANNEL_EMPTY)
			return
		}

		err := c.hub.Subscribe(c.Id, msg.Channel)
		if err != nil {
			response(false, err.Error())
			return
		}

		response(true, "Subscribed to channel "+msg.Channel)
	case TpQueueSubscribe:
		if msg.Channel == "" {
			response(false, ERR_CHANNEL_EMPTY)
			return
		}

		if msg.Queue == "" {
			response(false, ERR_QUEUE_EMPTY)
			return
		}

		err := c.hub.QueueSubscribe(c.Id, msg.Channel, msg.Queue)
		if err != nil {
			response(false, err.Error())
			return
		}

		response(true, strs.Format(`Subscribe to channel:%s queue:%s`, msg.Channel, msg.Queue))
	case TpStack:
		if msg.Channel == "" {
			response(false, ERR_CHANNEL_EMPTY)
			return
		}

		err := c.hub.Stack(c.Id, msg.Channel)
		if err != nil {
			response(false, err.Error())
			return
		}

		response(true, "Subscribe to channel "+msg.Channel)
	case TpUnsubscribe:
		if msg.Channel == "" {
			response(false, ERR_CHANNEL_EMPTY)
			return
		}
		if msg.Queue == "" {
			msg.Queue = utility.QUEUE_STACK
		}

		err := c.hub.Unsubscribe(c.Id, msg.Channel, msg.Queue)
		if err != nil {
			response(false, err.Error())
			return
		}

		response(true, "Unsubscribed from channel "+msg.Channel)
	case TpPublish:
		if msg.Channel == "" {
			response(false, ERR_CHANNEL_EMPTY)
			return
		}
		if msg.Queue == "" {
			msg.Queue = utility.QUEUE_STACK
		}

		msg.Ignored = []string{c.Id}
		go c.hub.Publish(msg.Channel, msg.Queue, msg, msg.Ignored, c.From())
	case TpDirect:
		msg.From = c.From()
		msg.Ignored = []string{c.Id}
		err := c.hub.SendMessage(msg.To, msg)
		if err != nil {
			response(false, err.Error())
			return
		}
	default:
		response(false, ERR_MESSAGE_UNFORMATTED)
	}

	logs.Logf(ServiceName, "Sender subscriber:%s message: %s", c.Id, msg.ToString())
}
