package ws

import (
	"net/url"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/gorilla/websocket"
)

// Create a Hub and run it
func connectHub() *Hub {
	hub := NewHub()
	go hub.Run()

	logs.Log("WS", "Run websocket server")

	return hub
}

// Connect to the server
func connectWs(host, scheme string) (*websocket.Conn, error) {
	if scheme == "" {
		scheme = "ws"
	}

	path := strs.Format("/%s", scheme)

	u := url.URL{Scheme: scheme, Host: host, Path: path}
	result, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}

	return result, nil
}
