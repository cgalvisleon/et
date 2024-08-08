package mongo

import (
	"context"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func connect() (*Conn, error) {
	host := envar.GetStr("", "MONGO_HOST")
	port := envar.GetInt(27017, "MONGO_PORT")
	dbname := envar.GetStr("", "MONGO_DB")

	if host == "" {
		return nil, logs.Errorf(ERR_ENV_REQUIRED, "REDIS_HOST")
	}

	ctx := context.TODO()
	url := strs.Format("mongodb://%s:%d", host, port)
	client := options.Client().ApplyURI(url)
	db, err := mongo.Connect(ctx, client)
	if err != nil {
		return nil, err
	}

	logs.Logf("Mongo", "Connected host:%s", host)

	return &Conn{
		ctx:    ctx,
		host:   host,
		port:   port,
		dbname: dbname,
		db:     db,
	}, nil
}
