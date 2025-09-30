package ettp

import (
	"fmt"
	"net/http"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
	"github.com/rs/cors"
)

type Config struct {
	Name         string
	Company      string
	Web          string
	Help         string
	Port         int
	PathApi      string
	PathApp      string
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
	addr            string
	mux             *http.ServeMux
	svr             *http.Server
	pathApi         string
	pathApp         string
	cors            *cors.Cors
	middlewares     []func(http.Handler) http.Handler
	authenticator   func(http.Handler) http.Handler
	notFoundHandler http.HandlerFunc
	router          []*Router
	solvers         []*Router
	packages        []*Package
	handlers        map[string]*ApiFunc
	proxys          map[string]*Proxy
	mutex           *sync.RWMutex
	readTimeout     time.Duration
	writeTimeout    time.Duration
	idleTimeout     time.Duration
	tls             bool
	certFile        string
	keyFile         string
	storageKey      string
	debug           bool
}

func New(config Config) (*Server, error) {
	/* Cache */
	err := cache.Load()
	if err != nil {
		return nil, err
	}

	/* Event */
	err = event.Load()
	if err != nil {
		return nil, err
	}

	/* Http ServeMux */
	version := "v0.0.1"
	hostName, _ := os.Hostname()
	mux := http.NewServeMux()
	srv := &Server{
		CreatedAt:       timezone.NowTime(),
		Id:              utility.UUID(),
		Name:            config.Name,
		Version:         version,
		Company:         config.Company,
		Web:             config.Web,
		Help:            config.Help,
		HostName:        hostName,
		addr:            fmt.Sprintf(":%d", config.Port),
		mux:             mux,
		pathApi:         config.PathApi,
		pathApp:         config.PathApp,
		cors:            CorsAllowAll([]string{}),
		notFoundHandler: notFoundHandler,
		middlewares:     make([]func(http.Handler) http.Handler, 0),
		router:          []*Router{},
		solvers:         []*Router{},
		packages:        []*Package{},
		handlers:        make(map[string]*ApiFunc),
		proxys:          make(map[string]*Proxy),
		mutex:           &sync.RWMutex{},
		readTimeout:     config.ReadTimeout,
		writeTimeout:    config.WriteTimeout,
		idleTimeout:     config.IdleTimeout,
		tls:             config.TLS,
		certFile:        config.CertFile,
		keyFile:         config.KeyFile,
		storageKey:      fmt.Sprintf("%s-%s", config.Name, version),
		debug:           config.Debug,
	}
	srv.svr = &http.Server{
		Addr:         srv.addr,
		Handler:      srv.cors.Handler(srv.mux),
		ReadTimeout:  srv.readTimeout,
		WriteTimeout: srv.writeTimeout,
		IdleTimeout:  srv.idleTimeout,
	}
	srv.mux.HandleFunc("/", srv.handlerResolver)

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

	if err := s.load(); err != nil {
		return err
	}

	s.initEvents()

	return nil
}

/**
* Reset
**/
func (s *Server) Reset() error {
	s.router = []*Router{}
	s.solvers = []*Router{}
	s.packages = []*Package{}

	if err := s.Save(); err != nil {
		return err
	}

	for _, handler := range s.handlers {
		s.setApiFunc(handler.Method, handler.Path, handler.HandlerFn, handler.PackageName)
	}

	return nil
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

	cache.Close()
	event.Close()
}

/**
* SetAddr
* @param port int
**/
func (s *Server) SetAddr(port int) {
	s.addr = fmt.Sprintf(":%d", port)
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
* GetPackages
* @return et.Items
**/
func (s *Server) GetPackages(name string) et.Items {
	var result = []et.Json{}
	if name != "" {
		idx := slices.IndexFunc(s.packages, func(e *Package) bool { return strs.Lowcase(e.Name) == strs.Lowcase(name) })
		if idx != -1 {
			pakage := s.packages[idx]
			result = append(result, pakage.ToJson())
		}
	} else {
		for _, pakage := range s.packages {
			result = append(result, pakage.ToJson())
		}
	}

	return et.Items{
		Ok:     len(result) > 0,
		Count:  len(result),
		Result: result,
	}
}
