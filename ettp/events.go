package ettp

import (
	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/event"
	rt "github.com/cgalvisleon/et/router"
)

func (s *Server) initEvents() {
	err := event.Subscribe(rt.APIGATEWAY_SET, s.eventSetResolve)
	if err != nil {
		console.Error(err)
	}

	err = event.Subscribe(rt.APIGATEWAY_DELETE, s.eventDeleteResolve)
	if err != nil {
		console.Error(err)
	}
}

func (s *Server) eventSetResolve(m event.EvenMessage) {
	data := m.Data
	fromId := data.Str("from_id")
	if fromId == s.Id {
		return
	}

	method := data.Str("method")
	path := data.Str("path")
	resolve := data.Str("resolve")
	header := data.Json("header")
	tpHeader := rt.ToTpHeader(data.Int("tp_header"))
	excludeHeader := data.ArrayStr("exclude_header")
	private := data.Bool("private")
	packageName := data.Str("package_name")
	id := data.ValStr("-1", "_id")
	_, err := s.SetResolve(private, id, method, path, resolve, header, tpHeader, excludeHeader, packageName, true)
	if err != nil {
		console.Alertf(`%s error:%s`, ServiceName, err.Error())
	}
}

func (s *Server) eventDeleteResolve(m event.EvenMessage) {
	data := m.Data
	fromId := data.Str("from_id")
	if fromId == s.Id {
		return
	}

	id := data.Str("_id")
	err := s.DeleteRouteById(id, true)
	if err != nil {
		console.Alertf(`%s error:%s`, ServiceName, err.Error())
	}
}
