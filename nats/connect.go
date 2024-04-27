package nats

import (
	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/nats-io/nats.go"
)

func connect(cache cache.Cache) (*nats.Conn, error) {
	host := envar.EnvarStr("", "NATS_HOST")
	if host == "" {
		return nil, logs.Alertm("NATS_HOST not found")
	}

	connect, err := nats.Connect(host)
	if err != nil {
		return nil, logs.Alert(err)
	}

	logs.Logf("NATS", `Connected host:%s`, host)

	return connect, nil
}
