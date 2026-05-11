package main

import (
	"github.com/cgalvisleon/et/jsql"
)

func main() {
	db, err := jsql.Load()
	if err != nil {
		panic(err)
	}

	defer db.Close()
}
