package ettp

import (
	"fmt"
	"net/http"
	"regexp"
	"slices"

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

type Router struct {
	server        *Server                           `json:"-"`
	middlewares   []func(http.Handler) http.Handler `json:"-"`
	pkg           *Package                          `json:"-"`
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
* newRoute
* @param server *Server, method string, routes []*Router, packageName s
* @return *Router, []*Router
**/
func newRoute(server *Server, method string, routes []*Router, packageName string) (*Router, []*Router) {
	pkg := getPackageByName(server, packageName)

	result := &Router{
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
		Routes:        []*Router{},
	}

	routes = append(routes, result)

	return result, routes
}

/**
* getRouteIndex
* @param tag string, routes []*Router
* @return int
**/
func getRouteIndex(tag string, routes []*Router) int {
	return slices.IndexFunc(routes, func(e *Router) bool { return strs.Lowcase(e.Tag) == strs.Lowcase(tag) })
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
* key
* @return string
**/
func (r *Router) key() string {
	return strs.Format(`[%s]:%s`, r.Method, r.Path)
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
* find
* @param tag string
* @return *Router
**/
func (r *Router) find(tag string) *Router {
	idx := getRouteIndex(tag, r.Routes)
	if idx == -1 {
		return nil
	}

	return r.Routes[idx]
}

/**
* addRoute
* @param id, method, tag string, kind TypeApi, header et.Json, tpHeader router.TpHeader, excludeHeader []string, private bool, tpParams TpParams
* @return *Router
**/
func (r *Router) addRoute(id, method, tag string, kind TypeApi, header et.Json, tpHeader router.TpHeader, excludeHeader []string, private bool, tpParams TpParams) *Router {
	result := &Router{
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
		Routes:        []*Router{},
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
* @return *Router
**/
func (r *Router) addMiddleware(middleware func(http.Handler) http.Handler) *Router {
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
* @return *Router
**/
func (r *Router) removeMiddleware(middleware func(http.Handler) http.Handler) *Router {
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
func (r *Router) deleteById(id string, save bool) bool {
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
* @param r *Router
* @return *Package
**/
func (r *Router) SetPakage(packageName string) *Router {
	if len(packageName) == 0 {
		return r
	}

	if r.PackageName == packageName {
		return r
	}

	old := getPackageByName(r.server, r.PackageName)
	if old != nil {
		old.deleteRoute(r)
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
	result := r.server.NewRoute()
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
