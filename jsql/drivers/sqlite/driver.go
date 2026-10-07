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

/**
* UseSchema: SQLite has no schemas: a model table is schema_name.
* @return bool
**/
func (s *Sqlite) UseSchema() bool {
	return false
}
