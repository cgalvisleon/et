package mysql

import (
	"errors"

	"github.com/cgalvisleon/et/jsql"
)

/**
* Mysql: Driver implementation for MySQL 8.0 or later.
* A model table is schema_name in the connection database (the schema is not a MySQL database); tables
* and columns are quoted with backticks. The SourceField is a JSON column. MySQL has no RETURNING, so a
* command runs as a batch (multiStatements) whose last SELECT returns the affected rows.
**/
type Mysql struct{}

// ErrFullJoin is returned for FULL JOIN, which MySQL does not support.
var ErrFullJoin = errors.New("mysql: FULL JOIN is not supported")

func init() {
	jsql.Register(jsql.DriverMysql, &Mysql{})
}

/**
* UseSchema: A MySQL schema is a separate database, so jsql does not use it: a model table is schema_name in the connection database.
* @return bool
**/
func (s *Mysql) UseSchema() bool {
	return false
}
