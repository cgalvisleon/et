package router

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
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
	LimitRate           int64
	authentication      []func(http.Handler) http.Handler
	idenpotency         []func(http.Handler) http.Handler
	authorization       func(method, path string) func(http.Handler) http.Handler
	registerEndpoint    func(packageName, group, method, path, name string) error
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
		LimitRate:      1000,
		authentication: make([]func(http.Handler) http.Handler, 0),
	}
}

func (s *Api) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + ":" + r.URL.Path
		duration := 1 * time.Second
		count := cache.Incr(key, duration)
		if count > s.LimitRate {
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

/**
* AddErrorStatusNotFound
* @param statuses ...string
**/
func (s *Api) AddErrorStatusNotFound(statuses ...string) {
	s.errorStatusNotFound = append(s.errorStatusNotFound, statuses...)
}

/**
* UseLimitRate
* @param limitRate int64
**/
func (s *Api) UseLimitRate(limitRate int64) {
	s.LimitRate = limitRate
}

/**
* Authentication
* @param middlewares ...func(http.Handler) http.Handler
**/
func (s *Api) UseAuthentication(middlewares ...func(http.Handler) http.Handler) {
	s.authentication = append(s.authentication, middlewares...)
}

/**
* UseIdenpotency
* @param middlewares ...func(http.Handler) http.Handler
**/
func (s *Api) UseIdenpotency(middlewares ...func(http.Handler) http.Handler) {
	s.idenpotency = append(s.idenpotency, middlewares...)
}

/**
* Authorization
* @param authorization func(method, path string) func(http.Handler) http.Handler
**/
func (s *Api) UseAuthorization(authorization func(method, path string) func(http.Handler) http.Handler) {
	s.authorization = authorization
}

func (s *Api) getPath(path string) string {
	path = strs.Append(s.Path, path, "/")
	path = strings.ReplaceAll(path, "//", "/")
	path = strings.ReplaceAll(path, "//", "/")
	return path
}

/**
* UseRegisterEndpoint
* @param registerEndpoint func(packageName, group, method, path, name string) error
**/
func (s *Api) UseRegisterEndpoint(registerEndpoint func(packageName, group, method, path, name string) error) {
	s.registerEndpoint = registerEndpoint
}

/**
* RegisterEndpoint
* @param packageName, group, method, path, name string
* @return error
**/
func (s *Api) RegisterEndpoint(packageName, group, method, path, name string) error {
	if s.registerEndpoint != nil {
		return s.registerEndpoint(packageName, group, method, path, name)
	}
	return nil
}

/**
* Public: A route without token; it is published to the endpoint catalog with its name.
* @param method, path, name string, handler http.HandlerFunc
**/
func (s *Api) Public(method, path, name string, handler http.HandlerFunc) {
	path = s.getPath(path)
	s.RegisterEndpoint(s.Name, "", method, path, name)
	middlewares := make([]func(http.Handler) http.Handler, 0)
	middlewares = append(middlewares, s.rateLimitMiddleware)
	if s.idenpotency != nil {
		middlewares = append(middlewares, s.idenpotency...)
	}
	With(s.Router, Route{
		Method:      method,
		Path:        path,
		Handler:     handler,
		Host:        s.Addr,
		PackageName: s.Name,
	}, middlewares)
}

/**
* Session: A route with token that does not check the role; it is published to the endpoint catalog with its name.
* @param method, path, name string, handler http.HandlerFunc
**/
func (s *Api) Session(method, path, name string, handler http.HandlerFunc) {
	path = s.getPath(path)
	s.RegisterEndpoint(s.Name, "", method, path, name)
	middlewares := make([]func(http.Handler) http.Handler, 0)
	middlewares = append(middlewares, s.rateLimitMiddleware)
	if s.authentication != nil {
		middlewares = append(middlewares, s.authentication...)
	}
	if s.idenpotency != nil {
		middlewares = append(middlewares, s.idenpotency...)
	}
	With(s.Router, Route{
		Method:      method,
		Path:        path,
		Handler:     handler,
		Host:        s.Addr,
		PackageName: s.Name,
		Version:     s.Version,
	}, middlewares)
}

/**
* Protected: A route with token and the role's authorization, published to the endpoint catalog in its group.
* @param group, method, path, name string, handler http.HandlerFunc
**/
func (s *Api) Protected(group, method, path, name string, handler http.HandlerFunc) {
	path = s.getPath(path)
	s.RegisterEndpoint(s.Name, group, method, path, name)
	middlewares := make([]func(http.Handler) http.Handler, 0)
	middlewares = append(middlewares, s.rateLimitMiddleware)
	if s.authentication != nil {
		middlewares = append(middlewares, s.authentication...)
	}
	if s.idenpotency != nil {
		middlewares = append(middlewares, s.idenpotency...)
	}
	if s.authorization != nil {
		middlewares = append(middlewares, s.authorization(method, path))
	}
	With(s.Router, Route{
		Method:      method,
		Path:        path,
		Handler:     handler,
		Host:        s.Addr,
		PackageName: s.Name,
		Version:     s.Version,
	}, middlewares)
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
