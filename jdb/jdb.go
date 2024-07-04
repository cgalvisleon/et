package jdb

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
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
		params := et.Json{
			"user":     user,
			"password": password,
			"host":     host,
			"port":     port,
			"database": name,
			"app":      app,
		}
		drivePg := &lpg.Postgres{
			Params: params,
		}
		drive = drivePg

		result, err := linq.NewDatabase(name, "", params, drive)
		if err != nil {
			return nil, logs.Alert(err)
		}

		return result, nil
	} else {
		return nil, logs.Alertm(MSG_DRIVER_NOT_FOUND)
	}
}
