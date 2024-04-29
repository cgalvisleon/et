package ws

import (
	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/logs"
)

// Create a Hub and run it
func connect(cache cache.Cache) *Hub {
	hub := NewHub()
	go hub.Run()

	logs.Log("WS", "Run websocket server")

	return hub
}
