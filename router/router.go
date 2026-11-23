package router

import (
	"fmt"
	"maps"
	"net/http"
	"strings"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/go-chi/chi/v5"
)

type Router interface {
	UseAuthentication(fn func(http.Handler) http.Handler)
	Protect(method, path string, handler func(http.ResponseWriter, *http.Request))
	Public(method, path string, handler func(http.ResponseWriter, *http.Request))
	With(method, path string, middlewares []func(http.Handler) http.Handler, handler func(http.ResponseWriter, *http.Request))
}

const (
	GET         = "GET"
	POST        = "POST"
	PUT         = "PUT"
	PATCH       = "PATCH"
	DELETE      = "DELETE"
	HEAD        = "HEAD"
	OPTIONS     = "OPTIONS"
	HandlerFunc = "HandlerFunc"
)

type TpHeader int

const (
	TpKeepHeader TpHeader = iota
	TpJoinHeader
	TpReplaceHeader
)

/**
* Routes: The routes this service has published to the gateway, by METHOD:path.
**/
type Routes struct {
	Name   string
	Routes map[string]et.Json
}

const (
	// V_1
	APIGATEWAY_SET_ROUTER    = "event:apigateway:set:router"
	APIGATEWAY_REMOVE_ROUTER = "event:apigateway:remove:router"
	APIGATEWAY_RESET_ROUTER  = "event:apigateway:reset:router"
	// V_0
	APIGATEWAY_SET_RESOLVE    = "apigateway/set/resolve"
	APIGATEWAY_DELETE_RESOLVE = "apigateway/delete/resolve"
	APIGATEWAY_RESET          = "apigateway/reset"
)

var (
	router *Routes
	// mu: guards router; PushApiGateway and RemoveApiGateway may run at any time from several goroutines (routes published
	// while the service runs) while eventActionReset walks the routes
	mu sync.RWMutex
)

/**
* initRouter: With mu held
* @param name string
**/
func initRouter(name string) {
	if router == nil {
		router = &Routes{
			Name:   name,
			Routes: map[string]et.Json{},
		}

		channel := fmt.Sprintf(`%s:%s`, APIGATEWAY_RESET_ROUTER, name)
		event.Stack(channel, eventActionReset)
		event.Stack(APIGATEWAY_RESET_ROUTER, eventActionReset)

		channel = fmt.Sprintf(`%s:%s`, APIGATEWAY_RESET, name)
		event.Stack(channel, eventActionReset)
		event.Stack(APIGATEWAY_RESET, eventActionReset)
	}
}

/**
* eventActionReset
* @param m event.Message
**/
func eventActionReset(m event.Message) {
	mu.RLock()
	if router == nil {
		mu.RUnlock()
		return
	}
	routes := maps.Clone(router.Routes)
	mu.RUnlock()

	for _, v := range routes {
		logs.Logf("Apigateway", `[RESET] %s:%s`, v.Str("method"), v.Str("path"))
		event.Publish(APIGATEWAY_SET_ROUTER, v)
		event.Publish(APIGATEWAY_SET_RESOLVE, v)
	}
}

/**
* String
* @return string
**/
func (t TpHeader) String() string {
	switch t {
	case TpKeepHeader:
		return "Keep the resolve header"
	case TpJoinHeader:
		return "Join request header with the resolve header"
	case TpReplaceHeader:
		return "Replace resolve header with request header"
	default:
		return "Unknown"
	}
}

/**
* IntToTpHeader
* @param tp int
* @return TpHeader
**/
func IntToTpHeader(tp int) TpHeader {
	switch tp {
	case 1:
		return TpJoinHeader
	case 2:
		return TpReplaceHeader
	default:
		return TpKeepHeader
	}
}

/**
* ToTpHeader
* @param str string
* @return TpHeader
**/
func ToTpHeader(tp int) TpHeader {
	switch tp {
	case 1:
		return TpJoinHeader
	case 2:
		return TpReplaceHeader
	default:
		return TpKeepHeader
	}
}

/**
* PushApiGateway
* @param method, path, resolve string, header et.Json, tpHeader TpHeader, excludeHeader []string, params, body et.Json, version int, packageName string
**/
func PushApiGateway(method, path, resolve string, tpHeader TpHeader, header et.Json, excludeHeader []string, params, body et.Json, version int, packageName string) {
	mu.Lock()
	initRouter(packageName)
	key := fmt.Sprintf("%s:%s", method, path)
	route := et.Json{
		"_id":            key,
		"id":             key,
		"kind":           "api",
		"method":         method,
		"path":           path,
		"resolve":        resolve,
		"tp_header":      tpHeader,
		"header":         header,
		"exclude_header": excludeHeader,
		"params":         params,
		"body":           body,
		"version":        version,
		"package_name":   packageName,
	}
	router.Routes[key] = route
	mu.Unlock()

	event.Publish(APIGATEWAY_SET_ROUTER, route)
	event.Publish(APIGATEWAY_SET_RESOLVE, route)
}

// The modes of a static route (PushStatic): the gateway copies the files (proxy) or redirects to the origin (redirect)
const (
	ModeProxy    = "proxy"
	ModeRedirect = "redirect"
)

/**
* StaticId: The id in the gateway of the static route of a prefix (GET:<path>/*), for RemoveApiGateway
* @param path string
* @return string
**/
func StaticId(path string) string {
	return fmt.Sprintf("%s:%s/*", GET, strings.TrimSuffix(path, "/"))
}

/**
* PushStatic: Publishes to the gateway the files of an origin under a prefix (ettp/v2 TpStatic): GET <path>/* resolves
* to <origin>/<rest>, copied (proxy) or answered with a 302 (redirect; the entry files —index.html, mf-manifest.json,
* remoteEntry.js— are always copied); fallback is the file of a SPA for its pages (index.html), empty for none. Like
* PushApiGateway, it is kept and published again when the gateway resets
* @param path, origin, mode, fallback string, version int, packageName string
* @return string
**/
func PushStatic(path, origin, mode, fallback string, version int, packageName string) string {
	prefix := strings.TrimSuffix(path, "/")
	resolve := strings.TrimSuffix(origin, "/") + "/*"
	id := StaticId(prefix)
	mu.Lock()
	initRouter(packageName)
	route := et.Json{
		"_id":          id,
		"id":           id,
		"kind":         "static",
		"method":       GET,
		"path":         prefix + "/*",
		"resolve":      resolve,
		"mode":         mode,
		"fallback":     fallback,
		"version":      version,
		"package_name": packageName,
	}
	router.Routes[id] = route
	mu.Unlock()

	event.Publish(APIGATEWAY_SET_ROUTER, route)
	return id
}

/**
* RemoveApiGateway
* @param id string
**/
func RemoveApiGateway(id string) {
	mu.Lock()
	if router == nil {
		mu.Unlock()
		return
	}

	delete(router.Routes, id)
	mu.Unlock()
	event.Publish(APIGATEWAY_REMOVE_ROUTER, et.Json{"id": id})
	event.Publish(APIGATEWAY_DELETE_RESOLVE, et.Json{"id": id})
}

/**
* GetRoutes: A copy of the routes published to the gateway
* @return map[string]et.Json
**/
func GetRoutes() map[string]et.Json {
	mu.RLock()
	defer mu.RUnlock()

	if router == nil {
		return map[string]et.Json{}
	}

	return maps.Clone(router.Routes)
}

/**
* pushApiGateway: Publishes a Route to the gateway with Host+Path as resolve and TpReplaceHeader.
* @param route Route
**/
func pushApiGateway(route Route) {
	resolve := route.Host + route.Path
	PushApiGateway(route.Method, route.Path, resolve, TpReplaceHeader, route.Header, route.ExcludeHeader, route.Params, route.Body, route.Version, route.PackageName)
}

/**
* Route: A route to mount (Publish, With) and publish to the gateway.
**/
type Route struct {
	Method        string
	Path          string
	Host          string
	Handler       http.HandlerFunc
	Header        et.Json
	ExcludeHeader []string
	Params        et.Json
	Body          et.Json
	Version       int
	PackageName   string
}

/**
* Publish
* @param r *chi.Mux, method string, path string, h http.HandlerFunc, packageName string, packagePath string, host string
* @return *chi.Mux
**/
func Publish(r *chi.Mux, route Route) *chi.Mux {
	switch route.Method {
	case "GET":
		r.Get(route.Path, route.Handler)
	case "POST":
		r.Post(route.Path, route.Handler)
	case "PUT":
		r.Put(route.Path, route.Handler)
	case "PATCH":
		r.Patch(route.Path, route.Handler)
	case "DELETE":
		r.Delete(route.Path, route.Handler)
	case "HEAD":
		r.Head(route.Path, route.Handler)
	case "OPTIONS":
		r.Options(route.Path, route.Handler)
	case "HandlerFunc":
		r.HandleFunc(route.Path, route.Handler)
	}

	pushApiGateway(route)
	return r
}

/**
* With
* @param r *chi.Mux, method string, path string, middlewares []func(http.Handler) http.Handler, h http.HandlerFunc, packageName string, packagePath string, host string
* @return *chi.Mux
**/
func With(r *chi.Mux, route Route, middlewares []func(http.Handler) http.Handler) *chi.Mux {
	switch route.Method {
	case "GET":
		r.With(middlewares...).Get(route.Path, route.Handler)
	case "POST":
		r.With(middlewares...).Post(route.Path, route.Handler)
	case "PUT":
		r.With(middlewares...).Put(route.Path, route.Handler)
	case "PATCH":
		r.With(middlewares...).Patch(route.Path, route.Handler)
	case "DELETE":
		r.With(middlewares...).Delete(route.Path, route.Handler)
	case "HEAD":
		r.With(middlewares...).Head(route.Path, route.Handler)
	case "OPTIONS":
		r.With(middlewares...).Options(route.Path, route.Handler)
	case "HandlerFunc":
		r.With(middlewares...).HandleFunc(route.Path, route.Handler)
	}

	pushApiGateway(route)
	return r
}
