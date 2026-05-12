package postgres

import "github.com/cgalvisleon/et/jsql"

/**
* Command: Generates the SQL DML string (INSERT, UPDATE, DELETE, UPSERT, BULK) for the given Command.
* @param command *jsql.Command
* @return string, error
**/
func (s *Postgres) Command(command *jsql.Command) (string, error) {
	return "", nil
}
