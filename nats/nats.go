package nats

import (
	"github.com/cgalvisleon/et/cache"
	"github.com/nats-io/nats.go"
)

var (
	conn *Conn
)

type Conn struct {
	conn             *nats.Conn
	cache            cache.Cache
	eventCreatedSub  *nats.Subscription
	eventCreatedChan chan Message
}

func (c *Conn) Lock(key string) bool {
	return c.cache.Del(key)
}

func Load(cache cache.Cache) (*Conn, error) {
	if conn != nil {
		return conn, nil
	}

	_conn, err := connect(cache)
	if err != nil {
		return nil, err
	}

	return &Conn{
		conn:             _conn,
		cache:            cache,
		eventCreatedChan: make(chan Message),
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
