package ettp

import (
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/router"
)

func (s *Server) initEvents() error {
	// V_1
	err := event.Subscribe(router.APIGATEWAY_SET_ROUTER, s.eventSetResolve)
	if err != nil {
		return err
	}

	err = event.Subscribe(router.APIGATEWAY_REMOVE_ROUTER, s.eventRemoveRouterById)
	if err != nil {
		return err
	}

	err = event.Subscribe(router.APIGATEWAY_RESET_ROUTER, s.eventReset)
	if err != nil {
		return err
	}

	// V_0
	err = event.Subscribe(router.APIGATEWAY_SET_RESOLVE, s.eventSetResolve)
	if err != nil {
		return err
	}

	err = event.Subscribe(router.APIGATEWAY_DELETE_RESOLVE, s.eventRemoveRouterById)
	if err != nil {
		return err
	}

	err = event.Subscribe(router.APIGATEWAY_RESET, s.eventReset)
	if err != nil {
		return err
	}

	return nil
}

/**
* eventSetResolve
* @param m event.Message
**/
func (s *Server) eventSetResolve(m event.Message) {
	if m.Myself {
		return
	}

	data := m.Data
	method := data.Str("method")
	path := data.Str("path")
	resolve := data.Str("resolve")
	header := data.Json("header")
	typeHeader := data.Int("tp_header")
	excludeHeader := data.ArrayStr("exclude_header")
	version := data.Int("version")
	packageName := data.Str("package_name")
	if StringToTypeRouter(data.Str("kind")) == TpStatic {
		mode := data.Str("mode")
		fallback := data.Str("fallback")
		_, err := s.SetStatic(path, resolve, mode, fallback, version, packageName, true)
		if err != nil {
			logs.Alertf(`eventSetRouter error:%s`, err.Error())
		}
		return
	}

	_, err := s.SetRouter(method, path, resolve, typeHeader, header, excludeHeader, version, packageName, true)
	if err != nil {
		logs.Alertf(`eventSetRouter error:%s`, err.Error())
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
		logs.Alertf(`eventRemoveRouterById error:%s`, err.Error())
	}
}

/**
* eventReset: Another replica reset (clear the routes: the services publish them again to every replica) or refreshes
* (keep: nothing to do). It does not publish the reset again: two replicas would answer each other forever
* @param m event.Message
**/
func (s *Server) eventReset(m event.Message) {
	if m.Myself || m.Data.Bool("keep") {
		return
	}

	s.clear()
}
