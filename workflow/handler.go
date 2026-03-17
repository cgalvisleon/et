package workflow

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/response"
	"github.com/go-chi/chi/v5"
)

/**
* New
* @return (*WorkFlow, error)
**/
func New() (*WorkFlow, error) {
	err := event.Load()
	if err != nil {
		return nil, err
	}

	result := &WorkFlow{
		Flows:     make(map[string]*Flow),
		Instances: make(map[string]*Instance),
		Results:   make(map[string]et.Json),
		mu:        sync.Mutex{},
		isDebug:   envar.GetBool("DEBUG", false),
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
* HttpGetInstance
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
	err = instance.setStatus(FlowStatus(status))
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
* HttpGetInstance
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
