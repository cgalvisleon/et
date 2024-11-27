package cache

import (
	"context"

	"github.com/cgalvisleon/et/logs"
	"github.com/redis/go-redis/v9"
)

const PackageName = "cache"

var conn *Conn

type Conn struct {
	*redis.Client
	ctx    context.Context
	host   string
	dbname int
}

func Load() (*Conn, error) {
	if conn != nil {
		return conn, nil
	}

	var err error
	conn, err = connect()
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func Close() {
	if conn == nil {
		return
	}

	conn.Close()

	logs.Log(PackageName, `Disconnect...`)
}
