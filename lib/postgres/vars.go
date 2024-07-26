package lib

import (
	"database/sql"
	"strconv"
)

/**
* defineVars define the vars table
* @param db *sql.DB
* @return error
**/
func defineVars(db *sql.DB) error {
	sql := `
	CREATE SCHEMA IF NOT EXISTS core;

  CREATE TABLE IF NOT EXISTS core.VARS(		
		VAR VARCHAR(00) DEFAULT '',
		VALUE VARCHAR(250) DEFAULT '',
		PRIMARY KEY(VAR)
	);`

	_, err := db.Exec(sql)
	if err != nil {
		return err
	}

	err = initVar(db, "REPLICA", "10000")
	if err != nil {
		return err
	}

	return nil
}

/**
* initVar init a var
* @param db *sql.DB
* @param name string
* @param value string
* @return error
**/
func initVar(db *sql.DB, name string, value string) error {
	sql := `
	INSERT INTO core.VARS (VAR, VALUE)
	VALUES ($1, $2);`

	_, err := db.Exec(sql, name, value)
	if err != nil {
		return err
	}

	return nil
}

/**
* setVar set a var
* @param db *sql.DB
* @param name string
* @param value string
* @return error
**/
func setVar(db *sql.DB, name string, value string) error {
	sql := `
	INSERT INTO core.VARS (VAR, VALUE)
	VALUES ($1, $2)
	ON CONFLICT (VAR) DO UPDATE SET
	VALUE = $2;`

	_, err := db.Exec(sql, name, value)
	if err != nil {
		return err
	}

	return nil
}

/**
* getVar set a var
* @param db *sql.DB
* @param name string
* @param def string
* @return string
* @return error
**/
func getVar(db *sql.DB, name, def string) (string, error) {
	sql := `
	SELECT VALUE
	FROM core.VARS
	WHERE VAR = $1;`

	rows, err := db.Query(sql, name)
	if err != nil {
		return def, err
	}
	defer rows.Close()

	var value string
	for rows.Next() {
		err := rows.Scan(&value)
		if err != nil {
			return def, err
		}
	}

	return value, nil
}

/**
* getVarInt set a var
* @param db *sql.DB
* @param name int64
* @param def int64
* @return int64
* @return error
**/
func getVarInt(db *sql.DB, name string, def int64) (int64, error) {
	sql := `
	SELECT VALUE
	FROM core.VARS
	WHERE VAR = $1;`

	rows, err := db.Query(sql, name)
	if err != nil {
		return def, err
	}
	defer rows.Close()

	var value string
	for rows.Next() {
		err := rows.Scan(&value)
		if err != nil {
			return def, err
		}
	}

	result, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return def, err
	}

	return result, nil
}

/**
* delVar delete a var
* @param db *sql.DB
* @param name string
* @return error
**/
func delVar(db *sql.DB, name string) error {
	sql := `
	DELETE FROM core.VARS
	WHERE VAR = $1;`

	_, err := db.Exec(sql, name)
	if err != nil {
		return err
	}

	return nil
}

/**
* SetVar set a var
* @param key string
* @param value string
* @return error
**/
func (d *Postgres) SetVar(key, value string) error {
	return setVar(d.DB, key, value)
}

/**
* DelVal delete a var
* @param key string
* @return error
**/
func (d *Postgres) DelVal(key string) error {
	return delVar(d.DB, key)
}

/**
* Var get a var
* @param key string
* @param def string
* @return string
* @return error
**/
func (d *Postgres) Var(key string, def string) (string, error) {
	return getVar(d.DB, key, def)
}

/**
* VarInt get a var
* @param key string
* @param def int64
* @return int64
* @return error
**/
func (d *Postgres) VarInt(key string, def int64) (int64, error) {
	return getVarInt(d.DB, key, def)
}
