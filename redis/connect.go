package redis

import (
	"context"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/redis/go-redis/v9"
)

func connect() {
	host := envar.EnvarStr("", "REDIS_HOST")
	password := envar.EnvarStr("", "REDIS_PASSWORD")
	dbname := envar.EnvarInt(0, "REDIS_DB")

	if host == "" {
		logs.Errorf(ERR_ENV_REQUIRED, "REDIS_HOST")
		return
	}

	if password == "" {
		logs.Errorf(ERR_ENV_REQUIRED, "REDIS_PASSWORD")
		return
	}

	client := redis.NewClient(&redis.Options{
		Addr:     host,
		Password: password,
		DB:       dbname,
	})

	logs.Logf("Redis", "Connected host:%s", host)

	conn = &Conn{
		ctx:    context.Background(),
		host:   host,
		dbname: dbname,
		db:     client,
	}
}
