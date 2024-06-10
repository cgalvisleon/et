package nats

import (
	"github.com/cgalvisleon/et/logs"
	"github.com/nats-io/nats.go"
)

// Connect to a host
func connect(host string) (*nats.Conn, error) {
	result, err := nats.Connect(host)
	if err != nil {
		return nil, logs.Alert(err)
	}

	logs.Logf("NATS", `Connected host:%s`, host)

	return result, nil
}
