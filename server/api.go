package server

import (
	"net/http"
	"os"

	"github.com/cgalvisleon/et/router"
	"github.com/cgalvisleon/et/strs"
	"github.com/go-chi/chi/v5"
)

type Api struct {
	Name     string
	Hostname string
	path     string
	host     string
	port     int
	addr     string
	r        *chi.Mux
	Version  string
	loaded   bool
}

func NewApi(name, path, host string, port int, version string) *Api {
	hostname, _ := os.Hostname()
	result := &Api{
		Name:     name,
		path:     path,
		host:     host,
		port:     port,
		Hostname: hostname,
		r:        chi.NewRouter(),
		Version:  version,
	}
	result.addr = strs.Format("%s:%d", result.host, result.port)

	return result
}

/**
* SetAutentication
* @param middleware func(http.Handler) http.Handler
**/
func (s *Api) SetAutentication(middleware func(http.Handler) http.Handler) {
	router.SetAutentication(middleware)
}

/**
* Use
* @param middlewares ...func(http.Handler) http.Handler
**/
func (s *Api) Use(middlewares ...func(http.Handler) http.Handler) {
	s.r.Use(middlewares...)
}

/**
* Public
* @param method, path string, handler http.HandlerFunc
**/
func (s *Api) Public(r *chi.Mux, method, path string, handler http.HandlerFunc) {
	router.Public(r, method, path, handler, s.Name, s.path, s.addr)
}

/**
* Private
* @param method, path string, handler http.HandlerFunc
**/
func (s *Api) Private(r *chi.Mux, method, path string, handler http.HandlerFunc) {
	router.Private(r, method, path, handler, s.Name, s.path, s.addr)
}
