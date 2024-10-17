package event

import (
	"github.com/cgalvisleon/et/logs"
	"github.com/nats-io/nats.go"
)

var conn *Conn

type Conn struct {
	conn             *nats.Conn
	eventCreatedSub  *nats.Subscription
	eventCreatedChan chan EvenMessage
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
	if conn != nil && conn.conn != nil {
		conn.conn.Close()
	}

	if conn.eventCreatedSub != nil {
		conn.eventCreatedSub.Unsubscribe()
	}

	if conn.eventCreatedChan != nil {
		close(conn.eventCreatedChan)
	}

	logs.Log("Event", `Disconnect...`)
}
