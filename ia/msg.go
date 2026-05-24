package ia

import "github.com/cgalvisleon/et/envar"

var (
	MSG_AGENT_NOT_FOUND      = "agente %s no encontrado"
	MSG_AGENT_ALREADY_EXISTS = "agente %s ya existe"
)

func init() {
	lang := envar.GetStr("LANG", "en")

	if lang == "es" {
		MSG_AGENT_NOT_FOUND = "agente %s no encontrado"
		MSG_AGENT_ALREADY_EXISTS = "agente %s ya existe"
	}
}
