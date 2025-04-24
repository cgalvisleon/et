package event

import (
	"sync"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/nats-io/nats.go"
)

const PackageName = "event"

var conn *Conn

type Conn struct {
	*nats.Conn
	Id              string
	eventCreatedSub map[string]*nats.Subscription
	mutex           *sync.RWMutex
}

/**
* FromId return the id of the connection
* @return string
**/
func FromId() string {
	return conn.Id
}

/**
* Load the connection to the service pubsub
* @return *Conn, error
**/
func Load() (*Conn, error) {
	if conn != nil {
		return conn, nil
	}

	host := envar.GetStr("", "NATS_HOST")
	user := envar.GetStr("", "NATS_USER")
	password := envar.GetStr("", "NATS_PASSWORD")

	if host == "" {
		return nil, logs.Alertf(msg.ERR_ENV_REQUIRED, "NATS_HOST")
	}

	var err error
	conn, err = ConnectTo(host, user, password)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

/**
* Close the connection to the service pubsub
**/
func Close() {
	if conn == nil {
		return
	}

	for _, sub := range conn.eventCreatedSub {
		sub.Unsubscribe()
	}

	conn.Close()

	logs.Log(PackageName, `Disconnect...`)
}
