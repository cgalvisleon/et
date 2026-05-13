package oracle

import "github.com/cgalvisleon/et/jsql"

/**
* Oracle: Driver implementation for Oracle Database (19c+).
**/
type Oracle struct{}

func init() {
	jsql.Register(jsql.DriverOracle, &Oracle{})
}
