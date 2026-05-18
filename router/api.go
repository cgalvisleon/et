package router

import (
	"net/http"

	"github.com/cgalvisleon/et/strs"
	"github.com/go-chi/chi/v5"
)

type Api struct {
	Name    string
	Path    string
	Version string
	Host    string
	Port    int
	Rpc     int
	Addr    string
	Router  *chi.Mux
}

/**
* NewApi
* @param name, path, host, port, rpc, version string
* @return *Api
**/
func NewApi(name, path, host string, port, rpc int, version string) *Api {
	addr := strs.Format("%s:%d", host, port)
	return &Api{
		Name:    name,
		Path:    path,
		Version: version,
		Host:    host,
		Port:    port,
		Rpc:     rpc,
		Addr:    addr,
		Router:  chi.NewRouter(),
	}
}

/**
* Public
* @param method, path string, handler func(http.ResponseWriter, *http.Request)
**/
func (s *Api) Public(method, path string, handler func(http.ResponseWriter, *http.Request)) {
	path = strs.Format("%s%s", s.Path, path)
	Public(s.Router, method, path, handler, s.Name, s.Path, s.Host)
}

/**
* Protect
* @param method, path string, handler func(http.ResponseWriter, *http.Request)
**/
func (s *Api) Protect(method, path string, handler func(http.ResponseWriter, *http.Request)) {
	path = strs.Format("%s/%s", s.Path, path)
	Protect(s.Router, method, path, handler, s.Name, s.Path, s.Host)
}
