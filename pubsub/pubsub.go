package pubsub

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/message"
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
