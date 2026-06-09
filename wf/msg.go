package workflow

import "github.com/cgalvisleon/et/envar"

var (
	MSG_STEP_IS_FUNCTION           = "step is function"
	MSG_WORKFLOW_STORE_IS_NIL      = "workflow store is nil"
	MSG_STEP_NOT_FOUND             = "step not found"
	MSG_STEP_DEFINITION_IS_UNKNOWN = "step definition is unknown"
	MSG_STEP_STATUS_INVALID        = "step status is invalid"
	MSG_FLOW_NOT_FOUND             = "flow not found"
)

func init() {
	lang := envar.GetStr("LANG", "en")

	if lang == "es" {
		MSG_STEP_IS_FUNCTION = "step is function"
		MSG_WORKFLOW_STORE_IS_NIL = "workflow store es nulo"
		MSG_STEP_NOT_FOUND = "step no encontrado"
		MSG_STEP_DEFINITION_IS_UNKNOWN = "definición de step desconocida"
		MSG_STEP_STATUS_INVALID = "status de step invalido"
		MSG_FLOW_NOT_FOUND = "flow no encontrado"
	}
}
