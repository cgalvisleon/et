package msg

import "github.com/cgalvisleon/et/envar"

var (
	MSG_ATRIB_REQUIRED    = "required attribute (%s)"
	ERR_NOT_CACHE_SERVICE = "caching service not available"
)

func init() {
	lang := envar.GetStr("LANG", "en")

	if lang == "es" {
		MSG_ATRIB_REQUIRED = "atributo requerido (%s)"
		ERR_NOT_CACHE_SERVICE = "no hay servicio de caching"
	}
}
