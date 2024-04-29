package ws

import (
	"net/url"
	"os"
	"os/signal"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/pubsub"
	"github.com/gorilla/websocket"
)

type PubSub struct {
	host      string
	socket    *websocket.Conn
	reciveFn  func(Message)
	channels  map[string]func(Message)
	from      et.Json
	connected bool
}

// Create a new client websocket connection
func NewPubSub(host string, from et.Json, recivefn func(Message)) *PubSub {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	if host == "" {
		host = ":3300"
	}

	// Connect to the server
	result := &PubSub{
		host:     host,
		from:     from,
		reciveFn: recivefn,
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
			_, data, err := p.socket.ReadMessage()
			if err != nil {
				logs.Alertm(err.Error())
				p.connected = false
				return
			}

			msg, err := DecodeMessage(data)
			if err != nil {
				logs.Alertm(err.Error())
				return
			}

			f, ok := p.channels[msg.Channel]
			if ok {
				f(msg)
			} else {
				p.reciveFn(msg)
			}
		}
	}()
}

// Send a message to the server
func (p *PubSub) send(message Message) error {
	if !p.connected {
		return logs.Alertm(ERR_NOT_CONNECT_WS)
	}

	if p.socket == nil {
		return logs.Alertm(ERR_NOT_CONNECT_WS)
	}

	msg, err := message.Encode()
	if err != nil {
		return err
	}

	err = p.socket.WriteMessage(websocket.TextMessage, msg)
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
	msg := NewMessage(p.from, et.Json{}, et.Json{})
	msg.Tp = pubsub.TpPing
	p.send(msg)
}

// Set the client parameters
func (p *PubSub) Params(params et.Json) {
	msg := NewMessage(p.from, et.Json{}, params)
	msg.Tp = pubsub.TpParams
	p.send(msg)
}

// Set the client system parameters
func (p *PubSub) System(params et.Json) {
	msg := NewMessage(p.from, et.Json{}, params)
	msg.Tp = pubsub.TpSystem
	p.send(msg)
}

// Subscribe to a channel
func (p *PubSub) Subscribe(channel string) {
	msg := NewMessage(p.from, et.Json{}, et.Json{})
	msg.Tp = pubsub.TpSubscribe
	msg.Channel = channel
	p.send(msg)
}

// Unsubscribe from a channel
func (p *PubSub) Unsubscribe(channel string) {
	msg := NewMessage(p.from, et.Json{}, et.Json{})
	msg.Tp = pubsub.TpUnsubscribe
	msg.Channel = channel
	p.send(msg)
}

// Publish a message to a channel
func (p *PubSub) Publish(channel string, message interface{}) {
	msg := NewMessage(p.from, et.Json{}, et.Json{
		"channel": channel,
		"message": message,
	})
	msg.Tp = pubsub.TpPublish
	msg.Channel = channel
	p.send(msg)
}

// Send a message to the server
func (p *PubSub) SendMessage(to et.Json, message interface{}) {
	msg := NewMessage(p.from, to, message)
	msg.Tp = pubsub.TpDirect
	p.send(msg)
}
