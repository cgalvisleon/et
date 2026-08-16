package sqlite

import (
	"github.com/cgalvisleon/et/jsql"
)

/**
* Sqlite: Driver implementation for SQLite databases.
**/
type Sqlite struct{}

func init() {
	jsql.Register(jsql.DriverSqlite, &Sqlite{})
}
