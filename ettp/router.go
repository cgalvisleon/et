package ettp

import (
	"net/http"
	"regexp"
	"slices"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/router"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
)

type TypeApi int

const (
	TpHandler TypeApi = iota
	TpApiRest
	TpProxy
	TpPortForward
)

func (t TypeApi) String() string {
	switch t {
	case TpHandler:
		return "Handler"
	case TpApiRest:
		return "Api REST"
	case TpProxy:
		return "Proxy"
	default:
		return "Unknown"
	}
}

func IntToTypeApi(i int) TypeApi {
	switch i {
	case 1:
		return TpApiRest
	case 2:
		return TpProxy
	default:
		return TpHandler
	}
}

const (
	CONNECT    = "CONNECT"
	DELETE     = "DELETE"
	GET        = "GET"
	HEAD       = "HEAD"
	OPTIONS    = "OPTIONS"
	PATCH      = "PATCH"
	POST       = "POST"
	PUT        = "PUT"
	TRACE      = "TRACE"
	ROUTER_KEY = "apigateway-router"
)

var methodMap = map[string]bool{
	CONNECT: true,
	DELETE:  true,
	GET:     true,
	HEAD:    true,
	OPTIONS: true,
	PATCH:   true,
	POST:    true,
	PUT:     true,
	TRACE:   true,
}

const QP = "?"

type TpParams int

const (
	TpPathParams TpParams = iota
	TpQueryParams
	TpMatrixParams
	TpNotParams
)

func (t TpParams) String() string {
	switch t {
	case TpPathParams:
		return "Path Params"
	case TpQueryParams:
		return "Query Params"
	case TpMatrixParams:
		return "Matrix Params"
	default:
		return "Not"
	}
}

type Router struct {
	server        *Server                           `json:"-"`
	middlewares   []func(http.Handler) http.Handler `json:"-"`
	pkg           *Package                          `json:"-"`
	main          *Router                           `json:"-"`
	Id            string                            `json:"id"`
	Tag           string                            `json:"tag"`
	TpParams      TpParams                          `json:"tp_params"`
	Kind          TypeApi                           `json:"kind"`
	Method        string                            `json:"method"`
	Resolve       string                            `json:"resolve"`
	Path          string                            `json:"path"`
	Header        et.Json                           `json:"header"`
	TpHeader      router.TpHeader                   `json:"tp_header"`
	ExcludeHeader []string                          `json:"exclude_header"`
	Private       bool                              `json:"private"`
	PackageName   string                            `json:"package_name"`
	Routes        []*Router                         `json:"routes"`
}

/**
* newMainRouter
* @param server *Server, method string, packageName string
* @return *Router
**/
func (s *Server) newRouter(method string, packageName string) *Router {
	pkg := getPackageByName(s, packageName)
	result := &Router{
		server:        s,
		middlewares:   make([]func(http.Handler) http.Handler, 0),
		pkg:           pkg,
		Id:            utility.UUID(),
		Tag:           method,
		TpParams:      TpNotParams,
		Kind:          TpHandler,
		Method:        method,
		Resolve:       "",
		Header:        et.Json{},
		TpHeader:      router.TpKeepHeader,
		ExcludeHeader: []string{},
		Private:       false,
		PackageName:   packageName,
		Routes:        []*Router{},
	}
	s.router = append(s.router, result)

	return result
}

/**
* getRouterIndexByTag
* @param tag string, routes []*Router
* @return int
**/
func getRouterIndexByTag(tag string, routes []*Router) int {
	return slices.IndexFunc(routes, func(e *Router) bool { return strs.Lowcase(e.Tag) == strs.Lowcase(tag) })
}

/**
* getRouterIndexById
* @param id string, routes []*Router
* @return int
**/
func getRouterIndexById(id string, routes []*Router) int {
	return slices.IndexFunc(routes, func(e *Router) bool { return e.Id == id })
}

/**
* getTpParams
* @param tag string
* @return TpParams
**/
func getTpParams(tag string) TpParams {
	/**
	* Path parameters
	* Example: /{id}
	**/
	regex := regexp.MustCompile(`^\{.*\}$`)
	if regex.MatchString(tag) {
		return TpPathParams
	}

	/**
	* Query parameters
	* Example: /users?name=5
	**/
	if strs.Contains(tag, QP) {
		return TpQueryParams
	}
	/**
	* Matrix parameters
	* Example: /users;name=5
	**/
	if strs.Contains(tag, ";") {
		return TpMatrixParams
	}

	return TpNotParams
}

/**
* getParams
* @param n int
* @return *Router, int
**/
func (r *Router) getParams(n int) (*Router, int) {
	if len(r.Routes) == 0 {
		return nil, -1
	}

	for i := n; i < len(r.Routes); i++ {
		route := r.Routes[i]
		if route.TpParams != TpNotParams {
			return route, i
		}
	}

	return nil, -1
}

/**
* getRouterByTag
* @param tag string
* @return *Router
**/
func (r *Router) getRouterByTag(tag string) *Router {
	idx := getRouterIndexByTag(tag, r.Routes)
	if idx == -1 {
		return nil
	}

	return r.Routes[idx]
}

/**
* getRouterIndexByTag
* @param id string, routes []*Router
* @return int
**/
func (r *Router) getRouterById(id string) *Router {
	idx := getRouterIndexById(id, r.Routes)
	if idx != -1 {
		return r.Routes[idx]
	}

	if r.Id == id {
		return nil
	}

	for _, route := range r.Routes {
		find := route.getRouterById(id)
		if find != nil {
			return find
		}
	}

	return nil
}

/**
* addRoute
* @param id, method, tag string, kind TypeApi, header et.Json, tpHeader router.TpHeader, excludeHeader []string, private bool, tpParams TpParams
* @return *Router
**/
func (r *Router) addRoute(id, method, tag string, kind TypeApi, header et.Json, tpHeader router.TpHeader, excludeHeader []string, private bool, tpParams TpParams) *Router {
	result := &Router{
		server:        r.server,
		middlewares:   r.server.middlewares,
		main:          r,
		Id:            utility.GenKey(id),
		Tag:           tag,
		TpParams:      tpParams,
		Kind:          kind,
		Method:        method,
		Resolve:       "",
		Header:        header,
		TpHeader:      tpHeader,
		ExcludeHeader: excludeHeader,
		Private:       private,
		Routes:        []*Router{},
	}
	r.Routes = append(r.Routes, result)

	return result
}

/**
* setPakage
* @param packageName string
* @param r *Router
* @return *Package
**/
func (r *Router) setPakage(packageName string) *Router {
	if len(packageName) == 0 {
		return r
	}

	if r.PackageName != packageName {
		old := getPackageByName(r.server, r.PackageName)
		if old != nil {
			old.deleteRoute(r)
		}
	}

	pkg := getPackageByName(r.server, packageName)
	if pkg == nil {
		pkg = newPakage(r.server, packageName)
	}

	r.PackageName = packageName
	r.pkg = pkg
	pkg.addRouter(r)

	return r
}

/**
* ToJson
* @return et.Json
**/
func (r *Router) ToJson() et.Json {
	var n int
	routes := make([]et.Json, 0)
	for _, route := range r.Routes {
		n++
		routes = append(routes, route.ToJson())
	}
	exlude := make([]string, 0)
	if r.ExcludeHeader != nil {
		exlude = r.ExcludeHeader
	}

	return et.Json{
		"Id":       r.Id,
		"Method":   r.Method,
		"Tag":      r.Tag,
		"TpParams": r.TpParams.String(),
		"Kind":     r.Kind.String(),
		"Resolve":  r.Resolve,
		"Header":   r.Header,
		"TpHeader": r.TpHeader.String(),
		"Exclude":  exlude,
		"Private":  r.Private,
		"Pakage":   r.PackageName,
		"Routes":   routes,
		"Count":    n,
	}
}

/**
* With
* @param middlewares ...func(http.HandlerFunc) http.HandlerFunc
* @return *Router
**/
func (r *Router) With(middlewares ...func(http.Handler) http.Handler) *Router {
	result := &Router{
		server:      r.server,
		middlewares: r.server.middlewares,
	}

	result.middlewares = append(result.middlewares, r.middlewares...)
	return result
}

/**
* Connect
* @param path string, handlerFn http.HandlerFunc, packageName string
**/
func (r *Router) Connect(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.setApiFunc(CONNECT, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}

/**
* Delete
* @param path string, handlerFn http.HandlerFunc, packageName string
**/
func (r *Router) Delete(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.setApiFunc(DELETE, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}

/**
* Get
* @param path string, handlerFn http.HandlerFunc, packageName string
**/
func (r *Router) Get(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.setApiFunc(GET, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}

/**
* Head
* @param path string, handlerFn http.HandlerFunc, packageName string
**/
func (r *Router) Head(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.setApiFunc(HEAD, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}

/**
* Options
* @param path string, handlerFn http.HandlerFunc, packageName string
**/
func (r *Router) Options(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.setApiFunc(OPTIONS, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}

/**
* Patch
* @param path string, handlerFn http.HandlerFunc, packageName string
**/
func (r *Router) Patch(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.setApiFunc(PATCH, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}

/**
* Post
* @param path string, handlerFn http.HandlerFunc, packageName string
**/
func (r *Router) Post(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.setApiFunc(POST, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}

/**
* Put
* @param path string, handlerFn http.HandlerFunc, packageName string
**/
func (r *Router) Put(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.setApiFunc(PUT, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}

/**
* Trace
* @param path string, handlerFn http.HandlerFunc, packageName string
**/
func (r *Router) Trace(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.setApiFunc(TRACE, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}
