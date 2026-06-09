package workflow

import (
	"net/http"
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
	r.Protect(POST, "/steps", s.httpSetStep)
	r.Protect(PUT, "/steps/{id}", s.httpStatusStep)
	r.Protect(DELETE, "/steps/{id}", s.httpDeleteStep)
}

/**
* HttpGetSteps
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) httpGetStep(w http.ResponseWriter, r *http.Request) {
}

/**
* httpSetStep
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) httpSetStep(w http.ResponseWriter, r *http.Request) {
}

/**
* httpStatusStep
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) httpStatusStep(w http.ResponseWriter, r *http.Request) {
}

/**
* httpDeleteStep
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) httpDeleteStep(w http.ResponseWriter, r *http.Request) {
}
