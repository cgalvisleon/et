package ws

import (
	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/redis/go-redis/v9"
)

type AdapterRedis struct {
	hub  *Hub
	conn *cache.Conn
}

func NewRedisAdapter() Adapter {
	return &AdapterRedis{}
}

/**
* ConnectTo
* @param params et.Json
* @return error
**/
func (s *AdapterRedis) ConnectTo(hub *Hub, params et.Json) error {
	if s.conn != nil {
		return nil
	}

	host := params.Str("host")
	if host == "" {
		return utility.NewError("WS Adapter, Redis host is required")
	}

	password := params.Str("password")
	dbname := params.Int("dbname")
	result, err := cache.ConnectTo(host, password, dbname)
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
func (s *AdapterRedis) Close() {}

/**
* Subscribed
* @param channel string
**/
func (s *AdapterRedis) Subscribed(channel string) {
	channel = clusterChannel(channel)
	s.conn.Sub(channel, func(receive *redis.Message) {
		data := []byte(receive.Payload)
		msg, err := DecodeMessage(data)
		if err != nil {
			return
		}

		if msg.Tp == TpDirect {
			s.hub.send(msg.To, msg)
		} else {
			s.hub.publish(msg.Channel, msg.Queue, msg, msg.Ignored, msg.From)
		}
	})
}

/**
* UnSubscribed
* @param sub channel string
**/
func (s *AdapterRedis) UnSubscribed(channel string) {
	channel = clusterChannel(channel)
	s.conn.Unsub(channel)
}

/**
* Publish
* @param sub channel string
**/
func (s *AdapterRedis) Publish(channel string, msg Message) error {
	channel = clusterChannel(channel)
	bt, err := msg.Encode()
	if err != nil {
		return err
	}

	return s.conn.Pub(channel, bt)
}
