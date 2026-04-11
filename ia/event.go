package ia

import (
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
)

const (
	packageName             = "ia"
	EVENT_AGENT_SET_CONTEXT = "agent:set_context"
	EVENT_AGENT_SET_MODEL   = "agent:set_model"
)

func (s *Agents) eventInit() {
	event.Subscribe(EVENT_AGENT_SET_CONTEXT, func(e event.Message) {
		if e.Myself {
			return
		}

		data := e.Data
		tag := data.String("tag")
		context := data.String("context")
		_, err := s.setContext(tag, context)
		if err != nil {
			logs.Error(err)
		}

		logs.Log(packageName, "eventInit", data)
	})

	event.Subscribe(EVENT_AGENT_SET_MODEL, func(e event.Message) {
		if e.Myself {
			return
		}

		data := e.Data
		tag := data.String("tag")
		model := data.String("model")
		_, err := s.setModel(tag, model)
		if err != nil {
			logs.Error(err)
		}

		logs.Log(packageName, "eventInit", data)
	})
}
