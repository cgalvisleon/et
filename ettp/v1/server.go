package ettp

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
	"github.com/rs/cors"
)

const packageName = "ettp"

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
	Name         string           `json:"name"`
	Company      string           `json:"company"`
	Web          string           `json:"web"`
	Help         string           `json:"help"`
	Port         int              `json:"port"`
	PathApi      string           `json:"path_api"`
	PathApp      string           `json:"path_app"`
	ReadTimeout  time.Duration    `json:"read_timeout"`
	WriteTimeout time.Duration    `json:"write_timeout"`
	IdleTimeout  time.Duration    `json:"idle_timeout"`
	IsTLS        bool             `json:"is_tls"`
	CertFile     string           `json:"cert_file"`
	KeyFile      string           `json:"key_file"`
	Transport    *TransportConfig `json:"transport"`
	Timeout      time.Duration    `json:"timeout"`
	Debug        bool             `json:"debug"`
}

type Server struct {
	CreatedAt       time.Time                         `json:"created_at"`
	Id              string                            `json:"id"`
	Name            string                            `json:"name"`
	Version         string                            `json:"version"`
	Company         string                            `json:"company"`
	Web             string                            `json:"web"`
	Help            string                            `json:"help"`
	HostName        string                            `json:"host_name"`
	addr            string                            `json:"-"`
	mux             *http.ServeMux                    `json:"-"`
	svr             *http.Server                      `json:"-"`
	client          *http.Client                      `json:"-"`
	pathApi         string                            `json:"-"`
	pathApp         string                            `json:"-"`
	cors            *cors.Cors                        `json:"-"`
	middlewares     []func(http.Handler) http.Handler `json:"-"`
	authenticator   func(http.Handler) http.Handler   `json:"-"`
	notFoundHandler http.HandlerFunc                  `json:"-"`
	router          map[string]*Router                `json:"-"`
	solvers         []*Router                         `json:"-"`
	packages        []*Package                        `json:"-"`
	handlers        map[string]*ApiFunc               `json:"-"`
	proxys          map[string]*Proxy                 `json:"-"`
	mu              sync.RWMutex                      `json:"-"`
	readTimeout     time.Duration                     `json:"-"`
	writeTimeout    time.Duration                     `json:"-"`
	idleTimeout     time.Duration                     `json:"-"`
	isTls           bool                              `json:"-"`
	certFile        string                            `json:"-"`
	keyFile         string                            `json:"-"`
	storageKey      string                            `json:"-"`
	debug           bool                              `json:"-"`
}

func New(cnf *Config) (*Server, error) {
	/* Cache */
	err := cache.Load(config.CNF)
	if err != nil {
		return nil, err
	}

	/* Event */
	err = event.Load(config.CNF)
	if err != nil {
		return nil, err
	}

	/* Http ServeMux */
	version := "v0.0.1"
	hostName, _ := os.Hostname()
	mux := http.NewServeMux()
	srv := &Server{
		CreatedAt:       timezone.Now(),
		Id:              utility.UUID(),
		Name:            cnf.Name,
		Version:         version,
		Company:         cnf.Company,
		Web:             cnf.Web,
		Help:            cnf.Help,
		HostName:        hostName,
		addr:            fmt.Sprintf(":%d", cnf.Port),
		mux:             mux,
		pathApi:         cnf.PathApi,
		pathApp:         cnf.PathApp,
		cors:            CorsAllowAll([]string{}),
		notFoundHandler: notFoundHandler,
		middlewares:     make([]func(http.Handler) http.Handler, 0),
		router:          make(map[string]*Router),
		solvers:         []*Router{},
		packages:        []*Package{},
		handlers:        make(map[string]*ApiFunc),
		proxys:          make(map[string]*Proxy),
		readTimeout:     cnf.ReadTimeout,
		writeTimeout:    cnf.WriteTimeout,
		idleTimeout:     cnf.IdleTimeout,
		isTls:           cnf.IsTLS,
		certFile:        cnf.CertFile,
		keyFile:         cnf.KeyFile,
		storageKey:      fmt.Sprintf("%s-%s", cnf.Name, version),
		debug:           cnf.Debug,
	}

	srv.svr = &http.Server{
		Addr:         srv.addr,
		Handler:      srv.cors.Handler(srv.mux),
		ReadTimeout:  srv.readTimeout,
		WriteTimeout: srv.writeTimeout,
		IdleTimeout:  srv.idleTimeout,
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: !cnf.IsTLS,
		},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		MaxConnsPerHost:     100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		ForceAttemptHTTP2:   true,
	}

	if cnf.Transport != nil {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: !cnf.IsTLS,
			},
			MaxIdleConns:        cnf.Transport.MaxIdleConns,
			MaxIdleConnsPerHost: cnf.Transport.MaxIdleConnsPerHost,
			MaxConnsPerHost:     cnf.Transport.MaxConnsPerHost,
			IdleConnTimeout:     time.Duration(cnf.Transport.IdleConnTimeout) * time.Second,
			TLSHandshakeTimeout: time.Duration(cnf.Transport.TLSHandshakeTimeout) * time.Second,
			ForceAttemptHTTP2:   cnf.Transport.ForceAttemptHTTP2,
		}
	}

	srv.client = &http.Client{
		Timeout:   cnf.Timeout,
		Transport: transport,
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
		if s.isTls {
			logs.Logf(packageName, `Load server on https://localhost%s`, s.addr)
			if err := s.svr.ListenAndServeTLS(s.certFile, s.keyFile); err != nil && err != http.ErrServerClosed {
				logs.Alert(err)
			}
		} else {
			logs.Logf(packageName, `Load server on http://localhost%s`, s.addr)
			if err := s.svr.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logs.Alert(err)
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
	s.mu.Lock()
	s.router = make(map[string]*Router)
	s.solvers = []*Router{}
	s.packages = []*Package{}
	s.mu.Unlock()

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

		if s.isTls {
			logs.Logf(packageName, "Shutting down server...")
		} else {
			logs.Logf(packageName, "Shutting down server...")
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
