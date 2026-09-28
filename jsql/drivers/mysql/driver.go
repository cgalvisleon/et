package mysql

import (
	"errors"

	"github.com/cgalvisleon/et/jsql"
)

/**
* Mysql: Driver implementation for MySQL 8.0 or later.
* A model schema is a MySQL database; tables and columns are quoted with backticks. The SourceField is a
* JSON column. MySQL has no RETURNING, so a command runs as a batch (multiStatements) whose last SELECT
* returns the affected rows.
**/
type Mysql struct{}

// ErrFullJoin is returned for FULL JOIN, which MySQL does not support.
var ErrFullJoin = errors.New("mysql: FULL JOIN is not supported")

func init() {
	jsql.Register(jsql.DriverMysql, &Mysql{})
}
