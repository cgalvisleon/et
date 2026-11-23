package router

import (
	"cmp"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/strs"
	"github.com/go-chi/chi/v5"
)

/**
* Invariants of the router package.
*
* Call event.Load() and cache.Load() before the first Public/Session/Protected: the first route
* subscribes the reset channels (initRouter) and every route is published to the gateway over
* NATS; without a connection they are silently skipped. initRouter ignores the error of
* event.Stack and never retries.
* Without a cache, Incr returns 0 and the rate limit never triggers.
**/

var (
	EVENT_SET_ENDPOINT = "event:set:endpoint"
)

/**
* Api: A chi router of one service: routes by group (Public, Session, Protected), each rate
* limited and published to the gateway and to the endpoint catalog.
**/
type Api struct {
	Name                string                                                           `json:"name"`
	Path                string                                                           `json:"path"`
	Version             int                                                              `json:"version"`
	Host                string                                                           `json:"host"`
	Port                int                                                              `json:"port"`
	Rpc                 int                                                              `json:"rpc"`
	Addr                string                                                           `json:"addr"`
	Router              *chi.Mux                                                         `json:"router"`
	LimitRate           int64                                                            `json:"limitRate"`
	AppId               string                                                           `json:"appId"`
	authentication      []func(http.Handler) http.Handler                                `json:"-"`
	idempotency         []func(http.Handler) http.Handler                                `json:"-"`
	authorization       func(method, path string) func(http.Handler) http.Handler        `json:"-"`
	registerEndpoint    func(packageName, group, method, path, name, appId string) error `json:"-"`
	errorStatusNotFound []string                                                         `json:"-"`
	endpoints           []Endpoint                                                       `json:"-"`
	rpcStarted          bool                                                             `json:"-"`
	mcp                 *MCPConfig                                                       `json:"-"`
}

/**
* Endpoint: A route mounted by the Api (Public, Session or Protected), with the path already prefixed; Access is public,
* session or protected and Doc its contract, if it has one.
**/
type Endpoint struct {
	PackageName string `json:"packageName"`
	Group       string `json:"group"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	Name        string `json:"name"`
	Access      string `json:"access"`
	Doc         *Doc   `json:"doc,omitempty"`
}

// Access of an Endpoint
const (
	AccessPublic    = "public"
	AccessSession   = "session"
	AccessProtected = "protected"
)

/**
* ToJson: Convert the Endpoint to a JSON object; doc only if the route has one.
* @return et.Json
**/
func (s *Endpoint) ToJson() et.Json {
	result := et.Json{
		"packageName": s.PackageName,
		"group":       s.Group,
		"method":      s.Method,
		"path":        s.Path,
		"name":        s.Name,
		"access":      s.Access,
	}
	if s.Doc != nil {
		result["doc"] = s.Doc.ToJson()
	}
	return result
}

/**
* NewApi: Builds the Api with Logger and Recoverer and a rate limit of 1000 requests per second per route.
* @param name, path, host string, port, rpc, version int
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
		key := "ratelimit:" + r.Method + ":" + r.URL.Path
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
* UseIdempotency
* @param middlewares ...func(http.Handler) http.Handler
**/
func (s *Api) UseIdempotency(middlewares ...func(http.Handler) http.Handler) {
	s.idempotency = append(s.idempotency, middlewares...)
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
func (s *Api) UseRegisterEndpoint(registerEndpoint func(packageName, group, method, path, name, appId string) error) {
	s.registerEndpoint = registerEndpoint
}

/**
* RegisterEndpoint
* @param packageName, group, method, path, name string
* @return error
**/
func (s *Api) RegisterEndpoint(packageName, group, method, path, name string) error {
	endpoint := Endpoint{PackageName: packageName, Group: group, Method: method, Path: path, Name: name}
	return s.registerRoute(endpoint)
}

/**
* registerRoute: Keeps the endpoint (with its access and doc) and registers it with the UseRegisterEndpoint hook
* @param endpoint Endpoint
* @return error
**/
func (s *Api) registerRoute(endpoint Endpoint) error {
	s.endpoints = append(s.endpoints, endpoint)
	packageName, group, method, path, name := endpoint.PackageName, endpoint.Group, endpoint.Method, endpoint.Path, endpoint.Name
	if s.registerEndpoint == nil {
		return nil
	}

	// Public, Session and Protected drop the error: log it so a route missing from the catalog is visible
	err := s.registerEndpoint(packageName, group, method, path, name, s.AppId)
	if err != nil {
		logs.Alert(err)
	}
	return err
}

/**
* Endpoints: The routes mounted by the Api, in the order they were mounted.
* @return []Endpoint
**/
func (s *Api) Endpoints() []Endpoint {
	return slices.Clone(s.endpoints)
}

/**
* ListEndpoints: List the endpoints of the Api.
* @return et.Items
**/
func (s *Api) ListEndpoints(r *http.Request, body et.Json) (et.Items, error) {
	items := et.Items{}
	for _, endpoint := range s.Endpoints() {
		items.Add(endpoint.ToJson())
	}
	return items, nil
}

/**
* MountRpc: Mounts the methods of services (net/rpc, "Struct.Method") in the RPC server of the Api, on its Rpc port, and
* starts it the first time; with Rpc 0 it does nothing. The host announced to the others is RPC_HOST, the Host of the Api
* or localhost: the address where the other services reach it
* @param services any
* @return error
**/
func (s *Api) MountRpc(services any) error {
	if s.Rpc <= 0 {
		return nil
	}

	host := cmp.Or(envar.GetStr("RPC_HOST", ""), s.Host, "localhost")
	if _, err := jrpc.Mount(host, s.Rpc, services, s.Name); err != nil {
		return err
	}

	if s.rpcStarted {
		return nil
	}

	if err := jrpc.Start(s.Rpc); err != nil {
		return err
	}
	s.rpcStarted = true
	return nil
}

/**
* DocRpc: The contract of an RPC method ("Struct.Method"), published by GET /rpcs; Query and Header do not apply
* @param method string, doc Doc
**/
func (s *Api) DocRpc(method string, doc Doc) {
	jrpc.SetDoc(method, doc.ToJson())
}

/**
* ListRpcs: The RPC methods mounted by the service (GET /rpcs): {packageName, method, host, port, inputs, output, doc}
* @param r *http.Request, body et.Json
* @return et.Items, error
**/
func (s *Api) ListRpcs(r *http.Request, body et.Json) (et.Items, error) {
	items := et.Items{}
	items.Add(jrpc.Methods()...)
	return items, nil
}

/**
* Public: A route without token; it is published to the endpoint catalog with its name and, if given, its contract.
* @param group, method, path, name string, handler http.HandlerFunc, docs ...Doc
**/
func (s *Api) Public(group, method, path, name string, handler http.HandlerFunc, docs ...Doc) {
	path = s.getPath(path)
	doc := docWithParams(docOf(docs), path)
	s.registerRoute(Endpoint{PackageName: s.Name, Group: group, Method: method, Path: path, Name: name, Access: AccessPublic, Doc: doc})
	middlewares := make([]func(http.Handler) http.Handler, 0)
	middlewares = append(middlewares, s.rateLimitMiddleware)
	if s.idempotency != nil {
		middlewares = append(middlewares, s.idempotency...)
	}
	middlewares = withValidate(middlewares, doc)
	With(s.Router, Route{
		Method:      method,
		Path:        path,
		Handler:     handler,
		Host:        s.Addr,
		PackageName: s.Name,
	}, middlewares)
}

/**
* Session: A route with token that does not check the role; it is published to the endpoint catalog with its name and,
* if given, its contract.
* @param group, method, path, name string, handler http.HandlerFunc, docs ...Doc
**/
func (s *Api) Session(group, method, path, name string, handler http.HandlerFunc, docs ...Doc) {
	path = s.getPath(path)
	doc := docWithParams(docOf(docs), path)
	s.registerRoute(Endpoint{PackageName: s.Name, Group: group, Method: method, Path: path, Name: name, Access: AccessSession, Doc: doc})
	middlewares := make([]func(http.Handler) http.Handler, 0)
	middlewares = append(middlewares, s.rateLimitMiddleware)
	if s.authentication != nil {
		middlewares = append(middlewares, s.authentication...)
	}
	if s.idempotency != nil {
		middlewares = append(middlewares, s.idempotency...)
	}
	middlewares = withValidate(middlewares, doc)
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
* Protected: A route with token and the role's authorization, published to the endpoint catalog in its group with, if
* given, its contract.
* @param group, method, path, name string, handler http.HandlerFunc, docs ...Doc
**/
func (s *Api) Protected(group, method, path, name string, handler http.HandlerFunc, docs ...Doc) {
	path = s.getPath(path)
	doc := docWithParams(docOf(docs), path)
	s.registerRoute(Endpoint{PackageName: s.Name, Group: group, Method: method, Path: path, Name: name, Access: AccessProtected, Doc: doc})
	middlewares := make([]func(http.Handler) http.Handler, 0)
	middlewares = append(middlewares, s.rateLimitMiddleware)
	if s.authentication != nil {
		middlewares = append(middlewares, s.authentication...)
	}
	if s.idempotency != nil {
		middlewares = append(middlewares, s.idempotency...)
	}
	if s.authorization != nil {
		middlewares = append(middlewares, s.authorization(method, path))
	}
	middlewares = withValidate(middlewares, doc)
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
* withValidate: The middlewares plus the validation of the query and the body when the doc asks for it (Validate)
* @param middlewares []func(http.Handler) http.Handler, doc *Doc
* @return []func(http.Handler) http.Handler
**/
func withValidate(middlewares []func(http.Handler) http.Handler, doc *Doc) []func(http.Handler) http.Handler {
	if doc == nil || !doc.Validate {
		return middlewares
	}

	return append(middlewares, validateMiddleware(doc))
}

/**
* ReadBody
* @param r *http.Request
* @return et.Json, error
**/
func (s *Api) readBody(r *http.Request) (et.Json, error) {
	// Sin cuerpo, un objeto vacío; un multipart/form-data queda sin leer para el handler (request.GetFile)
	if r.Body == nil || r.ContentLength == 0 || request.IsMultipart(r) {
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

// UseIdenpotency: Deprecated: use UseIdempotency.
func (s *Api) UseIdenpotency(middlewares ...func(http.Handler) http.Handler) {
	s.UseIdempotency(middlewares...)
}
