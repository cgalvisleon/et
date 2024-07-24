package linq

import (
	"database/sql"

	"github.com/cgalvisleon/et/js"
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

type Connection struct {
	Drive    TypeDriver
	User     string
	Password string
	Host     string
	Port     int
	Database string
	App      string
	UsedCore bool
}

type HandleListen func(res js.Json)

type Driver interface {
	Type() string
	Connect() (*sql.DB, error)
	UsedCore() bool
	// DDL (Data Definition Language)
	DefineSql(model *Model) string
	MutationSql(model *Model) string
	// Querys
	Query(query string, args ...any) (js.Items, error)
	QueryOne(query string, args ...any) (js.Item, error)
	Exec(query string, args ...any) error
	// Crud
	SelectSql(linq *Linq) string
	CurrentSql(linq *Linq) string
	InsertSql(linq *Linq) string
	UpdateSql(linq *Linq) string
	DeleteSql(linq *Linq) string
	// DCL (Data Control Language)
	DCL(command string, params js.Json) error
	// Listener
	SetListen(handler HandleListen)
	// Vars
	SetVar(key, value string) error
	DelVal(key string) error
	Var(key string) (string, error)
	VarInt(key string) (int64, error)
	// Serires
	UUIndex(tag string) (int64, error)
	NextSerie(tag string) (int, error)
	NextCode(tag, format string) (string, error)
	SetSerie(tag string, val int) error
	CurrentSerie(tag string) (int, error)
	DeleteSerie(tag string) error
	// Models
	GetModel(main, name, kind string) (js.Item, error)
	InsertModel(main, name, kind string, version int, data js.Json) error
	UpdateModel(main, name, kind string, version int, data js.Json) error
	DeleteModel(main, name, kind string) error
	// Migrations IDs
	UpSertMigrateId(old_id, _id, tag string) error
	GetMigrateId(old_id, tag string) (string, error)
}
