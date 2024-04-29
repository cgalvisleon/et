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
