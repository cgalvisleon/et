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
	envar.SetvarInt("port", 3000, "Port server", "PORT")
	envar.SetvarInt("rpc", 4200, "Port rpc server", "RPC")
	envar.SetvarStr("dbhost", "localhost", "Database host", "DB_HOST")
	envar.SetvarInt("dbport", 5432, "Database port", "DB_PORT")
	envar.SetvarStr("dbname", "", "Database name", "DB_NAME")
	envar.SetvarStr("dbuser", "", "Database user", "DB_USER")
	envar.SetvarStr("dbpass", "", "Database password", "DB_PASSWORD")

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
