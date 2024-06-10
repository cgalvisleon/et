package pubsub

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/message"
	"github.com/cgalvisleon/et/nats"
	"github.com/cgalvisleon/et/ws"
)

// PubSub interface
type PubSub interface {
	Type() string
	IsConnected() bool
	Close()
	Connect() (bool, error)
	Ping()
	Params(params et.Json) error
	Subscribe(channel string, reciveFn func(message.Message))
	Stack(channel string, reciveFn func(message.Message))
	Unsubscribe(channel string)
	Publish(channel string, message interface{})
	SendMessage(clientId string, message interface{}) error
}

var conn PubSub

const ERR_NOT_PUBSUB_SERVICE = "PubSub service not found func:%s"

func Load(tp string, clientId, name string, reciveFn func(message.Message)) (PubSub, error) {
	switch tp {
	case "nats":
		res, err := nats.NewPubSub(clientId, name, reciveFn)
		if err != nil {
			return nil, err
		}

		conn = res
	case "ws":
		res, err := ws.NewPubSub(clientId, name, reciveFn)
		if err != nil {
			return nil, err
		}

		conn = res
	}

	return conn, nil
}

// Return the connection pubsub type
func Type() string {
	if conn == nil {
		return ""
	}

	return conn.Type()
}

// Check if the client is connected
func IsConnected() bool {
	if conn == nil {
		return false
	}

	return conn.IsConnected()
}

// Close the client websocket connection
func Close() {
	if conn == nil {
		return
	}

	conn.Close()
}

// Connect to the service pubsub
func Connect() (bool, error) {
	if conn == nil {
		return false, logs.Alertf(ERR_NOT_PUBSUB_SERVICE, "Connect")
	}

	return conn.Connect()
}

// Ping the service pubsub
func Ping() {
	if conn == nil {
		logs.Alertf(ERR_NOT_PUBSUB_SERVICE, "Ping")
		return
	}

	conn.Ping()
}

// Set the params of the service pubsub
func Params(params et.Json) error {
	if conn == nil {
		return logs.Alertf(ERR_NOT_PUBSUB_SERVICE, "Params")
	}

	return conn.Params(params)
}

// Subscribe a client to a channel
func Subscribe(channel string, reciveFn func(message.Message)) {
	if conn == nil {
		logs.Alertf(ERR_NOT_PUBSUB_SERVICE, "Subscribe")
		return
	}

	conn.Subscribe(channel, reciveFn)
}

// Stack a client to a channel
func Stack(channel string, reciveFn func(message.Message)) {
	if conn == nil {
		logs.Alertf(ERR_NOT_PUBSUB_SERVICE, "Stack")
		return
	}

	conn.Stack(channel, reciveFn)
}

// Unsubscribe a client from a channel
func Unsubscribe(channel string) {
	if conn == nil {
		logs.Alertf(ERR_NOT_PUBSUB_SERVICE, "Unsubscribe")
		return
	}

	conn.Unsubscribe(channel)
}

// Publish a message to a channel
func Publish(channel string, message interface{}) {
	if conn == nil {
		logs.Alertf(ERR_NOT_PUBSUB_SERVICE, "Publish")
		return
	}

	conn.Publish(channel, message)
}

// Send a message to a client
func SendMessage(clientId string, message interface{}) error {
	if conn == nil {
		return logs.Alertf(ERR_NOT_PUBSUB_SERVICE, "SendMessage")
	}

	return conn.SendMessage(clientId, message)
}

// Send a telemetry message
func Telemetry(message interface{}) {
	if conn == nil {
		logs.Alertf(ERR_NOT_PUBSUB_SERVICE, "Telemetry")
		return
	}

	conn.Publish("telemetry", message)
}

// Send a overflow message
func Overflow(message interface{}) {
	if conn == nil {
		logs.Alertf(ERR_NOT_PUBSUB_SERVICE, "Overflow")
		return
	}

	conn.Publish("overflow", message)
}

// Send a token last use message
func TokeLastUse(message interface{}) {
	if conn == nil {
		logs.Alertf(ERR_NOT_PUBSUB_SERVICE, "TokeLastUse")
		return
	}

	conn.Publish("token_last_use", message)
}
