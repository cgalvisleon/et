package cache

import (
	"context"
	"fmt"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
)

/**
* pubCtx
* @params ctx context.Context
* @params channel string
* @params message interface{}
* @return error
**/
func pubCtx(ctx context.Context, channel string, message interface{}) error {
	if conn == nil {
		return logs.Errorm(msg.ERR_NOT_CACHE_SERVICE)
	}

	err := conn.Publish(ctx, channel, message).Err()
	if err != nil {
		return err
	}

	return nil
}

/**
* subCtx
* @params ctx context.Context
* @params channel string
* @params f func(interface{})
**/
func subCtx(ctx context.Context, channel string, f func(interface{})) {
	if conn == nil {
		return
	}

	pubsub := conn.Subscribe(ctx, channel)
	defer pubsub.Close()

	ch := pubsub.Channel()

	for msg := range ch {
		fmt.Println(msg.Channel, msg.Payload)
		f(msg.Payload)
	}
}

/**
* Pub
* @params channel string
* @params message interface{}
* @return error
**/
func Pub(channel string, message interface{}) error {
	ctx := context.Background()
	return pubCtx(ctx, channel, message)
}

/**
* Sub
* @params channel string
* @params f func(interface{})
**/
func Sub(channel string, f func(interface{})) {
	ctx := context.Background()
	subCtx(ctx, channel, f)
}
