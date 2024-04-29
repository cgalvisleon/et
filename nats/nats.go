package nats

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/nats-io/nats.go"
)

var (
	conn *Conn
)

type Conn struct {
	conn             *nats.Conn
	eventCreatedSub  *nats.Subscription
	eventCreatedChan chan Message
	connected        bool
}

func Load() (*Conn, error) {
	if conn != nil {
		return conn, nil
	}

	host := envar.EnvarStr("", "NATS_HOST")
	if host == "" {
		return nil, logs.Alertm("NATS_HOST not found")
	}

	_conn, err := connect(host)
	if err != nil {
		return nil, err
	}

	return &Conn{
		conn:             _conn,
		eventCreatedChan: make(chan Message),
		connected:        true,
	}, nil
}

func Close() {
	if conn.conn != nil {
		conn.conn.Close()
	}

	if conn.eventCreatedSub != nil {
		conn.eventCreatedSub.Unsubscribe()
	}

	if conn.eventCreatedChan == nil {
		return
	}

	close(conn.eventCreatedChan)
}
