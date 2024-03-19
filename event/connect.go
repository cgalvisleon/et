package event

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/nats-io/nats.go"
)

func connect() {
	host := envar.EnvarStr("", "NATS_HOST")
	if host == "" {
		return
	}

	connect, err := nats.Connect(host)
	if err != nil {
		return
	}

	logs.Logf("NATS", `Connected host:%s`, host)

	conn = &Conn{
		conn: connect,
	}
}
