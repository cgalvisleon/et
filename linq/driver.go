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
	InitModel(model *Model) string
	SelectSql(linq *Linq) string
	CurrentSql(linq *Linq) string
	InsertSql(linq *Linq) string
	UpdateSql(linq *Linq) string
	DeleteSql(linq *Linq) string
	MutationSql(linq *Linq) string
	DCL(command string, params et.Json) error
	Sql(command string) string
	// Serires
	NextSerie(tag string) int64
	NextCode(tag, format string) string
	SetSerie(tag string, val int) int64
	CurrentSerie(tag string) int64
	DeleteSerie(tag string) int64
	// Models
	UpSetModel(main, name, kind string, version int, data et.Json) (et.Item, error)
	GetModel(main, name, kind string) (et.Item, error)
	DeleteModel(main, name, kind string) error
}
