package ettp

import (
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	rt "github.com/cgalvisleon/et/router"
)

func (s *Server) initEvents() {
	err := event.Subscribe(rt.EVENT_SET_ROUTER, s.eventSetRouter)
	if err != nil {
		logs.Error(err)
	}

	err = event.Subscribe(rt.EVENT_REMOVE_ROUTER, s.eventDeleteRouter)
	if err != nil {
		logs.Error(err)
	}

	err = event.Subscribe(rt.EVENT_RESET_ROUTER, s.eventReset)
	if err != nil {
		logs.Error(err)
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
	tpHeader := rt.ToTpHeader(data.Int("tp_header"))
	excludeHeader := data.ArrayStr("exclude_header")
	private := data.Bool("private")
	packageName := data.Str("package_name")
	s.setRouter(method, path, resolve, TpApiRest, header, tpHeader, excludeHeader, private, packageName, true)
}

/**
* eventDeleteRouter
* @param m event.Message
**/
func (s *Server) eventDeleteRouter(m event.Message) {
	if m.Myself {
		return
	}

	data := m.Data
	id := data.Str("id")
	err := s.DeleteRouteById(id, true)
	if err != nil {
		logs.Error(err)
	}
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
	logs.Debug(packageName, "eventReset:", data.ToString())

	s.Reset()
}
