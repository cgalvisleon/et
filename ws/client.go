package ws

import (
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
	Addr     string
	socket   *websocket.Conn
	channels []string
	outbound chan []byte
}

func newClient(hub *Hub, socket *websocket.Conn, id, name string) (*Client, bool) {
	return &Client{
		hub:      hub,
		Id:       id,
		Name:     name,
		socket:   socket,
		channels: make([]string, 0),
		outbound: make(chan []byte),
	}, true
}

// SendMessage send a message to the client
func (c *Client) sendMessage(message []byte) bool {
	if c.socket == nil {
		return false
	}

	if c.outbound == nil {
		return false
	}

	c.outbound <- message

	return true
}

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

func (c *Client) Close() {
	c.socket.Close()
	close(c.outbound)
}

func (c *Client) subscribe(channels []string) {
	for _, channel := range channels {
		idx := slices.IndexFunc(c.channels, func(e string) bool { return e == strs.Lowcase(channel) })
		if idx == -1 {
			c.channels = append(c.channels, strs.Lowcase(channel))
		}
	}
}

func (c *Client) unsubscribe(channels []string) bool {
	var result bool

	ch := func() {
		if !result {
			result = true
		}
	}

	for _, channel := range channels {
		idx := slices.IndexFunc(c.channels, func(e string) bool { return e == strs.Lowcase(channel) })
		if idx != -1 {
			c.channels = append(c.channels[:idx], c.channels[idx+1:]...)
			ch()
		}
	}

	return result
}

func (c *Client) Channels() []string {
	return c.channels
}

func (c *Client) Clear() {
	c.unsubscribe(c.channels)
}
