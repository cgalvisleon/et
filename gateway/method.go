package gateway

import (
	"net/http"
	"strings"

	"github.com/cgalvisleon/elvis/console"
)

// MethodFunc adds the route `pattern` that matches `method` http method to
// execute the `handlerFn` http.HandlerFunc.
func (s *HttpServer) MethodFunc(method, pattern string, handlerFn http.HandlerFunc, packageName string) {
	method = strings.ToUpper(method)
	ok := methodMap[method]
	if !ok {
		console.PanicF(`'%s' http method is not supported.`, method)
	}

	s.AddHandleMethod(method, pattern, handlerFn, packageName)
}

// Connect adds the route `pattern` that matches `CONNECT` http method to
func (s *HttpServer) Connect(pattern string, handlerFn http.HandlerFunc, packageName string) {
	s.MethodFunc(CONNECT, pattern, handlerFn, packageName)
}

// Delete adds the route `pattern` that matches `DELETE` http method to
func (s *HttpServer) Delete(pattern string, handlerFn http.HandlerFunc, packageName string) {
	s.MethodFunc(DELETE, pattern, handlerFn, packageName)
}

// Get adds the route `pattern` that matches `GET` http method to
func (s *HttpServer) Get(pattern string, handlerFn http.HandlerFunc, packageName string) {
	s.MethodFunc(GET, pattern, handlerFn, packageName)
}

// Head adds the route `pattern` that matches `HEAD` http method to
func (s *HttpServer) Head(pattern string, handlerFn http.HandlerFunc, packageName string) {
	s.MethodFunc(HEAD, pattern, handlerFn, packageName)
}

// Options adds the route `pattern` that matches `OPTIONS` http method to
func (s *HttpServer) Options(pattern string, handlerFn http.HandlerFunc, packageName string) {
	s.MethodFunc(OPTIONS, pattern, handlerFn, packageName)
}

// Patch adds the route `pattern` that matches `PATCH` http method to
func (s *HttpServer) Patch(pattern string, handlerFn http.HandlerFunc, packageName string) {
	s.MethodFunc(PATCH, pattern, handlerFn, packageName)
}

// Post adds the route `pattern` that matches `POST` http method to
func (s *HttpServer) Post(pattern string, handlerFn http.HandlerFunc, packageName string) {
	s.MethodFunc(POST, pattern, handlerFn, packageName)
}

// Put adds the route `pattern` that matches `PUT` http method to
func (s *HttpServer) Put(pattern string, handlerFn http.HandlerFunc, packageName string) {
	s.MethodFunc(PUT, pattern, handlerFn, packageName)
}

// Trace adds the route `pattern` that matches `TRACE` http method to
func (s *HttpServer) Trace(pattern string, handlerFn http.HandlerFunc, packageName string) {
	s.MethodFunc(TRACE, pattern, handlerFn, packageName)
}
