package ws

import (
	"github.com/cgalvisleon/et/cache"
)

var (
	conn *Conn
)

type Conn struct {
	hub   *Hub
	cache cache.Cache
}

// Load the Websocket Hub
func Load(cache cache.Cache) (*Conn, error) {
	if conn != nil {
		return conn, nil
	}

	hub := connect(cache)
	conn = &Conn{
		hub:   hub,
		cache: cache,
	}

	return conn, nil
}

// Close the Websocket Hub
func Close() error {
	return nil
}
