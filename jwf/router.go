package jwf

import (
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/response"
)

const (
	GET    = "GET"
	POST   = "POST"
	PUT    = "PUT"
	DELETE = "DELETE"
)

type Router interface {
	Protect(method, path string, handler func(http.ResponseWriter, *http.Request))
}

func (s *WorkFlow) LoadRouter(r Router) {
	r.Protect(GET, "/steps/{id}", s.httpGetStep)
	r.Protect(POST, "/steps", s.httpNewStep)
	r.Protect(PUT, "/steps/{id}", s.httpUpdateStep)
	r.Protect(PUT, "/steps/{id}/definition", s.httpSetDefinitionStep)
	r.Protect(DELETE, "/steps/{id}", s.httpDeleteStep)
	// Flows
	r.Protect(GET, "/flows/{tag}", s.httpGetFlow)
	r.Protect(POST, "/flows", s.httpSetFlow)
	r.Protect(PUT, "/flows/{tag}", s.httpStatusFlow)
	r.Protect(DELETE, "/flows/{tag}", s.httpDeleteFlow)
	// Instances
	r.Protect(GET, "/instances/{id}", s.httpGetInstance)
	r.Protect(DELETE, "/instances/{id}", s.httpDeleteInstance)
	r.Protect(POST, "/instances", s.httpRunInstance)
}

/**
* HttpGetSteps
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) httpGetStep(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()
	step, exists := s.getStep(id)
	if !exists {
		response.HTTPError(w, r, http.StatusBadRequest, MSG_STEP_NOT_FOUND)
		return
	}

	response.JSON(w, r, http.StatusOK, step.ToJson())
}

/**
* httpNewStep
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) httpNewStep(w http.ResponseWriter, r *http.Request) {
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	kind := Kind(body.Str("kind"))
	if !KindList[kind] {
		response.HTTPError(w, r, http.StatusBadRequest, "invalid kind")
		return
	}

	tag := body.Str("tag")
	version := body.Str("version")
	title := body.Str("title")
	userId := request.UserId(r)

	step := s.newStep(kind, tag, version, title, userId)
	response.JSON(w, r, http.StatusOK, step.ToJson())
}

/**
* httpUpdateStep
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) httpUpdateStep(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	step, exists := s.getStep(id)
	if !exists {
		response.HTTPError(w, r, http.StatusBadRequest, MSG_STEP_NOT_FOUND)
		return
	}

	userId := request.UserId(r)

	err = step.put(body, userId)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.JSON(w, r, http.StatusOK, step.ToJson())
}

/**
* httpSetDefinitionStep
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) httpSetDefinitionStep(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	step, exists := s.getStep(id)
	if !exists {
		response.HTTPError(w, r, http.StatusBadRequest, MSG_STEP_NOT_FOUND)
		return
	}

	definition := body.Str("definition")
	userId := request.UserId(r)

	err = step.setDefinition(definition, userId)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.JSON(w, r, http.StatusOK, step.ToJson())
}

/**
* httpDeleteStep
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) httpDeleteStep(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()

	err := s.deleteStep(id)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.JSON(w, r, http.StatusOK, et.Json{
		"message": "ok",
	})
}

/**
* httpGetFlow
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) httpGetFlow(w http.ResponseWriter, r *http.Request) {
}

/**
* httpSetFlow
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) httpSetFlow(w http.ResponseWriter, r *http.Request) {}

/**
* httpStatusFlow
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) httpStatusFlow(w http.ResponseWriter, r *http.Request) {
}

/**
* httpDeleteFlow
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) httpDeleteFlow(w http.ResponseWriter, r *http.Request) {

}

/**
* httpGetInstance
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) httpGetInstance(w http.ResponseWriter, r *http.Request) {
}

/**
* httpDeleteInstance
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) httpDeleteInstance(w http.ResponseWriter, r *http.Request) {
}

/**
* httpRunInstance
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) httpRunInstance(w http.ResponseWriter, r *http.Request) {
}
