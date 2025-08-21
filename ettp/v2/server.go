package ettp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
)

type Config struct {
	PathApi      string
	PathApp      string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	AllowOrigin  []string
	TLS          bool
	CertFile     string
	KeyFile      string
	Debug        bool
}

type Server struct {
	Id            string                            `json:"id"`
	Host          string                            `json:"host"`
	Port          int                               `json:"port"`
	Addr          string                            `json:"addr"`
	PathApi       string                            `json:"path_api"`
	PathApp       string                            `json:"path_app"`
	Router        map[string]*Router                `json:"router"`
	Solvers       map[string]*Solver                `json:"solvers"`
	Packages      map[string]*Package               `json:"packages"`
	Requests      map[string]*Request               `json:"requests"`
	mux           *http.ServeMux                    `json:"-"`
	svr           *http.Server                      `json:"-"`
	middlewares   []func(http.Handler) http.Handler `json:"-"`
	authenticator func(http.Handler) http.Handler   `json:"-"`
	handlers      map[string]http.HandlerFunc       `json:"-"`
	readTimeout   time.Duration                     `json:"-"`
	writeTimeout  time.Duration                     `json:"-"`
	idleTimeout   time.Duration                     `json:"-"`
	tls           bool                              `json:"-"`
	certFile      string                            `json:"-"`
	keyFile       string                            `json:"-"`
	debug         bool                              `json:"-"`
}

/**
* NewServer
* @param port int
* @return *Server
**/
func NewServer(port int, config *Config) *Server {
	host, _ := os.Hostname()
	result := &Server{
		Id:           utility.UUID(),
		Host:         host,
		Port:         port,
		Addr:         fmt.Sprintf(":%d", port),
		PathApi:      config.PathApi,
		PathApp:      config.PathApp,
		Router:       make(map[string]*Router),
		Solvers:      make(map[string]*Solver),
		Packages:     make(map[string]*Package),
		Requests:     make(map[string]*Request),
		mux:          http.NewServeMux(),
		middlewares:  make([]func(http.Handler) http.Handler, 0),
		handlers:     make(map[string]http.HandlerFunc),
		readTimeout:  config.ReadTimeout,
		writeTimeout: config.WriteTimeout,
		idleTimeout:  config.IdleTimeout,
		tls:          config.TLS,
		certFile:     config.CertFile,
		keyFile:      config.KeyFile,
		debug:        config.Debug,
	}
	result.svr = &http.Server{
		Addr:         result.Addr,
		Handler:      CorsAllowAll(config.AllowOrigin).Handler(result.mux),
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}
	result.mux.HandleFunc("/", result.handler)

	return result
}

/**
* ToJson
* @return et.Json
**/
func (s *Server) ToJson() et.Json {
	packages := make([]et.Json, 0)
	for _, p := range s.Packages {
		packages = append(packages, p.ToJson())
	}

	return et.Json{
		"host":     s.Host,
		"port":     s.Port,
		"packages": packages,
	}
}

/**
* Start
* @return error
**/
func (s *Server) Start() error {
	go func() {
		if s.tls {
			console.Logf("Https", `Load server on https://localhost%s`, s.Addr)
			if err := s.svr.ListenAndServeTLS(s.certFile, s.keyFile); err != nil && err != http.ErrServerClosed {
				console.Fatal(err)
			}
		} else {
			console.Logf("Http", `Load server on http://localhost%s`, s.Addr)
			if err := s.svr.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				console.Fatal(err)
			}
		}
	}()
	return nil
}

/**
* Close
* @return error
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
}

/**
* Save
* @return error
**/
func (s *Server) Save() error {
	bytes, err := json.Marshal(s)
	if err != nil {
		return err
	}

	console.Log(string(bytes))

	return nil
}

/**
* Load
* @return error
**/
func (s *Server) Load() error {

	return nil
}

/**
* Empty
* @return error
**/
func (s *Server) Empty() error {
	s.Router = make(map[string]*Router)
	s.Solvers = make(map[string]*Solver)
	s.Packages = make(map[string]*Package)
	s.Requests = make(map[string]*Request)

	if err := s.Save(); err != nil {
		return err
	}

	return nil
}

/**
* addSolver
* @param kind, method, path, solver string, header et.Json, excludeHeader []string, version int, packageName string
* @return *Solver, error
**/
func (s *Server) addSolver(kind TypeApi, method, path, solver string, header map[string]string, excludeHeader []string, version int, packageName string) (*Solver, error) {
	if !methodMap[method] {
		return nil, fmt.Errorf("method %s not supported", method)
	}

	pkg := s.Packages[packageName]
	if pkg == nil {
		pkg = NewPackage(packageName, s)
	}

	key := fmt.Sprintf("%s:%s", method, path)
	result, ok := s.Solvers[key]
	if ok {
		result.Solver = solver
		result.Header = header
		result.ExcludeHeader = excludeHeader
		result.Version = version
		pkg.AddSolver(result)
		return result, nil
	}

	router, ok := s.Router[method]
	if !ok {
		router = NewRouter(s, method)
		s.Router[method] = router
	}

	result, err := router.addSolver(kind, key, method, path, s.PathApi, solver, header, excludeHeader, version, packageName)
	if err != nil {
		return nil, err
	}

	pkg.AddSolver(result)
	return result, nil
}

/**
* AddHandler
* @param method, path string, handlerFn http.HandlerFunc, packageName string
* @return *Solver, error
**/
func (s *Server) addHandler(method, path string, handlerFn http.HandlerFunc, packageName string) (*Solver, error) {
	solver := fmt.Sprintf("%s/%s", s.PathApi, path)
	solver = strings.ReplaceAll(solver, "//", "/")
	result, err := s.addSolver(TpHandler, method, path, solver, map[string]string{}, []string{}, 0, packageName)
	if err != nil {
		return nil, err
	}

	s.handlers[result.Id] = handlerFn
	return result, nil
}

/**
* Solver
* @param kind, method, path, solver string, header et.Json, excludeHeader []string, version int, packageName string
* @return *Solver, error
**/
func (s *Server) Solver(method, path, solver string, header map[string]string, excludeHeader []string, version int, packageName string) (*Solver, error) {
	result, err := s.addSolver(TpApiRest, method, path, solver, header, excludeHeader, version, packageName)
	if err != nil {
		return nil, err
	}

	s.Solvers[result.Id] = result
	return result, nil
}

/**
* Public
* @param method, path string, handlerFn http.HandlerFunc, packageName string
**/
func (s *Server) Public(method, path string, handlerFn http.HandlerFunc, packageName string) (*Solver, error) {
	return s.addHandler(method, path, handlerFn, packageName)
}

/**
* Private
* @param method, path string, handlerFn http.HandlerFunc, packageName string
**/
func (s *Server) Private(method, path string, handlerFn http.HandlerFunc, packageName string) (*Solver, error) {
	result, err := s.addHandler(method, path, handlerFn, packageName)
	if err != nil {
		return nil, err
	}

	result.middlewares = append(result.middlewares, s.authenticator)
	return result, nil
}

/**
* Use
* @param middlewares ...func(http.Handler) http.Handler
**/
func (s *Server) Use(middlewares ...func(http.Handler) http.Handler) {
	s.middlewares = append(s.middlewares, middlewares...)
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
* RemoveSolverById
* @param id string
* @return error
**/
func (s *Server) RemoveSolverById(id string) error {
	solver, ok := s.Solvers[id]
	if !ok {
		return fmt.Errorf("solver %s not found", id)
	}

	pkg := s.Packages[solver.PackageName]
	if pkg == nil {
		return fmt.Errorf("package %s not found", solver.PackageName)
	}

	pkg.RemoveSolver(solver)

	router := solver.router
	router.solver = nil
	delete(router.main.Router, router.Tag)
	delete(s.Solvers, id)

	return nil
}

/**
* FindRequest
* @param r *http.Request
* @return *Request, error
**/
func (s *Server) FindRequest(r *http.Request) (*Request, error) {
	method := r.Method
	router, ok := s.Router[method]
	if !ok {
		return nil, fmt.Errorf("router %s not found", method)
	}

	result, err := router.findRequest(r)
	if err != nil {
		return nil, err
	}

	s.Requests[result.Id] = result
	return result, nil
}
