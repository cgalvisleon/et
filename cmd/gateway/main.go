package main

import (
	"os"
	"os/signal"

	"github.com/cgalvisleon/et/envar"
	serv "github.com/cgalvisleon/et/gateway"
	"github.com/cgalvisleon/et/logs"
)

func main() {
	envar.SetInt("port", 3000, "Puerto Http", "PORT")
	envar.SetInt("rpc", 4200, "Puerto RPC", "RPC")
	envar.SetStr("dbhost", "localhost", "Database host", "DB_HOST")
	envar.SetInt("dbport", 5432, "Database port", "DB_PORT")
	envar.SetStr("dbname", "", "Database name", "DB_NAME")
	envar.SetStr("dbuser", "", "Database user", "DB_USER")
	envar.SetStr("dbpass", "", "Database password", "DB_PASSWORD")

	serv, err := serv.Load()
	if err != nil {
		logs.Fatal(err)
	}

	go serv.Start()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	serv.Close()
}
