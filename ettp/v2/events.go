package ettp

import (
	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/event"
)

const EVENT_SET_SOLVER = "event:set:solver"
const EVENT_REMOVE_SOLVER = "event:remove:solver"
const EVENT_RESET = "event:reset"

func (s *Server) initEvents() {
	err := event.Subscribe(EVENT_SET_SOLVER, s.eventSetRouter)
	if err != nil {
		console.Error(err)
	}

	err = event.Subscribe(EVENT_REMOVE_SOLVER, s.eventRemoveSolverById)
	if err != nil {
		console.Error(err)
	}

	err = event.Subscribe(EVENT_RESET, s.eventReset)
	if err != nil {
		console.Error(err)
	}
}

/**
* eventSetRouter
* @param m event.Message
**/
func (s *Server) eventSetRouter(m event.Message) {
	if m.Myself {
		return
	}

	data := m.Data
	method := data.Str("method")
	path := data.Str("path")
	resolve := data.Str("resolve")
	header := data.Json("header")
	excludeHeader := data.ArrayStr("exclude_header")
	version := data.Int("version")
	packageName := data.Str("package_name")
	_, err := s.SetSolver(method, path, resolve, header, excludeHeader, version, packageName)
	if err != nil {
		console.Alertf(`eventSetRouter error:%s`, err.Error())
	}

	console.Log("eventSetRouter", data.ToString())
}

/**
* eventRemoveSolverById
* @param m event.Message
**/
func (s *Server) eventRemoveSolverById(m event.Message) {
	if m.Myself {
		return
	}

	data := m.Data
	id := data.Str("id")
	err := s.RemoveSolverById(id, true)
	if err != nil {
		console.Alertf(`eventRemoveSolverById error:%s`, err.Error())
	}

	console.Log("eventRemoveSolverById", data.ToString())
}

/**
* eventReset
* @param m event.Message
**/
func (s *Server) eventReset(m event.Message) {
	if m.Myself {
		return
	}

	data := m.Data
	s.Reset()

	console.Log("eventReset", data.ToString())
}
