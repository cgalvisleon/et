package apigateway

import (
	"io"
	"net/http"

	"github.com/cgalvisleon/et/et"
)

func notFounder(w http.ResponseWriter, r *http.Request) {
	et.HTTPError(w, r, http.StatusNotFound, "404 Not Found.")
}

func handlerFn(w http.ResponseWriter, r *http.Request) {
	resolute := NewResolute(r)

	if resolute.Resolve == nil {
		conn.http.notFoundHandler(w, r)
		return
	}

	kind := resolute.Resolve.Node.Resolve.ValStr(HTTP, "kind")
	if kind == HANDLER {

		return
	}

	if kind == REST {
		request, err := http.NewRequest(resolute.Method, resolute.URL, resolute.Body)
		if err != nil {
			et.HTTPError(w, r, http.StatusInternalServerError, err.Error())
			return
		}

		request.Header = resolute.Header
		client := &http.Client{}
		res, err := client.Do(request)
		if err != nil {
			et.HTTPError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		defer res.Body.Close()

		for key, value := range res.Header {
			w.Header().Set(key, value[0])
		}
		_, err = io.Copy(w, res.Body)
		if err != nil {
			et.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		}

		return
	}

	/*
		if kind == WEBSOCKET {
			// TODO

		}
	*/

	http.Redirect(w, r, resolute.URL, http.StatusSeeOther)
}

func version(w http.ResponseWriter, r *http.Request) {
	result := Version()
	et.JSON(w, r, http.StatusOK, result)
}

func upsert(w http.ResponseWriter, r *http.Request) {
	body, _ := et.GetBody(r)
	method := body.Str("method")
	path := body.Str("path")
	resolve := body.Str("resolve")
	kind := body.ValStr("HTTP", "kind")
	stage := body.ValStr("default", "stage")
	packageName := body.Str("package")

	AddRoute(method, path, resolve, kind, stage, packageName)

	et.JSON(w, r, http.StatusOK, et.Json{
		"message": "Router added",
	})
}

func getAll(w http.ResponseWriter, r *http.Request) {
	_routes, err := et.Marshal(routes)
	if err != nil {
		et.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	et.JSON(w, r, http.StatusOK, _routes)
}
