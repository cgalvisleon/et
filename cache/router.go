package cache

import (
	"net/http"

	"github.com/cgalvisleon/et/response"
)

const (
	GET    = "GET"
	DELETE = "DELETE"
)

type Router interface {
	Protect(method, path string, handler func(http.ResponseWriter, *http.Request))
}

func LoadRouter(r Router) {
	r.Protect(GET, "/cache", HttpAll)
	r.Protect(GET, "/cache/{key}", HttpGet)
	r.Protect(DELETE, "/cache", HttpDelete)
}

/**
* HttpAll
* @params w http.ResponseWriter, r *http.Request
**/
func HttpAll(w http.ResponseWriter, r *http.Request) {
	query := response.GetQuery(r)
	search := query.Str("search")
	page := query.ValInt(1, "page")
	rows := query.ValInt(30, "rows")

	result, err := AllCache(search, page, rows)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, r, http.StatusOK, result)
}

/**
* HttpGet
* @params w http.ResponseWriter, r *http.Request
**/
func HttpGet(w http.ResponseWriter, r *http.Request) {
	query := response.GetQuery(r)
	key := query.Str("key")

	result, err := Get(key, "")
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, r, http.StatusOK, result)
}

/**
* HttpDelete
* @params w http.ResponseWriter, r *http.Request
**/
func HttpDelete(w http.ResponseWriter, r *http.Request) {
	query := response.GetQuery(r)
	key := query.Str("key")

	result, err := Delete(key)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, r, http.StatusOK, result)
}
