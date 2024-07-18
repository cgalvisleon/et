package nats

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/js"
	"github.com/cgalvisleon/et/logs"
	m "github.com/cgalvisleon/et/message"
	"github.com/cgalvisleon/et/utility"
	"github.com/nats-io/nats.go"
)

type PubSub struct {
	host         string
	ClientId     string
	Name         string
	from         js.Json
	subscription map[string]*nats.Subscription
	channels     map[string]func(m.Message)
	connected    bool
}

// Create a new client websocket connection
func NewPubSub() (*PubSub, error) {
	host := envar.GetStr("", "NATS_HOST")
	if host == "" {
		host = "localhost:4222"
	}

	clientId := utility.UUID()
	name := "Anonimo"
	result := &PubSub{
		host:     host,
		ClientId: clientId,
		Name:     name,
		from: js.Json{
			"id":   clientId,
			"name": name,
		},
		subscription: make(map[string]*nats.Subscription),
		channels:     make(map[string]func(m.Message)),
	}

	_, err := result.Connect()
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Subscribe to a channel
func (p *PubSub) subscribe(channel string, f func(m.Message)) error {
	if conn == nil {
		return logs.Alertm(ERR_NOT_CONNECT)
	}

	msg := Message{
		Channel: channel,
	}
	subscription, err := conn.conn.Subscribe(
		msg.Channel,
		func(m *nats.Msg) {
			f(msg)
		},
	)
	if err != nil {
		return err
	}

	p.subscription[channel] = subscription

	return nil
}

// Subscribe to a channel
func (p *PubSub) stack(channel string, f func(m.Message)) error {
	if conn == nil {
		return logs.Alertm(ERR_NOT_CONNECT)
	}

	msg := Message{
		Channel: channel,
	}
	subscription, err := conn.conn.QueueSubscribe(
		msg.Channel,
		"workers",
		func(m *nats.Msg) {
			f(msg)
		},
	)
	if err != nil {
		return err
	}

	p.subscription[channel] = subscription

	return nil
}

// Send a message to the server
func (p *PubSub) send(subj string, message Message) error {
	if conn == nil {
		return logs.Alertm(ERR_NOT_CONNECT)
	}

	msg, err := message.Encode()
	if err != nil {
		return err
	}

	err = conn.conn.Publish(subj, msg)
	if err != nil {
		return err
	}

	return nil
}

// Return type server pubsub
func (p *PubSub) Type() string {
	return "Nats"
}

// Check if the client is connected
func (p *PubSub) IsConnected() bool {
	if conn == nil {
		return false
	}

	return p.connected
}

// Close the client websocket connection
func (p *PubSub) Close() {
	if conn != nil {
		conn.conn.Close()
	}

	for _, sub := range p.subscription {
		sub.Unsubscribe()
	}
}

// Connect to the server
func (p *PubSub) Connect() (bool, error) {
	if p.connected {
		return true, nil
	}

	_, err := Load()
	if err != nil {
		return false, err
	}

	p.connected = true

	return p.connected, nil
}

// Ping the server
func (p *PubSub) Ping() {
	msg := NewMessage(p.from, js.Json{
		"ok":      true,
		"message": "pong",
	})
	msg.Tp = m.TpPing

	p.send(p.ClientId, msg)
}

// Set the client parameters
func (p *PubSub) Params(params js.Json) error {
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
	p.subscribe(channel, reciveFn)
	msg := NewMessage(p.from, js.Json{
		"ok":      true,
		"message": "Subscribed to channel " + channel,
	})

	p.send(p.ClientId, msg)
}

// Subscribe to a channel type fisrt, so send message to first client
func (p *PubSub) Stack(channel string, reciveFn func(m.Message)) {
	p.channels[channel] = reciveFn
	p.stack(channel, reciveFn)
	msg := NewMessage(p.from, js.Json{
		"ok":      true,
		"message": "Stacked to channel " + channel,
	})

	p.send(p.ClientId, msg)
}

// Unsubscribe from a channel
func (p *PubSub) Unsubscribe(channel string) {
	delete(p.channels, channel)
	delete(p.subscription, channel)
	msg := NewMessage(p.from, js.Json{
		"ok":      true,
		"message": "Unsubscribed from channel " + channel,
	})

	p.send(p.ClientId, msg)
}

// Publish a message to a channel
func (p *PubSub) Publish(channel string, message interface{}) {
	msg := NewMessage(p.from, message)
	msg.Tp = m.TpPublish
	msg.Channel = channel
	p.send(msg.Channel, msg)

	msg = NewMessage(p.from, js.Json{
		"ok":      true,
		"message": "Message published to " + channel,
	})

	p.send(p.ClientId, msg)
}

// Send a message to the server
func (p *PubSub) SendMessage(clientId string, message interface{}) error {
	msg := NewMessage(p.from, message)
	msg.to = clientId
	msg.Tp = m.TpDirect

	return p.send(msg.to, msg)
}
