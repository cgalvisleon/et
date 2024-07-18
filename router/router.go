package router

import (
	"net/http"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/js"
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

func eventApiGateway(method, path, packageName, packagePath, host string) {
	kind := "HTTP"
	develop := envar.GetStr("development", "ENV")
	if develop == "production" {
		kind = "REST"
	}

	path = packagePath + path
	resolve := host + path

	event.Publish("gateway/upsert", js.Json{
		"kind":    kind,
		"method":  method,
		"path":    path,
		"resolve": resolve,
		"package": packageName,
	})

}

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

	eventApiGateway(method, path, packageName, packagePath, host)

	return r
}

func Protect(r *chi.Mux, method, path string, h http.HandlerFunc, packageName, packagePath, host string) *chi.Mux {
	switch method {
	case "GET":
		r.With(middleware.Authorization).Get(path, h)
	case "POST":
		r.With(middleware.Authorization).Post(path, h)
	case "PUT":
		r.With(middleware.Authorization).Put(path, h)
	case "PATCH":
		r.With(middleware.Authorization).Patch(path, h)
	case "DELETE":
		r.With(middleware.Authorization).Delete(path, h)
	case "HEAD":
		r.With(middleware.Authorization).Head(path, h)
	case "OPTIONS":
		r.With(middleware.Authorization).Options(path, h)
	case "HandlerFunc":
		r.With(middleware.Authorization).HandleFunc(path, h)
	}

	eventApiGateway(method, path, packageName, packagePath, host)

	return r
}
