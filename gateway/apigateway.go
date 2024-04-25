package gateway

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/cgalvisleon/elvis/cache"
	"github.com/cgalvisleon/elvis/console"
	"github.com/cgalvisleon/elvis/envar"
	"github.com/cgalvisleon/elvis/et"
	"github.com/cgalvisleon/elvis/event"
	"github.com/cgalvisleon/elvis/strs"
	"github.com/cgalvisleon/elvis/ws"
	"github.com/dimiro1/banner"
	"github.com/mattn/go-colorable"
)

type Server struct {
	http *HttpServer
	rpc  *net.Listener
	ws   *ws.Conn
}

var PackageName = "gateway"
var PackageTitle = envar.EnvarStr("Apigateway", "PACKAGE_TITLE")
var PackagePath = "/api/gateway"
var PackageVersion = envar.EnvarStr("0.0.1", "VERSION")
var Company = envar.EnvarStr("", "COMPANY")
var Web = envar.EnvarStr("", "WEB")
var HostName, _ = os.Hostname()
var Host = strs.Format(`%s:%d`, envar.EnvarStr("http://localhost", "HOST"), envar.EnvarInt(3300, "PORT"))
var conn *Server

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

	// HTTP server
	httpServer := newHttpServer()

	// RPC server
	rpcServer := newRpc()

	// WS server
	wsServer, err := ws.Load()
	if err != nil {
		panic(err)
	}

	// Create a new server
	conn = &Server{
		http: httpServer,
		rpc:  &rpcServer,
		ws:   wsServer,
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
		console.LogKF("Http", "Running Api Gateway on http://localhost%s", svr.addr)
		console.Fatal(http.ListenAndServe(svr.addr, svr.handler))
	}()

	// Start RPC server
	go func() {
		if serv.rpc == nil {
			return
		}

		svr := *serv.rpc
		console.LogKF("RPC", "Running on tcp:localhost:%s", svr.Addr().String())
		http.Serve(svr, nil)
	}()

	// Init events
	initEvents()

	// Banner
	Banner()

	<-make(chan struct{})
}

func Version() et.Json {
	service := et.Json{
		"version": envar.EnvarStr("", "VERSION"),
		"service": PackageName,
		"host":    HostName,
		"company": Company,
		"web":     Web,
		"help":    "",
	}

	return service
}

func Banner() {
	time.Sleep(3 * time.Second)
	templ := fmt.Sprintf(`{{ .Title "%s V%s" "" 4 }}`, PackageName, PackageVersion)
	banner.InitString(colorable.NewColorableStdout(), true, true, templ)
	fmt.Println()
}
