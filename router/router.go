package router

import (
	"fmt"
	"net/http"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/middleware"
	"github.com/go-chi/chi/v5"
)

type TypeRoute int

const (
	HTTP TypeRoute = iota
	REST
)

const (
	Get         = "GET"
	Post        = "POST"
	Put         = "PUT"
	Patch       = "PATCH"
	Delete      = "DELETE"
	Head        = "HEAD"
	Options     = "OPTIONS"
	HandlerFunc = "HandlerFunc"
)

type TpHeader int

const (
	TpKeepHeader TpHeader = iota
	TpJoinHeader
	TpReplaceHeader
)

type Routes struct {
	Name   string
	Routes map[string]et.Json
}

var (
	router              *Routes
	APIGATEWAY_SET      = "apigateway/set"
	APIGATEWAY_DELETE   = "apigateway/delete"
	APIGATEWAY_RESET    = "apigateway/reset"
	PROXYGATEWAY_SET    = "proxygateway/set"
	PROXYGATEWAY_DELETE = "proxygateway/delete"
	PROXYGATEWAY_RESET  = "proxygateway/reset"
)

/**
* SetChannels
* @param vars et.Json
**/
func SetChannels(vars et.Json) {
	for k := range vars {
		switch k {
		case "APIGATEWAY_SET":
			APIGATEWAY_SET = vars.Str(k)
		case "APIGATEWAY_DELETE":
			APIGATEWAY_DELETE = vars.Str(k)
		case "APIGATEWAY_RESET":
			APIGATEWAY_RESET = vars.Str(k)
		case "PROXYGATEWAY_SET":
			PROXYGATEWAY_SET = vars.Str(k)
		case "PROXYGATEWAY_DELETE":
			PROXYGATEWAY_DELETE = vars.Str(k)
		case "PROXYGATEWAY_RESET":
			PROXYGATEWAY_RESET = vars.Str(k)
		}
	}
}

/**
* GetChannels
* @return et.Json
**/
func GetChanels() et.Json {
	return et.Json{
		"APIGATEWAY_SET":      APIGATEWAY_SET,
		"APIGATEWAY_DELETE":   APIGATEWAY_DELETE,
		"APIGATEWAY_RESET":    APIGATEWAY_RESET,
		"PROXYGATEWAY_SET":    PROXYGATEWAY_SET,
		"PROXYGATEWAY_DELETE": PROXYGATEWAY_DELETE,
		"PROXYGATEWAY_RESET":  PROXYGATEWAY_RESET,
	}
}

/**
* initRouter
* @param name string
**/
func initRouter(name string) {
	if router == nil {
		router = &Routes{
			Name:   name,
			Routes: map[string]et.Json{},
		}

		channel := fmt.Sprintf("%s/%s", APIGATEWAY_RESET, name)
		event.Stack(channel, eventAction)
	}
}

/**
* eventAction
* @param m event.Message
**/
func eventAction(m event.Message) {
	if router == nil {
		return
	}

	ResetRouter()
}

/**
* ResetRouter
**/
func ResetRouter() {
	if router == nil {
		return
	}

	for _, v := range router.Routes {
		console.Logf("Api gateway", `[RESET] %s:%s`, v.Str("method"), v.Str("path"))
		event.Publish(APIGATEWAY_SET, v)
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
* @param id, method, path, resolve string, header et.Json, tpHeader TpHeader, excludeHeader []string, private bool, packageName string
**/
func PushApiGateway(id, method, path, resolve string, header et.Json, tpHeader TpHeader, excludeHeader []string, private bool, packageName string) {
	initRouter(packageName)
	router.Routes[id] = et.Json{
		"_id":            id,
		"kind":           HTTP,
		"method":         method,
		"path":           path,
		"resolve":        resolve,
		"header":         header,
		"tp_header":      tpHeader,
		"exclude_header": excludeHeader,
		"private":        private,
		"package_name":   packageName,
	}

	event.Publish(APIGATEWAY_SET, router.Routes[id])
}

/**
* GetRoutes
* @return map[string]et.Json
**/
func GetRoutes() map[string]et.Json {
	if router == nil {
		return map[string]et.Json{}
	}

	return router.Routes
}

/**
* PushApiGateway
* @param method, path, packagePath, host, packageName string, private bool
**/
func pushApiGateway(method, path, packagePath, host, packageName string, private bool) {
	id := cache.GenKey(method, path, packageName)
	path = packagePath + path
	resolve := host + path

	PushApiGateway(id, method, path, resolve, et.Json{}, TpReplaceHeader, []string{}, private, packageName)
}

/**
* Public
* @param r *chi.Mux, method string, path string, h http.HandlerFunc, packageName string, packagePath string, host string
* @return *chi.Mux
**/
func Public(r *chi.Mux, method, path string, h http.HandlerFunc, packageName, packagePath, host string) *chi.Mux {
	switch method {
	case "GET":
		r.Get(path, h)
	case "POST":
		r.Post(path, h)
	case "PUT":
		r.Put(path, h)
	case "PATCH":
		r.Patch(path, h)
	case "DELETE":
		r.Delete(path, h)
	case "HEAD":
		r.Head(path, h)
	case "OPTIONS":
		r.Options(path, h)
	case "HandlerFunc":
		r.HandleFunc(path, h)
	}

	pushApiGateway(method, path, packagePath, host, packageName, false)

	return r
}

/**
* Private
* @param r *chi.Mux, method string, path string, h http.HandlerFunc, packageName string, packagePath string, host string
* @return *chi.Mux
**/
func Private(r *chi.Mux, method, path string, h http.HandlerFunc, packageName, packagePath, host string) *chi.Mux {
	switch method {
	case "GET":
		r.With(middleware.Autentication).Get(path, h)
	case "POST":
		r.With(middleware.Autentication).Post(path, h)
	case "PUT":
		r.With(middleware.Autentication).Put(path, h)
	case "PATCH":
		r.With(middleware.Autentication).Patch(path, h)
	case "DELETE":
		r.With(middleware.Autentication).Delete(path, h)
	case "HEAD":
		r.With(middleware.Autentication).Head(path, h)
	case "OPTIONS":
		r.With(middleware.Autentication).Options(path, h)
	case "HandlerFunc":
		r.With(middleware.Autentication).HandleFunc(path, h)
	}

	pushApiGateway(method, path, packagePath, host, packageName, true)

	return r
}

/**
* Protect
* @param r *chi.Mux, method string, path string, h http.HandlerFunc, packageName string, packagePath string, host string
* @return *chi.Mux
**/
func Protect(r *chi.Mux, method, path string, h http.HandlerFunc, packageName, packagePath, host string) *chi.Mux {
	return Private(r, method, path, h, packageName, packagePath, host)
}

/**
* Authorization
* @param r *chi.Mux, method string, path string, h http.HandlerFunc, packageName string, packagePath string, host string
* @return *chi.Mux
**/
func Authorization(r *chi.Mux, method, path string, h http.HandlerFunc, packageName, packagePath, host string) *chi.Mux {
	if middleware.AuthorizationMiddleware == nil {
		logs.Alertm("AuthorizationMiddleware not set")
		return r
	}

	router := r.With(middleware.Autentication).With(middleware.AuthorizationMiddleware)
	switch method {
	case "GET":
		router.Get(path, h)
	case "POST":
		router.Post(path, h)
	case "PUT":
		router.Put(path, h)
	case "PATCH":
		router.Patch(path, h)
	case "DELETE":
		router.Delete(path, h)
	case "HEAD":
		router.Head(path, h)
	case "OPTIONS":
		router.Options(path, h)
	case "HandlerFunc":
		router.HandleFunc(path, h)
	}

	pushApiGateway(method, path, packagePath, host, packageName, true)

	return r
}

/**
* With
* @param r *chi.Mux, method string, path string, middlewares []func(http.Handler) http.Handler, h http.HandlerFunc, packageName string, packagePath string, host string
* @return *chi.Mux
**/
func With(r *chi.Mux, method, path string, middlewares []func(http.Handler) http.Handler, h http.HandlerFunc, packageName, packagePath, host string) *chi.Mux {
	switch method {
	case "GET":
		r.With(middlewares...).Get(path, h)
	case "POST":
		r.With(middlewares...).Post(path, h)
	case "PUT":
		r.With(middlewares...).Put(path, h)
	case "PATCH":
		r.With(middlewares...).Patch(path, h)
	case "DELETE":
		r.With(middlewares...).Delete(path, h)
	case "HEAD":
		r.With(middlewares...).Head(path, h)
	case "OPTIONS":
		r.With(middlewares...).Options(path, h)
	case "HandlerFunc":
		r.With(middlewares...).HandleFunc(path, h)
	}

	pushApiGateway(method, path, packagePath, host, packageName, true)

	return r
}
