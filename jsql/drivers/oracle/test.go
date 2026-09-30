package oracle

import (
	"context"

	"github.com/cgalvisleon/et/jsql"
)

/**
* Test: Tests an Oracle connection (jsql.Tester): opens the service of the params, pings it and runs the query.
* @param ctx context.Context, connection jsql.Connection, query string
* @return error
**/
func (s *Oracle) Test(ctx context.Context, connection jsql.Connection, query string) error {
	db, err := connectTo(ctx, chain(connection.GetParams()))
	if err != nil {
		return err
	}

	return jsql.Probe(ctx, db, query)
}
