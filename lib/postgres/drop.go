package lib

import (
	"database/sql"

	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/strs"
)

/**
* DCL Data Control Language
* Drop database, schema, table, column, index, trigger, serie, user
**/

// DropDatabase drop a database if exists
func DropDatabase(db *sql.DB, name string) error {
	name = strs.Lowcase(name)
	sql := strs.Format(`DROP DATABASE %s;`, name)
	_, err := linq.Query(db, sql)
	if err != nil {
		return err
	}

	return nil
}

// DropSchema drop a schema if exists
func DropSchema(db *sql.DB, name string) error {
	name = strs.Lowcase(name)
	sql := strs.Format(`DROP SCHEMA %s CASCADE;`, name)
	_, err := linq.Query(db, sql)
	if err != nil {
		return err
	}

	return nil
}

// DropTable drop a table if exists
func DropTable(db *sql.DB, schema, name string) error {
	sql := strs.Format(`DROP TABLE %s.%s CASCADE;`, schema, name)
	_, err := linq.Query(db, sql)
	if err != nil {
		return err
	}

	return nil
}

// DropColumn drop a column if exists in the table
func DropColumn(db *sql.DB, schema, table, name string) error {
	sql := strs.Format(`ALTER TABLE %s.%s DROP COLUMN %s;`, schema, table, name)
	_, err := linq.Query(db, sql)
	if err != nil {
		return err
	}

	return nil
}

// DropIndex drop a index if exists in the table
func DropIndex(db *sql.DB, schema, table, field string) error {
	indexName := strs.Format(`%s_%s_IDX`, strs.Uppcase(table), strs.Uppcase(field))
	sql := strs.Format(`DROP INDEX %s.%s CASCADE;`, schema, indexName)
	_, err := linq.Query(db, sql)
	if err != nil {
		return err
	}

	return nil
}

// DropTrigger drop a trigger if exists in the table
func DropTrigger(db *sql.DB, schema, table, name string) error {
	sql := strs.Format(`DROP TRIGGER %s.%s CASCADE;`, schema, name)
	_, err := linq.Query(db, sql)
	if err != nil {
		return err
	}

	return nil
}

// DropSerie drop a serie if exists
func DropSerie(db *sql.DB, schema, name string) error {
	sql := strs.Format(`DROP SEQUENCE %s.%s CASCADE;`, schema, name)
	_, err := linq.Query(db, sql)
	if err != nil {
		return err
	}

	return nil
}

// DropUser drop a user if exists
func DropUser(db *sql.DB, name string) error {
	name = strs.Uppcase(name)
	sql := strs.Format(`DROP USER %s;`, name)
	_, err := linq.Query(db, sql)
	if err != nil {
		return err
	}

	return nil
}
