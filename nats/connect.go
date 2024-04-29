package nats

import (
	"github.com/cgalvisleon/et/logs"
	"github.com/nats-io/nats.go"
)

func connect(host string) (*nats.Conn, error) {
	connect, err := nats.Connect(host)
	if err != nil {
		return nil, logs.Alert(err)
	}

	logs.Logf("NATS", `Connected host:%s`, host)

	return connect, nil
}
