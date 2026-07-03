package server

import (
	"net/http"
	"os"

	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/router"
	"github.com/cgalvisleon/et/strs"
	"github.com/go-chi/chi/v5"
)

type Api struct {
	Name          string
	Hostname      string
	path          string
	host          string
	port          int
	addr          string
	r             *chi.Mux
	Version       string
	loaded        bool
	autentication []func(http.Handler) http.Handler
	authorization []func(http.Handler) http.Handler
}

func NewApi(name, path, host string, port int, version string) *Api {
	hostname, _ := os.Hostname()
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	result := &Api{
		Name:          name,
		path:          path,
		host:          host,
		port:          port,
		Hostname:      hostname,
		Version:       version,
		r:             r,
		autentication: make([]func(http.Handler) http.Handler, 0),
		authorization: make([]func(http.Handler) http.Handler, 0),
	}
	result.addr = strs.Format("%s:%d", result.host, result.port)

	return result
}

/**
* UseAutentication
* @param middleware func(http.Handler) http.Handler
**/
func (s *Api) UseAutentication(middleware func(http.Handler) http.Handler) {
	s.autentication = append(s.autentication, middleware)
}

/**
* UseAuthorization
* @param middleware func(http.Handler) http.Handler
**/
func (s *Api) UseAuthorization(middleware func(http.Handler) http.Handler) {
	s.authorization = append(s.authorization, middleware)
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
	router.Publish(r, method, path, handler, s.Name, s.path, s.addr)
}

/**
* Private
* @param method, path string, handler http.HandlerFunc
**/
func (s *Api) Private(r *chi.Mux, method, path string, handler http.HandlerFunc) {
	router.With(r, method, path, handler, s.Name, s.path, s.addr, s.autentication)
}

/**
* Authentication
* @param method, path string, handler http.HandlerFunc
**/
func (s *Api) Authentication(r *chi.Mux, method, path string, handler http.HandlerFunc) {
	router.With(r, method, path, handler, s.Name, s.path, s.addr, s.autentication)
}

/**
* Authorization
* @param method, path string, handler http.HandlerFunc
**/
func (s *Api) Authorization(r *chi.Mux, method, path string, handler http.HandlerFunc) {
	router.With(r, method, path, handler, s.Name, s.path, s.addr, s.authorization)
}
