package ettp

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/router"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
	"github.com/dimiro1/banner"
	"github.com/mattn/go-colorable"
)

const (
	Version = "v0.0.2"
)

type TransportConfig struct {
	InsecureSkipVerify  bool `json:"insecure_skip_verify"`
	MaxIdleConns        int  `json:"max_idle_conns"`
	MaxIdleConnsPerHost int  `json:"max_idle_conns_per_host"`
	MaxConnsPerHost     int  `json:"max_conns_per_host"`
	IdleConnTimeout     int  `json:"idle_conn_timeout"`
	TLSHandshakeTimeout int  `json:"tls_handshake_timeout"`
	ForceAttemptHTTP2   bool `json:"force_attempt_http2"`
}

type Config struct {
	Port         int              `json:"port"`
	Parent       string           `json:"parent"`
	ReadTimeout  time.Duration    `json:"read_timeout"`
	WriteTimeout time.Duration    `json:"write_timeout"`
	IdleTimeout  time.Duration    `json:"idle_timeout"`
	Timeout      time.Duration    `json:"timeout"`
	AllowOrigin  []string         `json:"allow_origin"`
	IsTLS        bool             `json:"is_tls"`
	CertFile     string           `json:"cert_file"`
	KeyFile      string           `json:"key_file"`
	Transport    *TransportConfig `json:"transport"`
	UseCache     bool             `json:"use_cache"`
	UseEvent     bool             `json:"use_event"`
	Debug        bool             `json:"debug"`
}

type Server struct {
	CreatedAt     time.Time                         `json:"created_at"`
	Name          string                            `json:"name"`
	Host          string                            `json:"host"`
	Port          int                               `json:"port"`
	Addr          string                            `json:"addr"`
	Parent        string                            `json:"parent"`
	Version       string                            `json:"version"`
	Solvers       map[string]*Solver                `json:"solvers"`
	Packages      map[string]*Package               `json:"packages"`
	router        map[string]*Router                `json:"-"`
	Requests      map[string]*Resolver              `json:"requests"`
	muRequests    sync.RWMutex                      `json:"-"`
	muRoutes      sync.RWMutex                      `json:"-"`
	mux           *http.ServeMux                    `json:"-"`
	svr           *http.Server                      `json:"-"`
	client        *http.Client                      `json:"-"`
	pipe          net.Listener                      `json:"-"`
	middlewares   []func(http.Handler) http.Handler `json:"-"`
	authenticator func(http.Handler) http.Handler   `json:"-"`
	readTimeout   time.Duration                     `json:"-"`
	writeTimeout  time.Duration                     `json:"-"`
	idleTimeout   time.Duration                     `json:"-"`
	istls         bool                              `json:"-"`
	certFile      string                            `json:"-"`
	keyFile       string                            `json:"-"`
	useCache      bool                              `json:"-"`
	useEvent      bool                              `json:"-"`
	debug         bool                              `json:"-"`
}

/**
* New
* @param name string
* @return (*Server, error)
**/
func New(name string, config *Config) (*Server, error) {
	now := timezone.Now()
	host, _ := os.Hostname()
	result := &Server{
		CreatedAt:     now,
		Name:          name,
		Host:          host,
		Port:          config.Port,
		Addr:          fmt.Sprintf(":%d", config.Port),
		Parent:        config.Parent,
		Solvers:       make(map[string]*Solver),
		Packages:      make(map[string]*Package),
		router:        make(map[string]*Router),
		Requests:      make(map[string]*Resolver),
		Version:       Version,
		mux:           http.NewServeMux(),
		middlewares:   make([]func(http.Handler) http.Handler, 0),
		authenticator: middleware.Authenticate,
		readTimeout:   config.ReadTimeout,
		writeTimeout:  config.WriteTimeout,
		idleTimeout:   config.IdleTimeout,
		istls:         config.IsTLS,
		certFile:      config.CertFile,
		keyFile:       config.KeyFile,
		useCache:      config.UseCache,
		useEvent:      config.UseEvent,
		debug:         config.Debug,
	}

	result.svr = &http.Server{
		Addr:         result.Addr,
		Handler:      CorsAllowAll(config.AllowOrigin).Handler(result.mux),
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: !config.IsTLS,
		},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		MaxConnsPerHost:     100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		ForceAttemptHTTP2:   true,
	}

	if config.Transport != nil {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: !config.IsTLS,
			},
			MaxIdleConns:        config.Transport.MaxIdleConns,
			MaxIdleConnsPerHost: config.Transport.MaxIdleConnsPerHost,
			MaxConnsPerHost:     config.Transport.MaxConnsPerHost,
			IdleConnTimeout:     time.Duration(config.Transport.IdleConnTimeout) * time.Second,
			TLSHandshakeTimeout: time.Duration(config.Transport.TLSHandshakeTimeout) * time.Second,
			ForceAttemptHTTP2:   config.Transport.ForceAttemptHTTP2,
		}
	}

	result.client = &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
	}

	result.mux.HandleFunc("/", result.handler)

	if config.UseCache {
		if err := cache.Load(); err != nil {
			return nil, err
		}
	}

	if config.UseEvent {
		if err := event.Load(); err != nil {
			return nil, err
		}
	}

	rpcPort := envar.GetInt("RPC_PORT", 4200)
	// tlsConfig := &tls.Config{}
	pipe, err := net.Listen("tcp", fmt.Sprintf(":%d", rpcPort))
	if err != nil {
		return nil, err
	}
	result.pipe = pipe

	return result, nil
}

/**
* ToJson
* @return (et.Json, error)
**/
func (s *Server) ToJson() (et.Json, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

/**
* banner
* @return void
**/
func (s *Server) banner() {
	templ := utility.BannerTitle(s.Name, 4)
	banner.InitString(colorable.NewColorableStdout(), true, true, templ)
	fmt.Println()
}

/**
* initHttpServer
* @return error
**/
func (s *Server) initHttpServer() error {
	go func() {
		if s.istls {
			logs.Logf("Https", `Load server on https://localhost%s`, s.Addr)
			if err := s.svr.ListenAndServeTLS(s.certFile, s.keyFile); err != nil && err != http.ErrServerClosed {
				logs.Fatal(err)
			}
		} else {
			logs.Logf("Http", `Load server on http://localhost%s`, s.Addr)
			if err := s.svr.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logs.Fatal(err)
			}
		}
	}()

	return nil
}

/**
* setSolver
* @param kind TypeRouter, method, path, solver string, typeHeader TpHeader, header map[string]string, excludeHeader []string, version int, packageName string, saved bool
* @return *Solver, error
**/
func (s *Server) setSolver(kind TypeRouter, method, path, solver string, typeHeader TpHeader, header map[string]string, excludeHeader []string, version int, packageName string, saved bool) (*Solver, error) {
	if !methods[method] {
		return nil, fmt.Errorf(msg.MSG_METHOD_NOT_SUPPORTED, method)
	}

	if !utility.ValidStr(path, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_PATH_INVALID, path)
	}

	log := func(action string) {
		if solver != "" {
			logs.Logf(s.Name, "%s Method:%s Path:%s Solver:%s Version:%d PackageName:%s", action, method, path, solver, version, packageName)
		} else {
			logs.Logf(s.Name, "%s Method:%s Path:%s Version:%d PackageName:%s", action, method, path, version, packageName)
		}
	}

	s.muRoutes.Lock()

	router, ok := s.router[method]
	if !ok {
		router = newRouter(method)
		s.router[method] = router
	}

	pkg, ok := s.Packages[packageName]
	if !ok {
		pkg = newPackage(packageName, s)
	}

	action := "Create"
	key := fmt.Sprintf("%s:%s", method, path)
	result, ok := s.Solvers[key]
	if ok {
		action = "Update"
	}

	result, err := router.set(kind, method, path, solver, typeHeader, header, excludeHeader, version)
	if err != nil {
		s.muRoutes.Unlock()
		return nil, err
	}

	if result.PackageName != packageName {
		old, ok := s.Packages[result.PackageName]
		if ok {
			old.removeSolver(result)
		}
	}

	result = pkg.addSolver(result)
	s.Solvers[key] = result

	s.muRoutes.Unlock()

	log(action)

	if saved {
		s.Save()
	}

	return result, nil
}

/**
* setHandler
* @param method, path string, handlerFn http.HandlerFunc, packageName string
* @return *Solver, error
**/
func (s *Server) setHandler(method, path string, handlerFn http.HandlerFunc, packageName string) (*Solver, error) {
	path = fmt.Sprintf("%s/%s", s.Parent, path)
	path = strings.ReplaceAll(path, "//", "/")
	result, err := s.setSolver(TpHandler, method, path, "", TpKeepHeader, map[string]string{}, []string{}, 0, packageName, false)
	if err != nil {
		return nil, err
	}

	result.handlerFn = handlerFn

	return result, nil
}

/**
* setRequest
* @param key string, resolver *Resolver
* @return void
**/
func (s *Server) setRequest(key string, resolver *Resolver) {
	s.muRequests.Lock()
	defer s.muRequests.Unlock()

	s.Requests[key] = resolver
}

/**
* getRequest
* @param key string
* @return (*Resolver, bool)
**/
func (s *Server) getRequest(key string) (*Resolver, bool) {
	s.muRequests.RLock()
	defer s.muRequests.RUnlock()

	resolver, ok := s.Requests[key]
	return resolver, ok
}

/**
* deleteRequest
* @param key string
* @return void
**/
func (s *Server) deleteRequest(key string) {
	s.muRequests.Lock()
	defer s.muRequests.Unlock()

	if r, ok := s.Requests[key]; ok && r.timer != nil {
		r.timer.Stop()
		r.timer = nil
	}
	delete(s.Requests, key)
}

/**
* listRequests
* @return map[string]*Resolver
**/
func (s *Server) listRequests() map[string]*Resolver {
	s.muRequests.RLock()
	defer s.muRequests.RUnlock()

	result := make(map[string]*Resolver)
	for k, v := range s.Requests {
		result[k] = v
	}

	return result
}

/**
* SetRouter
* @param method, path, resolve string, header et.Json, excludeHeader []string, version int, packageName string, saved bool
* @return *Solver, error
**/
func (s *Server) SetRouter(method, path, resolve string, typeHeader int, header et.Json, excludeHeader []string, version int, packageName string, saved bool) (*Solver, error) {
	headerMap := make(map[string]string)
	for k, v := range header {
		headerMap[k] = fmt.Sprintf("%v", v)
	}

	if !utility.ValidStr(resolve, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_RESOLVE_NOT_VALID, resolve)
	}

	result, err := s.setSolver(TpApiRest, method, path, resolve, TpHeader(typeHeader), headerMap, excludeHeader, version, packageName, saved)
	if err != nil {
		return nil, err
	}

	return result, nil
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

	if s.authenticator != nil {
		result.middlewares = append(result.middlewares, s.authenticator)
	}

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
	s.muRoutes.Lock()
	_, ok := s.Solvers[id]
	if !ok {
		s.muRoutes.Unlock()
		return fmt.Errorf(msg.MSG_SOLVER_NOT_FOUND, id)
	}
	delete(s.Solvers, id)
	s.muRoutes.Unlock()

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

	s.muRoutes.RLock()
	router, ok := s.router[method]
	s.muRoutes.RUnlock()

	if !ok {
		return nil, errors.New("router not found")
	}

	result, err := router.findResolver(r)
	if err != nil {
		return nil, err
	}

	if duration := s.idleTimeout + 300*time.Millisecond; duration > 0 {
		result.timer = time.AfterFunc(duration, func() {
			s.deleteRequest(result.ID)
		})
	}

	s.setRequest(result.ID, result)

	return result, nil
}

/**
* Start
* @return error
**/
func (s *Server) Start() {
	if err := s.load(); err != nil {
		logs.Fatal(err)
	}
	if err := s.basicRoutes(); err != nil {
		logs.Fatal(err)
	}
	if err := s.initEvents(); err != nil {
		logs.Fatal(err)
	}
	if err := s.initHttpServer(); err != nil {
		logs.Fatal(err)
	}
	s.banner()

	if s.debug {
		json, err := s.ToJson()
		if err != nil {
			logs.Fatal(err)
		}
		logs.Log("Start:", json.ToString())
	}

	utility.AppWait()

	s.Close()
}

/**
* Close
* @return error
**/
func (s *Server) Close() {
	if s.svr != nil {
		s.svr.Close()

		if s.istls {
			logs.Log("Https", "Shutting down server...")
		} else {
			logs.Log("Http", "Shutting down server...")
		}
	}
}

/**
* Reset
* @return error
**/
func (s *Server) Reset() {
	s.muRoutes.Lock()
	s.router = make(map[string]*Router)
	s.Solvers = make(map[string]*Solver)
	s.Packages = make(map[string]*Package)
	s.muRoutes.Unlock()

	if err := s.basicRoutes(); err != nil {
		logs.Fatal(err)
	}

	event.Publish(router.EVENT_RESET_ROUTER, et.Json{})
}
