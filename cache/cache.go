package cache

import (
	"context"
	"runtime"
	"slices"
	"sync"
	"time"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/utility"
	"github.com/redis/go-redis/v9"
)

var (
	packageName = "cache"
	os          = ""
	conn        *Conn
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
* LoadTo: Initializes the Redis connection from a Config struct.
* @param cfg Config
* @return error
**/
func New() (*Conn, error) {
	if !slices.Contains([]string{"linux", "darwin", "windows"}, os) {
		return nil, logs.Alertf(MSG_UNSUPPORTED_OS, os)
	}

	host := config.GetStr("REDIS_HOST", "")
	if !utility.ValidStr(host, 0, []string{}) {
		return nil, logs.Alertf(msg.MSG_ATRIB_REQUIRED, "host")
	}

	password := config.GetStr("REDIS_PASSWORD", "")
	dbname := config.GetInt("REDIS_DB", 0)
	client := redis.NewClient(&redis.Options{
		Addr:            host,
		Password:        password,
		DB:              dbname,
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
		dbname:   dbname,
		channels: make(map[string]*redis.PubSub),
		mutex:    &sync.RWMutex{},
	}, nil
}

/**
* Close terminates the Redis connection.
**/
func (s *Conn) Close() {
	s.Close()

	logs.Log(packageName, `Disconnect...`)
}

/**
* HealthCheck
* @return bool
**/
func (s *Conn) HealthCheck() bool {
	ctx, cancel := context.WithTimeout(s.ctx, 2*time.Second)
	defer cancel()

	return conn.Ping(ctx).Err() == nil
}
