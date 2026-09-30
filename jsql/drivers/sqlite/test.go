package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/cgalvisleon/et/jsql"
)

/**
* Test: Tests a SQLite connection (jsql.Tester): the file must already exist (opening a missing one would create it, as
* Connect does), then it is opened, pinged and the query runs.
* @param ctx context.Context, connection jsql.Connection, query string
* @return error
**/
func (s *Sqlite) Test(ctx context.Context, connection jsql.Connection, query string) error {
	path := connection.GetParams().Str("file")
	if path == "" {
		return fmt.Errorf("the sqlite file is required")
	}

	if _, err := os.Stat(path); err != nil {
		return err
	}

	db, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		return err
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return err
	}

	return jsql.Probe(ctx, db, query)
}
