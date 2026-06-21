package main

import (
	"github.com/cgalvisleon/et/jrex"
	"github.com/cgalvisleon/et/jsql"
	_ "github.com/cgalvisleon/et/jsql/drivers/postgres"
	"github.com/cgalvisleon/et/logs"
)

func main() {
	tenantId := "37860631"
	db, err := jsql.Load(tenantId)
	if err != nil {
		logs.Panic(err)
	}
	defer db.Close()

	logs.Debug("connected:", db.Name)

	model, err := db.DefineModel("apps", "users", 1, "admin")
	if err != nil {
		logs.Panic(err)
	}

	err = model.Init()
	if err != nil {
		logs.Panic(err)
	}

	v, err := jrex.Load("jrex", nil)
	if err != nil {
		logs.Panic(err)
	}

	v.Set("db", db)
	v.Set("getDb", jsql.GetDb)
	v.Set(model.Name, model)
	err = v.RunDev()
	if err != nil {
		logs.Panic(err)
	}
}
