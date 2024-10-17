package lib

import (
	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
)

/**
* DCL Data Control Language
* Create database, schema, table, column, index, trigger, serie, user
**/

// CreateDatabase create a database if not exists
func CreateDatabase(db *linq.DB, name string) error {
	name = strs.Lowcase(name)
	sql := strs.Format(`CREATE DATABASE %s;`, name)

	err := db.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

// CreateSchema create a schema if not exists
func CreateSchema(db *linq.DB, name string) error {
	sql := ddlSchema(name)

	err := db.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

// CreateColumn create a column if not exists in the table
func CreateColumn(db *linq.DB, schema, table, name, kind, _default string) error {
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

	err := db.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

// CreateIndex create a index if not exists in the table
func CreateIndex(db *linq.DB, schema, table, field string) error {
	sql := linq.SQLDDL(`
	CREATE INDEX IF NOT EXISTS $2_$3_IDX ON $1.$2($3);`,
		strs.Uppcase(schema), strs.Uppcase(table), strs.Uppcase(field))

	err := db.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

// CreateTrigger create a trigger if not exists in the table
func CreateTrigger(db *linq.DB, schema, table, name, when, event, function string) error {
	sql := linq.SQLDDL(`
	DROP TRIGGER IF EXISTS $3 ON $1.$2 CASCADE;
	CREATE TRIGGER $3
	$4 $5 ON $1.$2
	FOR EACH ROW
	EXECUTE PROCEDURE $6;`,
		strs.Uppcase(schema), strs.Uppcase(table), strs.Uppcase(name), when, event, function)

	err := db.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

// CreateSequence create a sequence if not exists
func CreateSequence(db *linq.DB, schema, tag string) error {
	sql := strs.Format(`CREATE SEQUENCE IF NOT EXISTS %s START 1;`, tag)

	err := db.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

// CreateUser create a user if not exists
func CreateUser(db *linq.DB, name, password string) error {
	passwordHash, err := utility.PasswordHash(password)
	if err != nil {
		return err
	}

	sql := strs.Format(`CREATE USER %s WITH PASSWORD '%s';`, name, passwordHash)

	err = db.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

// ChangePassword change the password of the user
func ChangePassword(db *linq.DB, name, password string) error {
	passwordHash, err := utility.PasswordHash(password)
	if err != nil {
		return err
	}

	sql := strs.Format(`ALTER USER %s WITH PASSWORD '%s';`, name, passwordHash)

	err = db.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}
