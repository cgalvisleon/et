package ws

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
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
func (h *Hub) Run() {
	if h.run {
		return
	}

	h.run = true
	host, _ := os.Hostname()
	logs.Logf("Websocket", "Run server host:%s", host)

	for {
		select {
		case client := <-h.register:
			h.onConnect(client)
		case client := <-h.unregister:
			h.onDisconnect(client)
		}
	}
}

// Connect a client to the hub
func (h *Hub) onConnect(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.clients = append(h.clients, client)
	client.Addr = client.socket.RemoteAddr().String()

	h.Publish("ws/connect", client.Params, client.Id)

	client.sendMessage([]byte(et.Json{
		"type":   "connect",
		"client": client.Params,
	}.ToString()))

	logs.Logf("Websocket", MSG_CLIENT_CONNECT, client.Id, h.Id)
}

// Disconnect a client from the hub
func (h *Hub) onDisconnect(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	client.Close()
	client.Clear()
	idx := slices.IndexFunc(h.clients, func(c *Client) bool { return c.Id == client.Id })

	copy(h.clients[idx:], h.clients[idx+1:])
	h.clients[len(h.clients)-1] = nil
	h.clients = h.clients[:len(h.clients)-1]

	h.Publish("ws/disconnect", client.Params, client.Id)

	logs.Logf("Websocket", MSG_CLIENT_DISCONNECT, client.Id, h.Id)
}

// Broadcast a message to all clients less the ignore client
func (h *Hub) broadcast(message interface{}, ignore *Client) {
	data, _ := json.Marshal(message)
	for _, client := range h.clients {
		if client != ignore {
			client.sendMessage(data)
		}
	}
}

// Create a client and connect to the hub
func (h *Hub) connect(socket *websocket.Conn, clientId, name string) (*Client, error) {
	idxC := slices.IndexFunc(h.clients, func(c *Client) bool { return c.Id == clientId })
	if idxC != -1 {
		return conn.clients[idxC], nil
	}

	client, isNew := newClient(h, socket, clientId, name)
	if isNew {
		h.register <- client

		go client.write()
		go client.read()
	}

	return client, nil
}

// Broadcast a message to all clients less the ignore client
func (h *Hub) Broadcast(message interface{}, ignoreId string) {
	var client *Client = nil
	idx := slices.IndexFunc(h.clients, func(c *Client) bool { return c.Id == ignoreId })
	if idx != -1 {
		client = h.clients[idx]
	}

	h.broadcast(message, client)
}

// Publish a message to a channel less the ignore client
func (h *Hub) Publish(channel string, message interface{}, ignoreId string) error {
	_channel := h.GetChannel(channel)
	if len(_channel.Subscribers) == 0 {
		return logs.Alertm(ERR_CHANNEL_NOT_SUBSCRIBERS)
	}

	data, _ := json.Marshal(message)
	for _, client := range _channel.Subscribers {
		if client.Id != ignoreId {
			client.sendMessage(data)
		}
	}

	return nil
}

// Send a message to a client in a channel
func (h *Hub) SendMessage(clientId string, message interface{}) error {
	data, _ := json.Marshal(message)
	idx := slices.IndexFunc(h.clients, func(c *Client) bool { return c.Id == clientId })
	if idx == -1 {
		return logs.Alertm(ERR_CLIENT_NOT_FOUND)
	}

	client := h.clients[idx]
	return client.sendMessage(data)
}

// Prune a channel if no subscribers
func (h *Hub) pruneChanner(channel *Channel) {
	if channel == nil {
		return
	}

	if channel.Count() == 0 {
		idx := slices.IndexFunc(h.channels, func(c *Channel) bool { return c.Low() == channel.Low() })
		if idx != -1 {
			h.channels = append(h.channels[:idx], h.channels[idx+1:]...)
		}
	}
}

func (h *Hub) GetChannel(name string) *Channel {
	var result *Channel

	clean := func() {
		logs.Log("Channel expired", name)
		h.pruneChanner(result)
	}

	idx := slices.IndexFunc(h.channels, func(c *Channel) bool { return c.Low() == strs.Lowcase(name) })
	if idx == -1 {
		logs.Log("New channel", name)
		result = newChannel(name)
		h.channels = append(h.channels, result)
	} else {
		result = h.channels[idx]
	}

	duration := 5 * time.Minute
	go time.AfterFunc(duration, clean)

	return result
}

// Subscribe a client to hub channels
func (h *Hub) Subscribe(clientId string, channel string) error {
	idx := slices.IndexFunc(h.clients, func(c *Client) bool { return c.Id == clientId })
	if idx == -1 {
		return logs.Alertm(ERR_CLIENT_NOT_FOUND)
	}

	client := h.clients[idx]
	_channel := h.GetChannel(channel)
	_channel.Subscribe(client)
	client.subscribe([]string{channel})
	return nil
}

// Unsubscribe a client from hub channels
func (h *Hub) Unsubscribe(clientId string, channel string) error {
	idx := slices.IndexFunc(h.clients, func(c *Client) bool { return c.Id == clientId })
	if idx == -1 {
		return logs.Alertm(ERR_CLIENT_NOT_FOUND)
	}

	client := h.clients[idx]
	client.unsubscribe([]string{channel})

	_channel := h.GetChannel(channel)
	_channel.Unsubcribe(clientId)
	h.pruneChanner(_channel)

	return nil
}

// Return client list subscribed to channel
func (h *Hub) GetSubscribers(channel string) []*Client {
	_channel := h.GetChannel(channel)
	return _channel.Subscribers
}
