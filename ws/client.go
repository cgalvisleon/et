package ws

import (
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
	hub      *Hub
	Id       string
	Name     string
	Params   *et.Json
	Addr     string
	socket   *websocket.Conn
	Channels []string
	outbound chan []byte
	close    bool
}

func newClient(hub *Hub, socket *websocket.Conn, id, name string) (*Client, bool) {
	return &Client{
		hub:  hub,
		Id:   id,
		Name: name,
		Params: &et.Json{
			"_id":  id,
			"name": name,
		},
		socket:   socket,
		Channels: make([]string, 0),
		outbound: make(chan []byte),
	}, true
}

// SendMessage send a message to the client
func (c *Client) sendMessage(message []byte) error {
	if c.close {
		return logs.Alertm(ERR_CLIENT_IS_CLOSED)
	}

	if c.socket == nil {
		return logs.Alertm(ERR_NOT_WS_SERVICE)
	}

	if c.outbound == nil {
		return logs.Alertm(ERR_NOT_WS_SERVICE)
	}

	c.outbound <- message

	return nil
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

// SetAtrib set a value to the client data
func (c *Client) SetParam(key string, value interface{}) {
	c.Params.Set(key, value)
}

// SetAtribs set a value to the client data
func (c *Client) SetParams(params et.Json) {
	c.Params = &params
}

// SendMessage send a message to the client
func (c *Client) SendMessage(message interface{}) {
	switch v := message.(type) {
	case string:
		c.sendMessage([]byte(v))
	case []byte:
		c.sendMessage(v)
	case et.Json:
		c.sendMessage([]byte(v.ToString()))
	default:
		c.sendMessage([]byte(message.(string)))
	}
}

// Close the client websocket connection
func (c *Client) Close() {
	c.close = true
	c.socket.Close()
	close(c.outbound)
}

// Clear the client channels
func (c *Client) Clear() {
	c.unsubscribe(c.Channels)
}
