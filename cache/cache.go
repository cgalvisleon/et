package cache

import (
	"context"
	"runtime"
	"slices"
	"sync"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/logs"
	"github.com/redis/go-redis/v9"
)

const PackageName = "cache"

var (
	os   = ""
	conn *Conn
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
* @return error
**/
func Load() error {
	if !slices.Contains([]string{"linux", "darwin", "windows"}, os) {
		return nil
	}

	if conn != nil {
		return nil
	}

	err := config.Validate([]string{
		"REDIS_HOST",
		"REDIS_PASSWORD",
		"REDIS_DB",
	})
	if err != nil {
		return err
	}

	host := config.String("REDIS_HOST", "")
	password := config.String("REDIS_PASSWORD", "")
	dbname := config.Int("REDIS_DB", 0)
	conn, err = ConnectTo(host, password, dbname)
	if err != nil {
		return err
	}

	return nil
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
