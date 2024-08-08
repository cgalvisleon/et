package gateway

import (
	"net/http"
	"regexp"

	"github.com/cgalvisleon/et/js"
	"github.com/cgalvisleon/et/utility"
)

type Route struct {
	Id          string
	middlewares []func(http.Handler) http.Handler
	server      *HttpServer
	IsWs        bool
	Tag         string
	Resolve     js.Json
	Routes      []*Route
}

/**
* ToJson
* @return js.Json
**/
func (r *Route) ToJson() js.Json {
	routes := make([]js.Json, 0)
	for _, route := range r.Routes {
		routes = append(routes, route.ToJson())
	}

	return js.Json{
		"id":      r.Id,
		"isWs":    r.IsWs,
		"tag":     r.Tag,
		"resolve": r.Resolve,
		"routes":  routes,
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

type Pakage struct {
	Name   string
	Routes []*Route
	Count  int
}

type Resolve struct {
	Route   *Route
	Params  []js.Json
	Resolve string
}

/**
* newRoute
* @param tag string
* @param routes []*Route
* @return *Route, []*Route
**/
func newRoute(tag string, server *HttpServer, routes []*Route) (*Route, []*Route) {
	result := &Route{
		Id:          utility.UUID(),
		middlewares: make([]func(http.Handler) http.Handler, 0),
		server:      server,
		Tag:         tag,
		Resolve:     js.Json{},
		Routes:      []*Route{},
	}

	routes = append(routes, result)

	return result, routes
}

/**
* findRoute
* @param tag string
* @param routes *Routes
**/
func findRoute(tag string, routes []*Route) *Route {
	for _, route := range routes {
		if route.Tag == tag {
			return route
		}
	}

	return nil
}

/**
* findResolve
* @param tag string
* @param routes *Routes
* @param route *Resolve
**/
func findResolve(tag string, routes []*Route, route *Resolve) (*Route, *Resolve) {
	node := findRoute(tag, routes)
	if node == nil {
		// Define regular expression
		regex := regexp.MustCompile(`^\{.*\}$`)
		// Find node by regular expression
		for _, n := range routes {
			if regex.MatchString(n.Tag) {
				if route == nil {
					route = &Resolve{
						Params: []js.Json{},
					}
				}
				route.Route = n
				route.Params = append(route.Params, js.Json{n.Tag: tag})
				return n, route
			}
		}
	} else if route == nil {
		route = &Resolve{
			Route:  node,
			Params: []js.Json{},
		}
	} else {
		route.Route = node
	}

	return node, route
}

/**
* basicRouter
* @param server *HttpServer
**/
func basicRouter(server *HttpServer) {
	server.Get("/version", version, "Api Gateway")
	server.Get("/gateway/all", getAll, "Api Gateway")
	server.Post("/gateway", upsert, "Api Gateway")
	server.Ws("/ws", server.handlerWS, "Api Gateway")
}

/**
* Connect
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (r *Route) Connect(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.MethodFunc(CONNECT, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}

/**
* Delete
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (r *Route) Delete(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.MethodFunc(DELETE, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}

/**
* Get
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (r *Route) Get(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.MethodFunc(GET, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}

/**
* Head
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (r *Route) Head(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.MethodFunc(HEAD, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}

/**
* Options
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (r *Route) Options(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.MethodFunc(OPTIONS, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}

/**
* Patch
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (r *Route) Patch(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.MethodFunc(PATCH, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}

/**
* Post
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (r *Route) Post(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.MethodFunc(POST, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}

/**
* Put
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (r *Route) Put(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.MethodFunc(PUT, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}

/**
* Trace
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (r *Route) Trace(path string, handlerFn http.HandlerFunc, packageName string) {
	result := r.server.MethodFunc(TRACE, path, handlerFn, packageName)
	result.middlewares = append(result.middlewares, r.middlewares...)
}
