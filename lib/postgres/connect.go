package lib

import (
	"database/sql"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	_ "github.com/lib/pq"
)

func connect(params et.Json) (*sql.DB, error) {
	if params["user"] == nil {
		return nil, logs.Errorm("User is required")
	}

	if params["password"] == nil {
		return nil, logs.Errorm("Password is required")
	}

	if params["host"] == nil {
		return nil, logs.Errorm("Host is required")
	}

	if params["port"] == nil {
		return nil, logs.Errorm("Port is required")
	}

	if params["database"] == nil {
		return nil, logs.Errorm("Database is required")
	}

	if params["app"] == nil {
		return nil, logs.Errorm("App name is required")
	}

	driver := "postgres"
	user := params.Str("user")
	password := params.Str("password")
	host := params.Str("host")
	port := params.Int("port")
	database := params.Str("database")
	app := params.Str("app")

	connStr := strs.Format(`%s://%s:%s@%s:%d/%s?sslmode=disable&application_name=%s`, driver, user, password, host, port, database, app)
	db, err := sql.Open(driver, connStr)
	if err != nil {
		return nil, err
	}

	err = defineCore(db)
	if err != nil {
		return nil, err
	}

	err = defineSeries(db)
	if err != nil {
		return nil, err
	}

	err = defineModels(db)
	if err != nil {
		return nil, err
	}

	err = defineSync(db)
	if err != nil {
		return nil, err
	}

	err = defineRecycling(db)
	if err != nil {
		return nil, err
	}

	go defineListen(connStr, []string{"sync", "recycling"})

	logs.Logf("DB", "Connected to %s database %s", driver, database)

	return db, nil
}
