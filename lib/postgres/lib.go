package lib

import (
	"database/sql"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	_ "github.com/lib/pq"
)

// Postgres struct to define a postgres database
type Postgres struct {
	DB     *sql.DB
	Params et.Json
	locks  map[string]*sync.RWMutex
	wgs    map[string]*sync.WaitGroup
}

func NewDriver(params et.Json) *Postgres {
	return &Postgres{
		Params: params,
		locks:  make(map[string]*sync.RWMutex),
		wgs:    make(map[string]*sync.WaitGroup),
	}
}

/**
* wg return a wait group
* @param tag string
* @return *sync.WaitGroup
**/
func (d *Postgres) wg(tag string) *sync.WaitGroup {
	if d.wgs[tag] == nil {
		d.wgs[tag] = &sync.WaitGroup{}
	}

	return d.wgs[tag]
}

/**
* wgAdd add a delta to a wait group
* @param tag string
* @param delta int
* @return *sync.WaitGroup
**/
func (d *Postgres) wgAdd(tag string, delta int) *sync.WaitGroup {
	result := d.wg(tag)
	result.Add(delta)

	return result
}

/**
* lock return a lock
* @param tag string
* @return *sync.RWMutex
**/
func (d *Postgres) lock(tag string) *sync.RWMutex {
	if d.locks[tag] == nil {
		d.locks[tag] = &sync.RWMutex{}
	}

	return d.locks[tag]
}

/**
* Type return the type of the database
* @return string
**/
func (d *Postgres) Type() string {
	return linq.Postgres.String()
}

/**
* Connect to the database
* @param params et.Json
* @return *sql.DB
* @return error
**/
func (d *Postgres) Connect(params et.Json) (*sql.DB, error) {
	if params["user"] == nil {
		return nil, logs.Errorm("User is required")
	}

	if params["password"] == nil {
		return nil, logs.Errorm("Password is required")
	}

	if params["host"] == nil {
		return nil, logs.Errorm("Host is required")
	}

	if params["port"] == nil {
		return nil, logs.Errorm("Port is required")
	}

	if params["database"] == nil {
		return nil, logs.Errorm("Database is required")
	}

	if params["app"] == nil {
		return nil, logs.Errorm("App name is required")
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
		return nil, err
	}

	err = defineSeries(db)
	if err != nil {
		return nil, err
	}

	err = defineModels(db)
	if err != nil {
		return nil, err
	}

	err = defineSync(db)
	if err != nil {
		return nil, err
	}

	err = defineRecycling(db)
	if err != nil {
		return nil, err
	}

	go defineListen(connStr, []string{"sync", "recycling"})

	logs.Logf("DB", "Connected to %s database %s", driver, database)

	params.Del("password")
	d.DB = db
	d.Params = params

	return d.DB, nil
}

/**
* Query execute a query
* @param query string
* @param args ...any
* @return et.Items
* @return error
**/
func (d *Postgres) Query(query string, args ...any) (et.Items, error) {
	rows, err := d.DB.Query(query, args...)
	if err != nil {
		return et.Items{}, err
	}

	defer rows.Close()

	result := linq.RowsItems(rows)

	return result, nil
}

/**
* QueryOne execute a query and return one row
* @param query string
* @param args ...any
* @return et.Item
* @return error
**/
func (d *Postgres) QueryOne(query string, args ...any) (et.Item, error) {
	items, err := d.Query(query, args...)
	if err != nil {
		return et.Item{}, err
	}

	if items.Count == 0 {
		return et.Item{}, nil
	}

	return et.Item{
		Ok:     true,
		Result: items.Result[0],
	}, nil
}

/**
* Exec execute a query
* @param query string
* @param args ...any
* @return error
**/
func (d *Postgres) Exec(query string, args ...any) error {
	_, err := d.DB.Exec(query, args...)
	if err != nil {
		return err
	}

	return nil
}

/**
* DefineSql return DDL sql to define a model
* @param m *linq.Model
* @return string
**/
func (d *Postgres) DefineSql(m *linq.Model) string {
	result := ddlTable(m)

	result = strs.Format(`%s`, result)

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
