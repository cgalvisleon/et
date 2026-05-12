package postgres

import (
	"database/sql"
	"fmt"

	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/logs"
)

/**
* ExistDatabase: Returns true when a database with the given name exists in the PostgreSQL instance.
* @param db *sql.DB
* @param name string
* @return bool, error
**/
func ExistDatabase(db *sql.DB, name string) (bool, error) {
	sql := `
	SELECT EXISTS(
	SELECT 1
	FROM pg_database
	WHERE UPPER(datname) = UPPER($1));`
	rows, err := db.Query(sql, name)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	items := jsql.RowsToItems(rows)
	if items.Count == 0 {
		return false, nil
	}

	return items.Bool(0, "exists"), nil
}

/**
* CreateDatabase: Creates a PostgreSQL database with the given name if it does not already exist.
* @param db *sql.DB
* @param name string
* @return error
**/
func CreateDatabase(db *sql.DB, name string) error {
	exist, err := ExistDatabase(db, name)
	if err != nil {
		return err
	}

	if exist {
		return nil
	}

	sql := fmt.Sprintf(`CREATE DATABASE %s;`, name)
	_, err = db.Exec(sql)
	if err != nil {
		return err
	}

	logs.Logf("Postgres", `Database %s created`, name)

	return nil
}

/**
* DropDatabase: Drops the PostgreSQL database with the given name.
* @param db *sql.DB
* @param name string
* @return error
**/
func DropDatabase(db *sql.DB, name string) error {
	sql := fmt.Sprintf(`DROP DATABASE %s;`, name)
	_, err := db.Exec(sql)
	if err != nil {
		return err
	}

	logs.Logf("Postgres", `Database %s droped`, name)

	return nil
}
