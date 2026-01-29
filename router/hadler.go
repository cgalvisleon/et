package router

import (
	"fmt"
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/response"
	"github.com/go-chi/chi/v5"
)

/**
* HttpSet handles POST requests to set solver data
* @param w http.ResponseWriter, r *http.Request
**/
func HttpSet(w http.ResponseWriter, r *http.Request) {
	body, _ := response.GetBody(r)
	method := body.String("method")
	path := body.String("path")
	resolve := body.String("resolve")
	tpHeader := TpHeader(body.Int("header"))
	excludeHeader := body.ArrayStr("exclude_header")
	version := body.Int("version")
	packageName := body.String("package_name")
	key := fmt.Sprintf("%s:%s", method, path)

	if setFn != nil {
		err := setFn(key, body)
		if err != nil {
			response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
	}

	PushApiGateway(method, path, resolve, tpHeader, et.Json{}, excludeHeader, version, packageName)
	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok: true,
		Result: et.Json{
			"message": "success",
			"id":      key,
		},
	})
}

/**
* HttpGet handles GET requests to get solver data
* @param w http.ResponseWriter, r *http.Request
**/
func HttpGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if getFn == nil {
		response.HTTPError(w, r, http.StatusInternalServerError, "get function not defined")
		return
	}

	data, err := getFn(id)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: data,
	})
}

/**
* HttpDelete handles DELETE requests to delete solver data
* @param w http.ResponseWriter, r *http.Request
**/
func HttpDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	RemoveApiGateway(id)
	if deleteFn == nil {
		response.HTTPError(w, r, http.StatusInternalServerError, "delete function not defined")
		return
	}

	err := deleteFn(id)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok: true,
		Result: et.Json{
			"message": "deleted",
		},
	})
}
