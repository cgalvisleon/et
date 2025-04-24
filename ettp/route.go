package ettp

import (
	"fmt"
	"net/http"
	"regexp"
	"slices"

	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/router"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
)

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

type Route struct {
	server        *Server
	middlewares   []func(http.Handler) http.Handler
	pkg           *Package
	Id            string
	Tag           string
	TpParams      TpParams
	Kind          TypeApi
	Method        string
	Resolve       string
	Path          string
	Header        et.Json
	TpHeader      router.TpHeader
	ExcludeHeader []string
	Private       bool
	PackageName   string
	Routes        []*Route
}

/**
* newRoute
* @param server *Server, method string, routes []*Route, packageName s
* @return *Route, []*Route
**/
func newRoute(server *Server, method string, routes []*Route, packageName string) (*Route, []*Route) {
	pkg := findPakage(server, packageName)

	result := &Route{
		server:        server,
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
		Routes:        []*Route{},
	}

	routes = append(routes, result)

	return result, routes
}

/**
* indexRoute
* @param tag string
* @param routes []*Route
* @return int
**/
func indexRoute(tag string, routes []*Route) int {
	return slices.IndexFunc(routes, func(e *Route) bool { return strs.Lowcase(e.Tag) == strs.Lowcase(tag) })
}

/**
* GetRoute
* @param tag string
* @param routes []*Route
* @return []*Route
**/
func GetRoute(id string, routes []*Route) (*Route, error) {
	idx := slices.IndexFunc(routes, func(e *Route) bool { return e.Id == id })
	if idx == -1 {
		return nil, console.Alertm(MSG_ROUTE_NOT_FOUND)
	}

	return routes[idx], nil
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
* isHasParams
* @return bool
**/
func (r *Route) getParamsRoute(n int) (*Route, int) {
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
* String
* @param tag string
* @return *Route
**/
func (r *Route) find(tag string) *Route {
	idx := indexRoute(tag, r.Routes)
	if idx == -1 {
		return nil
	}

	return r.Routes[idx]
}

/**
* newRoute
* @param id string
* @param method string
* @param tag string
* @param resolve string
* @param kind TypeApi
* @param header et.Json
* @param tpHeader router.TpHeader
* @param excludeHeader map[string]bool
* @param private bool
* @param tpParams TpParams
* @param packageName string
* @return *Route
**/
func (r *Route) newRoute(id, method, tag string, kind TypeApi, header et.Json, tpHeader router.TpHeader, excludeHeader []string, private bool, tpParams TpParams) *Route {
	result := &Route{
		Id:            utility.GenKey(id),
		server:        r.server,
		middlewares:   r.server.middlewares,
		Tag:           tag,
		TpParams:      tpParams,
		Kind:          kind,
		Method:        method,
		Resolve:       "",
		Header:        header,
		TpHeader:      tpHeader,
		ExcludeHeader: excludeHeader,
		Private:       private,
		Routes:        []*Route{},
	}

	if result.Private {
		result.addMiddleware(r.server.authenticator)
	}

	r.Routes = append(r.Routes, result)

	return result
}

/**
* addMiddleware
* @param middleware func(http.HandlerFunc) http.HandlerFunc
* @return *Route
**/
func (r *Route) addMiddleware(middleware func(http.Handler) http.Handler) *Route {
	isAdded := false
	for _, mw := range r.middlewares {
		if fmt.Sprintf("%p", mw) == fmt.Sprintf("%p", middleware) {
			isAdded = true
			break
		}
	}
	if !isAdded {
		r.middlewares = append(r.middlewares, middleware)
	}

	return r
}

/**
* removeMiddleware
* @param middleware func(http.HandlerFunc) http.HandlerFunc
* @return *Route
**/
func (r *Route) removeMiddleware(middleware func(http.Handler) http.Handler) *Route {
	var middlewares []func(http.Handler) http.Handler
	for _, mw := range r.middlewares {
		if fmt.Sprintf("%p", mw) != fmt.Sprintf("%p", middleware) {
			middlewares = append(middlewares, mw)
		}
	}
	r.middlewares = middlewares

	return r
}

/**
* deleteById
* @param id string
* @return bool
**/
func (r *Route) deleteById(id string, save bool) bool {
	result := false
	for i, route := range r.Routes {
		if route.Id == id {
			r.Routes = append(r.Routes[:i], r.Routes[i+1:]...)
			result = true
			break
		} else if route.deleteById(id, save) {
			result = true
			break
		}
	}

	if result && save {
		go r.server.Save()
	}

	return result
}

/**
* addPakageRoute
* @param packageName string
* @param r *Route
* @return *Package
**/
func (r *Route) SetPakage(packageName string) *Route {
	if len(packageName) == 0 {
		return r
	}

	pkg := findPakage(r.server, packageName)
	if pkg == nil {
		pkg = newPakage(r.server, packageName)
		r.PackageName = packageName
		r.pkg = pkg
		pkg.AddRoute(r.Method, r.Path, r)

		return r
	}

	if r.PackageName != packageName {
		old := findPakage(r.server, r.PackageName)
		if old != nil {
			old.DeleteRoute(r.Method, r.Path)
		}
	}

	r.PackageName = packageName
	r.pkg = pkg
	pkg.AddRoute(r.Method, r.Path, r)

	return r
}

/**
* ToJson
* @return et.Json
**/
func (r *Route) ToJson() et.Json {
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
* ToString
* @return string
**/
func (r *Route) ToString() string {
	result := r.ToJson()
	return result.ToString()
}

/**
* With
* @param middlewares ...func(http.HandlerFunc) http.HandlerFunc
**/
func (r *Route) With(middlewares ...func(http.Handler) http.Handler) *Route {
	result := r.server.NewRoute()
	result.middlewares = append(result.middlewares, r.middlewares...)
	return result
}

/**
* Connect
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (r *Route) Connect(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.setHandlerFunc(CONNECT, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}

/**
* Delete
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (r *Route) Delete(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.setHandlerFunc(DELETE, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}

/**
* Get
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (r *Route) Get(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.setHandlerFunc(GET, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}

/**
* Head
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (r *Route) Head(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.setHandlerFunc(HEAD, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}

/**
* Options
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (r *Route) Options(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.setHandlerFunc(OPTIONS, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}

/**
* Patch
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (r *Route) Patch(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.setHandlerFunc(PATCH, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}

/**
* Post
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (r *Route) Post(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.setHandlerFunc(POST, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}

/**
* Put
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (r *Route) Put(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.setHandlerFunc(PUT, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}

/**
* Trace
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (r *Route) Trace(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.setHandlerFunc(TRACE, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}
