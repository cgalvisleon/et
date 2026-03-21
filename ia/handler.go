package ia

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/instances"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/response"
	"github.com/go-chi/chi"
)

func New(store instances.Store) *Agents {
	result := &Agents{
		agents:  make(map[string]*Agent),
		mu:      sync.RWMutex{},
		isDebug: envar.GetBool("DEBUG", false),
	}

	if store != nil {
		result.getInstance = store.Get
		result.setInstance = store.Set
		result.queryInstance = store.Query
	}

	return result
}

/**
* HttpGet
* @params w http.ResponseWriter, r *http.Request
**/
func (s *Agents) HttpGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var instance Agent
	exists, err := s.getInstance(id, &instance)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if !exists {
		response.ITEM(w, r, http.StatusNotFound, et.Item{
			Ok:     true,
			Result: et.Json{"message": "instance not found"},
		})
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: instance.ToJson(),
	})
}

/**
* HttpState
* @params w http.ResponseWriter, r *http.Request
**/
func (s *Agents) HttpState(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var instance Agent
	exists, err := s.getInstance(id, &instance)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if !exists {
		response.ITEM(w, r, http.StatusNotFound, et.Item{
			Ok:     true,
			Result: et.Json{"message": "instance not found"},
		})
		return
	}

	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	status := body.Str("status")
	err = instance.setStatus(Status(status))
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: instance.ToJson(),
	})
}

/**
* HttpSetParams
* @params w http.ResponseWriter, r *http.Request
**/
func (s *Agents) HttpSetParams(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var instance Agent
	exists, err := s.getInstance(id, &instance)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if !exists {
		response.ITEM(w, r, http.StatusNotFound, et.Item{
			Ok:     true,
			Result: et.Json{"message": "instance not found"},
		})
		return
	}

	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	jsonData := instance.ToJson()
	for k, v := range body {
		keys := strings.Split(k, "->")
		jsonData.SetNested(keys, v)
	}

	bt, err := jsonData.ToByte()
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	err = json.Unmarshal(bt, &instance)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: instance.ToJson(),
	})
}

/**
* HttpQuery
* @params w http.ResponseWriter, r *http.Request
**/
func (s *Agents) HttpQuery(w http.ResponseWriter, r *http.Request) {
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	query := body.Json("query")
	result, err := s.queryInstance(query)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEMS(w, r, http.StatusOK, result)
}
