package ws

import (
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/cgalvisleon/et/js"
	"github.com/cgalvisleon/et/logs"
	m "github.com/cgalvisleon/et/message"
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

type HubParams struct {
	Id   string
	Name string
}

type Hub struct {
	Id         string
	Name       string
	Params     *HubParams
	clients    []*Client
	channels   []*Channel
	register   chan *Client
	unregister chan *Client
	mutex      *sync.Mutex
	adapter    *RedisAdapter
	run        bool
}

/**
* NewHub
* @return *Hub
**/
func NewHub() *Hub {
	id := utility.UUID()
	name := "Websocket Hub"
	result := &Hub{
		Id:   id,
		Name: name,
		Params: &HubParams{
			Id:   id,
			Name: name,
		},
		clients:    make([]*Client, 0),
		channels:   make([]*Channel, 0),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		mutex:      &sync.Mutex{},
		run:        false,
	}

	return result
}

/**
* Run
**/
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

/**
* Identify the hub
* @return js.Json
**/
func (h *Hub) Identify() js.Json {
	return js.Json{
		"id":   h.Id,
		"name": h.Name,
	}
}

/**
* onConnect
* @param client *Client
**/
func (h *Hub) onConnect(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.clients = append(h.clients, client)
	client.Addr = client.socket.RemoteAddr().String()

	logs.Debug("Client connected", client.Id, client.Name)

	msg := NewMessage(h.Identify(), js.Json{
		"ok":      true,
		"message": "Connected successfully",
		"client":  client.Identify(),
	}, m.TpConnect)
	msg.Channel = "ws/connect"

	h.Mute(msg.Channel, msg, []string{client.Id}, h.Identify())
	client.sendMessage(msg)

	logs.Logf("Websocket", MSG_CLIENT_CONNECT, client.Id, h.Id)
}

/**
* onDisconnect
* @param client *Client
**/
func (h *Hub) onDisconnect(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	client.cLose()
	client.clear()
	idx := slices.IndexFunc(h.clients, func(c *Client) bool { return c.Id == client.Id })

	copy(h.clients[idx:], h.clients[idx+1:])
	h.clients[len(h.clients)-1] = nil
	h.clients = h.clients[:len(h.clients)-1]

	msg := NewMessage(h.Identify(), js.Json{
		"ok":      true,
		"message": "Client disconnected",
		"client":  client.Identify(),
	}, m.TpDisconnect)
	msg.Channel = "ws/disconnect"

	h.Mute(msg.Channel, msg, []string{client.Id}, h.Identify())

	logs.Logf("Websocket", MSG_CLIENT_DISCONNECT, client.Id, h.Id)
}

/**
* connect
* @param socket *websocket.Conn
* @param clientId string
* @param name string
* @return *Client
* @return error
**/
func (h *Hub) connect(socket *websocket.Conn, clientId, name string) (*Client, error) {
	idxC := slices.IndexFunc(h.clients, func(c *Client) bool { return c.Id == clientId })
	if idxC != -1 {
		return servws.hub.clients[idxC], nil
	}

	client, isNew := newClient(h, socket, clientId, name)
	if isNew {
		h.register <- client

		go client.write()
		go client.read()
	}

	return client, nil
}

/**
* publish
* @param channel *Channel
* @param msg Message
* @param ignored []string
* @param from js.Json
* @return error
**/
func (h *Hub) broadcast(channel *Channel, msg Message, ignored []string, from js.Json) error {
	msg.Channel = channel.Low()
	msg.From = from
	msg.Ignored = ignored
	if len(channel.Group) > 0 {
		for queue, _ := range channel.Group {
			client := channel.NextTurn(queue)
			if client != nil {
				return client.sendMessage(msg)
			}
		}
	} else {
		for _, client := range channel.Subscribers {
			if !slices.Contains(ignored, client.Id) {
				client.sendMessage(msg)
			}
		}
	}

	if h.adapter != nil {
		h.adapter.Broadcast(channel.Name, msg, ignored, from)
	}

	return nil
}

/**
* listend
* @param msg interface{}
**/
func (h *Hub) listend(msg interface{}) {
	logs.Log("Broadcast", msg)

	m, err := decodeMessageBroadcat([]byte(msg.(string)))
	if err != nil {
		logs.Alert(err)
		return
	}

	switch m.Kind {
	case TpAll:
		h.Publish(m.To, m.Msg, m.Ignored, m.From)
	case TpDirect:
		idx := slices.IndexFunc(h.clients, func(c *Client) bool { return c.Id == m.To })
		if idx != -1 {
			client := h.clients[idx]
			client.sendMessage(m.Msg)
		}
	}
}

/**
* pruneChanner
* @param channel *Channel
**/
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

/**
* getChanel
* @param name string
* @return *Channel
**/
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

/**
* subscribe
* @param clientId string
* @param channel string
* @return *Channel
* @return error
**/
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

/**
* queueSubscribe
* @param clientId string
* @param channel string
* @param queue string
* @return *Channel
* @return error
**/
func (h *Hub) queueSubscribe(clientId string, channel, queue string) (*Channel, error) {
	idx := slices.IndexFunc(h.clients, func(c *Client) bool { return c.Id == clientId })
	if idx == -1 {
		return nil, logs.Alertm(ERR_CLIENT_NOT_FOUND)
	}

	client := h.clients[idx]
	result := h.getChanel(channel)
	result.QueueSubscribe(client, queue)
	client.subscribe([]string{channel})

	return result, nil
}

/**
* SetName
* @param name string
**/
func (h *Hub) SetName(name string) {
	h.Params.Name = name
}

/**
* Publish a message to a channel
* @param channel string
* @param msg Message
* @param ignored []string
* @param from js.Json
* @return error
**/
func (h *Hub) Publish(channel string, msg Message, ignored []string, from js.Json) error {
	ch := h.getChanel(channel)
	if len(ch.Subscribers) == 0 {
		return logs.Alertf(ERR_CHANNEL_NOT_SUBSCRIBERS, channel)
	}

	return h.broadcast(ch, msg, ignored, from)
}

/**
* Mute a message to a channel
* @param channel string
* @param msg Message
* @param ignored []string
* @param from js.Json
* @return error
**/
func (h *Hub) Mute(channel string, msg Message, ignored []string, from js.Json) error {
	ch := h.getChanel(channel)

	return h.broadcast(ch, msg, ignored, from)
}

/**
* SendMessage
* @param clientId string
* @param msg Message
* @return error
**/
func (h *Hub) SendMessage(clientId string, msg Message) error {
	idx := slices.IndexFunc(h.clients, func(c *Client) bool { return c.Id == clientId })
	if idx == -1 {
		if h.adapter != nil {
			h.adapter.Direct(clientId, msg)
		}
	}

	client := h.clients[idx]
	return client.sendMessage(msg)
}

/**
* Subscribe a client to hub channels
* @param clientId string
* @param channel string
* @return error
**/
func (h *Hub) Subscribe(clientId string, channel string) error {
	_, err := h.subscribe(clientId, channel)
	return err
}

/**
* Stack a client to hub channels
* @param clientId string
* @param channel string
* @return error
**/
func (h *Hub) Stack(clientId string, channel string) error {
	_, err := h.queueSubscribe(clientId, channel, "workers")
	if err != nil {
		return err
	}

	return nil
}

/**
* Unsubscribe a client from hub channels
* @param clientId string
* @param channel string
* @return error
**/
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

/**
* GetSubscribers
* @param channel string
* @return []*Client
**/
func (h *Hub) GetSubscribers(channel string) []*Client {
	ch := h.getChanel(channel)
	return ch.Subscribers
}
