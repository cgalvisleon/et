package ettp

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/router"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/et/ws"
	"github.com/rs/cors"
)

type TypeApi int

const (
	TpHandler TypeApi = iota
	TpRest
)

func (t TypeApi) String() string {
	switch t {
	case TpHandler:
		return "Handler"
	case TpRest:
		return "Rest"
	default:
		return "Unknown"
	}
}

func IntToTypeApi(i int) TypeApi {
	switch i {
	case 1:
		return TpRest
	default:
		return TpHandler
	}
}

const (
	CONNECT    = "CONNECT"
	DELETE     = "DELETE"
	GET        = "GET"
	HEAD       = "HEAD"
	OPTIONS    = "OPTIONS"
	PATCH      = "PATCH"
	POST       = "POST"
	PUT        = "PUT"
	TRACE      = "TRACE"
	ROUTER_KEY = "apigateway-router"
)

var methodMap = map[string]bool{
	CONNECT: true,
	DELETE:  true,
	GET:     true,
	HEAD:    true,
	OPTIONS: true,
	PATCH:   true,
	POST:    true,
	PUT:     true,
	TRACE:   true,
}

var ServiceName = "Api Gateway"
var Version = envar.GetStr("0.0.1", "VERSION")
var HostName, _ = os.Hostname()
var Company = envar.GetStr("", "COMPANY")
var Web = envar.GetStr("", "WEB")
var Help = envar.GetStr("", "HELP")

type Server struct {
	Id      string
	Name    string
	Storage *file.SyncFile
	// Db              *jdb.DB
	addr            string
	mux             *http.ServeMux
	svr             *http.Server
	ws              *ws.Hub
	host            string
	cors            *cors.Cors
	middlewares     []func(http.Handler) http.Handler
	authenticator   func(http.Handler) http.Handler
	notFoundHandler http.HandlerFunc
	solvers         []*Route
	router          []*Route
	packages        []*Package
	handlers        map[string]http.HandlerFunc
	mutex           *sync.RWMutex
	readTimeout     time.Duration
	writeTimeout    time.Duration
	idleTimeout     time.Duration
	tls             bool
	certFile        string
	keyFile         string
	debug           bool
}

func New() (*Server, error) {
	// Cache
	_, err := cache.Load()
	if err != nil {
		return nil, err
	}

	// Event
	_, err = event.Load()
	if err != nil {
		return nil, err
	}

	// Http ServeMux
	mux := http.NewServeMux()
	port := envar.GetInt(3000, "PORT")
	host := envar.GetStr("/", "HOST")
	readTimeout := envar.GetInt(0, "READ_TIMEOUT")
	writeTimeout := envar.GetInt(0, "WRITE_TIMEOUT")
	idleTimeout := envar.GetInt(24, "IDLE_TIMEOUT")
	tls := envar.GetBool(false, "TLS")
	certFile := envar.GetStr("", "CERT_FILE")
	keyFile := envar.GetStr("", "KEY_FILE")
	debug := envar.GetBool(false, "DEBUG")

	srv := &Server{
		Id:              utility.UUID(),
		addr:            strs.Format(":%d", port),
		mux:             mux,
		host:            host,
		cors:            CorsAllowAll([]string{}),
		notFoundHandler: notFoundHandler,
		middlewares:     make([]func(http.Handler) http.Handler, 0),
		solvers:         []*Route{},
		router:          []*Route{},
		packages:        []*Package{},
		handlers:        make(map[string]http.HandlerFunc),
		mutex:           &sync.RWMutex{},
		readTimeout:     time.Duration(readTimeout) * time.Second,
		writeTimeout:    time.Duration(writeTimeout) * time.Second,
		idleTimeout:     time.Duration(idleTimeout) * time.Hour,
		tls:             tls,
		certFile:        certFile,
		keyFile:         keyFile,
		debug:           debug,
	}
	srv.mux.HandleFunc(srv.host, srv.handlerResolve)
	srv.svr = &http.Server{
		Addr:         srv.addr,
		Handler:      srv.cors.Handler(srv.mux),
		ReadTimeout:  srv.readTimeout,
		WriteTimeout: srv.writeTimeout,
		IdleTimeout:  srv.idleTimeout,
	}

	return srv, nil
}

/**
* Empty
**/
func (s *Server) Empty() []string {
	s.Storage.Empty()
	result := []string{}
	for _, pk := range s.packages {
		result = append(result, pk.Name)
	}

	s.solvers = []*Route{}
	s.router = []*Route{}
	s.packages = []*Package{}

	return result
}

/**
* Reset
**/
func (s *Server) Reset() {
	pks := s.Empty()
	s.Load()
	s.LoadWS()
	s.Save()

	for _, pk := range pks {
		if pk == ServiceName {
			continue
		}

		channel := fmt.Sprintf(`%s/%s`, router.APIGATEWAY_RESET, pk)
		event.Publish(channel, et.Json{})
	}
}

/**
* Start
**/
func (s *Server) Start() error {
	go func() {
		if s.tls {
			console.Logf("Https", `Load server on https://localhost%s`, s.addr)
			if err := s.svr.ListenAndServeTLS(s.certFile, s.keyFile); err != nil && err != http.ErrServerClosed {
				console.Fatal(err)
			}
		} else {
			console.Logf("Http", `Load server on http://localhost%s`, s.addr)
			if err := s.svr.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				console.Fatal(err)
			}
		}
	}()

	err := s.Load()
	if err != nil {
		return err
	}

	s.initEvents()
	s.LoadWS()

	return nil
}

/**
* StartWS
**/
func (s *Server) LoadWS() {
	wsStart := envar.GetBool(false, "WS_START")
	if !wsStart {
		return
	}

	if s.ws == nil {
		s.ws = ws.NewHub()
		s.ws.Start()
		s.ws.JoinTo(et.Json{
			"adapter":  "redis",
			"host":     envar.GetStr("", "REDIS_HOST"),
			"dbname":   envar.GetInt(0, "REDIS_DB"),
			"password": envar.GetStr("", "REDIS_PASSWORD"),
		})
	}

	s.loadHandlerFuncWS()
}

/**
* Close
**/
func (s *Server) Close() {
	if s.svr != nil {
		s.svr.Close()

		if s.tls {
			console.Log("Https", "Shutting down server...")
		} else {
			console.Log("Http", "Shutting down server...")
		}
	}

	if s.ws != nil {
		s.ws.Close()
	}

	cache.Close()
	event.Close()
}

/**
* SetAddr
* @param port int
**/
func (s *Server) SetAddr(port int) {
	s.addr = strs.Format(":%d", port)
}

/**
* SetNotFoundHandler
* @param h http.Handler
**/
func (s *Server) SetReadTimeout(value time.Duration) {
	s.readTimeout = value
}

/**
* SetWriteTimeout
* @param h http.Handler
**/
func (s *Server) SetWriteTimeout(value time.Duration) {
	s.writeTimeout = value
}

/**
* SetIdleTimeout
* @param h http.Handler
**/
func (s *Server) SetIdleTimeout(value time.Duration) {
	s.idleTimeout = value
}

/**
* Use
* @param middlewares ...func(http.HandlerFunc) http.HandlerFunc
**/
func (s *Server) Use(middlewares ...func(http.Handler) http.Handler) {
	s.middlewares = append(s.middlewares, middlewares...)
}

/**
* With
* @param middlewares ...func(http.HandlerFunc) http.HandlerFunc
**/
func (s *Server) With(middlewares ...func(http.Handler) http.Handler) *Route {
	result := &Route{
		server:      s,
		middlewares: make([]func(http.Handler) http.Handler, 0),
	}

	result.middlewares = append(result.middlewares, middlewares...)

	return result
}

/**
* Authenticator
* @param middleware func(http.HandlerFunc) http.HandlerFunc
* @return *Server
**/
func (s *Server) Authenticator(middleware func(http.Handler) http.Handler) *Server {
	s.authenticator = middleware

	return s
}

/**
* Private
* @return *Route
**/
func (s *Server) Private() *Route {
	if s.authenticator == nil {
		return s.NewRoute()
	}

	return s.With(s.authenticator)
}

/**
* NewRoute
* @return *Route
**/
func (s *Server) NewRoute() *Route {
	return &Route{
		server:      s,
		middlewares: s.middlewares,
	}
}

/**
* PublicRoute
* @param method string
* @param path string
* @param h http.HandlerFunc
* @param packageName string
**/
func (s *Server) PublicRoute(method, path string, h http.HandlerFunc, packageName string) {
	switch method {
	case "GET":
		s.Get(path, h, packageName)
	case "POST":
		s.Post(path, h, packageName)
	case "PUT":
		s.Put(path, h, packageName)
	case "PATCH":
		s.Patch(path, h, packageName)
	case "DELETE":
		s.Delete(path, h, packageName)
	case "HEAD":
		s.Head(path, h, packageName)
	case "OPTIONS":
		s.Options(path, h, packageName)
	}
}

/**
* ProtectRoute
* @param method string
* @param path string
* @param h http.HandlerFunc
**/
func (s *Server) ProtectRoute(method, path string, h http.HandlerFunc, packageName string) {
	router := s.Private()
	switch method {
	case "GET":
		router.Get(path, h, packageName)
	case "POST":
		router.Post(path, h, packageName)
	case "PUT":
		router.Put(path, h, packageName)
	case "PATCH":
		router.Patch(path, h, packageName)
	case "DELETE":
		router.Delete(path, h, packageName)
	case "HEAD":
		router.Head(path, h, packageName)
	case "OPTIONS":
		router.Options(path, h, packageName)
	}
}

/**
* AuthorizationRoute
**/
func (s *Server) AuthorizationRoute(method, path string, h http.HandlerFunc, packageName string) {
	router := s.With(s.authenticator).With(middleware.Authorization)
	switch method {
	case "GET":
		router.Get(path, h, packageName)
	case "POST":
		router.Post(path, h, packageName)
	case "PUT":
		router.Put(path, h, packageName)
	case "PATCH":
		router.Patch(path, h, packageName)
	case "DELETE":
		router.Delete(path, h, packageName)
	case "HEAD":
		router.Head(path, h, packageName)
	case "OPTIONS":
		router.Options(path, h, packageName)
	}
}
