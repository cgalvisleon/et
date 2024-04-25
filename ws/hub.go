package ws

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
	"github.com/gorilla/websocket"
	"golang.org/x/exp/slices"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Hub struct {
	Id         string
	clients    []*Client
	channels   []*Channel
	register   chan *Client
	unregister chan *Client
	mutex      *sync.Mutex
	run        bool
}

// Create a new hub
func NewHub() *Hub {
	return &Hub{
		Id:         utility.NewId(),
		clients:    make([]*Client, 0),
		channels:   make([]*Channel, 0),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		mutex:      &sync.Mutex{},
		run:        false,
	}
}

// Run the hub
func (hub *Hub) Run() {
	if hub.run {
		return
	}

	hub.run = true
	host, _ := os.Hostname()
	logs.Logf("Websocket", "Run server host:%s", host)

	for {
		select {
		case client := <-hub.register:
			hub.onConnect(client)
		case client := <-hub.unregister:
			hub.onDisconnect(client)
		}
	}
}

// Broadcast a message to all clients less the ignore client
func (hub *Hub) broadcast(message interface{}, ignore *Client) {
	data, _ := json.Marshal(message)
	for _, client := range hub.clients {
		if client != ignore {
			client.sendMessage(data)
		}
	}
}

// Connect a client to the hub
func (hub *Hub) onConnect(client *Client) {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()

	hub.clients = append(hub.clients, client)
	client.Addr = client.socket.RemoteAddr().String()

	hub.Publish("ws/connect", et.Json{
		"client": client.Id,
		"hub":    hub.Id,
	}, client.Id)

	logs.Logf("Websocket", MSG_CLIENT_CONNECT, client.Id, hub.Id)
}

// Disconnect a client from the hub
func (hub *Hub) onDisconnect(client *Client) {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()

	client.Close()
	client.Clear()
	idx := slices.IndexFunc(hub.clients, func(c *Client) bool { return c.Id == client.Id })

	copy(hub.clients[idx:], hub.clients[idx+1:])
	hub.clients[len(hub.clients)-1] = nil
	hub.clients = hub.clients[:len(hub.clients)-1]

	hub.Publish("ws/disconnect", et.Json{
		"client": client.Id,
		"hub":    hub.Id,
	}, client.Id)

	logs.Logf("Websocket", MSG_CLIENT_DISCONNECT, client.Id, hub.Id)
}

// Get the index of a client in the hub
func (hub *Hub) indexClient(clientId string) int {
	return slices.IndexFunc(hub.clients, func(c *Client) bool { return c.Id == clientId })
}

// Create a client and connect to the hub
func (hub *Hub) connect(socket *websocket.Conn, id, name string) (*Client, error) {
	client, isNew := newClient(hub, socket, id, name)
	if isNew {
		hub.register <- client

		go client.write()
		go client.read()
	}

	return client, nil
}

// Listen a client message
func (hub *Hub) listen(client *Client, messageType int, message []byte) {
	data, err := et.ToJson(message)
	if err != nil {
		data = et.Json{
			"type":    messageType,
			"message": bytes.NewBuffer(message).String(),
		}
	}

	client.sendMessage([]byte(data.ToString()))
}

// Broadcast a message to all clients less the ignore client
func (hub *Hub) Broadcast(message interface{}, ignoreId string) {
	var client *Client = nil
	idx := slices.IndexFunc(hub.clients, func(c *Client) bool { return c.Id == ignoreId })
	if idx != -1 {
		client = hub.clients[idx]
	}

	hub.broadcast(message, client)
}

// Publish a message to a channel less the ignore client
func (hub *Hub) Publish(channel string, message interface{}, ignoreId string) {
	data, _ := json.Marshal(message)
	idx := slices.IndexFunc(hub.channels, func(c *Channel) bool { return c.Name == channel })
	if idx != -1 {
		_channel := hub.channels[idx]

		for _, client := range _channel.Subscribers {
			if client.Id != ignoreId {
				client.sendMessage(data)
			}
		}
	}
}

// Send a message to a client in a channel
func (hub *Hub) SendMessage(clientId, channel string, message interface{}) bool {
	data, _ := json.Marshal(message)
	idx := slices.IndexFunc(hub.clients, func(c *Client) bool { return c.Id == clientId })
	if idx != -1 {
		client := hub.clients[idx]

		idx = slices.IndexFunc(client.channels, func(c *Channel) bool { return c.Name == channel })
		if idx != -1 {
			client.sendMessage(data)
			return true
		}
	}

	return false
}

// Subscribe a client to hub channels
func (hub *Hub) Subscribe(clientId string, channel string) bool {
	idx := slices.IndexFunc(hub.clients, func(c *Client) bool { return c.Id == clientId })

	if idx != -1 {
		client := hub.clients[idx]
		client.Subscribe(channel)

		return true
	}

	return false
}

// Unsubscribe a client from hub channels
func (hub *Hub) Unsubscribe(clientId string, channel string) bool {
	idx := slices.IndexFunc(hub.clients, func(c *Client) bool { return c.Id == clientId })

	if idx != -1 {
		client := hub.clients[idx]
		client.Unsubscribe(channel)

		return true
	}

	return false
}

// Return client list subscribed to channel
func (hub *Hub) GetSubscribers(channel string) []*Client {
	idx := slices.IndexFunc(hub.channels, func(c *Channel) bool { return c.Name == channel })
	if idx != -1 {
		_channel := hub.channels[idx]
		return _channel.Subscribers
	}

	return []*Client{}
}
