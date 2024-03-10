package cache

import (
	"context"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
	"github.com/redis/go-redis/v9"
)

func connect() {
	host := envar.EnvarStr("", "REDIS_HOST")
	password := envar.EnvarStr("", "REDIS_PASSWORD")
	dbname := envar.EnvarInt(0, "REDIS_DB")

	if host == "" {
		et.Errorf(msg.T("ERR_ENV_REQUIRED"), "REDIS_HOST")
		return
	}

	if password == "" {
		et.Errorf(msg.T("ERR_ENV_REQUIRED"), "REDIS_PASSWORD")
		return
	}

	client := redis.NewClient(&redis.Options{
		Addr:     host,
		Password: password,
		DB:       dbname,
	})

	et.Logf("Redis", "Connected host:%s", host)

	conn = &Conn{
		ctx:    context.Background(),
		host:   host,
		dbname: dbname,
		db:     client,
	}
}
