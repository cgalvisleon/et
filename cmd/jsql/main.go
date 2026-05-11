package main

import (
	"fmt"

	"github.com/cgalvisleon/et/jsql"
	_ "github.com/cgalvisleon/et/jsql/drivers/postgres"
)

// demoDBConnect attempts a live connection using env vars
// (DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME).
func demoDBConnect() error {
	db, err := jsql.Load()
	if err != nil {
		return err
	}
	defer db.Close()

	fmt.Println("  connected:", db.Name)

	model, err := db.NewModel("public", "users", 1)
	if err != nil {
		return err
	}

	result, err := model.
		Where(jsql.Eq("id", 1)).
		Test().
		All()
	if err != nil {
		return err
	}

	fmt.Println("  result:", result)

	return nil
}

func main() {
	demoDBConnect()
}
