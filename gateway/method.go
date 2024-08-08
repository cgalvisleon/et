package gateway

import (
	"net/http"
	"strings"

	"github.com/cgalvisleon/et/js"
	"github.com/cgalvisleon/et/logs"
)

/**
* MethodFunc
* @param method string
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (s *HttpServer) MethodFunc(method, path string, handlerFn http.HandlerFunc, packageName string) *Route {
	isWS := method == WS
	method = GET
	method = strings.ToUpper(method)
	ok := methodMap[method]
	if !ok {
		logs.Panicf(`'%s' http method is not supported.`, method)
	}

	route := findRoute(method, s.routes)
	if route == nil {
		route, s.routes = newRoute(method, s, s.routes)
	}

	tags := strings.Split(path, "/")
	for _, tag := range tags {
		if len(tag) > 0 {
			find := findRoute(tag, route.Routes)
			if find == nil {
				route, route.Routes = newRoute(tag, route.server, route.Routes)
			} else {
				route = find
			}
		}
	}

	if route != nil {
		route.IsWs = isWS
		route.Resolve = js.Json{
			"method":  method,
			"kind":    "HANDLER",
			"resolve": "/",
		}
		s.handlers[route.Id] = handlerFn

		pakage := s.findPakage(packageName)
		if pakage == nil {
			pakage = s.newPakage(packageName)
		}
		pakage.Routes = append(pakage.Routes, route)
		pakage.Count = len(pakage.Routes)
	}

	return route
}

/**
* Use
* @param middlewares ...func(http.HandlerFunc) http.HandlerFunc
**/
func (s *HttpServer) Use(middlewares ...func(http.Handler) http.Handler) {
	s.middlewares = append(s.middlewares, middlewares...)
}

/**
* With
* @param middlewares ...func(http.HandlerFunc) http.HandlerFunc
**/
func (s *HttpServer) With(middlewares ...func(http.Handler) http.Handler) *Route {
	result := &Route{
		middlewares: make([]func(http.Handler) http.Handler, 0),
	}

	result.middlewares = append(result.middlewares, middlewares...)

	return result
}

/**
* Connect
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (s *HttpServer) Connect(path string, handlerFn http.HandlerFunc, packageName string) {
	s.MethodFunc(CONNECT, path, handlerFn, packageName)
}

/**
* Delete
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (s *HttpServer) Delete(path string, handlerFn http.HandlerFunc, packageName string) {
	s.MethodFunc(DELETE, path, handlerFn, packageName)
}

/**
* Get
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (s *HttpServer) Get(path string, handlerFn http.HandlerFunc, packageName string) {
	s.MethodFunc(GET, path, handlerFn, packageName)
}

/**
* Head
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (s *HttpServer) Head(path string, handlerFn http.HandlerFunc, packageName string) {
	s.MethodFunc(HEAD, path, handlerFn, packageName)
}

/**
* Options
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (s *HttpServer) Options(path string, handlerFn http.HandlerFunc, packageName string) {
	s.MethodFunc(OPTIONS, path, handlerFn, packageName)
}

/**
* Patch
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (s *HttpServer) Patch(path string, handlerFn http.HandlerFunc, packageName string) {
	s.MethodFunc(PATCH, path, handlerFn, packageName)
}

/**
* Post
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (s *HttpServer) Post(path string, handlerFn http.HandlerFunc, packageName string) {
	s.MethodFunc(POST, path, handlerFn, packageName)
}

/**
* Put
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (s *HttpServer) Put(path string, handlerFn http.HandlerFunc, packageName string) {
	s.MethodFunc(PUT, path, handlerFn, packageName)
}

/**
* Trace
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (s *HttpServer) Trace(path string, handlerFn http.HandlerFunc, packageName string) {
	s.MethodFunc(TRACE, path, handlerFn, packageName)
}

/**
* Ws
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (s *HttpServer) Ws(path string, handlerFn http.HandlerFunc, packageName string) {
	s.MethodFunc(WS, path, handlerFn, packageName)
}
