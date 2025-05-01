package ws

import (
	"net/http"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
)

type AdapterWS struct {
	hub  *Hub
	conn *Client
}

func NewWSAdapter() Adapter {
	return &AdapterWS{}
}

/**
* ConnectTo
* @param params et.Json
* @return error
**/
func (s *AdapterWS) ConnectTo(hub *Hub, params et.Json) error {
	if s.conn != nil {
		return nil
	}

	url := params.Str("url")
	if url == "" {
		return logs.Alertm("WS Adapter, url is required")
	}

	username := params.Str("username")
	if username == "" {
		return logs.Alertm("WS Adapter, username is required")
	}

	err := config.Validate([]string{
		"WS_USERNAME",
		"WS_PASSWORD",
		"WS_HOST",
	})
	if err != nil {
		return err
	}

	password := config.String("WS_PASSWORD", "")
	reconnect := config.Int("WS_RECONNECT", 3)
	name := params.ValStr("Anonime", "name")
	result, err := Login(&ClientConfig{
		ClientId:  utility.UUID(),
		Name:      name,
		Url:       url,
		Reconnect: reconnect,
		Header: http.Header{
			"username": []string{username},
			"password": []string{password},
		},
	})
	if err != nil {
		return err
	}

	s.hub = hub
	s.conn = result

	return nil
}

/**
* Close
**/
func (s *AdapterWS) Close() {}

/**
* Subscribed
* @param channel string
**/
func (s *AdapterWS) Subscribed(channel string) {
	channel = clusterChannel(channel)
	go s.conn.Subscribe(channel, func(msg Message) {
		if msg.Tp == TpDirect {
			s.hub.send(msg.Id, msg)
		} else {
			s.hub.publish(msg.Channel, msg.Queue, msg, msg.Ignored, msg.From)
		}
	})
}

/**
* UnSubscribed
* @param sub channel string
**/
func (s *AdapterWS) UnSubscribed(channel string) {
	channel = clusterChannel(channel)
	s.conn.Unsubscribe(channel)
}

/**
* Publish
* @param sub channel string
**/
func (s *AdapterWS) Publish(channel string, msg Message) error {
	channel = clusterChannel(channel)
	return s.conn.Publish(channel, msg)
}
