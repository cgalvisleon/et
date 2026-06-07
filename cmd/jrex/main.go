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

	v, err := jrex.New("jrex", db.Rules)
	if err != nil {
		logs.Panic(err)
	}

	err = v.RunDev("./cmd/jrex/src")
	if err != nil {
		logs.Panic(err)
	}

}
