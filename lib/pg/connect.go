package lib

import (
	"database/sql"

	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	_ "github.com/lib/pq"
)

/**
* ConnStr return a connection string
* @param params *linq.Connection
* @return string
* @return error
**/
func ConnStr(params *linq.Connection) (string, error) {
	if params.User == "" {
		return "", logs.Alertf(MSS_PARAM_REQUIRED, "User")
	}

	if params.Password == "" {
		return "", logs.Alertf(MSS_PARAM_REQUIRED, "Password")
	}

	if params.Host == "" {
		return "", logs.Alertf(MSS_PARAM_REQUIRED, "Host")
	}

	if params.Port == 0 {
		return "", logs.Alertf(MSS_PARAM_REQUIRED, "Port")
	}

	if params.Database == "" {
		return "", logs.Alertf(MSS_PARAM_REQUIRED, "Database")
	}

	if params.App == "" {
		return "", logs.Alertf(MSS_PARAM_REQUIRED, "App")
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

/**
* Connect return a connection to a database
* @param connStr string
* @return *sql.DB
* @return error
**/
func Connect(connStr string) (*sql.DB, error) {
	driver := "postgres"
	db, err := sql.Open(driver, connStr)
	if err != nil {
		return nil, err
	}

	return db, nil
}
