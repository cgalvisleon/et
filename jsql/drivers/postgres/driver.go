package postgres

import (
	"github.com/cgalvisleon/et/jsql"
)

type Postgres struct {
}

func init() {
	jsql.Register(jsql.DriverPostgres, &Postgres{})
}
