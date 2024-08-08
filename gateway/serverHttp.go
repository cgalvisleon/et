package gateway

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/js"
	"github.com/cgalvisleon/et/logs"
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
	WS      = "WS"
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
	routes          []*Route
	pakages         []*Pakage
	handlers        map[string]http.HandlerFunc
	middlewares     []func(http.Handler) http.Handler
	routesKey       string
	pakagesKey      string
}

func newHttpServer() *HttpServer {
	// Create a new server
	mux := http.NewServeMux()

	port := envar.GetInt(3300, "PORT")
	result := &HttpServer{
		addr:            strs.Format(":%d", port),
		handler:         cors.AllowAll().Handler(mux),
		mux:             mux,
		notFoundHandler: notFounder,
		handlerFn:       handlerExec,
		handlerWS:       wsConnect,
		routes:          []*Route{},
		pakages:         []*Pakage{},
		handlers:        make(map[string]http.HandlerFunc),
		middlewares:     make([]func(http.Handler) http.Handler, 0),
		routesKey:       "gateway/routes",
		pakagesKey:      "gateway/packages",
	}
	result.mux.HandleFunc("/", result.handlerFn)
	basicRouter(result)

	return result
}

/**
* NotFound
* @param handlerFn http.HandlerFunc
**/
func (s *HttpServer) NotFound(handlerFn http.HandlerFunc) {
	s.notFoundHandler = handlerFn
}

/**
* Handler
* @param handlerFn http.HandlerFunc
**/
func (s *HttpServer) Handler(handlerFn http.HandlerFunc) {
	s.handlerFn = handlerFn
}

/**
* HandlerWebSocket
* @param handlerFn http.HandlerFunc
**/
func (s *HttpServer) HandlerWebSocket(handlerFn http.HandlerFunc) {
	s.handlerWS = handlerFn
}

/**
* LoadRouter
* @return error
**/
func (s *HttpServer) Load() error {

	return nil
}

/**
* Save
* @return error
**/
func (s *HttpServer) Save() error {
	routesData, err := json.Marshal(s.routes)
	if err != nil {
		return err
	}

	pakagesData, err := json.Marshal(s.pakages)
	if err != nil {
		return err
	}

	logs.Debug(string(routesData))
	logs.Debug(string(pakagesData))
	return nil
}

/**
* findPakage
* @param name string
* @return *Pakage
**/
func (s *HttpServer) findPakage(name string) *Pakage {
	for _, pakage := range s.pakages {
		if pakage.Name == name {
			return pakage
		}
	}

	return nil
}

/**
* newPakage
* @param name string
* @return *Pakage
**/
func (s *HttpServer) newPakage(name string) *Pakage {
	pakage := &Pakage{
		Name:   name,
		Routes: []*Route{},
	}

	s.pakages = append(s.pakages, pakage)

	return pakage
}

/**
* GetResolve
* @param method string
* @param path string
* @return *Resolve
**/
func (s *HttpServer) GetResolve(method, path string) *Resolve {
	route := findRoute(method, s.routes)
	if route == nil {
		return nil
	}

	var result *Resolve
	tags := strings.Split(path, "/")
	for _, tag := range tags {
		if len(tag) > 0 {
			route, result = findResolve(tag, route.Routes, result)
			if route == nil {
				return nil
			}
		}
	}

	if result != nil {
		result.Resolve = route.Resolve.Str("resolve")
		for _, param := range result.Params {
			for key, value := range param {
				result.Resolve = strings.Replace(result.Resolve, key, "%v", -1)
				result.Resolve = strs.Format(result.Resolve, value)
			}
		}
	}

	return result
}

/**
* AddRoute
* @param method string
* @param path string
* @param resolve string
* @param kind string
* @param stage string
* @param packageName string
 */
func (s *HttpServer) AddRoute(method, path, resolve, kind, stage, packageName string) {
	isWs := method == WS
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
				route, route.Routes = newRoute(tag, s, route.Routes)
			} else {
				route = find
			}
		}
	}

	if route != nil {
		route.IsWs = isWs
		route.Resolve = js.Json{
			"method":  method,
			"kind":    kind,
			"stage":   stage,
			"resolve": resolve,
		}

		pakage := s.findPakage(packageName)
		if pakage == nil {
			pakage = s.newPakage(packageName)
		}
		pakage.Routes = append(pakage.Routes, route)
		pakage.Count = len(pakage.Routes)
	}
}
