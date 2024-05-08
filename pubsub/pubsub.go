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

const ERR_NOT_PUBSUB_SERVICE = "PubSub service not found"

func Load(tp string, clientId, name string, reciveFn func(message.Message)) PubSub {
	switch tp {
	case "nats":
		res, err := nats.NewPubSub(clientId, name, reciveFn)
		if err != nil {
			return nil
		}

		conn = res
	case "ws":
		res, err := ws.NewPubSub(clientId, name, reciveFn)
		if err != nil {
			return nil
		}

		conn = res
	}

	return nil
}

// Return the connection pubsub type
func Type() string {
	if conn == nil {
		logs.Panic(ERR_NOT_PUBSUB_SERVICE)
	}

	return conn.Type()
}

// Check if the client is connected
func IsConnected() bool {
	if conn == nil {
		logs.Panic(ERR_NOT_PUBSUB_SERVICE)
	}

	return conn.IsConnected()
}

// Close the client websocket connection
func Close() {
	if conn == nil {
		logs.Panic(ERR_NOT_PUBSUB_SERVICE)
	}

	conn.Close()
}

// Connect to the service pubsub
func Connect() (bool, error) {
	if conn == nil {
		logs.Panic(ERR_NOT_PUBSUB_SERVICE)
	}

	return conn.Connect()
}

// Ping the service pubsub
func Ping() {
	if conn == nil {
		logs.Panic(ERR_NOT_PUBSUB_SERVICE)
	}

	conn.Ping()
}

// Set the params of the service pubsub
func Params(params et.Json) error {
	if conn == nil {
		logs.Panic(ERR_NOT_PUBSUB_SERVICE)
	}

	return conn.Params(params)
}

// Subscribe a client to a channel
func Subscribe(channel string, reciveFn func(message.Message)) {
	if conn == nil {
		logs.Panic(ERR_NOT_PUBSUB_SERVICE)
	}

	conn.Subscribe(channel, reciveFn)
}

// Stack a client to a channel
func Stack(channel string, reciveFn func(message.Message)) {
	if conn == nil {
		logs.Panic(ERR_NOT_PUBSUB_SERVICE)
	}

	conn.Stack(channel, reciveFn)
}

// Unsubscribe a client from a channel
func Unsubscribe(channel string) {
	if conn == nil {
		logs.Panic(ERR_NOT_PUBSUB_SERVICE)
	}

	conn.Unsubscribe(channel)
}

// Publish a message to a channel
func Publish(channel string, message interface{}) {
	if conn == nil {
		logs.Panic(ERR_NOT_PUBSUB_SERVICE)
	}

	conn.Publish(channel, message)
}

// Send a message to a client
func SendMessage(clientId string, message interface{}) error {
	if conn == nil {
		logs.Panic(ERR_NOT_PUBSUB_SERVICE)
	}

	return conn.SendMessage(clientId, message)
}

// Send a telemetry message
func Telemetry(message interface{}) {
	if conn == nil {
		logs.Panic(ERR_NOT_PUBSUB_SERVICE)
	}

	switch v := message.(type) {
	case et.Json:
		logs.Log("telemetry", v.ToString())
	default:
		logs.Log("telemetry", message)
	}

	conn.Publish("telemetry", message)
}

// Send a overflow message
func Overflow(message interface{}) {
	if conn == nil {
		logs.Panic(ERR_NOT_PUBSUB_SERVICE)
	}

	switch v := message.(type) {
	case et.Json:
		logs.Log("overflow", v.ToString())
	default:
		logs.Log("overflow", message)
	}

	conn.Publish("overflow", message)
}

// Send a token last use message
func TokeLastUse(message interface{}) {
	if conn == nil {
		logs.Panic(ERR_NOT_PUBSUB_SERVICE)
	}

	conn.Publish("token_last_use", message)
}
