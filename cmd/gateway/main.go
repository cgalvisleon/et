package main

import (
	"os"
	"os/signal"

	"github.com/cgalvisleon/et/envar"
	serv "github.com/cgalvisleon/et/gateway"
	"github.com/cgalvisleon/et/logs"
	store "github.com/cgalvisleon/et/mem"
)

func main() {
	envar.SetInt("port", 3000, "Port server", "PORT")
	envar.SetInt("rpc", 4200, "Port rpc server", "RPC")
	envar.SetStr("dbhost", "localhost", "Database host", "DB_HOST")
	envar.SetInt("dbport", 5432, "Database port", "DB_PORT")
	envar.SetStr("dbname", "", "Database name", "DB_NAME")
	envar.SetStr("dbuser", "", "Database user", "DB_USER")
	envar.SetStr("dbpass", "", "Database password", "DB_PASSWORD")

	cache, err := store.Load()
	if err != nil {
		logs.Fatal(err)
	}

	serv, err := serv.Load(&cache)
	if err != nil {
		logs.Fatal(err)
	}

	go serv.Start()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	serv.Close()
}
