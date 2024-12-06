package cache

import (
	"context"
	"sync"

	"github.com/cgalvisleon/et/logs"
	"github.com/redis/go-redis/v9"
)

const PackageName = "cache"

var (
	conn *Conn
)

type Conn struct {
	*redis.Client
	_id      string
	ctx      context.Context
	host     string
	dbname   int
	channels map[string]bool
	mutex    *sync.RWMutex
}

/**
* FromId return the id of the connection
* @return string
**/
func FromId() string {
	return conn._id
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
