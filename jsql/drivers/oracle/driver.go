package oracle

import (
	"github.com/cgalvisleon/et/jsql"
)

/**
* Oracle: Driver implementation for Oracle databases (19c or later).
* Tables and columns are created as quoted lowercase identifiers ("public_users"."name"),
* so names keep the same spelling as in the model; a model table is schema_name in the
* connection schema (an Oracle schema is a user, so jsql does not use it).
* JSON is stored in CLOB columns with an IS JSON check constraint.
**/
type Oracle struct{}

func init() {
	jsql.Register(jsql.DriverOracle, &Oracle{})
}

/**
* UseSchema: An Oracle schema is a user, so jsql does not use it: a model table is schema_name in the connection schema.
* @return bool
**/
func (s *Oracle) UseSchema() bool {
	return false
}
