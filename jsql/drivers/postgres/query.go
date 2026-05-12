package postgres

import "github.com/cgalvisleon/et/jsql"

/**
* Query: Generates the SQL SELECT string for the given Query descriptor.
* @param query *jsql.Query
* @return string, error
**/
func (s *Postgres) Query(query *jsql.Query) (string, error) {
	return "", nil
}
