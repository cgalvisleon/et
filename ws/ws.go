package ws

import (
	"net/http"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/gorilla/websocket"
)

/**
* New
* @return *Hub
**/
func New() *Hub {
	result := &Hub{
		Channels:        make(map[string]*Channel),
		Subscribers:     make(map[string]*Client),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		onListener:      make([]func(*Client, []byte), 0),
		onConnection:    make([]func(*Client), 0),
		onDisconnection: make([]func(*Client), 0),
		onChannel:       make([]func(Channel), 0),
		onRemove:        make([]func(string), 0),
		onPublish:       make([]func(ch Channel, ms Message), 0),
		onSend:          make([]func(to string, ms Message), 0),
		mu:              &sync.RWMutex{},
		isStart:         false,
	}
	return result
}

/**
* Upgrader
* @params w http.ResponseWriter, r *http.Request
* @return  *websocket.Conn, error
**/
func Upgrader(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return upgrader.Upgrade(w, r, nil)
}

/**
* SendError
* @params socket *websocket.Conn, err error
* @return error
**/
func SendError(socket *websocket.Conn, err error) error {
	return socket.WriteJSON(et.Json{
		"ok": false,
		"result": et.Json{
			"message": err.Error(),
		},
	})
}
