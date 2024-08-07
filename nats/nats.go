package nats

import (
	"github.com/cgalvisleon/et/logs"
	"github.com/nats-io/nats.go"
)

type Conn struct {
	conn   *nats.Conn
	events *nats.Subscription
}

var conn *Conn

func Load() (*Conn, error) {
	if conn != nil {
		return conn, nil
	}

	c, err := connect()
	if err != nil {
		return nil, logs.Alert(err)
	}

	conn = &Conn{
		conn: c,
	}

	return conn, nil
}

func Close() {
	if conn.conn != nil {
		conn.conn.Close()
	}

	if conn.events != nil {
		conn.events.Unsubscribe()
	}
}
