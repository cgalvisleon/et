package ettp

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/router"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/timezone"
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

type Config struct {
	Name         string
	Version      string
	Company      string
	Web          string
	Help         string
	Port         int
	PathUrl      string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	TLS          bool
	CertFile     string
	KeyFile      string
	Debug        bool
}

type Server struct {
	CreatedAt       time.Time
	Id              string
	Name            string
	Version         string
	Company         string
	Web             string
	Help            string
	HostName        string
	Storage         *file.SyncFile
	addr            string
	mux             *http.ServeMux
	svr             *http.Server
	ws              *ws.Hub
	pathUrl         string
	cors            *cors.Cors
	middlewares     []func(http.Handler) http.Handler
	authenticator   func(http.Handler) http.Handler
	notFoundHandler http.HandlerFunc
	solvers         []*Router
	router          []*Router
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

func New(config Config) (*Server, error) {
	// Cache
	err := cache.Load()
	if err != nil {
		return nil, err
	}

	// Event
	err = event.Load()
	if err != nil {
		return nil, err
	}

	// Http ServeMux
	hostName, _ := os.Hostname()
	mux := http.NewServeMux()
	srv := &Server{
		CreatedAt:       timezone.NowTime(),
		Id:              utility.UUID(),
		Name:            config.Name,
		Version:         config.Version,
		Company:         config.Company,
		Web:             config.Web,
		Help:            config.Help,
		HostName:        hostName,
		addr:            strs.Format(":%d", config.Port),
		mux:             mux,
		pathUrl:         config.PathUrl,
		cors:            CorsAllowAll([]string{}),
		notFoundHandler: notFoundHandler,
		middlewares:     make([]func(http.Handler) http.Handler, 0),
		solvers:         []*Router{},
		router:          []*Router{},
		packages:        []*Package{},
		handlers:        make(map[string]http.HandlerFunc),
		mutex:           &sync.RWMutex{},
		readTimeout:     config.ReadTimeout,
		writeTimeout:    config.WriteTimeout,
		idleTimeout:     config.IdleTimeout,
		tls:             config.TLS,
		certFile:        config.CertFile,
		keyFile:         config.KeyFile,
		debug:           config.Debug,
	}
	srv.mux.HandleFunc(srv.pathUrl, srv.handlerResolve)
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
* version
* @return et.Json
**/
func (s *Server) version() et.Json {
	result := et.Json{
		"date_at": s.CreatedAt,
		"version": s.Version,
		"service": s.Name,
		"host":    s.HostName,
		"company": s.Company,
		"web":     s.Web,
		"help":    s.Help,
	}

	return result
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

	s.solvers = []*Router{}
	s.router = []*Router{}
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
		if pk == s.Name {
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
	wsStart := config.Bool("WS_START", false)
	if !wsStart {
		return
	}

	if config.Validate([]string{
		"REDIS_HOST",
		"REDIS_PASSWORD",
		"REDIS_DB",
	}) != nil {
		return
	}

	if s.ws == nil {
		s.ws = ws.NewHub()
		s.ws.Start()
		s.ws.JoinTo(et.Json{
			"adapter":  "redis",
			"host":     config.String("REDIS_HOST", ""),
			"dbname":   config.Int("REDIS_DB", 0),
			"password": config.String("REDIS_PASSWORD", ""),
		})
	}

	s.mountHandlerFuncWS()
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
func (s *Server) With(middlewares ...func(http.Handler) http.Handler) *Router {
	result := &Router{
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
* NewRoute
* @return *Router
**/
func (s *Server) NewRoute() *Router {
	return &Router{
		server:      s,
		middlewares: s.middlewares,
	}
}
