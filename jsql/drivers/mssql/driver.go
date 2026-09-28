package mssql

import (
	"github.com/cgalvisleon/et/jsql"
)

/**
* Mssql: Driver implementation for SQL Server 2022 or later.
* Tables and columns are quoted with brackets; the SourceField is an NVARCHAR(MAX) column checked with
* ISJSON. SQL Server has no JSON object builder that keeps types, so each row is returned as JSON text
* built by the driver; commands run as a batch whose last SELECT returns the affected rows.
**/
type Mssql struct{}

func init() {
	jsql.Register(jsql.DriverMssql, &Mssql{})
}
