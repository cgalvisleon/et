package oracle

import "github.com/cgalvisleon/et/jsql"

/**
* Query: Generates the SQL SELECT string for the given Query descriptor. Not implemented yet.
* @param query *jsql.Query
* @return string, error
**/
func (s *Oracle) Query(query *jsql.Query) (string, error) {
	return "", errNotImplemented
}
