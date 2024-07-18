package lib

import "github.com/cgalvisleon/et/js"

/**
* DCL execute a Data Control Language command
* @param command string
* @param params js.Json
* @return error
**/
func (d *Postgres) DCL(command string, params js.Json) error {
	switch command {
	case "exist_database":
		name := params.Str("name")
		_, err := ExistDatabase(d.DB, name)
		if err != nil {
			return err
		}

		return nil
	case "exist_schema":
		name := params.Str("name")
		_, err := ExistSchema(d.DB, name)
		if err != nil {
			return err
		}

		return nil
	case "exist_table":
		schema := params.Str("schema")
		name := params.Str("name")
		_, err := ExistTable(d.DB, schema, name)
		if err != nil {
			return err
		}

		return nil
	case "exist_column":
		schema := params.Str("schema")
		table := params.Str("table")
		name := params.Str("name")
		_, err := ExistColum(d.DB, schema, table, name)
		if err != nil {
			return err
		}

		return nil
	case "exist_index":
		schema := params.Str("schema")
		table := params.Str("table")
		name := params.Str("name")
		_, err := ExistIndex(d.DB, schema, table, name)
		if err != nil {
			return err
		}

		return nil
	case "exist_trigger":
		schema := params.Str("schema")
		table := params.Str("table")
		name := params.Str("name")
		_, err := ExistTrigger(d.DB, schema, table, name)
		if err != nil {
			return err
		}

		return nil
	case "exist_serie":
		schema := params.Str("schema")
		name := params.Str("name")
		_, err := ExistSerie(d.DB, schema, name)
		if err != nil {
			return err
		}

		return nil
	case "exist_user":
		name := params.Str("name")
		_, err := ExistUser(d.DB, name)
		if err != nil {
			return err
		}

		return nil
	case "create_database":
		name := params.Str("name")
		err := CreateDatabase(d.DB, name)
		if err != nil {
			return err
		}

		return nil
	case "create_schema":
		name := params.Str("name")
		err := CreateSchema(d.DB, name)
		if err != nil {
			return err
		}

		return nil
	case "create_column":
		schema := params.Str("schema")
		table := params.Str("table")
		name := params.Str("name")
		kind := params.Str("kind")
		_default := params.Str("default")
		err := CreateColumn(d.DB, schema, table, name, kind, _default)
		if err != nil {
			return err
		}

		return nil
	case "create_index":
		schema := params.Str("schema")
		table := params.Str("table")
		name := params.Str("name")
		err := CreateIndex(d.DB, schema, table, name)
		if err != nil {
			return err
		}

		return nil
	case "create_trigger":
		schema := params.Str("schema")
		table := params.Str("table")
		name := params.Str("name")
		when := params.Str("when")
		event := params.Str("event")
		function := params.Str("function")
		err := CreateTrigger(d.DB, schema, table, name, when, event, function)
		if err != nil {
			return err
		}

		return nil
	case "create_sequence":
		schema := params.Str("schema")
		name := params.Str("name")
		err := CreateSequence(d.DB, schema, name)
		if err != nil {
			return err
		}

		return nil
	case "create_user":
		name := params.Str("name")
		password := params.Str("password")
		err := CreateUser(d.DB, name, password)
		if err != nil {
			return err
		}

		return nil
	case "change_password":
		name := params.Str("name")
		password := params.Str("password")
		err := ChangePassword(d.DB, name, password)
		if err != nil {
			return err
		}

		return nil
	case "drop_database":
		name := params.Str("name")
		err := DropDatabase(d.DB, name)
		if err != nil {
			return err
		}

		return nil
	case "drop_schema":
		name := params.Str("name")
		err := DropSchema(d.DB, name)
		if err != nil {
			return err
		}

		return nil

	case "drop_table":
		schema := params.Str("schema")
		name := params.Str("name")
		err := DropTable(d.DB, schema, name)
		if err != nil {
			return err
		}

		return nil
	case "drop_column":
		schema := params.Str("schema")
		table := params.Str("table")
		name := params.Str("name")
		err := DropColumn(d.DB, schema, table, name)
		if err != nil {
			return err
		}

		return nil

	case "drop_index":
		schema := params.Str("schema")
		table := params.Str("table")
		name := params.Str("name")
		err := DropIndex(d.DB, schema, table, name)
		if err != nil {
			return err
		}

		return nil
	case "drop_trigger":
		schema := params.Str("schema")
		table := params.Str("table")
		name := params.Str("name")
		err := DropTrigger(d.DB, schema, table, name)
		if err != nil {
			return err
		}

		return nil
	case "drop_serie":
		schema := params.Str("schema")
		name := params.Str("name")
		err := DropSerie(d.DB, schema, name)
		if err != nil {
			return err
		}

		return nil
	case "drop_user":
		name := params.Str("name")
		err := DropUser(d.DB, name)
		if err != nil {
			return err
		}

		return nil
	default:
		return nil
	}
}
