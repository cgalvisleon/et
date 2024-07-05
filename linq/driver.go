package linq

import (
	"database/sql"

	"github.com/cgalvisleon/et/et"
)

type TypeDriver int

const (
	Postgres TypeDriver = iota
	Mysql
	Sqlite
	Oracle
	SQLServer
)

func (d TypeDriver) String() string {
	switch d {
	case Postgres:
		return "postgres"
	case Mysql:
		return "mysql"
	case Sqlite:
		return "sqlite"
	case Oracle:
		return "oracle"
	case SQLServer:
		return "sqlserver"
	}
	return ""
}

type Driver interface {
	Type() string
	Connect(params et.Json) (*sql.DB, error)
	// DDL (Data Definition Language)
	DefineSql(model *Model) string
	MutationSql(model *Model) string
	// Querys
	Query(query string, args ...any) (et.Items, error)
	QueryOne(query string, args ...any) (et.Item, error)
	Exec(query string, args ...any) error
	// Crud
	SelectSql(linq *Linq) string
	CurrentSql(linq *Linq) string
	InsertSql(linq *Linq) string
	UpdateSql(linq *Linq) string
	DeleteSql(linq *Linq) string
	// DCL (Data Control Language)
	DCL(command string, params et.Json) error
	// Serires
	NextSerie(tag string) (int, error)
	NextCode(tag, format string) (string, error)
	SetSerie(tag string, val int) error
	CurrentSerie(tag string) (int, error)
	DeleteSerie(tag string) error
	// Models
	GetModel(main, name, kind string) (et.Item, error)
	InsertModel(main, name, kind string, version int, data et.Json) error
	UpdateModel(main, name, kind string, version int, data et.Json) error
	DeleteModel(main, name, kind string) error
}
