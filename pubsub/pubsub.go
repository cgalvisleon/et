package pubsub

import "github.com/cgalvisleon/et/et"

type TpMessage int

const (
	TpPing TpMessage = iota
	TpParams
	TpSystem
	TpSubscribe
	TpUnsubscribe
	TpPublish
	TpDirect
	TpLog
)

func (t TpMessage) String() string {
	switch t {
	case TpPing:
		return "ping"
	case TpParams:
		return "params"
	case TpSystem:
		return "system"
	case TpSubscribe:
		return "subscribe"
	case TpUnsubscribe:
		return "unsubscribe"
	case TpPublish:
		return "publish"
	case TpDirect:
		return "direct"
	}

	return ""
}

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
