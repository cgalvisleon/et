package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/utility"
	"github.com/dimiro1/banner"
	"github.com/go-chi/chi/v5"
	"github.com/mattn/go-colorable"
	"github.com/rs/cors"
)

var packageName = "Ettp"

type Ettp struct {
	*chi.Mux
	http    *http.Server
	port    int
	pidFile string
	name    string
	onClose []func()
	onStart []func()
}

/**
* New
* @param appName string
* @return *Ettp
**/
func New(name string) *Ettp {
	result := &Ettp{
		port:    envar.GetInt("PORT", 3000),
		pidFile: ".pid",
		name:    name,
		onClose: make([]func(), 0),
		onStart: make([]func(), 0),
	}

	if result.port != 0 {
		result.Mux = chi.NewRouter()
		result.NotFound(func(w http.ResponseWriter, r *http.Request) {
			response.HTTPError(w, r, http.StatusNotFound, "404 Not Found")
		})

		addr := fmt.Sprintf(":%d", result.port)
		serv := &http.Server{
			Addr:    addr,
			Handler: cors.AllowAll().Handler(result.Mux),
		}

		result.http = serv
	}

	return result
}

/**
* banner
**/
func (s *Ettp) banner() {
	time.Sleep(3 * time.Second)
	templ := utility.BannerTitle(s.name, 4)
	banner.InitString(colorable.NewColorableStdout(), true, true, templ)
	fmt.Println()
}

/**
* Close
**/
func (s *Ettp) Close() {
	for _, fn := range s.onClose {
		fn()
	}

	logs.Log(packageName, "Shutting down server...")
}

/**
* OnClose
* @param onClose func()
**/
func (s *Ettp) OnClose(onClose func()) {
	s.onClose = append(s.onClose, onClose)
}

/**
* OnStart
* @param onStart func()
**/
func (s *Ettp) OnStart(onStart func()) {
	s.onStart = append(s.onStart, onStart)
}

/**
* Use
* @param middlewares ...func(http.Handler) http.Handler
**/
func (s *Ettp) Use(middlewares ...func(http.Handler) http.Handler) {
	if s.Mux == nil {
		return
	}

	s.Mux.Use(middlewares...)
}

/**
* NotFound
* @param handlerFn http.HandlerFunc
**/
func (s *Ettp) NotFound(handlerFn http.HandlerFunc) {
	if s.Mux == nil {
		return
	}

	s.Mux.NotFound(handlerFn)
}

/**
* HandleFunc
* @param pattern string, handlerFn http.HandlerFunc
**/
func (s *Ettp) HandleFunc(pattern string, handlerFn http.HandlerFunc) {
	if s.Mux == nil {
		return
	}

	s.Mux.HandleFunc(pattern, handlerFn)
}

/**
* Mount
* @param pattern string, handler http.Handler
**/
func (s *Ettp) Mount(pattern string, handler http.Handler) {
	if s.Mux == nil {
		return
	}

	s.Mux.Mount(pattern, handler)
}

/**
* startHttpServer
**/
func (s *Ettp) startHttpServer() error {
	if s.http == nil {
		return nil
	}

	svr := s.http
	logs.Logf(packageName, "Running on http://localhost%s", svr.Addr)
	if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logs.Logf(packageName, "Http error: %s", err)
		return err
	}

	return nil
}

/**
* start
**/
func (s *Ettp) start() {
	if err := s.startHttpServer(); err != nil {
		logs.Panic(err)
	}
}

/**
* Start
**/
func (s *Ettp) Start() {
	go s.start()
	for _, fn := range s.onStart {
		fn()
	}
	s.printRoutes()
	s.banner()
}

/**
* StartAndWait
**/
func (s *Ettp) StartWait() {
	s.Start()
	utility.AppWait()
}

/**
* printRoutes
**/
func (s *Ettp) printRoutes() {
	logs.Log(packageName, "📌 Rutas cargadas:")
	chi.Walk(s.Mux, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		logs.Logf(packageName, "%s:%s", method, route)
		return nil
	})
}
