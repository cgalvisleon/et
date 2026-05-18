package cache

import (
	"context"
	"runtime"
	"slices"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
	"github.com/redis/go-redis/v9"
)

const packageName = "cache"

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
* LoadTo: Initializes the Redis connection from a Config struct.
* @param params utility.Config
* @return error
**/
func LoadTo(params utility.Config) error {
	host := params.GetStr("REDIS_HOST", "")
	password := params.GetStr("REDIS_PASSWORD", "")
	dbname := params.GetInt("REDIS_DB", 0)

	var err error
	conn, err = connectTo(host, password, dbname)
	return err
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

	params := utility.NewConfig(et.Json{
		"REDIS_HOST":     "",
		"REDIS_PASSWORD": "",
		"REDIS_DB":       0,
	})
	err := LoadTo(params)
	if err != nil {
		return err
	}
	et.SetCacheStore(LoadStore())

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

	logs.Log(packageName, `Disconnect...`)
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
