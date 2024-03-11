package apigateway

import (
	"net/http"
	"strings"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/rs/cors"
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

type HttpServer struct {
	addr            string
	handler         http.Handler
	mux             *http.ServeMux
	notFoundHandler http.HandlerFunc
	handlerFn       http.HandlerFunc
	handlerWS       http.HandlerFunc
	// middlewares     []func(http.Handler) http.Handler
}

func NewHttpServer() *HttpServer {
	// Create a new server
	mux := http.NewServeMux()

	port := envar.EnvarInt(3300, "PORT")
	result := &HttpServer{
		addr:    et.Format(":%d", port),
		handler: cors.AllowAll().Handler(mux),
		mux:     mux,
	}
	result.notFoundHandler = notFounder
	result.handlerFn = handlerFn

	// Handler router
	mux.HandleFunc("/", result.handlerFn)

	return result
}

// NotFound sets the handler function for the server.
func (s *HttpServer) NotFound(handlerFn http.HandlerFunc) {
	s.notFoundHandler = handlerFn
}

// Handler sets the handler function for the server.
func (s *HttpServer) Handler(handlerFn http.HandlerFunc) {
	s.handlerFn = handlerFn
}

// HandlerWebSocket sets the handler function for the server.
func (s *HttpServer) HandlerWebSocket(handlerFn http.HandlerFunc) {
	s.handlerWS = handlerFn
}

// MethodFunc adds the route `pattern` that matches `method` http method to
// execute the `handlerFn` http.HandlerFunc.
func (s *HttpServer) MethodFunc(method, pattern string, handlerFn http.HandlerFunc) {
	method = strings.ToUpper(method)
	ok := methodMap[method]
	if !ok {
		et.Panicf(`'%s' http method is not supported.`, method)
	}

	AddHandleMethod(method, pattern, handlerFn)
}

// Connect adds the route `pattern` that matches `CONNECT` http method to
func (s *HttpServer) Connect(pattern string, handlerFn http.HandlerFunc) {
	s.MethodFunc(CONNECT, pattern, handlerFn)
}

// Delete adds the route `pattern` that matches `DELETE` http method to
func (s *HttpServer) Delete(pattern string, handlerFn http.HandlerFunc) {
	s.MethodFunc(DELETE, pattern, handlerFn)
}

// Get adds the route `pattern` that matches `GET` http method to
func (s *HttpServer) Get(pattern string, handlerFn http.HandlerFunc) {
	s.MethodFunc(GET, pattern, handlerFn)
}

// Head adds the route `pattern` that matches `HEAD` http method to
func (s *HttpServer) Head(pattern string, handlerFn http.HandlerFunc) {
	s.MethodFunc(HEAD, pattern, handlerFn)
}

// Options adds the route `pattern` that matches `OPTIONS` http method to
func (s *HttpServer) Options(pattern string, handlerFn http.HandlerFunc) {
	s.MethodFunc(OPTIONS, pattern, handlerFn)
}

// Patch adds the route `pattern` that matches `PATCH` http method to
func (s *HttpServer) Patch(pattern string, handlerFn http.HandlerFunc) {
	s.MethodFunc(PATCH, pattern, handlerFn)
}

// Post adds the route `pattern` that matches `POST` http method to
func (s *HttpServer) Post(pattern string, handlerFn http.HandlerFunc) {
	s.MethodFunc(POST, pattern, handlerFn)
}

// Put adds the route `pattern` that matches `PUT` http method to
func (s *HttpServer) Put(pattern string, handlerFn http.HandlerFunc) {
	s.MethodFunc(PUT, pattern, handlerFn)
}

// Trace adds the route `pattern` that matches `TRACE` http method to
func (s *HttpServer) Trace(pattern string, handlerFn http.HandlerFunc) {
	s.MethodFunc(TRACE, pattern, handlerFn)
}

// With adds a middleware to the server.
/*
func (s *HttpServer) With(middlewares ...func(http.Handler) http.Handler) Router {
  reurn nil
}
*/
