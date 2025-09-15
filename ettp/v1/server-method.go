package ettp

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/router"
	"github.com/cgalvisleon/et/strs"
)

/**
* setApiFunc
* @param method, path string, handlerFn http.HandlerFunc, packageName string
* @return *Router
**/
func (s *Server) setApiFunc(method, path string, handlerFn http.HandlerFunc, packageName string) *Router {
	method = strs.Uppcase(method)
	ok := methodMap[method]
	if !ok {
		console.Alertf(`'%s' http method is not supported.`, method)
		return nil
	}

	id := reg.GenKey(method, path)
	url := fmt.Sprintf("%s%s", s.pathApi, path)
	url = strings.ReplaceAll(url, "//", "/")
	route, err := s.setRouter(id, method, url, url, TpHandler, et.Json{}, router.TpReplaceHeader, []string{}, false, packageName, false)
	if err != nil {
		console.Alertf(err.Error())
		return nil
	}

	if route != nil {
		s.handlers[route.Id] = NewApiFunc(id, method, path, handlerFn, packageName)
	}

	return route
}

/**
* Private
* @return *Router
**/
func (s *Server) Private() *Router {
	if s.authenticator == nil {
		return &Router{
			server:      s,
			middlewares: s.middlewares,
		}
	}

	return s.With(s.authenticator)
}

/**
* Connect
* @param path string, handlerFn http.HandlerFunc, packageName string
**/
func (s *Server) Connect(path string, handlerFn http.HandlerFunc, packageName string) {
	s.setApiFunc(CONNECT, path, handlerFn, packageName)
}

/**
* Delete
* @param path string, handlerFn http.HandlerFunc, packageName string
**/
func (s *Server) Delete(path string, handlerFn http.HandlerFunc, packageName string) {
	s.setApiFunc(DELETE, path, handlerFn, packageName)
}

/**
* Get
* @param path string, handlerFn http.HandlerFunc, packageName string
**/
func (s *Server) Get(path string, handlerFn http.HandlerFunc, packageName string) {
	s.setApiFunc(GET, path, handlerFn, packageName)
}

/**
* Head
* @param path string, handlerFn http.HandlerFunc, packageName string
**/
func (s *Server) Head(path string, handlerFn http.HandlerFunc, packageName string) {
	s.setApiFunc(HEAD, path, handlerFn, packageName)
}

/**
* Options
* @param path string, handlerFn http.HandlerFunc, packageName string
**/
func (s *Server) Options(path string, handlerFn http.HandlerFunc, packageName string) {
	s.setApiFunc(OPTIONS, path, handlerFn, packageName)
}

/**
* Patch
* @param path string, handlerFn http.HandlerFunc, packageName string
**/
func (s *Server) Patch(path string, handlerFn http.HandlerFunc, packageName string) {
	s.setApiFunc(PATCH, path, handlerFn, packageName)
}

/**
* Post
* @param path string, handlerFn http.HandlerFunc, packageName string
**/
func (s *Server) Post(path string, handlerFn http.HandlerFunc, packageName string) {
	s.setApiFunc(POST, path, handlerFn, packageName)
}

/**
* Put
* @param path string, handlerFn http.HandlerFunc, packageName string
**/
func (s *Server) Put(path string, handlerFn http.HandlerFunc, packageName string) {
	s.setApiFunc(PUT, path, handlerFn, packageName)
}

/**
* Trace
* @param path string, handlerFn http.HandlerFunc, packageName string
**/
func (s *Server) Trace(path string, handlerFn http.HandlerFunc, packageName string) {
	s.setApiFunc(TRACE, path, handlerFn, packageName)
}

/**
* PublicRoute
* @param method, path, handlerFn, packageName string
**/
func (s *Server) PublicRoute(method, path string, h http.HandlerFunc, packageName string) {
	switch method {
	case "GET":
		s.Get(path, h, packageName)
	case "POST":
		s.Post(path, h, packageName)
	case "PUT":
		s.Put(path, h, packageName)
	case "PATCH":
		s.Patch(path, h, packageName)
	case "DELETE":
		s.Delete(path, h, packageName)
	case "HEAD":
		s.Head(path, h, packageName)
	case "OPTIONS":
		s.Options(path, h, packageName)
	}
}

/**
* PrivateRoute
* @param method, path, handlerFn, packageName string
**/
func (s *Server) PrivateRoute(method, path string, h http.HandlerFunc, packageName string) {
	router := s.Private()
	switch method {
	case "GET":
		router.Get(path, h, packageName)
	case "POST":
		router.Post(path, h, packageName)
	case "PUT":
		router.Put(path, h, packageName)
	case "PATCH":
		router.Patch(path, h, packageName)
	case "DELETE":
		router.Delete(path, h, packageName)
	case "HEAD":
		router.Head(path, h, packageName)
	case "OPTIONS":
		router.Options(path, h, packageName)
	}
}

/**
* ProtectRoute
* @param method, path, handlerFn, packageName string
**/
func (s *Server) ProtectRoute(method, path string, h http.HandlerFunc, packageName string) {
	s.PrivateRoute(method, path, h, packageName)
}
