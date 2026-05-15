package sqlite

import (
	"github.com/cgalvisleon/et/jsql"
)

type Sqlite struct {
}

func init() {
	jsql.Register(jsql.DriverSqlite, &Sqlite{})
}
