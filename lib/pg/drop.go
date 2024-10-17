package lib

import (
	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/strs"
)

/**
* DCL Data Control Language
* Drop database, schema, table, column, index, trigger, serie, user
**/

// DropDatabase drop a database if exists
func DropDatabase(db *linq.DB, name string) error {
	name = strs.Lowcase(name)
	sql := strs.Format(`DROP DATABASE %s;`, name)
	err := db.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

// DropSchema drop a schema if exists
func DropSchema(db *linq.DB, name string) error {
	name = strs.Lowcase(name)
	sql := strs.Format(`DROP SCHEMA %s CASCADE;`, name)
	err := db.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

// DropTable drop a table if exists
func DropTable(db *linq.DB, schema, name string) error {
	sql := strs.Format(`DROP TABLE %s.%s CASCADE;`, schema, name)
	err := db.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

// DropColumn drop a column if exists in the table
func DropColumn(db *linq.DB, schema, table, name string) error {
	sql := strs.Format(`ALTER TABLE %s.%s DROP COLUMN %s;`, schema, table, name)
	err := db.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

// DropIndex drop a index if exists in the table
func DropIndex(db *linq.DB, schema, table, field string) error {
	indexName := strs.Format(`%s_%s_IDX`, strs.Uppcase(table), strs.Uppcase(field))
	sql := strs.Format(`DROP INDEX %s.%s CASCADE;`, schema, indexName)
	err := db.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

// DropTrigger drop a trigger if exists in the table
func DropTrigger(db *linq.DB, schema, table, name string) error {
	sql := strs.Format(`DROP TRIGGER %s.%s CASCADE;`, schema, name)
	err := db.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

// DropSerie drop a serie if exists
func DropSerie(db *linq.DB, schema, name string) error {
	sql := strs.Format(`DROP SEQUENCE %s.%s CASCADE;`, schema, name)
	err := db.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

// DropUser drop a user if exists
func DropUser(db *linq.DB, name string) error {
	name = strs.Uppcase(name)
	sql := strs.Format(`DROP USER %s;`, name)
	err := db.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}
