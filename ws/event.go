package ws

import (
	"net/url"
	"os"
	"os/signal"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/gorilla/websocket"
)

type Event struct {
	host      string
	socket    *websocket.Conn
	reciveFn  func(messageType int, message []byte)
	connected bool
}

// Create a new client websocket connection
func NewPubSub(host string, reciveFn func(messageType int, message []byte)) *Event {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	if host == "" {
		host = ":3300"
	}

	// Connect to the server
	result := &Event{
		host:     host,
		reciveFn: reciveFn,
	}

	result.Connect()

	return result
}

// Read messages from the server
func (e *Event) read() {
	done := make(chan struct{})

	go func() {
		defer close(done)

		for {
			mt, message, err := e.socket.ReadMessage()
			if err != nil {
				logs.Alertm(err.Error())
				e.connected = false
				return
			}

			if e.reciveFn != nil {
				e.reciveFn(mt, message)
			}
		}
	}()
}

// Send a message to the server
func (e *Event) send(message string) error {
	if !e.connected {
		return logs.Alertm(ERR_NOT_CONNECT_WS)
	}

	if e.socket == nil {
		return logs.Alertm(ERR_NOT_CONNECT_WS)
	}

	msg := []byte(message)
	err := e.socket.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		return err
	}

	return nil
}

// Check if the client is connected
func (e *Event) IsConnected() bool {
	return e.connected
}

// Close the client websocket connection
func (e *Event) Close() {
	e.socket.Close()
}

// Connect to the server
func (e *Event) Connect() (bool, error) {
	if e.connected {
		return true, nil
	}

	u := url.URL{Scheme: "ws", Host: e.host, Path: "/ws"}
	socket, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return false, err
	}

	e.socket = socket
	e.connected = true
	go e.read()

	return e.connected, nil
}

// Ping the server
func (e *Event) Ping() {
	msg := et.Json{
		"type": "ping",
	}

	e.send(msg.ToString())
}

// Set the client parameters
func (e *Event) Params(params et.Json) {
	msg := et.Json{
		"type":   "params",
		"params": params,
	}

	e.send(msg.ToString())
}

// Set the client system parameters
func (e *Event) System(params et.Json) {
	msg := et.Json{
		"type":   "system",
		"params": params,
	}

	e.send(msg.ToString())
}

// Subscribe to a channel
func (e *Event) Subscribe(channel string) {
	msg := et.Json{
		"type":    "subscribe",
		"channel": channel,
	}

	e.send(msg.ToString())
}

// Unsubscribe from a channel
func (e *Event) Unsubscribe(channel string) {
	msg := et.Json{
		"type":    "unsubscribe",
		"channel": channel,
	}

	e.send(msg.ToString())
}

// Publish a message to a channel
func (e *Event) Publish(channel string, message interface{}) {
	msg := et.Json{
		"type":    "publish",
		"channel": channel,
		"message": message,
	}

	e.send(msg.ToString())
}

// Send a message to the server
func (e *Event) SendMessage(clientId string, message string) {
	msg := et.Json{
		"type":      "sendmessage",
		"client_id": clientId,
		"message":   message,
	}

	e.send(msg.ToString())
}
