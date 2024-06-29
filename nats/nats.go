package nats

import (
	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/nats-io/nats.go"
)

type Conn struct {
	conn   *nats.Conn
	events *nats.Subscription
}

func (c *Conn) Lock(key string) bool {
	return cache.Del(key)
}

var conn *Conn

func Load() (*Conn, error) {
	if conn != nil {
		return conn, nil
	}

	host := envar.GetStr("", "NATS_HOST")
	if host == "" {
		return nil, logs.Alertf(ERR_ENV_REQUIRED, "REDIS_HOST")
	}

	c, err := connect(host)
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
