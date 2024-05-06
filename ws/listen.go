package ws

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/pubsub"
)

// Listen a client message
func (c *Client) listen(messageType int, message []byte) {
	msg, err := DecodeMessage(message)
	if err != nil {
		msg = NewMessage(*c.hub.Params, et.Json{
			"ok":      false,
			"message": err.Error(),
		})
		c.sendMessage(msg)
		return
	}

	tp := msg.Type()
	switch tp {
	case pubsub.TpPing:
		msg := NewMessage(*c.hub.Params, et.Json{
			"ok":      true,
			"message": "pong",
		})
		c.sendMessage(msg)
	case pubsub.TpParams:
		params, err := msg.Json()
		if err != nil {
			msg := NewMessage(*c.hub.Params, et.Json{
				"ok":      false,
				"message": err.Error(),
			})
			c.sendMessage(msg)
			return
		}

		name := params.ValStr("", "name")
		if name != "" {
			c.Name = name
		}

		params.Set("id", c.Id)
		params.Set("name", c.Name)
		c.setParams(params)
		msg := NewMessage(*c.hub.Params, et.Json{
			"ok":      true,
			"message": PARAMS_UPDATED,
		})
		c.sendMessage(msg)
	case pubsub.TpSubscribe:
		channel := msg.Channel
		if channel == "" {
			msg := NewMessage(*c.hub.Params, et.Json{
				"ok":      false,
				"message": ERR_CHANNEL_EMPTY,
			})
			c.sendMessage(msg)
			return
		}

		err := c.hub.Subscribe(c.Id, channel)
		if err != nil {
			msg := NewMessage(*c.hub.Params, et.Json{
				"ok":      false,
				"message": err.Error(),
			})
			c.sendMessage(msg)
			return
		}

		msg := NewMessage(*c.hub.Params, et.Json{
			"ok":      true,
			"message": "Subscribed to channel " + channel,
		})
		c.sendMessage(msg)
	case pubsub.TpStack:
		channel := msg.Channel
		if channel == "" {
			msg := NewMessage(*c.hub.Params, et.Json{
				"ok":      false,
				"message": ERR_CHANNEL_EMPTY,
			})
			c.sendMessage(msg)
			return
		}

		err := c.hub.Stack(c.Id, channel)
		if err != nil {
			msg := NewMessage(*c.hub.Params, et.Json{
				"ok":      false,
				"message": err.Error(),
			})
			c.sendMessage(msg)
			return
		}

		msg := NewMessage(*c.hub.Params, et.Json{
			"ok":      true,
			"message": "Stacked to channel " + channel,
		})
		c.sendMessage(msg)
	case pubsub.TpUnsubscribe:
		channel := msg.Channel
		if channel == "" {
			msg := NewMessage(*c.hub.Params, et.Json{
				"ok":      false,
				"message": ERR_CHANNEL_EMPTY,
			})
			c.sendMessage(msg)
			return
		}

		err := c.hub.Subscribe(c.Id, channel)
		if err != nil {
			msg := NewMessage(*c.hub.Params, et.Json{
				"ok":      false,
				"message": err.Error(),
			})
			c.sendMessage(msg)
			return
		}

		msg := NewMessage(*c.hub.Params, et.Json{
			"ok":      true,
			"message": "Unsubscribed from channel " + channel,
		})
		c.sendMessage(msg)
	case pubsub.TpPublish:
		channel := msg.Channel
		if channel == "" {
			msg := NewMessage(*c.hub.Params, et.Json{
				"ok":      false,
				"message": ERR_CHANNEL_EMPTY,
			})
			c.sendMessage(msg)
			return
		}

		go c.hub.Publish(channel, msg, []string{c.Id}, *c.Params)
		msg := NewMessage(*c.hub.Params, et.Json{
			"ok":      true,
			"message": "Message published to " + channel,
		})
		c.sendMessage(msg)
	case pubsub.TpDirect:
		clientId := msg.to

		msg.From = *c.Params
		err := c.hub.SendMessage(clientId, msg)
		if err != nil {
			msg := NewMessage(*c.hub.Params, et.Json{
				"ok":      false,
				"message": err.Error(),
			})
			c.sendMessage(msg)
		}

		msg := NewMessage(*c.hub.Params, et.Json{
			"ok":      true,
			"message": "Message sent to " + clientId,
		})
		c.sendMessage(msg)
	default:
		msg := NewMessage(*c.hub.Params, et.Json{
			"ok":      false,
			"message": ERR_MESSAGE_UNFORMATTED,
		})
		c.sendMessage(msg)
	}

	logs.Logf("Websocket", "Client %s message: %s", c.Id, msg.ToString())
}
