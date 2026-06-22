package jia

import "github.com/cgalvisleon/et/envar"

var (
	MSG_AGENT_NOT_FOUND          = "agent not found"
	MSG_AGENT_ALREADY_EXISTS     = "agent already exists"
	MSG_AGENT_UPDATED            = "agente actualizado"
	MSG_PARTICIPANT_NOT_FOUND    = "participant not found"
	MSG_SENDER_NOT_FOUND         = "sender not found"
	MSG_CONVERSATION_NOT_FOUND   = "conversation not found"
	MSG_SKILL_HTTP_NOT_SUPPORTED = "skills must be registered programmatically, not via HTTP"
	MSG_ATRIB_REQUIRED           = "%s is required"
	MSG_INVALID_FROM             = "Invalid from: %s"
	MSG_PARTICIPANT_CREATED      = "participant created"
	MSG_PARTICIPANT_DELETED      = "participant deleted"
	MSG_PARTICIPANT_UPDATED      = "participant updated"
	MSG_STORE_IS_NIL             = "store is nil"
	MSG_IA_NOT_FOUND             = "ia not found"
)

func init() {
	lang := envar.GetStr("LANG", "en")

	if lang == "es" {
		MSG_AGENT_NOT_FOUND = "agent not found"
		MSG_AGENT_ALREADY_EXISTS = "agent already exists"
		MSG_AGENT_UPDATED = "agente actualizado"
		MSG_PARTICIPANT_NOT_FOUND = "participant not found"
		MSG_SENDER_NOT_FOUND = "sender not found"
		MSG_CONVERSATION_NOT_FOUND = "conversation not found"
		MSG_INVALID_FROM = "Invalid from: %s"
		MSG_PARTICIPANT_CREATED = "participant creado"
		MSG_PARTICIPANT_DELETED = "participante eliminado"
		MSG_PARTICIPANT_UPDATED = "participante actualizado"
		MSG_STORE_IS_NIL = "store es nulo"
		MSG_IA_NOT_FOUND = "ia no encontrada"
	}
}
