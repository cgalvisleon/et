package ws

import (
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
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
	Params     *et.Json
	Addr       string
	socket     *websocket.Conn
	Channels   []string
	IsNode     bool
	outbound   chan []byte
	close      bool
	allowed    bool
}

func newClient(hub *Hub, socket *websocket.Conn, id, name string) (*Client, bool) {
	return &Client{
		Created_at: time.Now(),
		hub:        hub,
		Id:         id,
		Name:       name,
		Params: &et.Json{
			"id":   id,
			"name": name,
		},
		socket:   socket,
		Channels: make([]string, 0),
		outbound: make(chan []byte),
		close:    false,
		allowed:  true,
	}, true
}

// Listen a client message
func (c *Client) read() {
	defer func() {
		if c.hub != nil {
			c.hub.unregister <- c
			c.socket.Close()
		}
	}()

	for {
		mt, message, err := c.socket.ReadMessage()
		if err != nil {
			break
		}

		c.listen(mt, message)
	}
}

// Write a message to the client
func (c *Client) write() {
	for {
		select {
		case message, ok := <-c.outbound:
			if !ok {
				c.socket.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.socket.WriteMessage(websocket.TextMessage, message)
		}
	}
}

// Subscribe a client to a channel
func (c *Client) subscribe(channels []string) {
	for _, channel := range channels {
		idx := slices.IndexFunc(c.Channels, func(e string) bool { return e == strs.Lowcase(channel) })
		if idx == -1 {
			c.Channels = append(c.Channels, strs.Lowcase(channel))
		}
	}
}

// Unsubscribe a client from a channel
func (c *Client) unsubscribe(channels []string) {
	for _, channel := range channels {
		idx := slices.IndexFunc(c.Channels, func(e string) bool { return e == strs.Lowcase(channel) })
		if idx != -1 {
			c.Channels = append(c.Channels[:idx], c.Channels[idx+1:]...)
		}
	}
}

// SetAtribs set a value to the client data
func (c *Client) setParams(params et.Json) {
	c.Params = &params
}

// SendMessage send a message to the client
func (c *Client) sendMessage(message Message) error {
	msg, err := message.Encode()
	if err != nil {
		return err
	}

	if c.close {
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

// Close the client websocket connection
func (c *Client) cLose() {
	c.close = true
	c.socket.Close()
	close(c.outbound)
}

// Clear the client channels
func (c *Client) clear() {
	c.unsubscribe(c.Channels)
}
