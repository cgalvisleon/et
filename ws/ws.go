package ws

import (
	"sync"
)

var (
	conn *Conn
	once sync.Once
)

type Conn struct {
	hub *Hub
}

func Load() (*Conn, error) {
	once.Do(connect)

	return conn, nil
}

func Close() error {
	return nil
}
