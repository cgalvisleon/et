package lib

import (
	"sync"

	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	_ "github.com/lib/pq"
)

// Postgres struct to define a postgres database
type Postgres struct {
	db         *linq.DB
	dm         *linq.DB
	params     *linq.Connection
	chain      string
	locks      map[string]*sync.RWMutex
	wgs        map[string]*sync.WaitGroup
	lastcomand int64
}

func NewDriver() linq.Driver {
	return &Postgres{
		locks: make(map[string]*sync.RWMutex),
		wgs:   make(map[string]*sync.WaitGroup),
	}
}

/**
* Type return the type of the database
* @return string
**/
func (d *Postgres) Type() string {
	return linq.Postgres.String()
}

/**
* Mode return the mode of the database
* @return linq.ModeDatabase
**/
func (d *Postgres) Mode() linq.ModeDatabase {
	return d.params.Mode
}

/**
* LoadMode return if the core is used
* @return bool
**/
func (d *Postgres) LoadMode() error {
	if d.params.Mode.IsNone() {
		return nil
	}

	err := defineCore(d.db)
	if err != nil {
		return err
	}

	err = defineSeries(d.db)
	if err != nil {
		return err
	}

	err = defineSync(d.db)
	if err != nil {
		return err
	}

	err = defineRecycling(d.db)
	if err != nil {
		return err
	}

	err = defineMigrateId(d.db)
	if err != nil {
		return err
	}

	return nil
}

/**
* Connect to the database
* @param params et.Json
* @return *linq.DB
* @return error
**/
func (d *Postgres) Connect(params *linq.Connection) (*linq.DB, error) {
	if d.db != nil {
		return d.db, nil
	}

	chain, err := ConnStr(params)
	if err != nil {
		return nil, err
	}

	db, err := Connect(chain)
	if err != nil {
		return nil, err
	}

	params.Password = ""
	d.chain = chain
	d.params = params
	d.db, err = linq.NewDB(params.Name, db, d)
	if err != nil {
		return nil, err
	}

	path := strs.Format(`%s:%d:%s:%s`, params.Host, params.Port, params.Database, params.Mode.String())
	logs.Logf("DB", "Connected to database:%s", path)

	err = d.LoadMode()
	if err != nil {
		return nil, err
	}

	return d.db, nil
}

/**
* Master execute a query
* @param db *linq.DB
* @param commandListener linq.HandlerListend
* @return error
**/
func (d *Postgres) Master(params *linq.Connection) (*linq.DB, error) {
	if d.dm != nil {
		return d.dm, nil
	}

	chain, err := ConnStr(params)
	if err != nil {
		return nil, err
	}

	db, err := Connect(chain)
	if err != nil {
		return nil, err
	}

	d.dm, err = linq.NewDB(params.Name, db, d)
	if err != nil {
		return nil, err
	}

	err = d.SyncCommand()
	if err != nil {
		return nil, err
	}

	path := strs.Format(`%s:%d/%s`, params.Host, params.Port, params.Database)
	logs.Logf("Master", "Connected to database:%s", path)

	return d.dm, nil
}

/**
* init
**/
func init() {
	linq.Register(linq.Postgres.String(), NewDriver)
}
