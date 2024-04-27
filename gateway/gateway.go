package gateway

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/store"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/ws"
	"github.com/dimiro1/banner"
	"github.com/mattn/go-colorable"
)

type Server struct {
	http  *HttpServer
	rpc   *net.Listener
	ws    *ws.Hub
	cache *store.Mem
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
	// HTTP server
	httpServer := newHttpServer()

	// RPC server
	rpcServer := newRpc()

	// WS server
	wsServer, err := ws.Load()
	if err != nil {
		panic(err)
	}

	memCache := store.NewMem()

	// Create a new server
	conn = &Server{
		http:  httpServer,
		rpc:   &rpcServer,
		ws:    wsServer,
		cache: memCache,
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
		logs.Logf("Http", "Running Api Gateway on http://localhost%s", svr.addr)
		logs.Fatal(http.ListenAndServe(svr.addr, svr.handler))
	}()

	// Start RPC server
	go func() {
		if serv.rpc == nil {
			return
		}

		svr := *serv.rpc
		logs.Logf("RPC", "Running on tcp:localhost:%s", svr.Addr().String())
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
