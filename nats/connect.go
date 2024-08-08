package nats

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/nats-io/nats.go"
)

// Connect to a host
func connect() (*nats.Conn, error) {
	host := envar.GetStr("", "NATS_HOST")

	if host == "" {
		return nil, logs.Alertf(ERR_ENV_REQUIRED, "REDIS_HOST")
	}

	result, err := nats.Connect(host)
	if err != nil {
		return nil, logs.Alert(err)
	}

	logs.Logf("NATS", `Connected host:%s`, host)

	return result, nil
}
