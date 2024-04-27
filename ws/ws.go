package ws

import (
	"sync"
)

var (
	conn *Hub
	once sync.Once
)

type Conn struct {
	hub *Hub
}

// Create a Hub and run it
func connect() {
	if conn != nil {
		return
	}

	conn = NewHub()
	go conn.Run()
}

// Load the Websocket Hub
func Load() (*Hub, error) {
	once.Do(connect)

	return conn, nil
}

// Close the Websocket Hub
func Close() error {
	return nil
}
