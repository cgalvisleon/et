package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/utility"
	"github.com/dimiro1/banner"
	"github.com/go-chi/chi/v5"
	"github.com/mattn/go-colorable"
	"github.com/rs/cors"
)

var packageName = "Ettp"

/**
* Ettp: Minimal HTTP server over chi.Mux (no cache, events or gateway) with start/close hooks;
* with port 0 nothing is served and the mux is nil.
**/
type Ettp struct {
	*chi.Mux
	http      *http.Server
	port      int
	pidFile   string
	name      string
	onClose   []func()
	onStart   []func()
	stop      chan os.Signal
	closeOnce sync.Once
}

// shutdownTimeout: cuánto espera Close a que terminen las peticiones en curso antes de cortar el HTTP.
const shutdownTimeout = 10 * time.Second

/**
* New
* @param appName string
* @return *Ettp
**/
func New(name string, port int) *Ettp {
	result := &Ettp{
		port:    port,
		pidFile: ".pid",
		name:    name,
		onClose: make([]func(), 0),
		onStart: make([]func(), 0),
		stop:    make(chan os.Signal, 1),
	}
	// From here, not from Start: a SIGTERM while the app loads (between New and Start)
	// waits for Start instead of ending the process without running OnClose
	signal.Notify(result.stop, os.Interrupt, syscall.SIGTERM)

	if result.port != 0 {
		result.Mux = chi.NewRouter()
		// Own 404: a JSON error in the library's format instead of chi's plain text
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
* banner: Prints the name banner. It waits 3 s first so the banner comes out after the
* startup logs of the server and of the OnStart hooks.
**/
func (s *Ettp) banner() {
	time.Sleep(3 * time.Second)
	templ := utility.BannerTitle(s.name, 4)
	banner.InitString(colorable.NewColorableStdout(), true, true, templ)
	fmt.Println()
}

/**
* Close: Stops the HTTP server (waiting up to shutdownTimeout for the requests in flight) and then runs the OnClose
* hooks, once: Start calls it on SIGINT/SIGTERM, and calling it again does nothing.
**/
func (s *Ettp) Close() {
	s.closeOnce.Do(func() {
		logs.Log(packageName, "Shutting down server...")
		signal.Stop(s.stop)
		if s.http != nil {
			ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
			defer cancel()
			if err := s.http.Shutdown(ctx); err != nil {
				logs.Logf(packageName, "Http shutdown: %s", err)
			}
		}

		for _, fn := range s.onClose {
			fn()
		}
	})
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
* Start: Serves HTTP, runs the OnStart hooks and blocks until SIGINT/SIGTERM; then Close (HTTP shutdown and the
* OnClose hooks) before returning.
**/
func (s *Ettp) Start() {
	go s.start()
	for _, fn := range s.onStart {
		fn()
	}
	s.printRoutes()
	s.banner()
	<-s.stop
	s.Close()
}

/**
* printRoutes
**/
func (s *Ettp) printRoutes() {
	logs.Log(packageName, "📌 "+msg.MSG_ROUTES_LOADED)
	chi.Walk(s.Mux, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		logs.Logf(packageName, "%s:%s", method, route)
		return nil
	})
}
