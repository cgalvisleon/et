package ws

import (
	"context"
	"encoding/json"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/redis/go-redis/v9"
)

// MessageBroadcast is a struct to manage the message to broadcast
type MessageBroadcast struct {
	Kind    TpBroadcast `json:"kind"`
	To      string      `json:"channel"`
	Msg     Message     `json:"msg"`
	Ignored []string    `json:"ignored"`
	From    et.Json     `json:"from"`
}

// Encode return the message as byte
func (m MessageBroadcast) Encode() ([]byte, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// Decode return the message as struct
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

// NewRedisAdapter create a new RedisAdapter
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

// Set a adapter to make a cluster hub
func (h *Hub) RedisAdapter(params *RedisAdapterParams) error {
	adapter, err := NewRedisAdapter(params)
	if err != nil {
		return err
	}
	adapter.subscribe(adapter.channel, h.listend)

	h.adapter = adapter

	return nil
}

// Close the connection
func (a *RedisAdapter) Close() error {
	return a.db.Close()
}

// Publish a message to a broadcast channel
func (a *RedisAdapter) publish(message MessageBroadcast) error {
	if a.db == nil {
		return logs.Errorm(ERR_REDISADAPTER_NOT_FOUND)
	}

	err := a.db.Publish(a.ctx, a.channel, message).Err()
	if err != nil {
		return err
	}

	return nil
}

// Subscribe to a channel with context
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

// Broadcast a message to all clients in the cluster hub
func (a *RedisAdapter) Broadcast(to string, msg Message, ignored []string, from et.Json) error {
	mbroadcast := MessageBroadcast{
		Kind:    TpAll,
		To:      to,
		Msg:     msg,
		Ignored: ignored,
		From:    from,
	}

	return a.publish(mbroadcast)
}

// Broadcast a message to all clients in the cluster hub
func (a *RedisAdapter) Direct(to string, msg Message) error {
	mbroadcast := MessageBroadcast{
		Kind:    TpDirect,
		To:      to,
		Msg:     msg,
		Ignored: []string{},
		From:    et.Json{},
	}

	return a.publish(mbroadcast)
}

func (a *RedisAdapter) Command(command string, params et.Json) error {
	mbroadcast := MessageBroadcast{
		Kind:    TpDirect,
		To:      command,
		Msg:     Message{},
		Ignored: []string{},
		From:    params,
	}

	return a.publish(mbroadcast)
}
