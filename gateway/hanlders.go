package gateway

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cgalvisleon/et/js"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/telemetry"
)

// Version information this package
func version(w http.ResponseWriter, r *http.Request) {
	result := Version()
	response.JSON(w, r, http.StatusOK, result)
}

// Handler for not found
func notFounder(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, r, http.StatusNotFound, js.Json{
		"message": "404 Not Found.",
		"route":   r.RequestURI,
	})
}

// Handler Router
func handlerRouter(w http.ResponseWriter, r *http.Request) {
	// Begin telemetry
	metric := telemetry.NewMetric(r)

	// Get resolute
	resolute := GetResolute(r)

	// Check if resolve is nil
	metric.CallExecute()

	if resolute.Resolve == nil || resolute.URL == "" {
		r.RequestURI = fmt.Sprintf(`%s://%s%s`, resolute.Scheme, resolute.Host, resolute.Path)
		conn.http.notFoundHandler(w, r)

		defer func() {
			go metric.NotFound(r)
		}()

		return
	}

	kind := resolute.Resolve.Node.Resolve.ValStr("HTTP", "kind")
	if kind == HANDLER {
		metric.Downtime = time.Since(metric.TimeBegin)
		handler := conn.http.handlers[resolute.Resolve.Node._id]
		if handler == nil {
			response.HTTPError(w, r, http.StatusNotFound, "404 Not Found.")
			return
		}

		defer func() {
			go metric.DoneHandler()
		}()

		handler(w, r)

		return
	}

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

// UpSert a update or new route
func upSert(w http.ResponseWriter, r *http.Request) {
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

// Getall list of routes
func getAll(w http.ResponseWriter, r *http.Request) {
	_pakages, err := js.Marshal(conn.http.pakages)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, r, http.StatusOK, _pakages)
}
