package event

import (
	"sync"

	"github.com/cgalvisleon/elvis/cache"
	"github.com/nats-io/nats.go"
)

var (
	conn *Conn
	once sync.Once
)

type Conn struct {
	conn             *nats.Conn
	eventCreatedSub  *nats.Subscription
	eventCreatedChan chan CreatedEvenMessage
}

func (c *Conn) LockStack(key string) bool {
	val, err := cache.Del(key)
	if err != nil {
		return false
	}

	return val == 1
}

func Load() (*Conn, error) {
	once.Do(connect)

	return conn, nil
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
