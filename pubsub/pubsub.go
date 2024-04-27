package pubsub

import "github.com/cgalvisleon/et/et"

type PubSub interface {
	Type() string
	IsConnected() bool
	Close()
	Connect() (bool, error)
	Ping()
	Params(params et.Json)
	System(params et.Json)
	Subscribe(channel string)
	Unsubscribe(channel string)
	Publish(channel string, message interface{})
	SendMessage(clientId string, message string)
}
