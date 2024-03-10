package cache

import (
	"context"
	"sync"

	"github.com/redis/go-redis/v9"
)

var (
	conn *Conn
	once sync.Once
)

type Conn struct {
	ctx    context.Context
	host   string
	dbname int
	db     *redis.Client
}

func Load() (*Conn, error) {
	once.Do(connect)

	return conn, nil
}

func Close() error {
	if conn.db == nil {
		return nil
	}

	return conn.db.Close()
}
