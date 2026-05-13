package mysql

import "github.com/cgalvisleon/et/jsql"

/**
* MySQL: Driver implementation for MySQL / MariaDB databases.
**/
type MySQL struct{}

func init() {
	jsql.Register(jsql.DriverMysql, &MySQL{})
}
