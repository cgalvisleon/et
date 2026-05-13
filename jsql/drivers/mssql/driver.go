package mssql

import "github.com/cgalvisleon/et/jsql"

/**
* MSSQL: Driver implementation for Microsoft SQL Server databases.
**/
type MSSQL struct{}

func init() {
	jsql.Register(jsql.DriverMssql, &MSSQL{})
}
