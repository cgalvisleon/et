package jws

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/utility"
	"github.com/gorilla/websocket"
)

const (
	packageName = "WebSocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Hub struct {
	Subscribers     map[string]*Client             `json:"subscribers"`
	Channels        map[string]*Channel            `json:"channels"`
	register        chan *Client                   `json:"-"`
	unregister      chan *Client                   `json:"-"`
	onListener      []func(*Client, []byte)        `json:"-"`
	onConnection    []func(*Client)                `json:"-"`
	onDisconnection []func(*Client)                `json:"-"`
	onChannel       []func(Channel)                `json:"-"`
	onRemove        []func(string)                 `json:"-"`
	onPublish       []func(ch Channel, ms Message) `json:"-"`
	onSend          []func(to string, ms Message)  `json:"-"`
	mu              *sync.RWMutex                  `json:"-"`
	isStart         bool                           `json:"-"`
	isDebug         bool                           `json:"-"`
}

/**
* run
**/
func (s *Hub) run() {
	for {
		select {
		case client := <-s.register:
			s.onConnect(client)
		case client := <-s.unregister:
			s.onDisconnect(client)
		}
	}
}

/**
* defOnConnect
* @param *Client client
**/
func (s *Hub) onConnect(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Subscribers[client.Name] = client
	logs.Logf(packageName, "Client connected: %s", client.Name)
	for _, fn := range s.onConnection {
		fn(client)
	}
}

/**
* onDisconnect
* @param *Client client
**/
func (s *Hub) onDisconnect(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.Subscribers[client.Name]
	if ok {
		client.Status = Disconnected
		for _, fn := range s.onDisconnection {
			fn(client)
		}

		delete(s.Subscribers, client.Name)
		client.retire()
	}
}

/**
* Start
**/
func (s *Hub) Start() {
	if s.isStart {
		return
	}

	logs.Logf(packageName, "Hub started")
	s.isStart = true
	go s.run()
}

/**
* Close
**/
func (s *Hub) Close() {
	s.isStart = false

	logs.Log(packageName, "Shutting down server...")
}

/**
* SetDebug
* @param debug bool
**/
func (s *Hub) SetDebug(debug bool) {
	s.isDebug = debug
}

/**
* Connect
* @param socket *websocket.Conn, context.Context
* @return *Client, error
**/
func (s *Hub) Connect(socket *websocket.Conn, ctx context.Context) (*Client, error) {
	if !s.isStart {
		return nil, errors.New(msg.MSG_HUB_NOT_STARTED)
	}

	username, ok := ctx.Value("username").(string)
	if !ok || !utility.ValidStr(username, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "username")
	}

	s.mu.RLock()
	client, exists := s.Subscribers[username]
	s.mu.RUnlock()
	if exists {
		outbound, done := client.rebind(socket)
		go client.read(socket, done)
		go client.write(socket, outbound, done)
		return client, nil
	}

	client = newSubscriber(s, ctx, username, socket)
	s.register <- client

	go client.write(client.socket, client.outbound, client.done)
	go client.read(client.socket, client.done)
	go client.SendHola()

	return client, nil
}

/**
* OnListener
* @param fn func(*Client, []byte)
**/
func (s *Hub) OnListener(fn func(*Client, []byte)) {
	s.onListener = append(s.onListener, fn)
}

/**
* OnConnection
* @param fn func(*Client)
**/
func (s *Hub) OnConnection(fn func(*Client)) {
	s.onConnection = append(s.onConnection, fn)
}

/**
* OnDisconnection
* @param fn func(*Client)
**/
func (s *Hub) OnDisconnection(fn func(*Client)) {
	s.onDisconnection = append(s.onDisconnection, fn)
}

/**
* OnChannel
* @param fn func(Channel)
**/
func (s *Hub) OnChannel(fn func(Channel)) {
	s.onChannel = append(s.onChannel, fn)
}

/**
* OnRemove
* @param fn func(string)
**/
func (s *Hub) OnRemove(fn func(string)) {
	s.onRemove = append(s.onRemove, fn)
}

/**
* OnPublish
* @param fn func(ch Channel, ms Message)
 */
func (s *Hub) OnPublish(fn func(ch Channel, ms Message)) {
	s.onPublish = append(s.onPublish, fn)
}

/**
* OnSend
* @param fn func(to string, ms Message)
 */
func (s *Hub) OnSend(fn func(to string, ms Message)) {
	s.onSend = append(s.onSend, fn)
}

/**
* addChannel
* @param *Channel ch
**/
func (s *Hub) addChannel(ch *Channel) {
	s.mu.Lock()
	s.Channels[ch.Name] = ch
	s.mu.Unlock()
	for _, fn := range s.onChannel {
		fn(*ch)
	}
}

/**
* SendTo
* @param to []string, message Message
**/
func (s *Hub) SendTo(to []string, message Message) ([]string, error) {
	result := []string{}

	sendObject := func(client *Client, m interface{}) {
		client.SendMessage(m)
		for _, fn := range s.onSend {
			fn(client.Name, message)
		}
	}

	s.mu.RLock()
	clients := make(map[string]*Client, len(to))
	for _, username := range to {
		if client, ok := s.Subscribers[username]; ok {
			clients[username] = client
		}
	}
	s.mu.RUnlock()

	for _, username := range to {
		client, ok := clients[username]
		if ok {
			idx := slices.IndexFunc(message.Ignored, func(user string) bool {
				return user == username
			})
			if idx != -1 {
				continue
			}

			if message.Verified {
				sendObject(client, message.Message)
			} else {
				go sendObject(client, message.Message)
			}

			result = append(result, username)
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf(msg.MSG_USER_NOT_FOUND, strings.Join(to, ", "))
	}

	return result, nil
}

/**
* Topic
* @param channel string
* @return *Channel
*
 */
func (s *Hub) Topic(channel string) *Channel {
	ch := newChannel(channel, TpTopic)
	s.addChannel(ch)
	return ch
}

/**
* Queue
* @param channel string
* @return *Channel
**/
func (s *Hub) Queue(channel string) *Channel {
	ch := newChannel(channel, TpQueue)
	s.addChannel(ch)
	return ch
}

/**
* Stack
* @param channel string
* @return *Channel
**/
func (s *Hub) Stack(channel string) *Channel {
	ch := newChannel(channel, TpStack)
	s.addChannel(ch)
	return ch
}

/**
* Remove
* @param channel string
* @return error
**/
func (s *Hub) Remove(channel string) error {
	s.mu.Lock()
	ch, ok := s.Channels[channel]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf(msg.MSG_CHANNEL_NOT_FOUND, channel)
	}
	delete(s.Channels, channel)
	s.mu.Unlock()

	for _, subscribe := range ch.list() {
		s.mu.RLock()
		client, ok := s.Subscribers[subscribe]
		s.mu.RUnlock()
		if ok {
			client.removeChannel(channel)
		}
	}

	for _, fn := range s.onRemove {
		fn(channel)
	}
	return nil
}

/**
* Subscribe
* @param cache string, subscribe string
* @return error
**/
func (s *Hub) Subscribe(channel string, subscribe string) error {
	s.mu.RLock()
	ch, ok := s.Channels[channel]
	if !ok {
		s.mu.RUnlock()
		return fmt.Errorf(msg.MSG_CHANNEL_NOT_FOUND, channel)
	}

	client, ok := s.Subscribers[subscribe]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf(msg.MSG_USER_NOT_FOUND, subscribe)
	}

	ch.subscriber(client.Name)
	client.addChannel(ch.Name)
	return nil
}

/**
* Unsubscribe
* @param cache string, subscribe string
* @return error
**/
func (s *Hub) Unsubscribe(cache string, subscribe string) error {
	s.mu.RLock()
	ch, ok := s.Channels[cache]
	if !ok {
		s.mu.RUnlock()
		return fmt.Errorf(msg.MSG_CHANNEL_NOT_FOUND, cache)
	}

	client, ok := s.Subscribers[subscribe]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf(msg.MSG_USER_NOT_FOUND, subscribe)
	}

	ch.remove(client.Name)
	client.removeChannel(ch.Name)
	return nil
}

/**
* Publish
* @param channel string, message Message
**/
func (s *Hub) Publish(channel string, message Message) ([]string, error) {
	s.mu.RLock()
	ch, ok := s.Channels[channel]
	s.mu.RUnlock()
	if !ok {
		return []string{}, fmt.Errorf(msg.MSG_CHANNEL_NOT_FOUND, channel)
	}

	for _, fn := range s.onPublish {
		fn(*ch, message)
	}

	switch ch.Type {
	case TpQueue:
		subscribe, ok := ch.nextQueueTarget()
		if !ok {
			return []string{}, fmt.Errorf(msg.MSG_USER_NOT_FOUND, channel)
		}
		return s.SendTo([]string{subscribe}, message)
	case TpStack:
		subscribe, ok := ch.nextStackTarget()
		if !ok {
			return []string{}, fmt.Errorf(msg.MSG_USER_NOT_FOUND, channel)
		}
		return s.SendTo([]string{subscribe}, message)
	case TpTopic:
		return s.SendTo(ch.list(), message)
	}

	return []string{}, fmt.Errorf(msg.MSG_USER_NOT_FOUND, channel)
}
