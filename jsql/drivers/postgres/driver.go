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

/**
* UseSchema: PostgreSQL has schemas: a model table is schema.name.
* @return bool
**/
func (s *Postgres) UseSchema() bool {
	return true
}
