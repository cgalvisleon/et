package main

import (
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

	logs.Debug("connected:", db.Name)

	model, err := db.DefineModel("public", "users", 1)
	if err != nil {
		return err
	}

	model.Debug()
	err = model.Init()
	if err != nil {
		return err
	}

	result, err := model.
		From("u").
		Where(jsql.Eq("u.id", 1)).
		Select("u.id", "u.name", "u.email", "u.full_name").
		Test().
		Debug().
		One()
	if err != nil {
		return err
	}

	logs.Debug("result:", result)

	return nil
}

func main() {
	demoDBConnect()
}
