package ws

import (
	"net/http"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
	"github.com/gorilla/websocket"
	"golang.org/x/exp/slices"
)

const ServiceName = "Websocket"

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Hub struct {
	Id         string
	Name       string
	Host       string
	clients    []*Subscriber
	channels   []*Channel
	queues     []*Queue
	register   chan *Subscriber
	unregister chan *Subscriber
	run        bool
	mutex      *sync.RWMutex
}

/**
* NewHub
* @return *Hub
**/
func NewHub() *Hub {
	result := &Hub{
		Id:         utility.UUID(),
		Name:       ServiceName,
		clients:    make([]*Subscriber, 0),
		channels:   make([]*Channel, 0),
		register:   make(chan *Subscriber),
		unregister: make(chan *Subscriber),
		run:        false,
		mutex:      &sync.RWMutex{},
	}

	return result
}

func (h *Hub) start() {
	for {
		select {
		case client := <-h.register:
			h.onConnect(client)
		case client := <-h.unregister:
			h.onDisconnect(client)
		}
	}
}

func (h *Hub) getClient(id string) *Subscriber {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	idx := slices.IndexFunc(h.clients, func(c *Subscriber) bool { return c.Id == id })
	if idx == -1 {
		return nil
	}

	return h.clients[idx]
}

func (h *Hub) addClient(value *Subscriber) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.clients = append(h.clients, value)
}

func (h *Hub) removeClient(value *Subscriber) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	idx := slices.IndexFunc(h.clients, func(c *Subscriber) bool { return c.Id == value.Id })
	if idx == -1 {
		return
	}

	h.clients = append(h.clients[:idx], h.clients[idx+1:]...)
}

func (h *Hub) getChannel(name string) *Channel {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	idx := slices.IndexFunc(h.channels, func(c *Channel) bool { return c.Name == name })
	if idx == -1 {
		return nil
	}

	return h.channels[idx]
}

func (h *Hub) addChannel(value *Channel) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.channels = append(h.channels, value)
}

func (h *Hub) removeChannel(value *Channel) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	idx := slices.IndexFunc(h.channels, func(c *Channel) bool { return c.Name == value.Name })
	if idx == -1 {
		return
	}

	h.ClusterUnSubscribed(value.Name)

	value.close()
	h.channels = append(h.channels[:idx], h.channels[idx+1:]...)
}

func (h *Hub) getQueue(name, queue string) *Queue {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	idx := slices.IndexFunc(h.queues, func(c *Queue) bool { return c.Name == name && c.Queue == queue })
	if idx == -1 {
		return nil
	}

	return h.queues[idx]
}

func (h *Hub) addQueue(value *Queue) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.queues = append(h.queues, value)
}

func (h *Hub) removeQueue(value *Queue) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	idx := slices.IndexFunc(h.queues, func(c *Queue) bool { return c.Name == value.Name })
	if idx == -1 {
		return
	}

	h.ClusterUnSubscribed(value.Name)

	value.close()

	h.queues = append(h.queues[:idx], h.queues[idx+1:]...)
}

func (h *Hub) onConnect(client *Subscriber) {
	if client == nil {
		return
	}

	h.addClient(client)
	msg := NewMessage(h.From(), et.Json{
		"ok":       true,
		"message":  MSG_CONNECT_SUCCESSFULLY,
		"clientId": client.Id,
		"name":     client.Name,
	}, TpConnect)
	msg.Channel = "ws/connect"
	client.sendMessage(msg)

	h.ClusterSubscribed(client.Id)

	logs.Logf(ServiceName, MSG_CLIENT_CONNECT, client.Id, client.Name, h.Id)
}

func (h *Hub) onDisconnect(client *Subscriber) {
	if client == nil {
		return
	}

	clientId := client.Id
	name := client.Name
	h.removeClient(client)

	h.ClusterUnSubscribed(clientId)

	logs.Logf(ServiceName, MSG_CLIENT_DISCONNECT, clientId, name, h.Id)
}

func (h *Hub) connect(socket *websocket.Conn, clientId, name string) (*Subscriber, error) {
	client := h.getClient(clientId)
	if client != nil {
		return client, nil
	}

	client, isNew := newSubscriber(h, socket, clientId, name)
	if isNew {
		h.register <- client

		go client.write()
		go client.read()
	}

	return client, nil
}

func (h *Hub) streaming(socket *websocket.Conn, clientId, name string) (*Subscriber, error) {
	client := h.getClient(clientId)
	if client != nil {
		return client, nil
	}

	client, isNew := newSubscriber(h, socket, clientId, name)
	if isNew {
		h.register <- client

		go client.write()
		go client.stream()
	}

	return client, nil
}

func (h *Hub) broadcast(channel, queue string, msg Message, ignored []string, from et.Json) {
	msg.From = from
	msg.Ignored = ignored

	n := 0
	_channel := h.getChannel(channel)
	if _channel != nil {
		n = _channel.broadcast(msg, ignored)
	}

	_queue := h.getQueue(channel, queue)
	if _queue != nil {
		n = _queue.broadcast(msg, ignored)
	}

	logs.Logf(ServiceName, "Broadcast channel:%s sent:%d", channel, n)
}
