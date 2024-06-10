package ws

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	m "github.com/cgalvisleon/et/message"
)

// Listen a client message
func (c *Client) listen(messageType int, message []byte) {
	send := func(ok bool, message string) {
		msg := NewMessage(c.hub.Identify(), et.Json{
			"ok":      ok,
			"message": message,
		})
		c.sendMessage(msg)
	}

	msg, err := DecodeMessage(message)
	if err != nil {
		send(false, err.Error())
		return
	}

	tp := msg.Type()
	switch tp {
	case m.TpPing:
		send(true, "pong")
	case m.TpParams:
		params, err := msg.Json()
		if err != nil {
			send(false, err.Error())
			return
		}

		name := params.ValStr("", "name")
		if name != "" {
			c.Name = name
		}

		send(true, PARAMS_UPDATED)
	case m.TpSubscribe:
		channel := msg.Channel
		if channel == "" {
			send(false, ERR_CHANNEL_EMPTY)
			return
		}

		err := c.hub.Subscribe(c.Id, channel)
		if err != nil {
			send(false, err.Error())
			return
		}

		send(true, "Subscribed to channel "+channel)
	case m.TpStack:
		channel := msg.Channel
		if channel == "" {
			send(false, ERR_CHANNEL_EMPTY)
			return
		}

		err := c.hub.Stack(c.Id, channel)
		if err != nil {
			send(false, err.Error())
			return
		}

		send(true, "Stacked to channel "+channel)
	case m.TpUnsubscribe:
		channel := msg.Channel
		if channel == "" {
			send(false, ERR_CHANNEL_EMPTY)
			return
		}

		err := c.hub.Subscribe(c.Id, channel)
		if err != nil {
			send(false, err.Error())
			return
		}

		send(true, "Unsubscribed from channel "+channel)
	case m.TpPublish:
		channel := msg.Channel
		if channel == "" {
			send(false, ERR_CHANNEL_EMPTY)
			return
		}

		go c.hub.Publish(channel, msg, []string{c.Id}, c.Identify())
		send(true, "Message published to "+channel)
	case m.TpDirect:
		clientId := msg.to

		msg.From = c.Identify()
		err := c.hub.SendMessage(clientId, msg)
		if err != nil {
			send(false, err.Error())
			return
		}

		send(true, "Message sent to "+clientId)
	default:
		send(false, ERR_MESSAGE_UNFORMATTED)
	}

	logs.Logf("Websocket", "Client %s message: %s", c.Id, msg.ToString())
}
