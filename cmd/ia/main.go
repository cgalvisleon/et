package main

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/ia"
	"github.com/cgalvisleon/et/jsql"
	_ "github.com/cgalvisleon/et/jsql/drivers/postgres"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/server"
)

/**
* main: Boots the ia RAG HTTP API: connects the default jsql DB, defines and
* initializes the RAG models under the given schema, and serves the
* document-ingestion and /conversation endpoints via the server package.
**/
func main() {
	db, err := jsql.Load()
	if err != nil {
		logs.Panic(err)
	}
	defer db.Close()

	schema := envar.GetStr("IA_SCHEMA", "public")
	rag, err := ia.Load(db, schema, ia.Config{})
	if err != nil {
		logs.Panic(err)
	}

	port := envar.GetInt("PORT", 3300)
	srv := server.New("ia", port)
	srv.Use(middleware.Logger, middleware.Recoverer)
	rag.LoadRouter(srv.Mux)

	srv.Start()
}
