package ws

import (
	"net/url"
	"os"
	"os/signal"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/gorilla/websocket"
)

type PubSub struct {
	host      string
	socket    *websocket.Conn
	reciveFn  func(messageType int, message []byte)
	connected bool
}

// Create a new client websocket connection
func NewPubSub(host string, reciveFn func(messageType int, message []byte)) *PubSub {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	if host == "" {
		host = ":3300"
	}

	// Connect to the server
	result := &PubSub{
		host:     host,
		reciveFn: reciveFn,
	}

	result.Connect()

	return result
}

// Read messages from the server
func (p *PubSub) read() {
	done := make(chan struct{})

	go func() {
		defer close(done)

		for {
			mt, message, err := p.socket.ReadMessage()
			if err != nil {
				logs.Alertm(err.Error())
				p.connected = false
				return
			}

			if p.reciveFn != nil {
				p.reciveFn(mt, message)
			}
		}
	}()
}

// Send a message to the server
func (p *PubSub) send(message string) error {
	if !p.connected {
		return logs.Alertm(ERR_NOT_CONNECT_WS)
	}

	if p.socket == nil {
		return logs.Alertm(ERR_NOT_CONNECT_WS)
	}

	msg := []byte(message)
	err := p.socket.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		return err
	}

	return nil
}

// Return type server pubsub
func (p *PubSub) Type() string {
	return "ETws"
}

// Check if the client is connected
func (p *PubSub) IsConnected() bool {
	return p.connected
}

// Close the client websocket connection
func (p *PubSub) Close() {
	p.socket.Close()
}

// Connect to the server
func (p *PubSub) Connect() (bool, error) {
	if p.connected {
		return true, nil
	}

	u := url.URL{Scheme: "ws", Host: p.host, Path: "/ws"}
	socket, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return false, err
	}

	p.socket = socket
	p.connected = true
	go p.read()

	return p.connected, nil
}

// Ping the server
func (p *PubSub) Ping() {
	msg := et.Json{
		"type": "ping",
	}

	p.send(msg.ToString())
}

// Set the client parameters
func (p *PubSub) Params(params et.Json) {
	msg := et.Json{
		"type":   "params",
		"params": params,
	}

	p.send(msg.ToString())
}

// Set the client system parameters
func (p *PubSub) System(params et.Json) {
	msg := et.Json{
		"type":   "system",
		"params": params,
	}

	p.send(msg.ToString())
}

// Subscribe to a channel
func (p *PubSub) Subscribe(channel string) {
	msg := et.Json{
		"type":    "subscribe",
		"channel": channel,
	}

	p.send(msg.ToString())
}

// Unsubscribe from a channel
func (p *PubSub) Unsubscribe(channel string) {
	msg := et.Json{
		"type":    "unsubscribe",
		"channel": channel,
	}

	p.send(msg.ToString())
}

// Publish a message to a channel
func (p *PubSub) Publish(channel string, message interface{}) {
	msg := et.Json{
		"type":    "publish",
		"channel": channel,
		"message": message,
	}

	p.send(msg.ToString())
}

// Send a message to the server
func (p *PubSub) SendMessage(clientId string, message string) {
	msg := et.Json{
		"type":      "sendmessage",
		"client_id": clientId,
		"message":   message,
	}

	p.send(msg.ToString())
}
