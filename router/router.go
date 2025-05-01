package router

import (
	"fmt"
	"net/http"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/mistake"
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

var router *Routes

const (
	APIGATEWAY_SET    = "apigateway/set/resolve"
	APIGATEWAY_DELETE = "apigateway/delete/resolve"
	APIGATEWAY_RESET  = "apigateway/reset"
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

		channel := fmt.Sprintf("%s/%s", APIGATEWAY_RESET, name)
		err := event.Stack(channel, eventAction)
		if err != nil {
			logs.Error(err)
		}
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

	for _, v := range router.Routes {
		console.Logf("Api gateway", `[RESET] %s:%s`, v.Str("method"), v.Str("path"))
		event.Publish(APIGATEWAY_SET, v)
	}
}

/**
* Delete
* @param name string
**/
func deleteRouter(id string) {
	if router == nil {
		return
	}

	delete(router.Routes, id)
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
* DeleteApiGatewayById
* @param id, method, path string
**/
func DeleteApiGatewayById(id, method, path string) {
	deleteRouter(id)

	event.Publish(APIGATEWAY_DELETE, et.Json{
		"_id":    id,
		"method": method,
		"path":   path,
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
	id := cache.GenKey(method, path)

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
* Protect
* @param r *chi.Mux, method string, path string, h http.HandlerFunc, packageName string, packagePath string, host string
* @return *chi.Mux
**/
func Protect(r *chi.Mux, method, path string, h http.HandlerFunc, packageName, packagePath, host string) *chi.Mux {
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
* Authorization
* @param r *chi.Mux, method string, path string, h http.HandlerFunc, packageName string, packagePath string, host string
* @return *chi.Mux
**/
func Authorization(r *chi.Mux, method, path string, h http.HandlerFunc, packageName, packagePath, host string) *chi.Mux {
	switch method {
	case "GET":
		r.With(middleware.Autentication).With(middleware.Authorization).Get(path, h)
	case "POST":
		r.With(middleware.Autentication).With(middleware.Authorization).Post(path, h)
	case "PUT":
		r.With(middleware.Autentication).With(middleware.Authorization).Put(path, h)
	case "PATCH":
		r.With(middleware.Autentication).With(middleware.Authorization).Patch(path, h)
	case "DELETE":
		r.With(middleware.Autentication).With(middleware.Authorization).Delete(path, h)
	case "HEAD":
		r.With(middleware.Autentication).With(middleware.Authorization).Head(path, h)
	case "OPTIONS":
		r.With(middleware.Autentication).With(middleware.Authorization).Options(path, h)
	case "HandlerFunc":
		r.With(middleware.Autentication).With(middleware.Authorization).HandleFunc(path, h)
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

/**
* authorization
* @param profile et.Json
* @return map[string]bool, error
**/
func authorization(profile et.Json) (map[string]bool, error) {
	method := config.String("AUTHORIZATION_METHOD", "Module.Services.GetPermissions")
	if method == "" {
		return map[string]bool{}, mistake.New("Authorization method not found")
	}

	result, err := jrpc.CallPermitios(method, profile)
	if err != nil {
		return map[string]bool{}, err
	}

	return result, nil
}

func init() {
	middleware.SetAuthorizationFunc(authorization)
}
