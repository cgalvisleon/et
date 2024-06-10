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

func Load() *Conn {
	if conn != nil {
		return conn
	}

	host := envar.GetStr("", "NATS_HOST")
	if host == "" {
		logs.Alertf(ERR_ENV_REQUIRED, "REDIS_HOST")
		return nil
	}

	c, err := connect(host)
	if err != nil {
		return nil
	}

	conn = &Conn{
		conn: c,
	}

	return conn
}

func Close() {
	if conn.conn != nil {
		conn.conn.Close()
	}

	if conn.events != nil {
		conn.events.Unsubscribe()
	}
}
