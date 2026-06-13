package main

import (
	"github.com/cgalvisleon/et/jrex"
	"github.com/cgalvisleon/et/jsql"
	_ "github.com/cgalvisleon/et/jsql/drivers/postgres"
	"github.com/cgalvisleon/et/logs"
)

func main() {
	db, err := jsql.Load(nil)
	if err != nil {
		logs.Panic(err)
	}
	defer db.Close()

	logs.Debug("connected:", db.Name)

	model, err := db.DefineModel("apps", "users", 1)
	if err != nil {
		logs.Panic(err)
	}

	err = model.Init()
	if err != nil {
		logs.Panic(err)
	}

	v, err := jrex.New("jrex", nil)
	if err != nil {
		logs.Panic(err)
	}

	v.Set("db", db)
	v.Set(model.Name, model)

	result, err := v.Run()
	if err != nil {
		logs.Panic(err)
	}

	logs.Debug("result:", result.ToString())
}
