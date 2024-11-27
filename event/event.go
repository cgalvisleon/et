package event

import (
	"sync"

	"github.com/cgalvisleon/et/logs"
	"github.com/nats-io/nats.go"
)

const PackageName = "event"

var conn *Conn

type Conn struct {
	*nats.Conn
	eventCreatedSub map[string]*nats.Subscription
	mutex           *sync.RWMutex
}

/**
* Load the connection to the service pubsub
* @return *Conn, error
**/
func Load() (*Conn, error) {
	if conn != nil {
		return conn, nil
	}

	var err error
	conn, err = connect()
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
