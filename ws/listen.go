package ws

import (
	"bytes"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
)

// Listen a client message
func (c *Client) listen(messageType int, message []byte) {
	data, err := et.ToJson(message)
	if err != nil {
		data = et.Json{
			"type":    messageType,
			"message": bytes.NewBuffer(message).String(),
		}
	}

	_type := data.Str("type")
	switch _type {
	case "subscribe":
		channel := data.ValStr("", "channel")
		if channel == "" {
			c.SendMessage(et.Json{
				"ok":    false,
				"error": "Channel not found",
			})
			return
		}

		err := c.hub.Subscribe(c.Id, channel)
		if err != nil {
			c.SendMessage(et.Json{
				"ok":    false,
				"error": err.Error(),
			})
			return
		}

		c.SendMessage(et.Json{
			"ok":      true,
			"message": "Subscribed to channel " + channel,
		})
	case "unsubscribe":
		channel := data.ValStr("", "channel")
		if channel == "" {
			c.SendMessage(et.Json{
				"ok":    false,
				"error": "Channel not found",
			})
			return
		}

		err := c.hub.Unsubscribe(c.Id, channel)
		if err != nil {
			c.SendMessage(et.Json{
				"ok":    false,
				"error": err.Error(),
			})
			return
		}

		c.SendMessage(et.Json{
			"ok":      true,
			"message": "Unsubscribed from channel " + channel,
		})
	case "params":
		params := data.Json("params")
		if params.Emptyt() {
			c.SendMessage(et.Json{
				"ok":    false,
				"error": "Params not found",
			})
			return
		}

		name := params.ValStr("", "name")
		if name != "" {
			c.Name = name
		}

		c.SetParams(params)
		c.SendMessage(et.Json{
			"ok":    false,
			"error": "Name not found",
		})
	case "ping":
		c.SendMessage(et.Json{
			"ok":      true,
			"message": "pong",
		})
	case "message":
		clientId := data.ValStr("-1", "client_id")
		if clientId == "-1" {
			c.SendMessage(et.Json{
				"ok":    false,
				"error": "client_id not found",
			})
			return
		}

		message := data.Get("message")
		if message == nil {
			c.SendMessage(et.Json{
				"ok":    false,
				"error": "Message not found",
			})
			return
		}

		msg := et.Json{
			"from":    c.Params,
			"message": message,
		}
		err := c.hub.SendMessage(clientId, msg)
		if err != nil {
			c.SendMessage(et.Json{
				"ok":    false,
				"error": err.Error(),
			})
			return
		}

		c.SendMessage(et.Json{
			"ok":      true,
			"message": "Message sent to " + clientId,
		})
	case "publish":
		channel := data.ValStr("", "channel")
		if channel == "" {
			c.SendMessage(et.Json{
				"ok":    false,
				"error": "Channel not found",
			})
			return
		}

		message := data.Get("message")
		if message == nil {
			c.SendMessage(et.Json{
				"ok":    false,
				"error": "Message not found",
			})
			return
		}

		msg := et.Json{
			"from":    c.Params,
			"message": message,
		}
		go c.hub.Publish(channel, msg, c.Id)

		c.SendMessage(et.Json{
			"ok":      true,
			"message": "Message published to " + channel,
		})
	}

	logs.Logf("Websocket", "Client %s message: %s", c.Id, data.ToString())
}
