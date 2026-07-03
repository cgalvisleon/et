package router

import (
	"fmt"
	"net/http"

	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/strs"
	"github.com/go-chi/chi/v5"
)

type Api struct {
	Name           string
	Path           string
	Version        string
	Host           string
	Port           int
	Rpc            int
	Addr           string
	Router         *chi.Mux
	authentication []func(http.Handler) http.Handler
	authorization  []func(http.Handler) http.Handler
}

/**
* NewApi
* @param name, path, host, port, rpc, version string
* @return *Api
**/
func NewApi(name, path, host string, port, rpc int, version string) *Api {
	addr := host
	portS := fmt.Sprintf("%d", port)
	addr = strs.Append(addr, portS, ":")
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	return &Api{
		Name:           name,
		Path:           path,
		Version:        version,
		Host:           host,
		Port:           port,
		Rpc:            rpc,
		Addr:           addr,
		Router:         r,
		authentication: make([]func(http.Handler) http.Handler, 0),
		authorization:  make([]func(http.Handler) http.Handler, 0),
	}
}

/**
* Authentication
* @param middlewares ...func(http.Handler) http.Handler
**/
func (s *Api) UseAuthentication(middlewares ...func(http.Handler) http.Handler) {
	s.authentication = append(s.authentication, middlewares...)
}

/**
* Authorization
* @param middlewares ...func(http.Handler) http.Handler
**/
func (s *Api) UseAuthorization(middlewares ...func(http.Handler) http.Handler) {
	s.authorization = append(s.authorization, middlewares...)
}

/**
* Public
* @param method, path string, handler func(http.ResponseWriter, *http.Request)
**/
func (s *Api) Public(method, path string, handler func(http.ResponseWriter, *http.Request)) {
	Publish(s.Router, method, path, handler, s.Name, s.Path, s.Addr)
}

/**
* Protect
* @param method, path string, handler func(http.ResponseWriter, *http.Request)
**/
func (s *Api) Authentication(method, path string, handler func(http.ResponseWriter, *http.Request)) {
	With(s.Router, method, path, handler, s.Name, s.Path, s.Addr, s.authentication)
}

/**
* Authorization
* @param method, path string, handler func(http.ResponseWriter, *http.Request)
**/
func (s *Api) Authorization(method, path string, handler func(http.ResponseWriter, *http.Request)) {
	With(s.Router, method, path, handler, s.Name, s.Path, s.Addr, s.authorization)
}
