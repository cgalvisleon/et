package jrpc

import (
	"errors"
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/response"
)

var (
	ErrorRpcNotConnected = errors.New("rpc not connected")
)

/**
* listRouters
* @return []et.Json
* @return error
**/
func listRouters() ([]et.Json, error) {
	result := []et.Json{}
	for name, pkg := range rpcs {
		result = append(result, et.Json{
			"name": name,
			"pkg":  pkg,
		})
	}

	return result, nil
}

/**
* HttpListRouters
* @param w http.ResponseWriter
* @param r *http.Request
**/
func HttpListRouters(w http.ResponseWriter, r *http.Request) {
	item, err := listRouters()
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
	}

	result := et.Items{}
	result.Add(item...)
	response.ITEMS(w, r, http.StatusOK, result)
}
