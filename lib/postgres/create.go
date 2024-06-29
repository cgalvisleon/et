package lib

import (
	"database/sql"

	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
)

/**
* DCL Data Control Language
* Create database, schema, table, column, index, trigger, serie, user
**/

// CreateDatabase create a database if not exists
func CreateDatabase(db *sql.DB, name string) (bool, error) {
	name = strs.Lowcase(name)
	exists, err := ExistDatabase(db, name)
	if err != nil {
		return false, err
	}

	if !exists {
		sql := strs.Format(`CREATE DATABASE %s;`, name)

		_, err := linq.Query(db, sql)
		if err != nil {
			return false, err
		}
	}

	return !exists, nil
}

// CreateSchema create a schema if not exists
func CreateSchema(db *sql.DB, name string) (bool, error) {
	name = strs.Lowcase(name)
	exists, err := ExistSchema(db, name)
	if err != nil {
		return false, err
	}

	if !exists {
		sql := strs.Format(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"; CREATE SCHEMA IF NOT EXISTS "%s";`, name)

		_, err := linq.Query(db, sql)
		if err != nil {
			return false, err
		}
	}

	return !exists, nil
}

// CreateColumn create a column if not exists in the table
func CreateColumn(db *sql.DB, schema, table, name, kind, _default string) (bool, error) {
	exists, err := ExistColum(db, schema, table, name)
	if err != nil {
		return false, err
	}

	if !exists {
		tableName := strs.Format(`%s.%s`, schema, strs.Uppcase(table))
		sql := linq.SQLDDL(`
		DO $$
		BEGIN
			BEGIN
				ALTER TABLE $1 ADD COLUMN $2 $3 DEFAULT $4;
			EXCEPTION
				WHEN duplicate_column THEN RAISE NOTICE 'column <column_name> already exists in <table_name>.';
			END;
		END;
		$$;`, tableName, strs.Uppcase(name), strs.Uppcase(kind), _default)

		_, err := linq.Query(db, sql)
		if err != nil {
			return false, err
		}
	}

	return !exists, nil
}

// CreateIndex create a index if not exists in the table
func CreateIndex(db *sql.DB, schema, table, field string) (bool, error) {
	exists, err := ExistIndex(db, schema, table, field)
	if err != nil {
		return false, err
	}

	if !exists {
		sql := linq.SQLDDL(`
		CREATE INDEX IF NOT EXISTS $2_$3_IDX ON $1.$2($3);`,
			strs.Uppcase(schema), strs.Uppcase(table), strs.Uppcase(field))

		_, err := linq.Query(db, sql)
		if err != nil {
			return false, err
		}
	}

	return !exists, nil
}

// CreateTrigger create a trigger if not exists in the table
func CreateTrigger(db *sql.DB, schema, table, name, when, event, function string) (bool, error) {
	exists, err := ExistTrigger(db, schema, table, name)
	if err != nil {
		return false, err
	}

	if !exists {
		sql := linq.SQLDDL(`
		DROP TRIGGER IF EXISTS $3 ON $1.$2 CASCADE;
		CREATE TRIGGER $3
		$4 $5 ON $1.$2
		FOR EACH ROW
		EXECUTE PROCEDURE $6;`,
			strs.Uppcase(schema), strs.Uppcase(table), strs.Uppcase(name), when, event, function)

		_, err := linq.Query(db, sql)
		if err != nil {
			return false, err
		}
	}

	return !exists, nil
}

// CreateSerie create a serie if not exists
func CreateSerie(db *sql.DB, schema, tag string) (bool, error) {
	exists, err := ExistSerie(db, schema, tag)
	if err != nil {
		return false, err
	}

	if !exists {
		sql := strs.Format(`CREATE SEQUENCE IF NOT EXISTS %s START 1;`, tag)

		_, err := linq.Query(db, sql)
		if err != nil {
			return false, err
		}
	}

	return !exists, nil
}

// CreateUser create a user if not exists
func CreateUser(db *sql.DB, name, password string) (bool, error) {
	name = strs.Uppcase(name)
	exists, err := ExistUser(db, name)
	if err != nil {
		return false, err
	}

	if !exists {
		passwordHash, err := utility.PasswordHash(password)
		if err != nil {
			return false, err
		}

		sql := strs.Format(`CREATE USER %s WITH PASSWORD '%s';`, name, passwordHash)

		_, err = linq.Query(db, sql)
		if err != nil {
			return false, err
		}
	}

	return !exists, nil
}

// ChangePassword change the password of the user
func ChangePassword(db *sql.DB, name, password string) (bool, error) {
	exists, err := ExistUser(db, name)
	if err != nil {
		return false, err
	}

	if !exists {
		return false, logs.Errorm("User not exists")
	}

	passwordHash, err := utility.PasswordHash(password)
	if err != nil {
		return false, err
	}

	sql := strs.Format(`ALTER USER %s WITH PASSWORD '%s';`, name, passwordHash)

	_, err = linq.Query(db, sql)
	if err != nil {
		return false, err
	}

	return true, nil
}
