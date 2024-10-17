package ws

import (
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/gorilla/websocket"
	"golang.org/x/exp/slices"
)

type WsMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type Client struct {
	Created_at time.Time
	hub        *Hub
	Id         string
	Name       string
	Addr       string
	socket     *websocket.Conn
	Channels   []string
	outbound   chan []byte
	closed     bool
	allowed    bool
}

/**
* NewClient
* @param *Hub
* @param *websocket.Conn
* @param string
* @param string
* @return *Client
* @return bool
**/
func newClient(hub *Hub, socket *websocket.Conn, id, name string) (*Client, bool) {
	return &Client{
		Created_at: timezone.NowTime(),
		hub:        hub,
		Id:         id,
		Name:       name,
		socket:     socket,
		Channels:   make([]string, 0),
		outbound:   make(chan []byte),
		closed:     false,
		allowed:    true,
	}, true
}

/**
* Describe
* @return et.Json
**/
func (c *Client) Describe() et.Json {
	result, err := et.Object(c)
	if err != nil {
		return et.Json{}
	}

	return result
}

/**
* read
**/
func (c *Client) read() {
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

		c.listen(message)
	}
}

/**
* write
**/
func (c *Client) write() {
	for message := range c.outbound {
		c.socket.WriteMessage(websocket.TextMessage, message)
	}
	c.socket.WriteMessage(websocket.CloseMessage, []byte{})
}

/**
* subscribe a client to a channel
**/
func (c *Client) subscribe(channels []string) {
	for _, channel := range channels {
		idx := slices.IndexFunc(c.Channels, func(e string) bool { return e == strs.Lowcase(channel) })
		if idx == -1 {
			c.Channels = append(c.Channels, strs.Lowcase(channel))
		}
	}
}

/**
* unsubscribe a client from a channel
**/
func (c *Client) unsubscribe(channels []string) {
	for _, channel := range channels {
		idx := slices.IndexFunc(c.Channels, func(e string) bool { return e == strs.Lowcase(channel) })
		if idx != -1 {
			c.Channels = append(c.Channels[:idx], c.Channels[idx+1:]...)
		}
	}
}

/**
* sendMessage
* @param Message
* @return error
**/
func (c *Client) sendMessage(message Message) error {
	msg, err := message.Encode()
	if err != nil {
		return err
	}

	if c.closed {
		return logs.Alertm(ERR_CLIENT_IS_CLOSED)
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
* clear
**/
func (c *Client) clear() {
	c.unsubscribe(c.Channels)
}

/**
* close
**/
func (c *Client) close() {
	c.closed = true
	c.socket.Close()
	close(c.outbound)
}

/**
* From
* @return et.Json
**/
func (c *Client) From() et.Json {
	return et.Json{
		"id":   c.Id,
		"name": c.Name,
	}
}

/**
* listen
* @param []byte
**/
func (c *Client) listen(message []byte) {
	response := func(ok bool, message string) {
		msg := NewMessage(c.hub.From(), et.Json{
			"ok":      ok,
			"message": message,
		}, TpDirect)
		c.sendMessage(msg)
	}

	msg, err := DecodeMessage(message)
	if err != nil {
		response(false, err.Error())
		return
	}

	tp := msg.Tp
	switch tp {
	case TpPing:
		response(true, "pong")
	case TpSetFrom:
		data, err := et.Object(msg.Data)
		if err != nil {
			response(false, err.Error())
			return
		}

		name := data.ValStr("", "name")
		if name != "" {
			c.Name = name
		}

		response(true, PARAMS_UPDATED)
	case TpSubscribe:
		channel := msg.Channel
		if channel == "" {
			response(false, ERR_CHANNEL_EMPTY)
			return
		}

		err := c.hub.Subscribe(c.Id, channel)
		if err != nil {
			response(false, err.Error())
			return
		}

		response(true, "Subscribed to channel "+channel)
	case TpUnsubscribe:
		channel := msg.Channel
		if channel == "" {
			response(false, ERR_CHANNEL_EMPTY)
			return
		}

		err := c.hub.Unsubscribe(c.Id, channel)
		if err != nil {
			response(false, err.Error())
			return
		}

		response(true, "Unsubscribed from channel "+channel)
	case TpQueue:
		channel := msg.Channel
		if channel == "" {
			response(false, ERR_CHANNEL_EMPTY)
			return
		}

		queue := msg.Queue
		if queue == "" {
			response(false, ERR_QUEUE_EMPTY)
		}

		err := c.hub.Queue(c.Id, channel, queue)
		if err != nil {
			response(false, err.Error())
			return
		}

		response(true, "Subscribe to channel "+channel)
	case TpPublish:
		channel := msg.Channel
		if channel == "" {
			response(false, ERR_CHANNEL_EMPTY)
			return
		}

		go c.hub.Publish(channel, msg, []string{c.Id}, c.From())
	case TpDirect:
		clientId := msg.To

		msg.From = c.From()
		err := c.hub.SendMessage(clientId, msg)
		if err != nil {
			response(false, err.Error())
			return
		}
	default:
		response(false, ERR_MESSAGE_UNFORMATTED)
	}

	logs.Logf("Websocket", "Client %s message: %s", c.Id, msg.ToString())
}
