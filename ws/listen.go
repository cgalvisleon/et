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
		c.hub.Subscribe(c.Id, data.Str("channel"))
	}

	logs.Logf("Websocket", "Client %s message: %s", c.Id, data.ToString())

	c.sendMessage([]byte(data.ToString()))
}
