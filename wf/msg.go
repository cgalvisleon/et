package workflow

import "github.com/cgalvisleon/et/envar"

var (
	MSG_STEP_IS_FUNCTION              = "step is function"
	MSG_WORKFLOW_STORE_IS_NIL         = "workflow store is nil"
	MSG_STEP_NOT_FOUND                = "step not found"
	MSG_STEP_DEFINITION_IS_UNKNOWN    = "step definition is unknown"
	MSG_STEP_STATUS_INVALID           = "step status is invalid"
	MSG_FLOW_NOT_FOUND                = "flow not found"
	MSG_INSTANCE_NOT_FOUND            = "instance not found"
	MSG_INSTANCE_STATUS               = "Instancia:%s tag:%s status:%s step:%d"
	MSG_INSTANCE_ERROR                = "Instancia:%s tag:%s step:%d error:%s"
	MSG_INSTANCE_ALREADY_DONE         = "instance already done"
	MSG_INSTANCE_CONNECTION_NOT_FOUND = "instance connection not found"
	MSG_INSTANCE_INVALID_CONNECTION   = "instance invalid connection"
	MSG_INSTANCE_INVALID_STEP         = "instance invalid step"
	MSG_STEPER_NOT_FOUND              = "steper not found"
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
		MSG_INSTANCE_NOT_FOUND = "instance no encontrado"
		MSG_INSTANCE_STATUS = "Instancia:%s tag:%s status:%s step:%d"
		MSG_INSTANCE_ERROR = "Instancia:%s tag:%s step:%d error:%s"
		MSG_INSTANCE_ALREADY_DONE = "Instancia ya finalizada"
		MSG_INSTANCE_CONNECTION_NOT_FOUND = "Conexión de instance no encontrada"
		MSG_INSTANCE_INVALID_CONNECTION = "Conexión de instance invalido"
		MSG_INSTANCE_INVALID_STEP = "Step de instance invalido"
		MSG_STEPER_NOT_FOUND = "Steper no encontrado"
	}
}
