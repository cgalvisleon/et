package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/utility"
	"github.com/dimiro1/banner"
	"github.com/go-chi/chi"
	"github.com/mattn/go-colorable"
	"github.com/rs/cors"
)

var packageName = "Ettp"

type Ettp struct {
	*chi.Mux
	http    *http.Server
	port    int
	rpc     int
	pidFile string
	appName string
	onClose func()
}

/**
* New
* @param appName string
* @return *Ettp, error
**/
func New(appName string) (*Ettp, error) {
	result := &Ettp{
		port:    envar.GetInt("PORT", 3000),
		rpc:     envar.GetInt("RPC_PORT", 4200),
		pidFile: ".pid",
		appName: appName,
	}

	if result.port != 0 {
		result.Mux = chi.NewRouter()
		result.Mux.Use(middleware.Logger)
		result.Mux.Use(middleware.Recoverer)
		result.Mux.NotFound(func(w http.ResponseWriter, r *http.Request) {
			response.HTTPError(w, r, http.StatusNotFound, "404 Not Found")
		})

		addr := fmt.Sprintf(":%d", result.port)
		serv := &http.Server{
			Addr:    addr,
			Handler: cors.AllowAll().Handler(result.Mux),
		}

		result.http = serv
	}

	return result, nil
}

/**
* banner
**/
func (s *Ettp) banner() {
	time.Sleep(3 * time.Second)
	templ := utility.BannerTitle(s.appName, 4)
	banner.InitString(colorable.NewColorableStdout(), true, true, templ)
	fmt.Println()
}

/**
* Close
**/
func (s *Ettp) Close() {
	if s.onClose != nil {
		s.onClose()
	}

	logs.Log("Http", "Shutting down server...")
}

/**
* OnClose
* @param onClose func()
**/
func (s *Ettp) OnClose(onClose func()) {
	s.onClose = onClose
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
func (s *Ettp) startHttpServer() {
	if s.http == nil {
		return
	}

	svr := s.http
	logs.Logf(packageName, "Running on http://localhost%s", svr.Addr)
	if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logs.Logf(packageName, "Http error: %s", err)
	}
}

/**
* Background
**/
func (s *Ettp) background() {
	s.startHttpServer()
}

/**
* Start
**/
func (s *Ettp) Start() {
	go s.background()
	s.banner()

	utility.AppWait()

	s.Close()
}
