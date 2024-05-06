package nats

import (
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/pubsub"
)

// Subscribe to a channel
func Subscribe(channel string, f func(pubsub.Message)) error {
	if conn == nil {
		return logs.Alertm(ERR_NOT_CONNECT_NATS)
	}

	conn.Subscribe(channel, f)

	return nil
}

// Unsubscribe to a channel
func Unsubscribe(channel string) error {
	if conn == nil {
		return logs.Alertm(ERR_NOT_CONNECT_NATS)
	}

	conn.Unsubscribe(channel)

	return nil
}

// Stack to a channel
func Stack(channel string, f func(pubsub.Message)) error {
	if conn == nil {
		return logs.Alertm(ERR_NOT_CONNECT_NATS)
	}

	err := conn.stack(channel, f)
	if err != nil {
		return err
	}

	return nil
}

// Publish to a channel
func Publish(channel string, message pubsub.Message) error {
	if conn == nil {
		return logs.Alertm(ERR_NOT_CONNECT_NATS)
	}

	conn.Publish(channel, message)

	return nil
}

// SendMessage to a client
func SendMessage(clientId string, message pubsub.Message) error {
	if conn == nil {
		return logs.Alertm(ERR_NOT_CONNECT_NATS)
	}

	conn.SendMessage(clientId, message)

	return nil
}
