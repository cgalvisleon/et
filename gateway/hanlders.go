package gateway

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cgalvisleon/et/js"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/response"
)

/**
* version
* @params w http.ResponseWriter
* @params r *http.Request
**/
func version(w http.ResponseWriter, r *http.Request) {
	result := Version()
	response.JSON(w, r, http.StatusOK, result)
}

/**
* notFounder
* @params w http.ResponseWriter
* @params r *http.Request
**/
func notFounder(w http.ResponseWriter, r *http.Request) {
	result := js.Json{
		"message": "404 Not Found.",
		"route":   r.RequestURI,
	}
	response.JSON(w, r, http.StatusNotFound, result)
}

/**
* upsert
* @params w http.ResponseWriter
* @params r *http.Request
**/
func upsert(w http.ResponseWriter, r *http.Request) {
	body, _ := response.GetBody(r)
	method := body.Str("method")
	path := body.Str("path")
	resolve := body.Str("resolve")
	kind := body.ValStr("HTTP", "kind")
	stage := body.ValStr("default", "stage")
	packageName := body.Str("package")

	conn.http.AddRoute(method, path, resolve, kind, stage, packageName)

	response.JSON(w, r, http.StatusOK, js.Json{
		"message": "Router added",
	})
}

/**
* getAll
* @params w http.ResponseWriter
* @params r *http.Request
**/
func getAll(w http.ResponseWriter, r *http.Request) {
	_pakages, err := js.Marshal(conn.http.pakages)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, r, http.StatusOK, _pakages)
}

/**
* handlerFn
* @params w http.ResponseWriter
* @params r *http.Request
**/
func handlerFn(w http.ResponseWriter, r *http.Request) {
	finalHandler := http.HandlerFunc(handlerExec)
	middleware.Authorization(finalHandler).ServeHTTP(w, r)
}

/**
* handlerExec
* @params w http.ResponseWriter
* @params r *http.Request
**/
func handlerExec(w http.ResponseWriter, r *http.Request) {
	// Begin telemetry
	metric := middleware.NewMetric(r)

	// Get resolute
	resolute := GetResolute(r)

	// Call search time since begin
	metric.CallExecute()

	// If not found
	if resolute.Resolve == nil || resolute.URL == "" {
		r.RequestURI = fmt.Sprintf(`%s://%s%s`, resolute.Scheme, resolute.Host, resolute.Path)
		metric.NotFound(conn.http.notFoundHandler, w, r)
		return
	}

	// If HandlerFunc is handler
	kind := resolute.Resolve.Route.Resolve.ValStr("HTTP", "kind")
	if kind == HANDLER {
		handler := conn.http.handlers[resolute.Resolve.Route.Id]
		if handler == nil {
			r.RequestURI = fmt.Sprintf(`%s://%s%s`, resolute.Scheme, resolute.Host, resolute.Path)
			metric.NotFound(conn.http.notFoundHandler, w, r)
			return
		}

		if resolute.Resolve.Route.IsWs {
			handler(w, r)
			go metric.DoneFn(http.StatusOK, w, r)
			return
		}

		metric.Handler(handler, w, r)
		return
	}

	// If REST is handler
	request, err := http.NewRequest(resolute.Method, resolute.URL, resolute.Body)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	metric.Downtime = time.Since(metric.TimeBegin)
	request.Header = resolute.Header
	client := &http.Client{}
	res, err := client.Do(request)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadGateway, err.Error())
		return
	}

	defer func() {
		go metric.Done(res)
		res.Body.Close()
	}()

	for key, value := range res.Header {
		w.Header().Set(key, value[0])
	}
	w.WriteHeader(res.StatusCode)
	_, err = io.Copy(w, res.Body)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
	}
}
