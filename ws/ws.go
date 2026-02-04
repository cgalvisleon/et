package ws

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

/**
* New
* @return *Hub
**/
func New() *Hub {
	result := &Hub{
		Channels:        make(map[string]*Channel),
		Subscribers:     make(map[string]*Subscriber),
		register:        make(chan *Subscriber),
		unregister:      make(chan *Subscriber),
		onConnection:    make([]func(*Subscriber), 0),
		onDisconnection: make([]func(*Subscriber), 0),
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
