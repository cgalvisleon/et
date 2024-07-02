package lib

import (
	"database/sql"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	_ "github.com/lib/pq"
)

// Postgres struct to define a postgres database
type Postgres struct {
	Params et.Json
	DB     *sql.DB
}

/**
* Type return the type of the database
* @return string
**/
func (d Postgres) Type() string {
	return linq.Postgres.String()
}

/**
* Connect to the database
* @param params et.Json
* @return *sql.DB
* @return error
**/
func (d Postgres) Connect(params et.Json) error {
	if params["user"] == nil {
		logs.Errorm("User is required")
	}

	if params["password"] == nil {
		logs.Errorm("Password is required")
	}

	if params["host"] == nil {
		logs.Errorm("Host is required")
	}

	if params["port"] == nil {
		logs.Errorm("Port is required")
	}

	if params["database"] == nil {
		logs.Errorm("Database is required")
	}

	if params["app"] == nil {
		logs.Errorm("App name is required")
	}

	driver := "postgres"
	user := params.Str("user")
	password := params.Str("password")
	host := params.Str("host")
	port := params.Int("port")
	database := params.Str("database")
	app := params.Str("app")

	connStr := strs.Format(`%s://%s:%s@%s:%d/%s?sslmode=disable&application_name=%s`, driver, user, password, host, port, database, app)
	db, err := sql.Open(driver, connStr)
	if err != nil {
		return err
	}

	err = db.Ping()
	if err != nil {
		return err
	}

	_, err = defineSeries(db)
	if err != nil {
		return err
	}

	_, err = defineModels(db)
	if err != nil {
		return err
	}

	_, err = defineSync(db)
	if err != nil {
		return err
	}

	_, err = defineRecycling(db)
	if err != nil {
		return err
	}

	logs.Infof("Connected to %s database %s", driver, database)

	params.Del("password")
	d.DB = db
	d.Params = params

	return nil
}

/**
* DefineSql return DDL sql to define a model
* @param m *linq.Model
* @return string
**/
func (d *Postgres) DefineSql(m *linq.Model) string {
	var result string

	result = ddlTable(m)

	result = strs.Format(`%s;`, result)

	return result
}

/**
* MutationSql return DDL mutation the sql to mutate
* @param l *linq.Linq
* @return string
**/
func (d *Postgres) MutationSql(m *linq.Model) string {

	return ""
}

/**
* SelectSql return the sql to select
* @param l *linq.Linq
* @return string
**/
func (d *Postgres) SelectSql(l *linq.Linq) string {
	l.Clear()

	sqlSelect(l)

	sqlFrom(l)

	sqlJoin(l)

	sqlWhere(l)

	sqlGroupBy(l)

	sqlHaving(l)

	sqlOrderBy(l)

	sqlLimit(l)

	sqlOffset(l)

	return l.SQL()
}

/**
* CurrentSql return the sql to select
* @param l *linq.Linq
* @return string
**/
func (d *Postgres) CurrentSql(l *linq.Linq) string {
	l.Clear()

	sqlCurrent(l)

	sqlFrom(l)

	sqlWhere(l)

	sqlLimit(l)

	return l.SQL()
}

/**
* InsertSql return the sql to insert
* @param l *linq.Linq
* @return string
**/
func (d *Postgres) InsertSql(l *linq.Linq) string {
	l.Clear()

	sqlInsert(l)

	sqlReturns(l)

	return l.SQL()
}

/**
* UpdateSql return the sql to update
* @param l *linq.Linq
* @return string
**/
func (d *Postgres) UpdateSql(l *linq.Linq) string {
	l.Clear()

	sqlUpdate(l)

	sqlWhere(l)

	sqlReturns(l)

	return l.SQL()
}

/**
* DeleteSql return the sql to delete
* @param l *linq.Linq
* @return string
**/
func (d *Postgres) DeleteSql(l *linq.Linq) string {
	l.Clear()

	sqlDelete(l)

	sqlWhere(l)

	sqlReturns(l)

	return l.SQL()
}

/**
* DCL execute a Data Control Language command
* @param command string
* @param params et.Json
* @return error
**/
func (d *Postgres) DCL(command string, params et.Json) error {
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
		_, err := CreateDatabase(d.DB, name)
		if err != nil {
			return err
		}

		return nil
	case "create_schema":
		name := params.Str("name")
		_, err := CreateSchema(d.DB, name)
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
		_, err := CreateColumn(d.DB, schema, table, name, kind, _default)
		if err != nil {
			return err
		}

		return nil
	case "create_index":
		schema := params.Str("schema")
		table := params.Str("table")
		name := params.Str("name")
		_, err := CreateIndex(d.DB, schema, table, name)
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
		_, err := CreateTrigger(d.DB, schema, table, name, when, event, function)
		if err != nil {
			return err
		}

		return nil
	case "create_sequence":
		schema := params.Str("schema")
		name := params.Str("name")
		_, err := CreateSequence(d.DB, schema, name)
		if err != nil {
			return err
		}

		return nil
	case "create_user":
		name := params.Str("name")
		password := params.Str("password")
		_, err := CreateUser(d.DB, name, password)
		if err != nil {
			return err
		}

		return nil
	case "change_password":
		name := params.Str("name")
		password := params.Str("password")
		_, err := ChangePassword(d.DB, name, password)
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
