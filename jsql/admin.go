package jsql

import (
	"fmt"
	"strings"

	"github.com/cgalvisleon/et/et"
)

/**
* Administration descriptors (spec §8.3): drop and truncate of a model, a schema or a
* database, and create, alter (password) and drop of a user, in JSON:
*   {"drop": {"model": "schema.model"}}                {"truncate": {"model": "schema.model"}}
*   {"drop": {"schema": "name", "cascade": true}}      {"truncate": {"schema": "name"}}
*   {"drop": {"database": "name"}}                     {"truncate": {"database": "name"}}
*   {"drop": {"user": "name"}}
*   {"create": {"user": "name", "password": "secret"}}
*   {"alter": {"user": "name", "password": "secret"}}
* "if_exists": true makes a missing object not an error. Each becomes the SQL of the
* engine: postgres and josefina all of them; mysql and mssql models, databases and users;
* sqlite and oracle models (sqlite empties a table with DELETE). Anything else answers
* MSG_ADMIN_UNSUPPORTED.
**/

// keywordsAdmin are the keys of an administration descriptor.
var keywordsAdmin = []string{"drop", "truncate", "create", "alter"}

/**
* parseAdmin: Runs an administration descriptor and returns what it did
* @param op string, value any
* @return et.Items, error
**/
func (s *DB) parseAdmin(op string, value any) (et.Items, error) {
	desc, ok := jsonOf(value)
	if !ok {
		return et.Items{}, errJsonCommandInvalid
	}
	kind, name := "", ""
	for _, k := range []string{"model", "schema", "database", "user"} {
		if v := strings.TrimSpace(desc.Str(k)); v != "" {
			kind, name = k, v
			break
		}
	}
	if kind == "" {
		return et.Items{}, fmt.Errorf(MSG_ADMIN_INVALID, op, "model|schema|database|user: name")
	}

	statements, err := s.adminSQL(op, kind, name, desc)
	if err != nil {
		return et.Items{}, err
	}
	for _, sql := range statements {
		if _, err := s.Sql(sql); err != nil {
			return et.Items{}, err
		}
	}
	s.forgetDropped(op, kind, name)

	result := et.Items{}
	result.Add(et.Json{"op": op, kind: name, "ok": true})
	return result, nil
}

/**
* forgetDropped: After a drop, the models it took are removed from the DB in memory
* @param op, kind, name string
**/
func (s *DB) forgetDropped(op, kind, name string) {
	if op != "drop" {
		return
	}
	switch kind {
	case "model":
		schema, model, _, _ := splitFromRef(name)
		if sch, ok := s.lookupSchema(schema); ok {
			sch.removeModel(model)
		}
	case "schema":
		s.muSchemas.Lock()
		delete(s.Schemas, name)
		s.muSchemas.Unlock()
	}
}

/**
* adminSQL: The statements of an administration descriptor in the dialect of the driver
* @param op, kind, name string, desc et.Json
* @return []string, error
**/
func (s *DB) adminSQL(op, kind, name string, desc et.Json) ([]string, error) {
	pg := s.Driver == DriverPostgres || s.Driver == DriverJosefina
	ifExists := ""
	if desc.Bool("if_exists") {
		ifExists = "IF EXISTS "
	}
	unsupported := fmt.Errorf(MSG_ADMIN_UNSUPPORTED, s.Driver, op+" "+kind)
	password := strings.ReplaceAll(desc.Str("password"), "'", "''")

	switch op + " " + kind {
	case "drop model":
		return []string{"DROP TABLE " + ifExists + s.adminTable(name)}, nil
	case "truncate model":
		if s.Driver == DriverSqlite {
			return []string{"DELETE FROM " + s.adminTable(name)}, nil
		}
		return []string{"TRUNCATE TABLE " + s.adminTable(name)}, nil
	case "drop schema":
		if !pg {
			return nil, unsupported
		}
		sql := "DROP SCHEMA " + ifExists + quoteIdent(name)
		if desc.Bool("cascade") {
			sql += " CASCADE"
		}
		return []string{sql}, nil
	case "truncate schema", "truncate database":
		if !pg {
			return nil, unsupported
		}
		if kind == "database" && name != s.Name {
			return nil, fmt.Errorf(MSG_INVALID_DATABASE, name, "truncate")
		}
		return s.adminTruncateAll(kind, name)
	case "drop database":
		switch s.Driver {
		case DriverPostgres, DriverJosefina, DriverMssql:
			return []string{"DROP DATABASE " + ifExists + quoteIdent(name)}, nil
		case DriverMysql:
			return []string{"DROP DATABASE " + ifExists + "`" + name + "`"}, nil
		}
	case "drop user":
		switch s.Driver {
		case DriverPostgres, DriverJosefina:
			return []string{"DROP USER " + ifExists + quoteIdent(name)}, nil
		case DriverMysql:
			return []string{"DROP USER " + ifExists + "'" + name + "'"}, nil
		case DriverMssql:
			return []string{"DROP LOGIN " + quoteIdent(name)}, nil
		}
	case "create user", "alter user":
		if password == "" {
			return nil, fmt.Errorf(MSG_ADMIN_INVALID, op, "user, password")
		}
		verb := strings.ToUpper(op)
		switch s.Driver {
		case DriverPostgres, DriverJosefina:
			return []string{verb + " USER " + quoteIdent(name) + " WITH PASSWORD '" + password + "'"}, nil
		case DriverMysql:
			return []string{verb + " USER '" + name + "' IDENTIFIED BY '" + password + "'"}, nil
		case DriverMssql:
			return []string{verb + " LOGIN " + quoteIdent(name) + " WITH PASSWORD = '" + password + "'"}, nil
		}
	}
	return nil, unsupported
}

/**
* adminTable: A model reference ("schema.model" or "model") as a table of the engine: the table of the
* model when the DB knows it (sqlite names it schema_model), else the reference quoted
* @param ref string
* @return string
**/
func (s *DB) adminTable(ref string) string {
	schema, table, _, _ := splitFromRef(ref)
	if sch, ok := s.lookupSchema(schema); ok {
		if model, err := sch.getModel(table); err == nil && model.Table != "" {
			return model.Table
		}
	}
	if schema == "" || s.Driver == DriverSqlite || s.Driver == DriverMysql {
		return quoteIdent(table)
	}
	return quoteIdent(schema) + "." + quoteIdent(table)
}

/**
* adminTruncateAll: TRUNCATE of every table of a schema, or of every user schema of the
* database, read from information_schema (nothing to truncate: no statement)
* @param kind, name string
* @return []string, error
**/
func (s *DB) adminTruncateAll(kind, name string) ([]string, error) {
	query := "SELECT table_schema, table_name FROM information_schema.tables WHERE table_type = 'BASE TABLE' AND table_schema NOT IN ('pg_catalog', 'information_schema')"
	args := []any{}
	if kind == "schema" {
		query += " AND table_schema = $1"
		args = append(args, name)
	}
	items, err := s.Sql(query, args...)
	if err != nil {
		return nil, err
	}
	tables := []string{}
	for _, row := range items.Result {
		tables = append(tables, quoteIdent(row.Str("table_schema"))+"."+quoteIdent(row.Str("table_name")))
	}
	if len(tables) == 0 {
		return []string{}, nil
	}
	return []string{"TRUNCATE TABLE " + strings.Join(tables, ", ")}, nil
}

/**
* quoteIdent: An identifier in double quotes (the SQL standard)
* @param name string
* @return string
**/
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
