package server

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/cgalvisleon/et/config"
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
	stdout  bool
	pidFile string
	appName string
	version string
	onClose func()
}

/**
* New
* @param appName string
* @return *Ettp, error
**/
func New(appName string) (*Ettp, error) {
	err := config.Validate([]string{
		"PORT",
		"RPC_PORT",
		"VERSION",
	})
	if err != nil {
		return nil, err
	}

	result := Ettp{
		port:    config.Int("PORT", 3000),
		rpc:     config.Int("RPC_PORT", 4200),
		stdout:  config.Bool("STDOUT", false),
		pidFile: ".pid",
		appName: appName,
		version: config.String("VERSION", "0.0.1"),
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

	return &result, nil
}

/**
* savePID
* @param pid int
* @return error
**/
func (s *Ettp) savePID(pid int) error {
	return os.WriteFile(s.pidFile, []byte(strconv.Itoa(pid)), 0644)
}

/**
* getPID
* @return int, error
**/
func (s *Ettp) getPID() (int, error) {
	data, err := os.ReadFile(s.pidFile)
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(string(data))
}

/**
* stopServer
* @param pid int
**/
func (s *Ettp) stopServer(pid int) {
	if pid == -1 {
		return
	}

	_, err := exec.Command("kill", fmt.Sprintf("%d", pid)).CombinedOutput()
	if err != nil {
		logs.Alertf("Error al detener el servidor: %s", err.Error())
	} else {
		logs.Logf(s.appName, "Servidor detenido PID:%d", pid)
		s.savePID(-1)
	}
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
* StartHttpServer
**/
func (s *Ettp) StartHttpServer() {
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
func (s *Ettp) Background() {
	s.StartHttpServer()
}

/**
* Start
**/
func (s *Ettp) Start() {
	go s.Background()
	s.banner()

	utility.AppWait()

	s.Close()
}
