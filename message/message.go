package message

import "github.com/cgalvisleon/et/et"

type TpMessage string

const (
	TpPing        = TpMessage("ping")
	TpParams      = TpMessage("params")
	TpSubscribe   = TpMessage("subscribe")
	TpUnsubscribe = TpMessage("unsubscribe")
	TpStack       = TpMessage("stack")
	TpPublish     = TpMessage("publish")
	TpDirect      = TpMessage("direct")
	TpLog         = TpMessage("log")
	TpError       = TpMessage("error")
	TpConnect     = TpMessage("connect")
	TpDisconnect  = TpMessage("disconnect")
)

type Message interface {
	Type() TpMessage
	ToString() string
	Encode() ([]byte, error)
	Json() (et.Json, error)
}
