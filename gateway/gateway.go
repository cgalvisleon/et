package gateway

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/message"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/ws"
	"github.com/dimiro1/banner"
	"github.com/mattn/go-colorable"
)

type Server struct {
	http *HttpServer
	ws   *ws.Conn
}

var PackageName = "gateway"
var PackageTitle = envar.GetStr("Apigateway", "PACKAGE_TITLE")
var PackagePath = "/api/gateway"
var PackageVersion = envar.GetStr("0.0.1", "VERSION")
var Company = envar.GetStr("", "COMPANY")
var Web = envar.GetStr("", "WEB")
var HostName, _ = os.Hostname()
var Host = strs.Format(`%s:%d`, envar.GetStr("http://localhost", "HOST"), envar.GetInt(3300, "PORT"))
var conn *Server

func Load() (*Server, error) {
	if conn != nil {
		return conn, nil
	}

	err := cache.Load("")
	if err != nil {
		return nil, err
	}

	// HTTP server
	http := newHttpServer()

	// WS server
	ws, err := ws.Load()
	if err != nil {
		panic(err)
	}

	// Create a new server
	conn = &Server{
		http: http,
		ws:   ws,
	}

	return conn, nil
}

func inbox(msg message.Message) {
	logs.Debug(msg.ToString())
}

// Server close
func (serv *Server) Close() error {
	return nil
}

// Server start
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

	// Init events
	initEvents()

	// Banner
	Banner()

	<-make(chan struct{})
}

func Version() et.Json {
	service := et.Json{
		"version": envar.GetStr("", "VERSION"),
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
