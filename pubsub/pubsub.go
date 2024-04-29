package pubsub

import "github.com/cgalvisleon/et/et"

type TpMessage string

const (
	TpPing        = TpMessage("ping")
	TpParams      = TpMessage("params")
	TpSystem      = TpMessage("system")
	TpSubscribe   = TpMessage("subscribe")
	TpUnsubscribe = TpMessage("unsubscribe")
	TpPublish     = TpMessage("publish")
	TpDirect      = TpMessage("direct")
	TpLog         = TpMessage("log")
)

type Message interface {
	Type() TpMessage
	ToString() string
	Encode() ([]byte, error)
	Json() (et.Json, error)
}

// PubSub interface
type PubSub interface {
	Type() string
	IsConnected() bool
	Close()
	Connect() (bool, error)
	Ping()
	Params(params et.Json) error
	System(params et.Json) error
	Subscribe(channel string, reciveFn func(Message))
	Unsubscribe(channel string)
	Publish(channel string, message interface{})
	SendMessage(to et.Json, message string) error
}
