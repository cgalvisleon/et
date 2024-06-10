package nats

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/nats-io/nats.go"
)

// Subscribe to a channel
func Subscribe(channel string, reciveFn func(Message)) error {
	if conn == nil {
		return logs.Errorm(ERR_NOT_PUBSUB_SERVICE)
	}

	msg := Message{
		Channel: channel,
	}
	var err error
	conn.events, err = conn.conn.Subscribe(
		msg.Channel,
		func(m *nats.Msg) {
			reciveFn(msg)
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func Stack(channel string, reciveFn func(Message)) error {
	if conn == nil {
		return logs.Errorm(ERR_NOT_PUBSUB_SERVICE)
	}

	msg := Message{
		Channel: channel,
	}
	var err error
	conn.events, err = conn.conn.Subscribe(
		msg.Channel,
		func(m *nats.Msg) {
			ok := conn.Lock(msg.Id)
			if !ok {
				return
			}

			reciveFn(msg)
		},
	)
	if err != nil {
		return err
	}

	return nil
}

// Publish a message to a channel
func Publish(channel string, message interface{}) error {
	if conn == nil {
		return logs.Errorm(ERR_NOT_PUBSUB_SERVICE)
	}

	msg := NewMessage(et.Json{}, message)
	bt, err := msg.Encode()
	if err != nil {
		return err
	}

	err = conn.conn.Publish(channel, bt)
	if err != nil {
		return err
	}

	return nil
}

// Send a telemetry message
func Telemetry(message interface{}) {
	if conn == nil {
		logs.Errorm(ERR_NOT_PUBSUB_SERVICE)
	}

	switch v := message.(type) {
	case et.Json:
		logs.Log("telemetry", v.ToString())
	default:
		logs.Log("telemetry", message)
	}

	Publish("telemetry", message)
}

// Send a overflow message
func Overflow(message interface{}) {
	if conn == nil {
		logs.Errorm(ERR_NOT_PUBSUB_SERVICE)
	}

	switch v := message.(type) {
	case et.Json:
		logs.Log("overflow", v.ToString())
	default:
		logs.Log("overflow", message)
	}

	Publish("overflow", message)
}

// Send a token last use message
func TokeLastUse(message interface{}) {
	if conn == nil {
		logs.Errorm(ERR_NOT_PUBSUB_SERVICE)
	}

	Publish("token_last_use", message)
}
