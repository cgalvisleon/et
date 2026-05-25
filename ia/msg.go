package ia

import "github.com/cgalvisleon/et/envar"

var (
	MSG_AGENT_NOT_FOUND             = "agente %s no encontrado"
	MSG_AGENT_ALREADY_EXISTS        = "agente %s ya existe"
	MSG_PARTICIPANT_NOT_FOUND       = "participant not found"
	MSG_SENDER_NOT_FOUND            = "sender not found"
	MSG_CONVERSATION_NOT_FOUND      = "conversation not found"
	MSG_SKILL_HTTP_NOT_SUPPORTED    = "skills must be registered programmatically, not via HTTP"
)

func init() {
	lang := envar.GetStr("LANG", "en")

	if lang == "es" {
		MSG_AGENT_NOT_FOUND = "agente %s no encontrado"
		MSG_AGENT_ALREADY_EXISTS = "agente %s ya existe"
		MSG_PARTICIPANT_NOT_FOUND = "participant not found"
		MSG_SENDER_NOT_FOUND = "sender not found"
		MSG_CONVERSATION_NOT_FOUND = "conversation not found"
	}
}
