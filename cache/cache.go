package cache

import (
	"context"
	"runtime"
	"slices"
	"sync"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/redis/go-redis/v9"
)

const PackageName = "cache"

var (
	os     = ""
	conn   *Conn
	loadMu sync.Mutex
)

func init() {
	os = runtime.GOOS
}

type Conn struct {
	*redis.Client
	Id       string
	ctx      context.Context
	host     string
	dbname   int
	channels map[string]*redis.PubSub
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
* @return error
**/
func Load() error {
	if !slices.Contains([]string{"linux", "darwin", "windows"}, os) {
		return nil
	}

	loadMu.Lock()
	defer loadMu.Unlock()

	if conn != nil {
		return nil
	}

	err := envar.Validate([]string{
		"REDIS_HOST",
		"REDIS_PASSWORD",
		"REDIS_DB",
	})
	if err != nil {
		return err
	}

	host := envar.GetStr("REDIS_HOST", "")
	password := envar.GetStr("REDIS_PASSWORD", "")
	dbname := envar.GetInt("REDIS_DB", 0)
	conn, err = ConnectTo(host, password, dbname)
	if err != nil {
		return err
	}

	return nil
}

/**
* Close terminates the Redis connection.
**/
func Close() {
	if conn == nil {
		return
	}

	conn.Close()

	logs.Log(PackageName, `Disconnect...`)
}

/**
* IsLoad
* @return bool
**/
func IsLoad() bool {
	return conn != nil
}

/**
* HealthCheck
* @return bool
**/
func HealthCheck() bool {
	if conn == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(conn.ctx, 2*time.Second)
	defer cancel()

	return conn.Ping(ctx).Err() == nil
}
