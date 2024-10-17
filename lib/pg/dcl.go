package lib

import "github.com/cgalvisleon/et/et"

/**
* DCL execute a Data Control Language command
* @param command string
* @param params et.Json
* @return error
**/
func (d *Postgres) DCL(command string, params et.Json) (et.Item, error) {
	switch command {
	case "exist_database":
		name := params.Str("name")
		_, err := ExistDatabase(d.db, name)
		if err != nil {
			return et.Item{}, err
		}

		return et.Item{}, nil
	case "exist_schema":
		name := params.Str("name")
		_, err := ExistSchema(d.db, name)
		if err != nil {
			return et.Item{}, err
		}

		return et.Item{}, nil
	case "exist_table":
		schema := params.Str("schema")
		name := params.Str("name")
		_, err := ExistTable(d.db, schema, name)
		if err != nil {
			return et.Item{}, err
		}

		return et.Item{}, nil
	case "exist_column":
		schema := params.Str("schema")
		table := params.Str("table")
		name := params.Str("name")
		_, err := ExistColum(d.db, schema, table, name)
		if err != nil {
			return et.Item{}, err
		}

		return et.Item{}, nil
	case "exist_index":
		schema := params.Str("schema")
		table := params.Str("table")
		name := params.Str("name")
		_, err := ExistIndex(d.db, schema, table, name)
		if err != nil {
			return et.Item{}, err
		}

		return et.Item{}, nil
	case "exist_trigger":
		schema := params.Str("schema")
		table := params.Str("table")
		name := params.Str("name")
		_, err := ExistTrigger(d.db, schema, table, name)
		if err != nil {
			return et.Item{}, err
		}

		return et.Item{}, nil
	case "exist_serie":
		schema := params.Str("schema")
		name := params.Str("name")
		_, err := ExistSerie(d.db, schema, name)
		if err != nil {
			return et.Item{}, err
		}

		return et.Item{}, nil
	case "exist_user":
		name := params.Str("name")
		_, err := ExistUser(d.db, name)
		if err != nil {
			return et.Item{}, err
		}

		return et.Item{}, nil
	case "create_database":
		name := params.Str("name")
		err := CreateDatabase(d.db, name)
		if err != nil {
			return et.Item{}, err
		}

		return et.Item{}, nil
	case "create_schema":
		name := params.Str("name")
		err := CreateSchema(d.db, name)
		if err != nil {
			return et.Item{}, err
		}

		return et.Item{}, nil
	case "create_column":
		schema := params.Str("schema")
		table := params.Str("table")
		name := params.Str("name")
		kind := params.Str("kind")
		_default := params.Str("default")
		err := CreateColumn(d.db, schema, table, name, kind, _default)
		if err != nil {
			return et.Item{}, err
		}

		return et.Item{}, nil
	case "create_index":
		schema := params.Str("schema")
		table := params.Str("table")
		name := params.Str("name")
		err := CreateIndex(d.db, schema, table, name)
		if err != nil {
			return et.Item{}, err
		}

		return et.Item{}, nil
	case "create_trigger":
		schema := params.Str("schema")
		table := params.Str("table")
		name := params.Str("name")
		when := params.Str("when")
		event := params.Str("event")
		function := params.Str("function")
		err := CreateTrigger(d.db, schema, table, name, when, event, function)
		if err != nil {
			return et.Item{}, err
		}

		return et.Item{}, nil
	case "create_sequence":
		schema := params.Str("schema")
		name := params.Str("name")
		err := CreateSequence(d.db, schema, name)
		if err != nil {
			return et.Item{}, err
		}

		return et.Item{}, nil
	case "create_user":
		name := params.Str("name")
		password := params.Str("password")
		err := CreateUser(d.db, name, password)
		if err != nil {
			return et.Item{}, err
		}

		return et.Item{}, nil
	case "change_password":
		name := params.Str("name")
		password := params.Str("password")
		err := ChangePassword(d.db, name, password)
		if err != nil {
			return et.Item{}, err
		}

		return et.Item{}, nil
	case "drop_database":
		name := params.Str("name")
		err := DropDatabase(d.db, name)
		if err != nil {
			return et.Item{}, err
		}

		return et.Item{}, nil
	case "drop_schema":
		name := params.Str("name")
		err := DropSchema(d.db, name)
		if err != nil {
			return et.Item{}, err
		}

		return et.Item{}, nil

	case "drop_table":
		schema := params.Str("schema")
		name := params.Str("name")
		err := DropTable(d.db, schema, name)
		if err != nil {
			return et.Item{}, err
		}

		return et.Item{}, nil
	case "drop_column":
		schema := params.Str("schema")
		table := params.Str("table")
		name := params.Str("name")
		err := DropColumn(d.db, schema, table, name)
		if err != nil {
			return et.Item{}, err
		}

		return et.Item{}, nil

	case "drop_index":
		schema := params.Str("schema")
		table := params.Str("table")
		name := params.Str("name")
		err := DropIndex(d.db, schema, table, name)
		if err != nil {
			return et.Item{}, err
		}

		return et.Item{}, nil
	case "drop_trigger":
		schema := params.Str("schema")
		table := params.Str("table")
		name := params.Str("name")
		err := DropTrigger(d.db, schema, table, name)
		if err != nil {
			return et.Item{}, err
		}

		return et.Item{}, nil
	case "drop_serie":
		schema := params.Str("schema")
		name := params.Str("name")
		err := DropSerie(d.db, schema, name)
		if err != nil {
			return et.Item{}, err
		}

		return et.Item{}, nil
	case "drop_user":
		name := params.Str("name")
		err := DropUser(d.db, name)
		if err != nil {
			return et.Item{}, err
		}

		return et.Item{}, nil
	default:
		return et.Item{}, nil
	}
}
