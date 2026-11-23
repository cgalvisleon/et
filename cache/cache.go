package cache

import (
	"context"
	"runtime"
	"slices"
	"sync"
	"time"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
	"github.com/redis/go-redis/v9"
)

var (
	packageName = "cache"
	os          = ""
	conn        *Conn
	local       *memStore
)

func init() {
	os = runtime.GOOS
}

/**
* Conn: The Redis client of the package plus its id, context and pub/sub channels.
**/
type Conn struct {
	*redis.Client
	Id       string
	ctx      context.Context
	host     string
	dbname   int
	channels map[string]*redis.PubSub
	mutex    *sync.RWMutex
}

type Params struct {
	Host       string
	Password   string
	Dbname     int
	MaxRetries int
}

/**
* New: Opens the Redis connection from REDIS_HOST, REDIS_PASSWORD and REDIS_DB and pings it.
* @return *Conn, error
**/
func New(params Params) (*Conn, error) {
	if !slices.Contains([]string{"linux", "darwin", "windows"}, os) {
		return nil, logs.Alertf(MSG_UNSUPPORTED_OS, os)
	}

	if !utility.ValidStr(params.Host, 0, []string{}) {
		return nil, logs.Alertf(MSG_ATRIB_REQUIRED, "host")
	}

	maxRetries := params.MaxRetries
	if maxRetries == 0 {
		maxRetries = 1000
	}
	client := redis.NewClient(&redis.Options{
		Addr:            params.Host,
		Password:        params.Password,
		DB:              params.Dbname,
		MaxRetries:      maxRetries,
		MinRetryBackoff: 1 * time.Second,
		MaxRetryBackoff: 2 * time.Second,
	})

	ctx := context.Background()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	logs.Logf("Redis", "Connected host:%s", params.Host)

	return &Conn{
		Client:   client,
		Id:       utility.UUID(),
		ctx:      ctx,
		host:     params.Host,
		dbname:   params.Dbname,
		channels: make(map[string]*redis.PubSub),
		mutex:    &sync.RWMutex{},
	}, nil
}

/**
* Close terminates the Redis connection.
**/
func (s *Conn) Close() {
	s.Client.Close()

	logs.Log(packageName, `Disconnect...`)
}

/**
* HealthCheck
* @return bool
**/
func (s *Conn) HealthCheck() bool {
	ctx, cancel := context.WithTimeout(s.ctx, 2*time.Second)
	defer cancel()

	return s.Ping(ctx).Err() == nil
}
