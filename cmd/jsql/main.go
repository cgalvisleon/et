package main

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	_ "github.com/cgalvisleon/et/jsql/drivers/postgres"
	"github.com/cgalvisleon/et/logs"
)

// demoDBConnect attempts a live connection using env vars
// (DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME).
func demoDBConnect() error {
	db, err := jsql.Load()
	if err != nil {
		return err
	}
	defer db.Close()

	result, err := db.Query(et.Json{})
	if err != nil {
		return err
	}

	logs.Debug("result:", result.ToString())

	return nil
}

func main() {
	demoDBConnect()
}
