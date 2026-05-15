package postgres

import (
	"github.com/cgalvisleon/et/jsql"
)

/**
* Postgres: Driver implementation for PostgreSQL databases.
**/
type Postgres struct{}

func init() {
	jsql.Register(jsql.DriverPostgres, &Postgres{})
}
