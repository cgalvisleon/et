package ws

import (
	"os"
	"os/signal"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	m "github.com/cgalvisleon/et/message"
	"github.com/cgalvisleon/et/utility"
	"github.com/gorilla/websocket"
)

type PubSub struct {
	host      string
	ClientId  string
	Name      string
	socket    *websocket.Conn
	channels  map[string]func(m.Message)
	from      et.Json
	connected bool
}

// Create a new client websocket connection
func NewPubSub() (*PubSub, error) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	host := envar.GetStr("", "WS_HOST")
	if host == "" {
		host = ":3300"
	}

	clientId := utility.UUID()
	name := "Anonimo"
	result := &PubSub{
		host:     host,
		ClientId: clientId,
		Name:     name,
		from: et.Json{
			"id":   clientId,
			"name": name,
		},
	}

	_, err := result.Connect()
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Read messages from the server
func (p *PubSub) read() {
	done := make(chan struct{})

	go func() {
		defer close(done)

		for {
			_, data, err := p.socket.ReadMessage()
			if err != nil {
				logs.Alert(err)
				p.connected = false
				return
			}

			msg, err := DecodeMessage(data)
			if err != nil {
				logs.Alert(err)
				return
			}

			f, ok := p.channels[msg.Channel]
			if ok {
				f(msg)
			}
		}
	}()
}

// Send a message to the server
func (p *PubSub) send(message Message) error {
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

	scheme := envar.GetStr("", "WS_SCHEME")
	if scheme == "" {
		scheme = "ws"
	}

	socket, err := ConnectWs(p.host, scheme, p.ClientId, p.Name)
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
	msg := NewMessage(p.from, et.Json{}, m.TpPing)

	p.send(msg)
}

// Set the client parameters
func (p *PubSub) Params(params et.Json) error {
	if params.Empty() {
		return logs.Alertm(ERR_PARAM_NOT_FOUND)
	}

	name := params.ValStr("", "name")
	if name != "" {
		p.Name = name
	}

	params.Set("id", p.ClientId)
	params.Set("name", p.Name)
	p.from = params

	return nil
}

// Subscribe to a channel
func (p *PubSub) Subscribe(channel string, reciveFn func(m.Message)) {
	p.channels[channel] = reciveFn

	msg := NewMessage(p.from, et.Json{}, m.TpSubscribe)
	msg.Channel = channel

	p.send(msg)
}

// Unsubscribe from a channel
func (p *PubSub) Unsubscribe(channel string) {
	delete(p.channels, channel)

	msg := NewMessage(p.from, et.Json{}, m.TpUnsubscribe)
	msg.Channel = channel

	p.send(msg)
}

// Subscribe to a channel type fisrt, so send message to first client
func (p *PubSub) Stack(channel string, reciveFn func(m.Message)) {
	p.channels[channel] = reciveFn

	msg := NewMessage(p.from, et.Json{}, m.TpStack)
	msg.Channel = channel

	p.send(msg)
}

// Publish a message to a channel
func (p *PubSub) Publish(channel string, message interface{}) {
	msg := NewMessage(p.from, message, m.TpPublish)
	msg.Ignored = []string{p.ClientId}
	msg.Channel = channel

	p.send(msg)
}

// Send a message to the server
func (p *PubSub) SendMessage(clientId string, message interface{}) error {
	msg := NewMessage(p.from, message, m.TpDirect)
	msg.Ignored = []string{p.ClientId}
	msg.to = clientId

	return p.send(msg)
}
