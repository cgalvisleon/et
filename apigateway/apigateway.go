package apigateway

import (
	"net"
	"net/http"
	"os"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/ws"
)

var PackageName = "apigateway"
var PackageTitle = et.EnvarStr("Apigateway", "PACKAGE_TITLE")
var PackagePath = "/api/apigateway"
var PackageVersion = et.EnvarStr("0.0.1", "VERSION")
var Company = et.EnvarStr("", "COMPANY")
var Web = et.EnvarStr("", "WEB")
var HostName, _ = os.Hostname()
var Host = et.Format(`%s:%d`, et.EnvarStr("http://localhost", "HOST"), et.EnvarInt(3300, "PORT"))

type Server struct {
	http *HttpServer
	rpc  *net.Listener
}

func New() (*Server, error) {
	// Create cache server
	_, err := cache.Load()
	if err != nil {
		panic(err)
	}

	// Create event server
	_, err = event.Load()
	if err != nil {
		panic(err)
	}

	// Create ws server
	_, err = ws.Load()
	if err != nil {
		panic(err)
	}

	// HTTP server
	httpServer := NewHttpServer()

	// RPC server
	rpcServer := NewRpc()

	// Create a new server
	result := &Server{
		http: httpServer,
		rpc:  &rpcServer,
	}

	return result, nil
}

func (serv *Server) Close() error {
	return nil
}

func (serv *Server) Start() {
	// Start HTTP server
	go func() {
		if serv.http == nil {
			return
		}

		svr := *serv.http
		et.Logf("Http", "Running Api Gateway on http://localhost%s", svr.addr)
		et.Fatal(http.ListenAndServe(svr.addr, svr.handler))
	}()

	// Start RPC server
	go func() {
		if serv.rpc == nil {
			return
		}

		svr := *serv.rpc
		et.Logf("RPC", "Running on tcp:localhost:%s", svr.Addr().String())
		http.Serve(svr, nil)
	}()

	// Init events
	initEvents()

	<-make(chan struct{})
}

func Version() et.Json {
	service := et.Json{
		"version": et.EnvarStr("", "VERSION"),
		"service": PackageName,
		"host":    HostName,
		"company": Company,
		"web":     Web,
		"help":    "",
	}

	return service
}
