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

	v := jrex.New("jrex", db.Rules)
	err = v.RunDev("./cmd/jrex/src")
	if err != nil {
		logs.Panic(err)
	}

	err = v.Build(jrex.Same)
	if err != nil {
		logs.Panic(err)
	}
}
