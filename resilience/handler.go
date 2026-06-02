package resilience

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/response"
)

/**
* HttpGetInstace
* @params w http.ResponseWriter, r *http.Request
**/
func (s *Resilience) HttpGetInstace(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()
	instance, exists := s.GetInstance(id)
	if !exists {
		response.HTTPError(w, r, http.StatusNotFound, "instance not found")
		return
	}

	jsonData := instance.ToJson()
	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: jsonData,
	})
}

/**
* HttpState
* @params w http.ResponseWriter, r *http.Request
**/
func (s *Resilience) HttpState(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()
	instance, exists := s.GetInstance(id)
	if !exists {
		response.HTTPError(w, r, http.StatusNotFound, "instance not found")
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

	jsonData := instance.ToJson()
	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: jsonData,
	})
}

/**
* HttpSetInstance
* @params w http.ResponseWriter, r *http.Request
**/
func (s *Resilience) HttpSetInstance(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()
	instance, exists := s.GetInstance(id)
	if !exists {
		response.HTTPError(w, r, http.StatusNotFound, "instance not found")
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

	err = instance.save()
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	jsonDataStr := instance.ToJson()
	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: jsonDataStr,
	})
}

/**
* HttpQueryInstance
* @params w http.ResponseWriter, r *http.Request
**/
func (s *Resilience) HttpQueryInstance(w http.ResponseWriter, r *http.Request) {
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	query := body.Json("query")
	result, err := s.Query(query)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEMS(w, r, http.StatusOK, result)
}
