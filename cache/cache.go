package cache

import (
	"context"
	"sync"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/utility"
	"github.com/redis/go-redis/v9"
)

const PackageName = "cache"

var conn *Conn

type Conn struct {
	*redis.Client
	Id       string
	ctx      context.Context
	host     string
	dbname   int
	channels map[string]bool
	mutex    *sync.RWMutex
}

/**
* FromId
* @return string
**/
func FromId() string {
	if conn == nil {
		return ""
	}

	return conn.Id
}

/**
* Load
* @return *Conn, error
**/
func Load() (*Conn, error) {
	if conn != nil {
		return conn, nil
	}

	host := envar.GetStr("", "REDIS_HOST")
	password := envar.GetStr("", "REDIS_PASSWORD")
	dbname := envar.GetInt(0, "REDIS_DB")

	if !utility.ValidStr(host, 0, []string{}) {
		return nil, logs.Alertf(msg.ERR_ENV_REQUIRED, "REDIS_HOST")
	}

	if !utility.ValidStr(password, 0, []string{}) {
		return nil, logs.Alertf(msg.ERR_ENV_REQUIRED, "REDIS_PASSWORD")
	}

	var err error
	conn, err = ConnectTo(host, password, dbname)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

/**
* Close
**/
func Close() {
	if conn == nil {
		return
	}

	conn.Close()

	logs.Log(PackageName, `Disconnect...`)
}
