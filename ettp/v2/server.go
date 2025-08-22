package ettp

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/utility"
	"github.com/dimiro1/banner"
	"github.com/mattn/go-colorable"
)

type Config struct {
	Port         int
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
	CreatedAt     time.Time                         `json:"created_at"`
	Id            string                            `json:"id"`
	Name          string                            `json:"name"`
	Host          string                            `json:"host"`
	Port          int                               `json:"port"`
	Addr          string                            `json:"addr"`
	PathApi       string                            `json:"path_api"`
	PathApp       string                            `json:"path_app"`
	Router        map[string]*Router                `json:"router"`
	Solvers       map[string]*Solver                `json:"solvers"`
	Packages      map[string]*Package               `json:"packages"`
	Resolvers     map[string]*Resolver              `json:"resolvers"`
	Version       string                            `json:"version"`
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
* @param name string
* @return *Server
**/
func NewServer(name string, config *Config) *Server {
	now := time.Now()
	host, _ := os.Hostname()
	result := &Server{
		CreatedAt:     now,
		Id:            utility.UUID(),
		Name:          name,
		Host:          host,
		Port:          config.Port,
		Addr:          fmt.Sprintf(":%d", config.Port),
		PathApi:       config.PathApi,
		PathApp:       config.PathApp,
		Router:        make(map[string]*Router),
		Solvers:       make(map[string]*Solver),
		Packages:      make(map[string]*Package),
		Resolvers:     make(map[string]*Resolver),
		Version:       "v0.0.2",
		mux:           http.NewServeMux(),
		middlewares:   make([]func(http.Handler) http.Handler, 0),
		authenticator: middleware.Autentication,
		handlers:      make(map[string]http.HandlerFunc),
		readTimeout:   config.ReadTimeout,
		writeTimeout:  config.WriteTimeout,
		idleTimeout:   config.IdleTimeout,
		tls:           config.TLS,
		certFile:      config.CertFile,
		keyFile:       config.KeyFile,
		debug:         config.Debug,
	}
	result.svr = &http.Server{
		Addr:         result.Addr,
		Handler:      CorsAllowAll(config.AllowOrigin).Handler(result.mux),
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}
	result.mux.HandleFunc("/", result.handlerRouteTable)

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
* banner
* @return void
**/
func (s *Server) banner() {
	time.Sleep(3 * time.Second)
	templ := utility.BannerTitle(s.Name, 4)
	banner.InitString(colorable.NewColorableStdout(), true, true, templ)
	fmt.Println()
}

func (s *Server) initHttpServer() error {
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
* Start
* @return error
**/
func (s *Server) Start() {
	if err := s.load(); err != nil {
		console.Fatal(err)
	}
	if err := s.initRouteTable(); err != nil {
		console.Fatal(err)
	}
	if err := s.initEvents(); err != nil {
		console.Fatal(err)
	}
	if err := s.initHttpServer(); err != nil {
		console.Fatal(err)
	}
	s.banner()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	s.Close()
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
* Reset
* @return error
**/
func (s *Server) Reset() {
	event.Publish(EVENT_RESET, et.Json{})
}

/**
* Empty
* @return error
**/
func (s *Server) Empty() error {
	s.Router = make(map[string]*Router)
	s.Solvers = make(map[string]*Solver)
	s.Packages = make(map[string]*Package)
	s.Resolvers = make(map[string]*Resolver)

	if err := s.Save(); err != nil {
		return err
	}

	return nil
}

/**
* setRouter
* @param kind TypeRouter, method, path, solver string, typeHeader TpHeader, header et.Json, excludeHeader []string, version int, packageName string
* @return *Solver, error
**/
func (s *Server) setRouter(kind TypeRouter, method, path, solver string, typeHeader TpHeader, header map[string]string, excludeHeader []string, version int, packageName string) (*Solver, error) {
	if !methodMap[method] {
		return nil, fmt.Errorf("method %s not supported", method)
	}

	if !utility.ValidStr(path, 1, []string{""}) {
		return nil, fmt.Errorf("path %s is not valid", path)
	}

	router, ok := s.Router[method]
	if !ok {
		router = NewRouter(s, method)
		s.Router[method] = router
	}

	parentPath := s.PathApi
	if kind == TpWepApp {
		parentPath = s.PathApp
	}

	result, err := router.setRouter(kind, method, path, parentPath, solver, typeHeader, header, excludeHeader, version, packageName)
	if err != nil {
		return nil, err
	}

	if solver != "" {
		console.Logf(s.Name, "Method:%s Path:%s Solver:%s Version:%d PackageName:%s", method, path, solver, version, packageName)
	} else {
		console.Logf(s.Name, "Method:%s Path:%s Version:%d PackageName:%s", method, path, version, packageName)
	}

	return result, nil
}

/**
* setHandler
* @param method, path string, handlerFn http.HandlerFunc, packageName string
* @return *Solver, error
**/
func (s *Server) setHandler(method, path string, handlerFn http.HandlerFunc, packageName string) (*Solver, error) {
	result, err := s.setRouter(TpHandler, method, path, "", TpKeepHeader, map[string]string{}, []string{}, 0, packageName)
	if err != nil {
		return nil, err
	}

	s.handlers[result.Id] = handlerFn
	return result, nil
}

/**
* setSolver
* @param kind TypeRouter, method, path, solver string, typeHeader TpHeader, header map[string]string, excludeHeader []string, version int, packageName string
* @return *Solver, error
**/
func (s *Server) setSolver(kind TypeRouter, method, path, solver string, typeHeader TpHeader, header map[string]string, excludeHeader []string, version int, private bool, packageName string) (*Solver, error) {
	result, err := s.setRouter(kind, method, path, solver, typeHeader, header, excludeHeader, version, packageName)
	if err != nil {
		return nil, err
	}

	result.Private = private
	s.Solvers[result.Id] = result
	return result, nil
}

/**
* SetRouter
* @param method, path, solver string, header et.Json, excludeHeader []string, version int, private bool, packageName string
* @return *Solver, error
**/
func (s *Server) SetRouter(method, path, solver string, typeHeader int, header et.Json, excludeHeader []string, version int, private bool, packageName string) (*Solver, error) {
	headerMap := make(map[string]string)
	for k, v := range header {
		headerMap[k] = fmt.Sprintf("%v", v)
	}

	return s.setSolver(TpApiRest, method, path, solver, TpHeader(typeHeader), headerMap, excludeHeader, version, private, packageName)
}

/**
* Public
* @param method, path string, handlerFn http.HandlerFunc, packageName string
**/
func (s *Server) Public(method, path string, handlerFn http.HandlerFunc, packageName string) (*Solver, error) {
	return s.setHandler(method, path, handlerFn, packageName)
}

/**
* Private
* @param method, path string, handlerFn http.HandlerFunc, packageName string
**/
func (s *Server) Private(method, path string, handlerFn http.HandlerFunc, packageName string) (*Solver, error) {
	result, err := s.setHandler(method, path, handlerFn, packageName)
	if err != nil {
		return nil, err
	}

	result.Private = true
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
* RemoveRouterById
* @param id string
* @return error
**/
func (s *Server) RemoveRouterById(id string, save bool) error {
	_, ok := s.Solvers[id]
	if !ok {
		return fmt.Errorf("solver %s not found", id)
	}

	delete(s.Solvers, id)

	if save {
		s.Save()
	}

	return nil
}

/**
* FindResolver
* @param r *http.Request
* @return *Request, error
**/
func (s *Server) FindResolver(r *http.Request) (*Resolver, error) {
	method := r.Method
	router, ok := s.Router[method]
	if !ok {
		return nil, fmt.Errorf("router %s not found", method)
	}

	result, err := router.findResolver(r)
	if err != nil {
		return nil, err
	}

	s.Resolvers[result.Id] = result

	clean := func() {
		delete(s.Resolvers, result.Id)
	}

	duration := 24 * time.Hour
	if duration != 0 {
		go time.AfterFunc(duration, clean)
	}

	return result, nil
}

/**
* StatusResolver
* @param r *Resolver, status Status
**/
func (s *Server) HTTPError(resolver *Resolver, metric *middleware.Metrics, w http.ResponseWriter, r *http.Request, status int, message string) {
	resolver.SetStatus(TpStatusFailed)
	metric.HTTPError(w, r, status, message)

	s.Save()
}

/**
* HTTPSuccess
* @param resolver *Resolver, metric *middleware.Metrics, rw *middleware.ResponseWriterWrapper
**/
func (s *Server) HTTPSuccess(resolver *Resolver, metric *middleware.Metrics, rw *middleware.ResponseWriterWrapper) {
	resolver.SetStatus(TpStatusSuccess)
	delete(s.Resolvers, resolver.Id)
	metric.DoneFn(rw)

	s.Save()
}
