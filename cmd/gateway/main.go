package main

import (
	"os"
	"os/signal"

	"github.com/cgalvisleon/et/envar"
	serv "github.com/cgalvisleon/et/gateway"
	"github.com/cgalvisleon/et/logs"
)

func main() {
	envar.SetInt("port", 3000, "PORT")
	envar.SetInt("rpc", 4200, "RPC")
	envar.SetStr("dbhost", "localhost", "DB_HOST")
	envar.SetInt("dbport", 5432, "DB_PORT")
	envar.SetStr("dbname", "", "DB_NAME")
	envar.SetStr("dbuser", "", "DB_USER")
	envar.SetStr("dbpass", "", "DB_PASSWORD")

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
