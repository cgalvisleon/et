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
		msg = NewMessage(*c.hub.Params, *c.Params, et.Json{
			"ok": false,
			"result": et.Json{
				"message": err.Error(),
			},
		})
		c.sendMessage(msg)
		return
	}

	_type := msg.Type()
	switch _type {
	case pubsub.TpPing:
		msg := NewMessage(*c.hub.Params, *c.Params, et.Json{
			"ok": true,
			"result": et.Json{
				"message": "pong",
			},
		})
		c.sendMessage(msg)
	case pubsub.TpParams:
		params, err := msg.Json()
		if err != nil {
			msg := NewMessage(*c.hub.Params, *c.Params, et.Json{
				"ok": false,
				"result": et.Json{
					"message": err.Error(),
				},
			})
			c.sendMessage(msg)
			return
		}

		if params.Emptyt() {
			msg := NewMessage(*c.hub.Params, *c.Params, et.Json{
				"ok": false,
				"result": et.Json{
					"message": ERR_PARAM_NOT_FOUND,
				},
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
		msg := NewMessage(*c.hub.Params, *c.Params, et.Json{
			"ok": true,
			"result": et.Json{
				"message": PARAMS_UPDATED,
			},
		})
		c.sendMessage(msg)
	case pubsub.TpSystem:
		params, err := msg.Json()
		if err != nil {
			msg := NewMessage(*c.hub.Params, *c.Params, et.Json{
				"ok": false,
				"result": et.Json{
					"message": err.Error(),
				},
			})
			c.sendMessage(msg)
			return
		}

		if params.Emptyt() {
			msg := NewMessage(*c.hub.Params, *c.Params, et.Json{
				"ok": false,
				"result": et.Json{
					"message": ERR_PARAM_NOT_FOUND,
				},
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
		c.hub.SetParams(params)
		msg := NewMessage(*c.hub.Params, *c.Params, et.Json{
			"ok": true,
			"result": et.Json{
				"message": PARAMS_UPDATED,
			},
		})
		c.sendMessage(msg)
	case pubsub.TpSubscribe:
		channel := msg.Channel
		if channel == "" {
			msg := NewMessage(*c.hub.Params, *c.Params, et.Json{
				"ok": false,
				"result": et.Json{
					"message": ERR_CHANNEL_EMPTY,
				},
			})
			c.sendMessage(msg)
			return
		}

		err := c.hub.Subscribe(c.Id, channel)
		if err != nil {
			msg := NewMessage(*c.hub.Params, *c.Params, et.Json{
				"ok": false,
				"result": et.Json{
					"message": err.Error(),
				},
			})
			c.sendMessage(msg)
			return
		}

		msg := NewMessage(*c.hub.Params, *c.Params, et.Json{
			"ok": true,
			"result": et.Json{
				"message": "Subscribed to channel " + channel,
			},
		})
		c.sendMessage(msg)
	case pubsub.TpUnsubscribe:
		channel := msg.Channel
		if channel == "" {
			msg := NewMessage(*c.hub.Params, *c.Params, et.Json{
				"ok": false,
				"result": et.Json{
					"message": ERR_CHANNEL_EMPTY,
				},
			})
			c.sendMessage(msg)
			return
		}

		err := c.hub.Subscribe(c.Id, channel)
		if err != nil {
			msg := NewMessage(*c.hub.Params, *c.Params, et.Json{
				"ok": false,
				"result": et.Json{
					"message": err.Error(),
				},
			})
			c.sendMessage(msg)
			return
		}

		msg := NewMessage(*c.hub.Params, *c.Params, et.Json{
			"ok": true,
			"result": et.Json{
				"message": "Unsubscribed from channel " + channel,
			},
		})
		c.sendMessage(msg)
	case pubsub.TpPublish:
		channel := msg.Channel
		if channel == "" {
			msg := NewMessage(*c.hub.Params, *c.Params, et.Json{
				"ok": false,
				"result": et.Json{
					"message": ERR_CHANNEL_EMPTY,
				},
			})
			c.sendMessage(msg)
			return
		}

		go c.hub.Publish(channel, msg, []string{c.Id}, *c.Params)
		msg := NewMessage(*c.hub.Params, *c.Params, et.Json{
			"ok": true,
			"result": et.Json{
				"message": "Message published to " + channel,
			},
		})
		c.sendMessage(msg)
	default:
		clientId := msg.To.ValStr("-1", "client_id")
		if clientId == "-1" {
			msg := NewMessage(*c.hub.Params, *c.Params, et.Json{
				"ok": false,
				"result": et.Json{
					"message": ERR_CLIENT_ID_EMPTY,
				},
			})
			c.sendMessage(msg)
			return
		}

		msg.From = *c.Params
		err := c.hub.SendMessage(clientId, msg)
		if err != nil {
			msg := NewMessage(*c.hub.Params, *c.Params, et.Json{
				"ok": false,
				"result": et.Json{
					"message": err.Error(),
				},
			})
			c.sendMessage(msg)
		}
	}

	logs.Logf("Websocket", "Client %s message: %s", c.Id, msg.ToString())
}
