package router

import (
	"fmt"
	"net/http"

	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/middleware"
	"github.com/go-chi/chi/v5"
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

type Channels struct {
	EVENT_SET_ROUTER    string
	EVENT_REMOVE_ROUTER string
	EVENT_RESET_ROUTER  string
}

var (
	router              *Routes
	EVENT_SET_ROUTER    = "event:set:router"
	EVENT_REMOVE_ROUTER = "event:remove:router"
	EVENT_RESET_ROUTER  = "event:reset:router"
)

/**
* SetChannels
* @param vars et.Json
**/
func SetChannels(channels *Channels) {
	EVENT_SET_ROUTER = channels.EVENT_SET_ROUTER
	EVENT_REMOVE_ROUTER = channels.EVENT_REMOVE_ROUTER
	EVENT_RESET_ROUTER = channels.EVENT_RESET_ROUTER
}

/**
* GetChannels
* @return et.Json
**/
func GetChanels() et.Json {
	return et.Json{
		"EVENT_SET_ROUTER":    EVENT_SET_ROUTER,
		"EVENT_REMOVE_ROUTER": EVENT_REMOVE_ROUTER,
		"EVENT_RESET_ROUTER":  EVENT_RESET_ROUTER,
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

		channel := fmt.Sprintf(`%s:%s`, EVENT_RESET_ROUTER, name)
		event.Stack(channel, eventActionReset)
		event.Stack(EVENT_RESET_ROUTER, eventActionReset)
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
		console.Logf("Api gateway", `[RESET] %s:%s`, v.Str("method"), v.Str("path"))
		event.Publish(EVENT_SET_ROUTER, v)
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
* @param method, path, resolve string, header et.Json, tpHeader TpHeader, excludeHeader []string, private bool, version int, packageName string
**/
func PushApiGateway(method, path, resolve string, tpHeader TpHeader, header et.Json, excludeHeader []string, private bool, version int, packageName string) {
	initRouter(packageName)
	key := fmt.Sprintf("%s:%s", method, path)
	router.Routes[key] = et.Json{
		"kind":           "api",
		"method":         method,
		"path":           path,
		"resolve":        resolve,
		"tp_header":      tpHeader,
		"header":         header,
		"exclude_header": excludeHeader,
		"private":        private,
		"version":        version,
		"package_name":   packageName,
	}

	event.Publish(EVENT_SET_ROUTER, router.Routes[key])
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
	event.Publish(EVENT_REMOVE_ROUTER, et.Json{
		"id": id,
	})
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
	path = packagePath + path
	resolve := host + path

	PushApiGateway(method, path, resolve, TpReplaceHeader, et.Json{}, []string{}, private, 0, packageName)
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
