package ettp

import (
	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/event"
	rt "github.com/cgalvisleon/et/router"
)

func (s *Server) initEvents() {
	err := event.Subscribe(rt.EVENT_SET_ROUTER, s.eventSetRouter)
	if err != nil {
		console.Error(err)
	}

	err = event.Subscribe(rt.EVENT_REMOVE_ROUTER, s.eventDeleteRouter)
	if err != nil {
		console.Error(err)
	}

	err = event.Subscribe(rt.EVENT_RESET_ROUTER, s.eventReset)
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
	id := data.ValStr("", "id")
	method := data.Str("method")
	path := data.Str("path")
	resolve := data.Str("resolve")
	header := data.Json("header")
	tpHeader := rt.ToTpHeader(data.Int("tp_header"))
	excludeHeader := data.ArrayStr("exclude_header")
	private := data.Bool("private")
	packageName := data.Str("package_name")
	s.setRouter(id, method, path, resolve, TpApiRest, header, tpHeader, excludeHeader, private, packageName, true)
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
		console.Alertf(`%s error:%s`, s.Name, err.Error())
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
	console.Debug("eventReset:", data.ToString())

	s.Reset()
}
