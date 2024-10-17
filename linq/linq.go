package linq

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
)

// Global variables
var (
	MaxUpdate = 1000
	MaxDelete = 1000
	MaxRows   = 1000
	drivers   map[string]func() Driver
	dbs       []*DB
	schemas   []*Schema
	models    []*Model
)

// Linq struct
type Linq struct {
	DB        *DB
	Froms     []*Lfrom
	Columns   []*Lselect
	Atribs    []*Lselect
	Selects   *Lcolumns
	Datas     *Lcolumns
	Returns   *Lcolumns
	Details   *Lcolumns
	Wheres    []*Lwhere
	Groups    []*Lgroup
	Havings   []*Lwhere
	isHaving  bool
	Orders    []*Lorder
	Joins     []*Ljoin
	Union     []*Linq
	Limit     int
	Offset    int
	Values    *Values
	TypeQuery TypeQuery
	Sql       string
	Args      []any
	Result    *et.Item
	as        int
	sets      et.Json
	debug     bool
	showModel bool
}

/**
* Describe return a json with the definition of the linq
* @return et.Json
**/
func (l *Linq) Describe() *et.Json {
	var froms []et.Json = []et.Json{}
	for _, f := range l.Froms {
		froms = append(froms, f.Describe())
	}

	var columns []et.Json = []et.Json{}
	for _, c := range l.Columns {
		columns = append(columns, c.Describe())
	}

	var atribs []et.Json = []et.Json{}
	for _, a := range l.Atribs {
		atribs = append(atribs, a.Describe())
	}

	var wheres []et.Json = []et.Json{}
	for _, w := range l.Wheres {
		wheres = append(wheres, w.Describe())
	}

	var groups []et.Json = []et.Json{}
	for _, g := range l.Groups {
		groups = append(groups, g.Describe())
	}

	var havings []et.Json = []et.Json{}
	for _, h := range l.Havings {
		havings = append(havings, h.Describe())
	}

	var orders []et.Json = []et.Json{}
	for _, o := range l.Orders {
		orders = append(orders, o.Describe())
	}

	var joins []et.Json = []et.Json{}
	for _, j := range l.Joins {
		joins = append(joins, j.Describe())
	}

	var unions []et.Json = []et.Json{}
	for _, u := range l.Union {
		unions = append(unions, *u.Describe())
	}

	return &et.Json{
		"as":        l.as,
		"froms":     froms,
		"columns":   columns,
		"atribs":    atribs,
		"selects":   l.Selects.Describe(),
		"data":      l.Datas.Describe(),
		"returns":   l.Returns.Describe(),
		"details":   l.Details.Describe(),
		"wheres":    wheres,
		"groups":    groups,
		"havings":   havings,
		"orders":    orders,
		"joins":     joins,
		"unions":    unions,
		"limit":     l.Limit,
		"offset":    l.Offset,
		"values":    l.Values.Describe(),
		"typeQuery": l.TypeQuery.String(),
		"sql":       l.Sql,
	}
}

/**
* Debug show sql in debug mode
* @return *Linq
**/
func (l *Linq) Debug() *Linq {
	l.debug = true

	return l
}

/**
* ShowModel show model id debug model
* @return *Linq
**/
func (l *Linq) ShowModel() *Linq {
	l.showModel = true

	return l
}

/**
* Register register a driver
* @return *Linq
**/
func Register(name string, driver func() Driver) {
	drivers[name] = driver
}

/**
* Load a database
* @return *DB
* @return error
**/
func Load(params *Connection) (*DB, error) {
	newDrive, ok := drivers[params.Drive.String()]
	if !ok {
		return nil, logs.Alertm(MSG_DRIVER_NOT_FOUND)
	}

	drive := newDrive()
	result, err := drive.Connect(params)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* Core a database
* @return *linq.DB
* @return error
**/
func Core() (*DB, error) {
	kind := envar.GetStr("postgre", "DB_DRIVE")
	host := envar.GetStr("localhost", "DB_HOST")
	port := envar.GetInt(5432, "DB_PORT")
	name := envar.GetStr("test", "DB_NAME")
	user := envar.GetStr("test", "DB_USER")
	password := envar.GetStr("test", "DB_PASSWORD")
	app := envar.GetStr("test", "DB_APP_NAME")
	masterHost := envar.GetStr("", "DB_MASTER_HOST")

	connect := &Connection{
		Name:     "core",
		Drive:    StrToDriver(kind),
		User:     user,
		Password: password,
		Host:     host,
		Port:     port,
		Database: name,
		App:      app,
		Mode:     ModeCore,
	}

	result, err := Load(connect)
	if err != nil {
		return nil, logs.Alert(err)
	}

	if len(masterHost) == 0 {
		return result, nil
	}

	result.Master(&Connection{
		Name:     "master",
		Drive:    StrToDriver(kind),
		User:     user,
		Password: password,
		Host:     envar.GetStr(host, "DB_MASTER_HOST"),
		Port:     envar.GetInt(port, "DB_MASTER_PORT"),
		Database: name,
		App:      app,
		Mode:     ModeCore,
	})

	return result, nil
}

/**
* Master a database
* @return *linq.DB
* @return error
**/
func Master() (*DB, error) {
	kind := envar.GetStr("postgre", "DB_DRIVE")
	host := envar.GetStr("localhost", "DB_HOST")
	port := envar.GetInt(5432, "DB_PORT")
	name := envar.GetStr("test", "DB_NAME")
	user := envar.GetStr("test", "DB_USER")
	password := envar.GetStr("test", "DB_PASSWORD")
	app := envar.GetStr("test", "DB_APP_NAME")

	connect := &Connection{
		Name:     "core",
		Drive:    StrToDriver(kind),
		User:     user,
		Password: password,
		Host:     host,
		Port:     port,
		Database: name,
		App:      app,
		Mode:     ModeMaster,
	}

	result, err := Load(connect)
	if err != nil {
		return nil, logs.Alert(err)
	}

	logs.Log("Master", "Server master Up")

	return result, nil

}

/**
* Read a database
* @return *linq.DB
* @return error
**/
func Read() (*DB, error) {
	kind := envar.GetStr("postgre", "DB_DRIVE")
	host := envar.GetStr("localhost", "DB_HOST")
	port := envar.GetInt(5432, "DB_PORT")
	name := envar.GetStr("test", "DB_NAME")
	user := envar.GetStr("test", "DB_USER")
	password := envar.GetStr("test", "DB_PASSWORD")
	app := envar.GetStr("test", "DB_APP_NAME")

	connect := &Connection{
		Name:     "read",
		Drive:    StrToDriver(kind),
		User:     user,
		Password: password,
		Host:     host,
		Port:     port,
		Database: name,
		App:      app,
		Mode:     ModeRead,
	}

	result, err := Load(connect)
	if err != nil {
		return nil, logs.Alert(err)
	}

	return result, nil
}

/**
* Mocks a database
* @return *linq.DB
* @return error
**/
func Mocks() (*DB, error) {
	kind := envar.GetStr("postgre", "DB_DRIVE")
	host := envar.GetStr("localhost", "DB_HOST")
	port := envar.GetInt(5432, "DB_PORT")
	name := envar.GetStr("test", "DB_NAME")
	user := envar.GetStr("test", "DB_USER")
	password := envar.GetStr("test", "DB_PASSWORD")
	app := envar.GetStr("test", "DB_APP_NAME")

	connect := &Connection{
		Name:     "read",
		Drive:    StrToDriver(kind),
		User:     user,
		Password: password,
		Host:     host,
		Port:     port,
		Database: name,
		App:      app,
		Mode:     ModeMocks,
	}

	result, err := Load(connect)
	if err != nil {
		return nil, logs.Alert(err)
	}

	return result, nil
}

/**
* None a database
* @return *linq.DB
* @return error
**/
func None() (*DB, error) {
	kind := envar.GetStr("postgre", "DB_DRIVE")
	host := envar.GetStr("localhost", "DB_HOST")
	port := envar.GetInt(5432, "DB_PORT")
	name := envar.GetStr("test", "DB_NAME")
	user := envar.GetStr("test", "DB_USER")
	password := envar.GetStr("test", "DB_PASSWORD")
	app := envar.GetStr("test", "DB_APP_NAME")

	connect := &Connection{
		Name:     "read",
		Drive:    StrToDriver(kind),
		User:     user,
		Password: password,
		Host:     host,
		Port:     port,
		Database: name,
		App:      app,
		Mode:     ModeNone,
	}

	result, err := Load(connect)
	if err != nil {
		return nil, logs.Alert(err)
	}

	return result, nil
}

// Init linq
func init() {
	drivers = map[string]func() Driver{}
	dbs = []*DB{}
	schemas = []*Schema{}
	models = []*Model{}
}
