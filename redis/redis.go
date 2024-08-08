package redis

import (
	"context"
	"sync"

	"github.com/redis/go-redis/v9"
)

type Conn struct {
	ctx    context.Context
	locks  map[string]*sync.RWMutex
	host   string
	dbname int
	db     *redis.Client
}

var conn *Conn

/**
* Load redis connection
* @return (*Conn, error)
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
* Close redis connection
* @return error
**/
func Close() error {
	if conn.db == nil {
		return nil
	}

	return conn.db.Close()
}
