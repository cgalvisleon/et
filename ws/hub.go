package ws

import (
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/cgalvisleon/et/envar"
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
	Name       string
	Params     *et.Json
	main       *websocket.Conn
	clients    []*Client
	channels   []*Channel
	register   chan *Client
	unregister chan *Client
	mutex      *sync.Mutex
	run        bool
}

// Create a new hub
func NewHub() *Hub {
	id := utility.UUID()
	name := "Hub"
	main := envar.GetStr("", "WS_MAIN_HOST")
	scheme := envar.GetStr("ws", "WS_SCHEME")
	result := &Hub{
		Id:   id,
		Name: name,
		Params: &et.Json{
			"id":   id,
			"name": name,
			"main": et.Json{
				"host":   main,
				"scheme": scheme,
			},
		},
		clients:    make([]*Client, 0),
		channels:   make([]*Channel, 0),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		mutex:      &sync.Mutex{},
		run:        false,
	}

	if main != "" {
		var err error
		result.main, err = connectWs(main, scheme)
		if err != nil {
			result.main = nil
		}
	}

	return result
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

	msg := NewMessage(*h.Params, et.Json{
		"ok":      true,
		"message": "Connected successfully",
		"client":  client.Params,
	})

	h.Mute("ws/connect", msg, []string{client.Id}, *h.Params)
	client.sendMessage(msg)

	logs.Logf("Websocket", MSG_CLIENT_CONNECT, client.Id, h.Id)
}

// Disconnect a client from the hub
func (h *Hub) onDisconnect(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	client.cLose()
	client.clear()
	idx := slices.IndexFunc(h.clients, func(c *Client) bool { return c.Id == client.Id })

	copy(h.clients[idx:], h.clients[idx+1:])
	h.clients[len(h.clients)-1] = nil
	h.clients = h.clients[:len(h.clients)-1]

	msg := NewMessage(*h.Params, et.Json{
		"ok":      true,
		"message": "Client disconnected",
		"client":  client.Params,
	})

	h.Mute("ws/disconnect", msg, []string{client.Id}, *h.Params)

	logs.Logf("Websocket", MSG_CLIENT_DISCONNECT, client.Id, h.Id)
}

// Create a client and connect to the hub
func (h *Hub) connect(socket *websocket.Conn, clientId, name string) (*Client, error) {
	idxC := slices.IndexFunc(h.clients, func(c *Client) bool { return c.Id == clientId })
	if idxC != -1 {
		return conn.hub.clients[idxC], nil
	}

	client, isNew := newClient(h, socket, clientId, name)
	if isNew {
		h.register <- client

		go client.write()
		go client.read()
	}

	return client, nil
}

// SetAtribs set a value to the client data
func (h *Hub) SetParams(params et.Json) {
	h.Params = &params
}

// Publish a message to a channel less the ignore client
func (h *Hub) publish(channel *Channel, msg Message, ignored []string, from et.Json) error {
	msg.Channel = channel.Low()
	msg.From = from
	msg.Ignored = ignored
	if channel.TpBroadcast == TpRoundRobin {
		client := channel.NextTurn()
		if client != nil {
			return client.sendMessage(msg)
		}
	} else {
		for _, client := range channel.Subscribers {
			if !slices.Contains(ignored, client.Id) {
				client.sendMessage(msg)
			}
		}
	}

	if h.main == nil {
		for _, client := range h.clients {
			if client.IsNode {
				client.sendMessage(msg)
			}
		}
	}

	return nil
}

// Publish a message to a channel less the ignore client
func (h *Hub) Publish(channel string, msg Message, ignored []string, from et.Json) error {
	ch := h.getChanel(channel)
	if len(ch.Subscribers) == 0 {
		return logs.Alertm(ERR_CHANNEL_NOT_SUBSCRIBERS)
	}

	return h.publish(ch, msg, ignored, from)
}

// Publish mute a message to a channel less the ignore client
func (h *Hub) Mute(channel string, msg Message, ignored []string, from et.Json) error {
	ch := h.getChanel(channel)

	return h.publish(ch, msg, ignored, from)
}

// Send a message to a client in a channel
func (h *Hub) SendMessage(clientId string, msg Message) error {
	idx := slices.IndexFunc(h.clients, func(c *Client) bool { return c.Id == clientId })
	if idx == -1 {
		return logs.Alertm(ERR_CLIENT_NOT_FOUND)
	}

	client := h.clients[idx]
	return client.sendMessage(msg)
}

// Prune a channel if no subscribers
func (h *Hub) pruneChanner(channel *Channel) {
	if channel == nil {
		return
	}

	if channel.Count() == 0 {
		logs.Log("Channel prune", channel.Name)
		idx := slices.IndexFunc(h.channels, func(c *Channel) bool { return c.Low() == channel.Low() })
		if idx != -1 {
			h.channels = append(h.channels[:idx], h.channels[idx+1:]...)
		}
	}
}

func (h *Hub) getChanel(name string) *Channel {
	var result *Channel

	clean := func() {
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

func (h *Hub) subscribe(clientId string, channel string) (*Channel, error) {
	idx := slices.IndexFunc(h.clients, func(c *Client) bool { return c.Id == clientId })
	if idx == -1 {
		return nil, logs.Alertm(ERR_CLIENT_NOT_FOUND)
	}

	client := h.clients[idx]
	result := h.getChanel(channel)
	result.Subscribe(client)
	client.subscribe([]string{channel})

	return result, nil
}

// Subscribe a client to hub channels
func (h *Hub) Subscribe(clientId string, channel string) error {
	_, err := h.subscribe(clientId, channel)
	return err
}

// Subscribe a client to hub channels
func (h *Hub) Stack(clientId string, channel string) error {
	ch, err := h.subscribe(clientId, channel)
	if err != nil {
		return err
	}

	ch.TpBroadcast = TpRoundRobin

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

	ch := h.getChanel(channel)
	ch.Unsubcribe(clientId)
	h.pruneChanner(ch)

	return nil
}

// Return client list subscribed to channel
func (h *Hub) GetSubscribers(channel string) []*Client {
	ch := h.getChanel(channel)
	return ch.Subscribers
}
