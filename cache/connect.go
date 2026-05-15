package cache

import (
	"context"
	"sync"
	"time"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/utility"
	"github.com/redis/go-redis/v9"
)

/**
* connectTo to a host
* @param host, password string, db int
* @return *Conn, error
**/
func connectTo(host, password string, db int) (*Conn, error) {
	if !utility.ValidStr(host, 0, []string{}) {
		return nil, logs.Alertf(msg.MSG_ATRIB_REQUIRED, "host")
	}

	client := redis.NewClient(&redis.Options{
		Addr:            host,
		Password:        password,
		DB:              db,
		MaxRetries:      1000,
		MinRetryBackoff: 1 * time.Second,
		MaxRetryBackoff: 2 * time.Second,
	})

	ctx := context.Background()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	logs.Logf("Redis", "Connected host:%s", host)

	return &Conn{
		Client:   client,
		Id:       utility.UUID(),
		ctx:      ctx,
		host:     host,
		dbname:   db,
		channels: make(map[string]*redis.PubSub),
		mutex:    &sync.RWMutex{},
	}, nil
}
