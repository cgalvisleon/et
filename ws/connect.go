package ws

import (
	"net/url"
	"os"
	"os/signal"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/gorilla/websocket"
)

type ClientWS struct {
	host      string
	socket    *websocket.Conn
	reciveFn  func(messageType int, message []byte)
	connected bool
}

// Create a new client websocket connection
func NewClientWS(host string, reciveFn func(messageType int, message []byte)) *ClientWS {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	if host == "" {
		host = ":3300"
	}

	// Connect to the server
	result := &ClientWS{
		host:     host,
		reciveFn: reciveFn,
	}

	result.Connect()

	return result
}

func (c *ClientWS) IsConnected() bool {
	return c.connected
}

// Read messages from the server
func (c *ClientWS) read() {
	done := make(chan struct{})

	go func() {
		defer close(done)

		for {
			mt, message, err := c.socket.ReadMessage()
			if err != nil {
				logs.Alertm(err.Error())
				c.connected = false
				return
			}

			if c.reciveFn != nil {
				c.reciveFn(mt, message)
			}
		}
	}()
}

// Close the client websocket connection
func (c *ClientWS) Close() {
	c.socket.Close()
}

func (c *ClientWS) Connect() (bool, error) {
	if c.connected {
		return true, nil
	}

	u := url.URL{Scheme: "ws", Host: c.host, Path: "/ws"}
	socket, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return false, err
	}

	c.socket = socket
	c.connected = true
	go c.read()

	return c.connected, nil
}

// Send a message to the server
func (c *ClientWS) SendMessage(message string) error {
	if !c.connected {
		return logs.Alertm(ERR_NOT_CONNECT_WS)
	}

	if c.socket == nil {
		return logs.Alertm(ERR_NOT_CONNECT_WS)
	}

	msg := []byte(message)
	err := c.socket.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		return err
	}

	return nil
}

func (s *ClientWS) Subscribe(channel string) {
	msg := et.Json{
		"type":    "subscribe",
		"channel": channel,
	}

	s.SendMessage(msg.ToString())
}
