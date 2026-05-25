package ia

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

}
