package ws

import (
	"sync"

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
	channels []*Channel
	outbound chan []byte
	mutex    *sync.Mutex
}

func newClient(hub *Hub, socket *websocket.Conn, id, name string) (*Client, bool) {
	return &Client{
		hub:      hub,
		Id:       id,
		Name:     name,
		socket:   socket,
		channels: make([]*Channel, 0),
		outbound: make(chan []byte),
		mutex:    &sync.Mutex{},
	}, true
}

func (c *Client) read() {
	defer func() {
		c.hub.unregister <- c
	}()

	for {
		mt, message, err := c.socket.ReadMessage()
		if err != nil {
			break
		}

		if c.hub != nil {
			c.hub.listen(c, mt, message)
		}
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
	c.mutex.Lock()
	c.socket.Close()
	close(c.outbound)
	c.mutex.Unlock()
}

func (c *Client) sendMessage(message []byte) {
	if c.socket == nil {
		return
	}

	if c.outbound == nil {
		return
	}

	c.outbound <- message
}

func (c *Client) Channels() []*Channel {
	return c.channels
}

func (c *Client) Subscribe(channel string) {
	idx := slices.IndexFunc(c.channels, func(e *Channel) bool { return e.Name == channel })
	if idx == -1 {
		_channel := NewChanel(c.hub, channel)
		_channel.Subscribers = append(_channel.Subscribers, c)
		c.channels = append(c.channels, _channel)
	} else {
		_channel := c.channels[idx]

		idxS := slices.IndexFunc(_channel.Subscribers, func(e *Client) bool { return e.Id == c.Id })
		if idxS == -1 {
			_channel.Subscribers = append(_channel.Subscribers, c)
		}
	}
}

func (c *Client) Unsubscribe(channel string) {
	idx := slices.IndexFunc(c.channels, func(e *Channel) bool { return e.Name == channel })
	if idx == -1 {
		return
	}

	c.channels[idx].Unsubcribe(c.Id)
	c.channels = append(c.channels[:idx], c.channels[idx+1:]...)
}

func (c *Client) Subscribers(channels []string) {
	for _, channel := range channels {
		c.Subscribe(channel)
	}
}

func (c *Client) Unsubscribers(channels []string) {
	for _, channel := range channels {
		c.Unsubscribe(channel)
	}
}

func (c *Client) Clear() {
	for _, channel := range c.channels {
		channel.Unsubcribe(c.Id)
	}
}
