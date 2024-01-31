package event

import (
	"github.com/cgalvisleon/elvis/envar"
	"github.com/cgalvisleon/elvis/logs"
	_ "github.com/joho/godotenv/autoload"
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
