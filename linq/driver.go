package linq

import "github.com/cgalvisleon/et/et"

type TypeDriver int

const (
	Postgres TypeDriver = iota
	Mysql
	Sqlite
	Oracle
	SQLServer
)

/**
* String return the string of the driver
* @return string
**/
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

func StrToDriver(s string) TypeDriver {
	switch s {
	case "postgres":
		return Postgres
	case "mysql":
		return Mysql
	case "sqlite":
		return Sqlite
	case "oracle":
		return Oracle
	case "sqlserver":
		return SQLServer
	}
	return Postgres
}

type Connection struct {
	Name     string
	Drive    TypeDriver
	User     string
	Password string
	Host     string
	Port     int
	Database string
	App      string
	Mode     ModeDatabase
}

type HandlerListend func(res et.Json)

type Driver interface {
	Type() string
	Mode() ModeDatabase
	LoadMode() error
	Connect(params *Connection) (*DB, error)
	Master(params *Connection) (*DB, error)
	// Listen
	SetListen(channels []string, handler HandlerListend)
	SetCommand(query string) error
	SetMutex(id, query string, index int64) error
	GetCommand(id string) (et.Item, error)
	SyncCommand() error
	// DDL (Data Definition Language)
	DefineSql(model *Model) string
	MutationSql(model *Model) string
	// Crud
	SelectSql(linq *Linq) string
	CurrentSql(linq *Linq) string
	InsertSql(linq *Linq) string
	UpdateSql(linq *Linq) string
	DeleteSql(linq *Linq) string
	// DCL (Data Control Language)
	DCL(command string, params et.Json) (et.Item, error)
	// Series
	NextSerie(tag string) (int, error)
	NextCode(tag, format string) (string, error)
	SetSerie(tag string, val int) error
	CurrentSerie(tag string) (int, error)
	DeleteSerie(tag string) error
	// Migration IDs
	UpSertMigrateId(old_id, _id, tag string) error
	GetMigrateId(old_id, tag string) (string, error)
	// Models
	ModelExist(schema, name string) (bool, error)
}
