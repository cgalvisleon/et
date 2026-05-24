package ia

import (
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
)

const (
	EVENT_IA_SET              = "ia:set"
	EVENT_IA_DELETE           = "ia:delete"
	EVENT_AGENT_SET           = "ia:agent:set"
	EVENT_AGENT_DELETE        = "ia:agent:delete"
	EVENT_MESSAGE_SET         = "ia:message:set"
	EVENT_MESSAGE_DELETE      = "ia:message:delete"
	EVENT_CONVERSATION_SET    = "ia:conversation:set"
	EVENT_CONVERSATION_DELETE = "ia:conversation:delete"
	EVENT_PARTICIPANT_SET     = "ia:participant:set"
	EVENT_PARTICIPANT_DELETE  = "ia:participant:delete"
)

func (s *Ia) eventInit() {
	event.Subscribe(EVENT_AGENT_SET, func(e event.Message) {
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
