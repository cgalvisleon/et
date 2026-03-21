package workflow

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/instances"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/resilience"
	"github.com/cgalvisleon/et/response"
	"github.com/go-chi/chi/v5"
)

/**
* New
* @return (*WorkFlow, error)
**/
func New(store instances.Store) (*WorkFlow, error) {
	err := event.Load()
	if err != nil {
		return nil, err
	}

	resetInstance, err := resilience.New(store)
	if err != nil {
		return nil, err
	}

	result := &WorkFlow{
		Flows:      make(map[string]*Flow),
		Instances:  make(map[string]*Instance),
		Results:    make(map[string]et.Json),
		mu:         sync.Mutex{},
		resilience: resetInstance,
		isDebug:    envar.GetBool("DEBUG", false),
	}
	if store != nil {
		result.getInstance = store.Get
		result.setInstance = store.Set
		result.queryInstance = store.Query
	}

	return result, nil
}

/**
* HttpGet
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) HttpGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var instance Instance
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
func (s *WorkFlow) HttpState(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var instance Instance
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
func (s *WorkFlow) HttpSetParams(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var instance Instance
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
func (s *WorkFlow) HttpQuery(w http.ResponseWriter, r *http.Request) {
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
