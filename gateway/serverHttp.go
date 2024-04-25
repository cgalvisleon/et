package gateway

import (
	"net/http"

	"github.com/cgalvisleon/elvis/envar"
	"github.com/cgalvisleon/elvis/strs"
	"github.com/rs/cors"
)

const (
	// Types
	HANDLER   = "HANDLER"
	HTTP      = "HTTP"
	REST      = "REST"
	WEBSOCKET = "WEBSOCKET"
	// Methods
	CONNECT = "CONNECT"
	DELETE  = "DELETE"
	GET     = "GET"
	HEAD    = "HEAD"
	OPTIONS = "OPTIONS"
	PATCH   = "PATCH"
	POST    = "POST"
	PUT     = "PUT"
	TRACE   = "TRACE"
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
	routes          *Nodes
	pakages         *Pakages
	handlers        Handlers
	routesKey       string
	pakagesKey      string
}

func newHttpServer() *HttpServer {
	// Create a new server
	mux := http.NewServeMux()

	port := envar.EnvarInt(3300, "PORT")
	result := &HttpServer{
		addr:       strs.Format(":%d", port),
		handler:    cors.AllowAll().Handler(mux),
		mux:        mux,
		routes:     newRouters(),
		pakages:    newPakages(),
		handlers:   newHandlers(),
		routesKey:  "gateway/routes",
		pakagesKey: "gateway/packages",
	}
	result.notFoundHandler = notFounder
	result.handlerFn = handlerFn
	result.Get("/version", version, "Api Gateway")
	result.Get("/gateway/all", getAll, "Api Gateway")
	result.Post("/gateway", upsert, "Api Gateway")
	result.Get("/ws", wsConnect, "Api Gateway")

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
