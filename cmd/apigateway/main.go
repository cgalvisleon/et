package main

import (
	"os"
	"os/signal"

	serv "github.com/cgalvisleon/et/apigateway"
	"github.com/cgalvisleon/et/et"

	"github.com/cgalvisleon/et/envar"
)

func main() {
	envar.SetvarInt("port", 3000, "Port server", "PORT")
	envar.SetvarInt("rpc", 4200, "Port rpc server", "RPC")
	envar.SetvarStr("dbhost", "localhost", "Database host", "DB_HOST")
	envar.SetvarInt("dbport", 5432, "Database port", "DB_PORT")
	envar.SetvarStr("dbname", "", "Database name", "DB_NAME")
	envar.SetvarStr("dbuser", "", "Database user", "DB_USER")
	envar.SetvarStr("dbpass", "", "Database password", "DB_PASSWORD")

	serv, err := serv.New()
	if err != nil {
		et.Fatal(err)
	}

	go serv.Start()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	serv.Close()
}
