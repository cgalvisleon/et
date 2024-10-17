package rt

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/et/ws"
	"github.com/gorilla/websocket"
)

/**
* read
**/
func (c *ClientWS) read() {
	done := make(chan struct{})

	go func() {
		defer close(done)

		for {
			_, data, err := c.socket.ReadMessage()
			if err != nil {
				logs.Alert(err)
				c.connected = false
				return
			}

			msg, err := ws.DecodeMessage(data)
			if err != nil {
				logs.Alert(err)
				return
			}

			f, ok := c.channels[msg.Channel]
			if ok {
				f(msg)
			}
		}
	}()
}

/**
* send
* @param message ws.ws.Message
* @return error
**/
func (c *ClientWS) send(message ws.Message) error {
	if c.socket == nil {
		return logs.Alertm(ERR_NOT_CONNECT_WS)
	}

	msg, err := message.Encode()
	if err != nil {
		return err
	}

	err = conn.socket.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		return err
	}

	return nil
}

/**
* wsConnect
* @param ws.Message
**/
func (c *ClientWS) wsConnect(ws.Message) {

}

/**
* IsConnected
* @return bool
**/
func IsConnected() bool {
	if conn == nil {
		return false
	}

	return conn.connected
}

/**
* From
* @return et.Json
**/
func From() et.Json {
	return et.Json{
		"id":   conn.ClientId,
		"name": conn.Name,
	}
}

/**
* Ping
**/
func Ping() {
	msg := ws.NewMessage(From(), et.Json{}, ws.TpPing)

	conn.send(msg)
}

/**
* SetFrom
* @param params et.Json
* @return error
**/
func SetFrom(name string) error {
	if !utility.ValidName(name) {
		return logs.Alertm(ERR_INVALID_NAME)
	}

	conn.Name = name
	msg := ws.NewMessage(From(), From(), ws.TpSetFrom)
	return conn.send(msg)
}

/**
* Subscribe to a channel
* @param channel string
* @param reciveFn func(message.ws.Message)
**/
func Subscribe(channel string, reciveFn func(ws.Message)) {
	conn.channels[channel] = reciveFn

	msg := ws.NewMessage(From(), et.Json{}, ws.TpSubscribe)
	msg.Channel = channel

	conn.send(msg)
}

/**
* Queue to a channel
* @param channel string
* @param reciveFn func(message.ws.Message)
**/
func Queue(channel, queue string, reciveFn func(ws.Message)) {
	conn.channels[channel] = reciveFn

	msg := ws.NewMessage(From(), et.Json{}, ws.TpQueue)
	msg.Channel = channel
	msg.Queue = queue

	conn.send(msg)
}

/**
* Unsubscribe to a channel
* @param channel string
**/
func Unsubscribe(channel string) {
	delete(conn.channels, channel)

	msg := ws.NewMessage(From(), et.Json{}, ws.TpUnsubscribe)
	msg.Channel = channel

	conn.send(msg)
}

/**
* Publish a message to a channel
* @param channel string
* @param message interface{}
**/
func Publish(channel string, message interface{}) {
	msg := ws.NewMessage(From(), message, ws.TpPublish)
	msg.Ignored = []string{conn.ClientId}
	msg.Channel = channel

	conn.send(msg)
}

/**
* SendMessage
* @param clientId string
* @param message interface{}
* @return error
**/
func SendMessage(clientId string, message interface{}) error {
	msg := ws.NewMessage(From(), message, ws.TpDirect)
	msg.Ignored = []string{conn.ClientId}
	msg.To = clientId

	return conn.send(msg)
}
