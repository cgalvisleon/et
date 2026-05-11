package postgres

import (
	"github.com/cgalvisleon/et/jsql"
)

type Postgres struct {
}

func (s *Postgres) Load(model *jsql.Model) (string, error) {
	return "", nil
}

func (s *Postgres) Query(query *jsql.Query) (string, error) {
	return "", nil
}

func (s *Postgres) Command(command *jsql.Command) (string, error) {
	return "", nil
}

func init() {
	jsql.Register(jsql.DriverPostgres, &Postgres{})
}
