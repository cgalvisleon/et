package nats

import (
	"slices"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/pubsub"
	"github.com/cgalvisleon/et/utility"
	"github.com/nats-io/nats.go"
)

type PubSub struct {
	host         string
	ClientId     string
	Name         string
	from         et.Json
	conn         *nats.Conn
	subscription map[string]*nats.Subscription
	reciveFn     func(pubsub.Message)
	channels     map[string]func(pubsub.Message)
	connected    bool
}

// Create a new client websocket connection
func NewPubSub(host, clientId, name string, reciveFn func(pubsub.Message)) *PubSub {
	if host == "" {
		host = "localhost:4222"
	}

	if !slices.Contains([]string{"", "-1", "new"}, clientId) {
		clientId = utility.UUID()
	}

	if !slices.Contains([]string{"", "-1"}, name) {
		name = "Anonimo"
	}

	result := &PubSub{
		host:     host,
		ClientId: clientId,
		Name:     name,
		from: et.Json{
			"id":   clientId,
			"name": name,
		},
		subscription: make(map[string]*nats.Subscription),
		reciveFn:     reciveFn,
	}

	result.Connect()

	return result
}

// Subscribe to a channel
func (p PubSub) subscribe(channel string, f func(pubsub.Message)) error {
	if p.conn == nil {
		return logs.Alertm(ERR_NOT_CONNECT_NATS)
	}

	msg := Message{
		Channel: channel,
	}
	subscription, err := p.conn.Subscribe(
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

func (p PubSub) unsubscribe(channel string, f func(pubsub.Message)) error {
	if p.conn == nil {
		return logs.Alertm(ERR_NOT_CONNECT_NATS)
	}

	msg := Message{
		Channel: channel,
	}
	subscription, err := p.conn.Subscribe(
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

// Send a message to the server
func (p PubSub) send(subj string, message Message) error {
	if p.conn == nil {
		return logs.Alertm(ERR_NOT_CONNECT_NATS)
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
func (p PubSub) Type() string {
	return "Nats"
}

// Check if the client is connected
func (p PubSub) IsConnected() bool {
	if conn == nil {
		return false
	}

	return conn.connected
}

// Close the client websocket connection
func (p PubSub) Close() {
	if p.conn != nil {
		p.conn.Close()
	}

	for _, sub := range p.subscription {
		sub.Unsubscribe()
	}
}

// Connect to the server
func (p PubSub) Connect() (bool, error) {
	if p.connected {
		return true, nil
	}

	conn, err := connect(p.host)
	if err != nil {
		return false, err
	}

	p.conn = conn
	err = p.subscribe(p.ClientId, p.reciveFn)
	if err != nil {
		return false, err
	}

	p.connected = true

	return p.connected, nil
}

// Ping the server
func (p PubSub) Ping() {
	msg := NewMessage(et.Json{}, p.from, et.Json{
		"ok":      true,
		"message": "pong",
	})
	msg.Tp = pubsub.TpPing
	p.send(p.ClientId, msg)
}

// Set the client parameters
func (p PubSub) Params(params et.Json) error {
	if params.Emptyt() {
		return logs.Alertm(ERR_PARAM_NOT_FOUND)
	}

	name := params.ValStr("", "name")
	if name != "" {
		p.Name = name
	}

	params.Set("id", p.ClientId)
	params.Set("name", p.Name)
	p.from = params
	msg := NewMessage(et.Json{}, p.from, et.Json{
		"ok":      true,
		"message": PARAMS_UPDATED,
	})

	return p.send(p.ClientId, msg)
}

// Set the client system parameters
func (p PubSub) System(params et.Json) error {
	msg := NewMessage(et.Json{}, p.from, et.Json{
		"ok":      true,
		"message": PARAMS_UPDATED,
	})
	return p.send(p.ClientId, msg)
}

// Subscribe to a channel
func (p PubSub) Subscribe(channel string, reciveFn func(pubsub.Message)) {
	p.channels[channel] = reciveFn
	p.subscribe(channel, reciveFn)

	msg := NewMessage(et.Json{}, p.from, et.Json{
		"ok":      true,
		"message": "Subscribed to channel " + channel,
	})
	p.send(p.ClientId, msg)
}

// Unsubscribe from a channel
func (p PubSub) Unsubscribe(channel string) {
	delete(p.channels, channel)
	delete(p.subscription, channel)

	msg := NewMessage(et.Json{}, p.from, et.Json{
		"ok":      true,
		"message": "Unsubscribed from channel " + channel,
	})
	p.send(p.ClientId, msg)
}

// Publish a message to a channel
func (p *PubSub) Publish(channel string, message interface{}) {
	msg := NewMessage(p.from, et.Json{}, message)
	msg.Tp = pubsub.TpPublish
	msg.Channel = channel
	p.send(msg.Channel, msg)

	msg = NewMessage(et.Json{}, p.from, et.Json{
		"ok":      true,
		"message": "Message published to " + channel,
	})
	p.send(p.ClientId, msg)
}

// Send a message to the server
func (p *PubSub) SendMessage(to et.Json, message interface{}) error {
	clientId := to.ValStr("-1", "client_id")
	if clientId == "-1" {
		return logs.Alertm(ERR_CLIENT_ID_EMPTY)
	}

	msg := NewMessage(p.from, to, message)
	msg.Tp = pubsub.TpDirect
	err := p.send(clientId, msg)
	if err != nil {
		return err
	}

	msg = NewMessage(et.Json{}, p.from, et.Json{
		"ok":      true,
		"message": "Message sent to " + clientId,
	})
	return p.send(p.ClientId, msg)
}
