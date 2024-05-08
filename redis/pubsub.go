package redis

import (
	"context"
	"fmt"

	"github.com/cgalvisleon/et/logs"
)

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

func Pub(channel string, message interface{}) error {
	ctx := context.Background()
	return PubCtx(ctx, channel, message)
}

func SubCtx(ctx context.Context, channel string, f func(interface{})) {
	if conn == nil {
		return
	}

	pubsub := conn.db.Subscribe(ctx, channel)
	defer pubsub.Close()

	ch := pubsub.Channel()

	for msg := range ch {
		fmt.Println(msg.Channel, msg.Payload)
		f(msg.Payload)
	}
}

func Sub(channel string, f func(interface{})) {
	ctx := context.Background()
	SubCtx(ctx, channel, f)
}
