package apigateway

import (
	"net"
	"net/http"
	"os"

	"github.com/cgalvisleon/et/et"
)

const (
	HANDLER   = "HANDLER"
	HTTP      = "HTTP"
	REST      = "REST"
	WEBSOCKET = "WEBSOCKET"
	// Methods
	CONNECT = "CONNECT"
	DELETE  = "DELETE"
	GET     = "GET"
	HEAD    = "HEAD"
	OPTIONS = "OPTIONS"
	PATCH   = "PATCH"
	POST    = "POST"
	PUT     = "PUT"
	TRACE   = "TRACE"
)

var (
	PackageName    = "apigateway"
	PackageTitle   = et.EnvarStr("Apigateway", "PACKAGE_TITLE")
	PackagePath    = "/api/apigateway"
	PackageVersion = et.EnvarStr("0.0.1", "VERSION")
	Company        = et.EnvarStr("", "COMPANY")
	Web            = et.EnvarStr("", "WEB")
	HostName, _    = os.Hostname()
	Host           = et.Format(`%s:%d`, et.EnvarStr("http://localhost", "HOST"), et.EnvarInt(3300, "PORT"))
)

type Server struct {
	http *HttpServer
	rpc  *net.Listener
}

var conn *Server

func New() (*Server, error) {
	// HTTP server
	httpServer := NewHttpServer()

	// RPC server
	rpcServer := NewRpc()

	// Create a new server
	conn = &Server{
		http: httpServer,
		rpc:  &rpcServer,
	}

	return conn, nil
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
