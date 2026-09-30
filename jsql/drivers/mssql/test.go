package mssql

import (
	"context"

	"github.com/cgalvisleon/et/jsql"
)

/**
* Test: Tests a SQL Server connection (jsql.Tester): opens the database of the params (master if none), pings it and
* runs the query; it never creates the database, unlike Connect.
* @param ctx context.Context, connection jsql.Connection, query string
* @return error
**/
func (s *Mssql) Test(ctx context.Context, connection jsql.Connection, query string) error {
	params := connection.GetParams()
	database := params.Str("database")
	if database == "" {
		database = "master"
	}

	db, err := connectTo(ctx, dsn(params, database))
	if err != nil {
		return err
	}

	return jsql.Probe(ctx, db, query)
}
