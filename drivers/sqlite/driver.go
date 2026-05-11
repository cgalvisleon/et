package sqlite

import (
	"database/sql"

	"github.com/cgalvisleon/et/jsql"
)

type Sqlite struct {
}

func (s *Sqlite) Connect(db *jsql.DB) (*sql.DB, error) {
	return nil, nil
}

func (s *Sqlite) Load(model *jsql.Model) (string, error) {
	return "", nil
}

func (s *Sqlite) Query(query *jsql.Query) (string, error) {
	return "", nil
}

func (s *Sqlite) Command(command *jsql.Command) (string, error) {
	return "", nil
}

func init() {
	jsql.Register(jsql.DriverSqlite, &Sqlite{})
}
