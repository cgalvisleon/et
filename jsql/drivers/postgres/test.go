package postgres

import (
	"context"

	"github.com/cgalvisleon/et/jsql"
)

/**
* Test: Tests a PostgreSQL connection (jsql.Tester): opens the database of the params (postgres if none), pings it and
* runs the query; it never creates the database, unlike Connect.
* @param ctx context.Context, connection jsql.Connection, query string
* @return error
**/
func (s *Postgres) Test(ctx context.Context, connection jsql.Connection, query string) error {
	params := connection.GetParams()
	if params.Str("database") == "" {
		params["database"] = "postgres"
	}

	db, err := connectTo(ctx, chain(params))
	if err != nil {
		return err
	}

	return jsql.Probe(ctx, db, query)
}
