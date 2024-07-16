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
	DB      *sql.DB
	Params  *linq.Connection
	connStr string
	locks   map[string]*sync.RWMutex
	wgs     map[string]*sync.WaitGroup
}

func NewDriver(params *linq.Connection) *Postgres {
	params.Drive = linq.Postgres
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
func (d *Postgres) Lock(tag string) *sync.RWMutex {
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
func (d *Postgres) Connect() (*sql.DB, error) {
	if d.DB != nil {
		return d.DB, nil
	}

	connStr, err := ConnStr(d.Params)
	if err != nil {
		return nil, err
	}

	db, err := Connect(connStr)
	if err != nil {
		return nil, err
	}

	d.Params.Password = ""
	d.connStr = connStr
	d.DB = db

	if !d.Params.UsedCore {
		logs.Logf("DB", "Connected to database:%s", strs.Uppcase(d.Params.Database))

		return d.DB, nil
	}

	err = defineCore(db)
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

	err = defineMigrateId(db)
	if err != nil {
		return nil, err
	}

	logs.Logf("DB", "Connected to database:%s", strs.Uppcase(d.Params.Database))

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

	sqlReturns(l)

	return l.SQL()
}

/**
* UpSertMigrateId upsert a migrate id
* @param old_id string
* @param _id string
* @param tag string
* @return error
**/
func (d *Postgres) UpSertMigrateId(old_id, _id, tag string) error {
	return upSertMigrateId(d.DB, old_id, _id, tag)
}

/**
* GetMigrateId get a migrate id
* @param old_id string
* @param tag string
* @return string
* @return error
**/
func (d *Postgres) GetMigrateId(old_id, tag string) (string, error) {
	return getMigrateId(d.DB, old_id, tag)
}
