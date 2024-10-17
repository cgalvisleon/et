package lib

import (
	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/strs"
)

/**
* DCL Data Control Language
* Exist database, schema, table, column, index, trigger, serie, user
**/

/**
* ExistDatabase check if the database exists
* @param db *linq.DB
* @param name string
* @return bool, error
**/
func ExistDatabase(db *linq.DB, name string) (bool, error) {
	name = strs.Lowcase(name)
	sql := `
	SELECT EXISTS(
		SELECT 1
		FROM pg_database
		WHERE UPPER(datname) = UPPER($1));`

	item, err := db.QueryOne(sql, name)
	if err != nil {
		return false, err
	}

	return item.Bool("exists"), nil
}

// ExistSchema check if the schema exists
func ExistSchema(db *linq.DB, name string) (bool, error) {
	name = strs.Lowcase(name)
	sql := `
	SELECT EXISTS(
		SELECT 1
		FROM pg_namespace
		WHERE UPPER(nspname) = UPPER($1));`

	item, err := db.QueryOne(sql, name)
	if err != nil {
		return false, err
	}

	return item.Bool("exists"), nil
}

// ExistTable check if the table exists
func ExistTable(db *linq.DB, schema, name string) (bool, error) {
	sql := `
	SELECT EXISTS(
		SELECT 1
		FROM information_schema.tables
		WHERE UPPER(table_schema) = UPPER($1)
		AND UPPER(table_name) = UPPER($2));`

	item, err := db.QueryOne(sql, schema, name)
	if err != nil {
		return false, err
	}

	return item.Bool("exists"), nil
}

// ExistColum check if the column exists in the table
func ExistColum(db *linq.DB, schema, table, name string) (bool, error) {
	sql := `
	SELECT EXISTS(
		SELECT 1
		FROM information_schema.columns
		WHERE UPPER(table_schema) = UPPER($1)
		AND UPPER(table_name) = UPPER($2)
		AND UPPER(column_name) = UPPER($3));`

	item, err := db.QueryOne(sql, schema, table, name)
	if err != nil {
		return false, err
	}

	return item.Bool("exists"), nil
}

// ExistIndex check if the index exists in the table
func ExistIndex(db *linq.DB, schema, table, field string) (bool, error) {
	indexName := strs.Format(`%s_%s_IDX`, strs.Uppcase(table), strs.Uppcase(field))
	sql := `
	SELECT EXISTS(
		SELECT 1
		FROM pg_indexes
		WHERE UPPER(schemaname) = UPPER($1)
		AND UPPER(tablename) = UPPER($2)
		AND UPPER(indexname) = UPPER($3));`

	item, err := db.QueryOne(sql, schema, table, indexName)
	if err != nil {
		return false, err
	}

	return item.Bool("exists"), nil
}

// ExistTrigger check if the trigger exists in the table
func ExistTrigger(db *linq.DB, schema, table, name string) (bool, error) {
	sql := `
	SELECT EXISTS(
		SELECT 1
		FROM information_schema.triggers
		WHERE UPPER(event_object_schema) = UPPER($1)
		AND UPPER(event_object_table) = UPPER($2)
		AND UPPER(trigger_name) = UPPER($3));`

	item, err := db.QueryOne(sql, schema, table, name)
	if err != nil {
		return false, err
	}

	return item.Bool("exists"), nil
}

// ExistSerie check if the serie exists
func ExistSerie(db *linq.DB, schema, name string) (bool, error) {
	sql := `
	SELECT EXISTS(
		SELECT 1
		FROM pg_sequences
		WHERE UPPER(schemaname) = UPPER($1)
		AND UPPER(sequencename) = UPPER($2));`

	item, err := db.QueryOne(sql, schema, name)
	if err != nil {
		return false, err
	}

	return item.Bool("exists"), nil
}

// ExistUser check if the user exists
func ExistUser(db *linq.DB, name string) (bool, error) {
	name = strs.Uppcase(name)
	sql := `
	SELECT EXISTS(
		SELECT 1
		FROM pg_roles
		WHERE UPPER(rolname) = UPPER($1));`

	item, err := db.QueryOne(sql, name)
	if err != nil {
		return false, err
	}

	return item.Bool("exists"), nil
}
