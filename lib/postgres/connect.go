package lib

import (
	"database/sql"

	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	_ "github.com/lib/pq"
)

func ConnStr(params *linq.Connection) (string, error) {
	if params.User == "" {
		return "", logs.Errorm("User is required")
	}

	if params.Password == "" {
		return "", logs.Errorm("Password is required")
	}

	if params.Host == "" {
		return "", logs.Errorm("Host is required")
	}

	if params.Port == 0 {
		return "", logs.Errorm("Port is required")
	}

	if params.Database == "" {
		return "", logs.Errorm("Database is required")
	}

	if params.App == "" {
		return "", logs.Errorm("App name is required")
	}

	driver := params.Drive.String()
	user := params.User
	password := params.Password
	host := params.Host
	port := params.Port
	database := params.Database
	app := params.App

	result := strs.Format(`%s://%s:%s@%s:%d/%s?sslmode=disable&application_name=%s`, driver, user, password, host, port, database, app)

	return result, nil
}

func Connect(connStr string) (*sql.DB, error) {
	driver := "postgres"
	db, err := sql.Open(driver, connStr)
	if err != nil {
		return nil, err
	}

	return db, nil
}
