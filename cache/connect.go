package cache

import (
	"context"
	"log"
	"sync"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/utility"
	"github.com/redis/go-redis/v9"
)

func ConnectTo(host, password string, dbname int) (*Conn, error) {
	if !utility.ValidStr(host, 0, []string{}) {
		return nil, logs.Alertf(msg.MSG_ATRIB_REQUIRED, "redist_host")
	}

	client := redis.NewClient(&redis.Options{
		Addr:     host,
		Password: password,
		DB:       dbname,
	})

	ctx := context.Background()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		log.Fatal(err)
	}

	logs.Logf("Redis", "Connected host:%s", host)

	return &Conn{
		Client:   client,
		_id:      utility.UUID(),
		ctx:      ctx,
		host:     host,
		dbname:   dbname,
		channels: make(map[string]bool),
		mutex:    &sync.RWMutex{},
	}, nil
}

func connect() (*Conn, error) {
	host := envar.GetStr("", "REDIS_HOST")
	password := envar.GetStr("", "REDIS_PASSWORD")
	dbname := envar.GetInt(0, "REDIS_DB")

	if !utility.ValidStr(host, 0, []string{}) {
		return nil, logs.Alertf(msg.ERR_ENV_REQUIRED, "REDIS_HOST")
	}

	if !utility.ValidStr(password, 0, []string{}) {
		return nil, logs.Alertf(msg.ERR_ENV_REQUIRED, "REDIS_PASSWORD")
	}

	return ConnectTo(host, password, dbname)
}
