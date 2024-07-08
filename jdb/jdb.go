package jdb

import (
	"database/sql"

	"github.com/cgalvisleon/et/envar"
	lpg "github.com/cgalvisleon/et/lib/postgres"
	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/logs"
)

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
		drivePg := lpg.NewDriver(&lpg.Connection{
			User:     user,
			Password: password,
			Host:     host,
			Port:     port,
			Database: name,
			App:      app,
		})

		drive = drivePg

		result, err := linq.NewDatabase(name, "", drive)
		if err != nil {
			return nil, logs.Alert(err)
		}

		return result, nil
	} else {
		return nil, logs.Alertm(MSG_DRIVER_NOT_FOUND)
	}
}

func Connect(params *lpg.Connection) (*sql.DB, error) {
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
