package ws

import (
	"context"
	"encoding/json"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/redis/go-redis/v9"
)

type TpBroadcast int

const (
	BroadcastAll TpBroadcast = iota
	BroadcastDirect
)

/**
* MessageBroadcast is a struct to manage the message to broadcast
**/
type MessageBroadcast struct {
	Kind    TpBroadcast `json:"kind"`
	To      string      `json:"channel"`
	Msg     Message     `json:"msg"`
	Ignored []string    `json:"ignored"`
	From    et.Json     `json:"from"`
}

/**
* Encode return the message as byte array
* @return []byte
* @return error
**/
func (m MessageBroadcast) Encode() ([]byte, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	return b, nil
}

/**
* decodeMessageBroadcat return a MessageBroadcast from a byte array
* @param data []byte
* @return MessageBroadcast
* @return error
**/
func decodeMessageBroadcat(data []byte) (MessageBroadcast, error) {
	var m MessageBroadcast
	err := json.Unmarshal(data, &m)
	if err != nil {
		return MessageBroadcast{}, err
	}

	return m, nil
}

/**
* RedisAdapter is a struct to manage the Redis connection
* to broadcast messages to all clients in the cluster hub
**/
type RedisAdapterParams struct {
	Addr     string
	Password string
	DB       int
}

type RedisAdapter struct {
	db      *redis.Client
	ctx     context.Context
	channel string
}

/**
* NewRedisAdapter return a new RedisAdapter
* @param params *RedisAdapterParams
* @return *RedisAdapter
* @return error
**/
func NewRedisAdapter(params *RedisAdapterParams) (*RedisAdapter, error) {
	db := redis.NewClient(&redis.Options{
		Addr:     params.Addr,
		Password: params.Password,
		DB:       params.DB,
	})

	return &RedisAdapter{
		db:      db,
		ctx:     context.Background(),
		channel: "broadcast",
	}, nil
}

/**
* RedisAdapter is a struct to manage the Redis connection
* to broadcast messages to all clients in the cluster hub
* @param params *RedisAdapterParams
* @return error
**/
func (h *Hub) RedisAdapter(params *RedisAdapterParams) error {
	adapter, err := NewRedisAdapter(params)
	if err != nil {
		return err
	}
	h.adapter = adapter
	go h.adapter.subscribe(adapter.channel, h.listend)

	logs.Log("WebSocket", "RedisAdapter is ready")

	return nil
}

/**
* Close the RedisAdapter
* @return error
**/
func (a *RedisAdapter) Close() error {
	return a.db.Close()
}

/**
* publish a message to the Redis channel
* @param message MessageBroadcast
* @return error
**/
func (a *RedisAdapter) publish(message MessageBroadcast) error {
	if a.db == nil {
		return logs.Alertm(ERR_REDISADAPTER_NOT_FOUND)
	}

	err := a.db.Publish(a.ctx, a.channel, message).Err()
	if err != nil {
		return err
	}

	return nil
}

/**
* subscribe to the Redis channel
* @param channel string
* @param f func(interface{})
**/
func (a *RedisAdapter) subscribe(channel string, f func(interface{})) {
	if a.db == nil {
		return
	}

	pubsub := a.db.Subscribe(a.ctx, channel)
	defer pubsub.Close()

	ch := pubsub.Channel()

	for msg := range ch {
		f(msg.Payload)
	}
}

/**
* Broadcast a message to all clients in the cluster hub
* @param to string
* @param msg Message
* @param ignored []string
* @param from et.Json
* @return error
**/
func (a *RedisAdapter) Broadcast(to string, msg Message, ignored []string, from et.Json) error {
	mbroadcast := MessageBroadcast{
		Kind:    BroadcastAll,
		To:      to,
		Msg:     msg,
		Ignored: ignored,
		From:    from,
	}

	return a.publish(mbroadcast)
}

/**
* Direct send a message to a specific client in the cluster hub
* @param to string
* @param msg Message
* @return error
**/
func (a *RedisAdapter) Direct(to string, msg Message) error {
	mbroadcast := MessageBroadcast{
		Kind:    BroadcastDirect,
		To:      to,
		Msg:     msg,
		Ignored: []string{},
		From:    et.Json{},
	}

	return a.publish(mbroadcast)
}

/**
* Command send a command to all clients in the cluster hub
* @param command string
* @param params et.Json
* @return error
**/
func (a *RedisAdapter) Command(command string, params et.Json) error {
	mbroadcast := MessageBroadcast{
		Kind:    BroadcastDirect,
		To:      command,
		Msg:     Message{},
		Ignored: []string{},
		From:    params,
	}

	return a.publish(mbroadcast)
}
