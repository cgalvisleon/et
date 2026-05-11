package main

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/utility"
)

func main() {
	config := utility.NewConfig(et.Json{})
	jsql.LoadTo(config)
}
