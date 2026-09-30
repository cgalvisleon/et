package mysql

import (
	"context"

	"github.com/cgalvisleon/et/jsql"
)

/**
* Test: Tests a MySQL connection (jsql.Tester): opens the server (and the database of the params, if any), pings it and
* runs the query; it never creates the database, unlike Connect.
* @param ctx context.Context, connection jsql.Connection, query string
* @return error
**/
func (s *Mysql) Test(ctx context.Context, connection jsql.Connection, query string) error {
	params := connection.GetParams()
	db, err := connectTo(ctx, dsn(params, params.Str("database")))
	if err != nil {
		return err
	}

	return jsql.Probe(ctx, db, query)
}
