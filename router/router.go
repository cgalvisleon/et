package router

import (
	"fmt"
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/go-chi/chi/v5"
)

type Router interface {
	UseAutentication(fn func(http.Handler) http.Handler)
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
)

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
	if router == nil {
		return
	}

	for _, v := range router.Routes {
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
* @param method, path, resolve string, header et.Json, tpHeader TpHeader, excludeHeader []string, version int, packageName string
**/
func PushApiGateway(method, path, resolve string, tpHeader TpHeader, header et.Json, excludeHeader []string, version int, packageName string) {
	initRouter(packageName)
	key := fmt.Sprintf("%s:%s", method, path)
	router.Routes[key] = et.Json{
		"_id":            key,
		"kind":           "api",
		"method":         method,
		"path":           path,
		"resolve":        resolve,
		"tp_header":      tpHeader,
		"header":         header,
		"exclude_header": excludeHeader,
		"version":        version,
		"package_name":   packageName,
	}

	event.Publish(APIGATEWAY_SET_ROUTER, router.Routes[key])
	event.Publish(APIGATEWAY_SET_RESOLVE, router.Routes[key])
}

/**
* RemoveApiGateway
* @param id string
**/
func RemoveApiGateway(id string) {
	if router == nil {
		return
	}

	delete(router.Routes, id)
	event.Publish(APIGATEWAY_REMOVE_ROUTER, et.Json{"id": id})
	event.Publish(APIGATEWAY_DELETE_RESOLVE, et.Json{"id": id})
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
* @param method, path, packagePath, host, packageName string
**/
func pushApiGateway(method, path, packagePath, host, packageName string) {
	path = packagePath + path
	resolve := host + path

	PushApiGateway(method, path, resolve, TpReplaceHeader, et.Json{}, []string{}, 0, packageName)
}

/**
* Publish
* @param r *chi.Mux, method string, path string, h http.HandlerFunc, packageName string, packagePath string, host string
* @return *chi.Mux
**/
func Publish(r *chi.Mux, method, path string, h http.HandlerFunc, packageName, packagePath, host string) *chi.Mux {
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

	pushApiGateway(method, path, packagePath, host, packageName)

	return r
}

/**
* With
* @param r *chi.Mux, method string, path string, middlewares []func(http.Handler) http.Handler, h http.HandlerFunc, packageName string, packagePath string, host string
* @return *chi.Mux
**/
func With(r *chi.Mux, method, path string, h http.HandlerFunc, packageName, packagePath, host string, middlewares []func(http.Handler) http.Handler) *chi.Mux {
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

	pushApiGateway(method, path, packagePath, host, packageName)

	return r
}
