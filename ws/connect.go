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

	u := url.URL{Scheme: "ws", Host: host, Path: "/ws"}

	// Connect to the server
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil
	}

	result := &ClientWS{
		socket:    c,
		reciveFn:  reciveFn,
		connected: true,
	}

	go result.read()

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
				logs.Alertf(`Error al leer mensaje:%v`, err)
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

// Send a message to the server
func (s *ClientWS) SendMessage(message string) error {
	if !s.connected {
		return logs.Log(ERR_NOT_CONNECT_WS)
	}

	msg := []byte(message)
	err := s.socket.WriteMessage(websocket.TextMessage, msg)
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
