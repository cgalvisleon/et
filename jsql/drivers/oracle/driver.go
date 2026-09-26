package oracle

import (
	"github.com/cgalvisleon/et/jsql"
)

/**
* Oracle: Driver implementation for Oracle databases (19c or later).
* Tables and columns are created as quoted lowercase identifiers ("users"."name"),
* so names keep the same spelling as in the model; a model schema maps to an existing
* Oracle schema (user), written unquoted so it resolves in uppercase.
* JSON is stored in CLOB columns with an IS JSON check constraint.
**/
type Oracle struct{}

func init() {
	jsql.Register(jsql.DriverOracle, &Oracle{})
}
