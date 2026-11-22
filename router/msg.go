package router

import "github.com/cgalvisleon/et/envar"

// Messages of the route contract (Doc), its validation and the MCP server
var (
	MSG_SCHEMA_REQUIRED    = "%s is required"
	MSG_SCHEMA_TYPE        = "%s must be %s"
	MSG_SCHEMA_ENUM        = "%s must be one of: %s"
	MSG_SCHEMA_INVALID     = "invalid request: %s"
	MSG_MCP_PARSE_ERROR    = "parse error"
	MSG_MCP_INVALID        = "invalid request"
	MSG_MCP_METHOD         = "method not found: %s"
	MSG_MCP_TOOL_NOT_FOUND = "tool not found: %s"
	MSG_MCP_TOOL_PARAMS    = "invalid tool arguments: %s"
	MSG_MCP_ORIGIN         = "origin not allowed: %s"
	MSG_PATH_PARAM         = "Path param %s"
)

func init() {
	lang := envar.GetStr("LANG", "en")

	if lang == "es" {
		MSG_SCHEMA_REQUIRED = "%s es obligatorio"
		MSG_SCHEMA_TYPE = "%s debe ser %s"
		MSG_SCHEMA_ENUM = "%s debe ser uno de: %s"
		MSG_SCHEMA_INVALID = "petición inválida: %s"
		MSG_MCP_PARSE_ERROR = "error de sintaxis"
		MSG_MCP_INVALID = "petición inválida"
		MSG_MCP_METHOD = "método no encontrado: %s"
		MSG_MCP_TOOL_NOT_FOUND = "herramienta no encontrada: %s"
		MSG_MCP_TOOL_PARAMS = "argumentos inválidos de la herramienta: %s"
		MSG_MCP_ORIGIN = "origen no permitido: %s"
		MSG_PATH_PARAM = "Parámetro de ruta %s"
	}
}
