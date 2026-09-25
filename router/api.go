package router

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/strs"
	"github.com/go-chi/chi/v5"
)

var (
	EVENT_SET_ENDPOINT = "event:set:endpoint"
)

type Api struct {
	Name                string
	Path                string
	Version             int
	Host                string
	Port                int
	Rpc                 int
	Addr                string
	Router              *chi.Mux
	authentication      []func(http.Handler) http.Handler
	authorization       func(method, path string) func(http.Handler) http.Handler
	errorStatusNotFound []string
}

/**
* NewApi
* @param name, path, host, port, rpc, version int
* @return *Api
**/
func NewApi(name, path, host string, port, rpc int, version int) *Api {
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
	}
}

/**
* AddErrorStatusNotFound
* @param statuses ...string
**/
func (s *Api) AddErrorStatusNotFound(statuses ...string) {
	s.errorStatusNotFound = append(s.errorStatusNotFound, statuses...)
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
* @param authorization func(method, path string) func(http.Handler) http.Handler
**/
func (s *Api) UseAuthorization(authorization func(method, path string) func(http.Handler) http.Handler) {
	s.authorization = authorization
}

/**
* public
* @param method, path string, handler http.HandlerFunc
**/
func (s *Api) Public(method, path string, handler http.HandlerFunc) {
	With(s.Router, Route{
		Method:      method,
		Path:        path,
		Handler:     handler,
		Host:        s.Addr,
		PackageName: s.Path,
	}, []func(http.Handler) http.Handler{})
}

/**
* registerEndpoint
* @param packageName, group, method, path, name string
* @return error
**/
func (s *Api) RegisterEndpoint(packageName, group, method, path, name string) error {
	data := et.Json{
		"package_name": packageName,
		"group_name":   group,
		"method":       method,
		"path":         path,
		"name":         name,
	}
	return event.Publish(EVENT_SET_ENDPOINT, data)
}

/**
* Session
* @param method, path string, handler http.HandlerFunc
**/
func (s *Api) Session(method, path string, handler http.HandlerFunc) {
	With(s.Router, Route{
		Method:      method,
		Path:        path,
		Handler:     handler,
		Host:        s.Addr,
		PackageName: s.Path,
		Version:     s.Version,
	}, s.authentication)
}

/**
* session
* @param method, path string, handler http.HandlerFunc
**/
func (s *Api) Protected(group, method, path, name string, handler http.HandlerFunc) {
	s.RegisterEndpoint(s.Name, group, method, path, name)
	authorize := make([]func(http.Handler) http.Handler, 0)
	authorize = append(authorize, s.authentication...)
	if s.authorization != nil {
		authorize = append(authorize, s.authorization(method, path))
	}
	With(s.Router, Route{
		Method:      method,
		Path:        path,
		Handler:     handler,
		Host:        s.Addr,
		PackageName: s.Path,
	}, authorize)
}

// ---------- Helpers ----------

/**
* ReadBody
* @param r *http.Request
* @return et.Json, error
**/
func (s *Api) readBody(r *http.Request) (et.Json, error) {
	// Sin cuerpo, un objeto vacío
	if r.Body == nil || r.ContentLength == 0 {
		return et.Json{}, nil
	}
	return request.GetBody(r)
}

/**
* fail
* @param w http.ResponseWriter, r *http.Request, err error
**/
func (s *Api) fail(w http.ResponseWriter, r *http.Request, err error) {
	// 404 si no se encontró; 400 lo demás. Los MSG_* se traducen al arrancar, así que se compara el texto
	status := http.StatusBadRequest
	if slices.Contains(s.errorStatusNotFound, err.Error()) {
		status = http.StatusNotFound
	}
	response.HTTPError(w, r, status, err.Error())
}

// ---------- Wrappers ----------

/**
* Item
* @param status int, fn func(r *http.Request, body et.Json) (et.Item, error)
* @return http.HandlerFunc
**/
func (s *Api) Item(status int, fn func(r *http.Request, body et.Json) (et.Item, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := s.readBody(r)
		if err != nil {
			response.HTTPError(w, r, http.StatusBadRequest, err.Error())
			return
		}

		result, err := fn(r, body)
		if err != nil {
			s.fail(w, r, err)
			return
		}

		response.ITEM(w, r, status, result)
	}
}

/**
* Items
* @param fn func(r *http.Request, body et.Json) (et.Items, error)
* @return http.HandlerFunc
**/
func (s *Api) Items(fn func(r *http.Request, body et.Json) (et.Items, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := s.readBody(r)
		if err != nil {
			response.HTTPError(w, r, http.StatusBadRequest, err.Error())
			return
		}

		result, err := fn(r, body)
		if err != nil {
			s.fail(w, r, err)
			return
		}

		response.ITEMS(w, r, http.StatusOK, result)
	}
}

/**
* List
* @param fn func(r *http.Request) (et.List, error)
* @return http.HandlerFunc
**/
func (s *Api) List(fn func(r *http.Request) (et.List, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := fn(r)
		if err != nil {
			s.fail(w, r, err)
			return
		}

		response.LIST(w, r, http.StatusOK, result)
	}
}
