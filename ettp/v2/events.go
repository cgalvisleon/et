package ettp

import (
	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/router"
)

func (s *Server) initEvents() error {
	err := event.Subscribe(router.EVENT_SET_ROUTER, s.eventSetRouter)
	if err != nil {
		return err
	}

	err = event.Subscribe(router.EVENT_REMOVE_ROUTER, s.eventRemoveRouterById)
	if err != nil {
		return err
	}

	err = event.Subscribe(router.EVENT_RESET_ROUTER, s.eventReset)
	if err != nil {
		return err
	}

	return nil
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
	typeHeader := data.Int("type_header")
	header := data.Json("header")
	excludeHeader := data.ArrayStr("exclude_header")
	version := data.Int("version")
	private := data.Bool("private")
	packageName := data.Str("package_name")
	_, err := s.SetRouter(method, path, resolve, typeHeader, header, excludeHeader, version, private, packageName, true)
	if err != nil {
		console.Alertf(`eventSetRouter error:%s`, err.Error())
	}
}

/**
* eventRemoveRouterById
* @param m event.Message
**/
func (s *Server) eventRemoveRouterById(m event.Message) {
	if m.Myself {
		return
	}

	data := m.Data
	id := data.Str("id")
	err := s.RemoveRouterById(id, true)
	if err != nil {
		console.Alertf(`eventRemoveRouterById error:%s`, err.Error())
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

	s.Reset()
}
