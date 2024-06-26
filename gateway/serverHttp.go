package gateway

import (
	"net/http"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/strs"
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
	//
	PACKAGE_NAME = "Api Gateway"
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

	port := envar.GetInt(3300, "PORT")
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
	result.handlerFn = handlerRouter
	result.Get("/version", version, PACKAGE_NAME)
	result.Get("/gateway/all", getAll, PACKAGE_NAME)
	result.Post("/gateway", upSert, PACKAGE_NAME)
	result.Get("/ws", wsConnect, PACKAGE_NAME)

	// Handler router
	result.mux.HandleFunc("/", result.handlerFn)

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
