package oracle

import "github.com/cgalvisleon/et/jsql"

/**
* Command: Generates the DML SQL (INSERT/UPDATE/DELETE/UPSERT) for the given Command. Not implemented yet.
* @param command *jsql.Command
* @return string, error
**/
func (s *Oracle) Command(command *jsql.Command) (string, error) {
	return "", errNotImplemented
}
