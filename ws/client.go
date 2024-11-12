package ws

import (
	"net/http"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/race"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
	"github.com/gorilla/websocket"
)

type ClientConfig struct {
	ClientId  string
	Name      string
	Url       string
	Header    http.Header
	Reconcect int
}

/**
* From
* @return et.Json
**/
func (s *ClientConfig) From() et.Json {
	return et.Json{
		"id":   s.ClientId,
		"name": s.Name,
	}
}

type Client struct {
	Channels          map[string]func(Message)
	Attempts          *race.Value
	Connected         *race.Value
	clientId          string
	name              string
	url               string
	header            http.Header
	reconcect         int
	directMessage     func(Message)
	reconnectCallback func(*Client)
	socket            *websocket.Conn
	mutex             *sync.Mutex
}

/**
* NewClient
* @config config ConectPatams
* @return erro
**/
func NewClient(config *ClientConfig) (*Client, error) {
	result := &Client{
		Channels:  make(map[string]func(Message)),
		Attempts:  race.NewValue(0),
		Connected: race.NewValue(false),
		mutex:     &sync.Mutex{},
		clientId:  config.ClientId,
		name:      config.Name,
		url:       config.Url,
		header:    config.Header,
		reconcect: config.Reconcect,
	}

	err := result.Connect()
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Client) setChannel(channel string, reciveFn func(Message)) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.Channels[channel] = reciveFn
}

func (c *Client) getChannel(channel string) (func(Message), bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	resul, ok := c.Channels[channel]
	return resul, ok
}

func (c *Client) deleteChannel(channel string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.Channels, channel)
}

/**
* Connect
* @return error
**/
func (c *Client) Connect() error {
	if c.Connected.Bool() {
		return nil
	}

	path := strs.Format(`%s?clientId=%s&name=%s`, c.url, c.clientId, c.name)
	socket, _, err := websocket.DefaultDialer.Dial(path, c.header)
	if err != nil {
		return err
	}

	c.socket = socket
	c.Connected.Set(true)
	c.Attempts.Set(0)

	go c.Listener()

	logs.Log(ServiceName, "Connected websocket")

	return nil
}

func (c *Client) Reconnect() {
	if c.reconcect == 0 {
		logs.Log(ServiceName, "Reconnect disabled")
		return
	}

	ticker := time.NewTicker(time.Duration(c.reconcect) * time.Second)
	for range ticker.C {
		if !c.Connected.Bool() {
			err := c.Connect()
			if err != nil {
				c.Attempts.Increase(1)
				logs.Logf(ServiceName, `Reconnect attempts:%d`, c.Attempts.Int())
			} else {
				c.ReconnectCallback()
			}
		}
	}
}

/**
* Close
**/
func (c *Client) Close() {
	if c.socket == nil {
		return
	}

	c.Channels = nil
	c.socket.Close()
}

/**
* read
**/
func (c *Client) Listener() {
	done := make(chan struct{})

	go func() {
		defer close(done)

		for {
			_, data, err := c.socket.ReadMessage()
			if err != nil {
				c.Connected.Set(false)
				c.Reconnect()
				return
			}

			msg, err := DecodeMessage(data)
			if err != nil {
				logs.Alert(err)
				return
			}

			f, ok := c.getChannel(msg.Channel)
			if ok {
				f(msg)
			} else {
				c.DirectMessage(msg)
			}
		}
	}()
}

/**
* SetDirectMessage
* @param reciveFn func(message.Message)
**/
func (c *Client) SetDirectMessage(reciveFn func(Message)) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.directMessage = reciveFn
}

/**
* DirectMessage
* @param msg Message
**/
func (c *Client) DirectMessage(msg Message) {
	if c.directMessage != nil {
		c.directMessage(msg)
	}
}

/**
* SetReconnectCallback
* @param reciveFn func()
**/
func (c *Client) SetReconnectCallback(reciveFn func(c *Client)) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.reconnectCallback = reciveFn
}

/**
* ReconnectCallback
**/
func (c *Client) ReconnectCallback() {
	if c.reconnectCallback != nil {
		c.reconnectCallback(c)
	}
}

/**
* send
* @param message Message
* @return error
**/
func (c *Client) send(message Message) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.socket == nil {
		return logs.Alertm(ERR_NOT_CONNECT_WS)
	}

	if !c.Connected.Bool() {
		return logs.Alertm(ERR_CLIENT_DISCONNECTED)
	}

	msg, err := message.Encode()
	if err != nil {
		return err
	}

	err = c.socket.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		return err
	}

	return nil
}

/**
* From
* @return et.Json
**/
func (c *Client) From() et.Json {
	return et.Json{
		"id":   c.clientId,
		"name": c.name,
	}
}

/**
* Ping
**/
func (c *Client) Ping() {
	msg := NewMessage(c.From(), et.Json{}, TpPing)

	c.send(msg)
}

/**
* SetFrom
* @param config et.Json
* @return error
**/
func (c *Client) SetFrom(name string) error {
	if !utility.ValidName(name) {
		return logs.Alertm(ERR_INVALID_NAME)
	}

	c.name = name
	msg := NewMessage(c.From(), c.From(), TpSetFrom)
	return c.send(msg)
}

/**
* Subscribe to a channel
* @param channel string
* @param reciveFn func(message.Message)
**/
func (c *Client) Subscribe(channel string, reciveFn func(Message)) {
	c.setChannel(channel, reciveFn)

	msg := NewMessage(c.From(), et.Json{}, TpSubscribe)
	msg.Channel = channel

	c.send(msg)
}

/**
* Queue to a channel
* @param channel, queue string
* @param reciveFn func(message.Message)
**/
func (c *Client) Queue(channel, queue string, reciveFn func(Message)) {
	c.setChannel(channel, reciveFn)

	msg := NewMessage(c.From(), et.Json{}, TpStack)
	msg.Channel = channel
	msg.Data = queue

	c.send(msg)
}

/**
* Stack to a channel
* @param channel string
* @param reciveFn func(message.Message)
**/
func (c *Client) Stack(channel string, reciveFn func(Message)) {
	c.Queue(channel, utility.QUEUE_STACK, reciveFn)
}

/**
* Unsubscribe to a channel
* @param channel string
**/
func (c *Client) Unsubscribe(channel string) {
	c.deleteChannel(channel)

	msg := NewMessage(c.From(), et.Json{}, TpUnsubscribe)
	msg.Channel = channel

	c.send(msg)
}

/**
* Publish a message to a channel
* @param channel string
* @param message interface{}
**/
func (c *Client) Publish(channel string, message interface{}) {
	msg := NewMessage(c.From(), message, TpPublish)
	msg.Ignored = []string{c.clientId}
	msg.Channel = channel

	c.send(msg)
}

/**
* SendMessage
* @param clientId string
* @param message interface{}
* @return error
**/
func (c *Client) SendMessage(clientId string, message interface{}) error {
	msg := NewMessage(c.From(), message, TpDirect)
	msg.Ignored = []string{c.clientId}
	msg.To = clientId

	return c.send(msg)
}
