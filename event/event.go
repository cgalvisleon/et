package event

import (
	"regexp"
	"strings"

	"github.com/cgalvisleon/et/js"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/message"
	"github.com/cgalvisleon/et/nats"
	"github.com/cgalvisleon/et/pubsub"
)

var conn pubsub.PubSub

const ERR_NOT_PUBSUB_SERVICE = "PubSub service not found func:%s"

// Load the event adapter
func Load() error {
	if conn != nil {
		return nil
	}

	res, err := nats.NewPubSub()
	if err != nil {
		return err
	}

	conn = res

	return nil
}

// Prune a channel
func ChannelPrune(channel string) string {
	// Encuentra la cadena que no comienza con "/"
	re := regexp.MustCompile(`^\/*(.+)$`)
	// Agrega "/" al inicio si no está presente
	result := re.ReplaceAllString(channel, "/$1")
	// Reemplazar espacios con "-"
	result = strings.ReplaceAll(result, " ", "-")
	// Convertir a minúsculas
	result = strings.ToLower(result)

	return result
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
func Params(params js.Json) error {
	if conn == nil {
		return logs.Alertf(ERR_NOT_PUBSUB_SERVICE, "Params")
	}

	return conn.Params(params)
}

// Subscribe a client to a channel
func Subscribe(channel string, reciveFn func(message.Message)) error {
	if conn == nil {
		return logs.Alertf(ERR_NOT_PUBSUB_SERVICE, "Subscribe")
	}

	conn.Subscribe(channel, reciveFn)

	return nil
}

// Stack a client to a channel
func Stack(channel string, reciveFn func(message.Message)) error {
	if conn == nil {
		return logs.Alertf(ERR_NOT_PUBSUB_SERVICE, "Stack")
	}

	conn.Stack(channel, reciveFn)

	return nil
}

// Unsubscribe a client from a channel
func Unsubscribe(channel string) error {
	if conn == nil {
		return logs.Alertf(ERR_NOT_PUBSUB_SERVICE, "Unsubscribe")
	}

	conn.Unsubscribe(channel)

	return nil
}

// Publish a message to a channel
func Publish(channel string, message interface{}) error {
	if conn == nil {
		return logs.Alertf(ERR_NOT_PUBSUB_SERVICE, "Publish")
	}

	conn.Publish(channel, message)

	return nil
}

// Send a message to a client
func SendMessage(clientId string, message interface{}) error {
	if conn == nil {
		return logs.Alertf(ERR_NOT_PUBSUB_SERVICE, "SendMessage")
	}

	return conn.SendMessage(clientId, message)
}

// Send a message to a event worker
func Worker(event string, data interface{}) error {
	if conn == nil {
		return logs.Alertf(ERR_NOT_PUBSUB_SERVICE, "Worker")
	}

	conn.Publish(event, data)

	return nil
}

// Send a log message
func Log(message interface{}) error {
	if conn == nil {
		return logs.Alertf(ERR_NOT_PUBSUB_SERVICE, "Log")
	}

	conn.Publish("/log", message)

	return nil
}

// Send a telemetry message
func Telemetry(message interface{}) error {
	if conn == nil {
		return logs.Alertf(ERR_NOT_PUBSUB_SERVICE, "Telemetry")
	}

	conn.Publish("/telemetry", message)

	return nil
}

// Send a overflow message
func Overflow(message interface{}) error {
	if conn == nil {
		return logs.Alertf(ERR_NOT_PUBSUB_SERVICE, "Overflow")
	}

	conn.Publish("/overflow", message)

	return nil
}

// Send a token last use message
func TokeLastUse(message interface{}) error {
	if conn == nil {
		return logs.Alertf(ERR_NOT_PUBSUB_SERVICE, "TokeLastUse")
	}

	conn.Publish("/token_last_use", message)

	return nil
}
