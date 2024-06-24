package redis

import (
	"context"
	"fmt"

	"github.com/cgalvisleon/et/logs"
)

// Publish a message to a channel with context
func PubCtx(ctx context.Context, channel string, message interface{}) error {
	if conn == nil {
		return logs.Errorm(ERR_NOT_CACHE_SERVICE)
	}

	err := conn.db.Publish(ctx, channel, message).Err()
	if err != nil {
		return err
	}

	return nil
}

// Subscribe to a channel with context
func SubCtx(ctx context.Context, channel string, reciveFn func(interface{})) {
	if conn == nil {
		return
	}

	pubsub := conn.db.Subscribe(ctx, channel)
	defer pubsub.Close()

	ch := pubsub.Channel()

	for msg := range ch {
		fmt.Println(msg.Channel, msg.Payload)
		reciveFn(msg.Payload)
	}
}

// Publish a message to a channel
func Pub(channel string, message interface{}) error {
	ctx := context.Background()
	return PubCtx(ctx, channel, message)
}

// Subscribe to a channel
func Sub(channel string, reciveFn func(interface{})) {
	ctx := context.Background()
	SubCtx(ctx, channel, reciveFn)
}
