package jdb

import (
	"database/sql"

	"github.com/cgalvisleon/et/envar"
	lpg "github.com/cgalvisleon/et/lib/postgres"
	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/logs"
)

/**
* Load a database
* @return *linq.Database
* @return error
**/
func Load() (*linq.Database, error) {
	kind := envar.GetStr("postgre", "DB_DRIVE")
	host := envar.GetStr("localhost", "DB_HOST")
	port := envar.GetInt(5432, "DB_PORT")
	name := envar.GetStr("test", "DB_NAME")
	user := envar.GetStr("test", "DB_USER")
	password := envar.GetStr("test", "DB_PASSWORD")
	app := envar.GetStr("test", "DB_APP_NAME")

	if kind == "postgres" {
		var drive linq.Driver
		drivePg := lpg.NewDriver(&linq.Connection{
			Drive:    linq.Postgres,
			User:     user,
			Password: password,
			Host:     host,
			Port:     port,
			Database: name,
			App:      app,
			UsedCore: true,
		})

		drive = drivePg

		result, err := linq.NewDatabase(name, "", drive)
		if err != nil {
			return nil, err
		}

		return result, nil
	} else {
		return nil, logs.Alertm(MSG_DRIVER_NOT_FOUND)
	}
}

func LoadTo(params *linq.Connection) (*linq.Database, error) {
	var drive linq.Driver
	drivePg := lpg.NewDriver(params)
	drive = drivePg

	result, err := linq.NewDatabase(params.Database, "", drive)
	if err != nil {
		return nil, logs.Alert(err)
	}

	return result, nil
}

func Connect(params *linq.Connection) (*sql.DB, error) {
	connStr, err := lpg.ConnStr(params)
	if err != nil {
		return nil, logs.Alert(err)
	}

	result, err := lpg.Connect(connStr)
	if err != nil {
		return nil, logs.Alert(err)
	}

	return result, nil
}
