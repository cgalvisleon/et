package ws

import (
	"github.com/cgalvisleon/et/logs"
)

// Create a Hub and run it
func connect() *Hub {
	hub := NewHub()
	go hub.Run()

	logs.Log("WS", "Run websocket server")

	return hub
}
