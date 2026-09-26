package oracle

import (
	"errors"

	"github.com/cgalvisleon/et/jsql"
)

var errNotImplemented = errors.New("oracle: not implemented")

/**
* Oracle: Driver implementation for Oracle databases.
**/
type Oracle struct{}

func init() {
	jsql.Register(jsql.DriverOracle, &Oracle{})
}
