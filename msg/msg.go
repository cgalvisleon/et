package msg

import "github.com/cgalvisleon/et/envar"

var (
	MSG_ATRIB_REQUIRED                 = "required attribute (%s)"
	MSG_FAILED_TO_UNMARSHAL_JSON_VALUE = "failed to unmarshal JSON value:%s"
	MSG_NOT_CACHE_SERVICE              = "caching service not available"
)

func init() {
	lang := envar.GetStr("LANG", "en")

	if lang == "es" {
		MSG_ATRIB_REQUIRED = "atributo requerido (%s)"
		MSG_FAILED_TO_UNMARSHAL_JSON_VALUE = "no se pudo deserializar el JSON value:%s"
		MSG_NOT_CACHE_SERVICE = "no hay servicio de caching"
	}
}
