package mongo

import (
	"context"

	"github.com/cgalvisleon/et/logs"
	"go.mongodb.org/mongo-driver/mongo"
)

type Conn struct {
	ctx    context.Context
	host   string
	port   int
	dbname string
	db     *mongo.Client
}

var conn *Conn

func Load() (*Conn, error) {
	if conn != nil {
		return conn, nil
	}

	var err error
	conn, err = connect()
	if err != nil {
		return nil, logs.Alert(err)
	}

	return conn, nil
}

func Close() {
	if conn != nil {
		conn.db.Disconnect(conn.ctx)
	}
}
