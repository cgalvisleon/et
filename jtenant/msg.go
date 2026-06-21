package jtenant

import "github.com/cgalvisleon/et/config"

var (
	MSG_TENANT_NOT_FOUND = "tenant not found"
	MSG_STORE_IS_NIL     = "store is nil"
)

func init() {
	lang := config.GetStr("LANG", "en")
	if lang == "es" {
		MSG_TENANT_NOT_FOUND = "tenant no encontrado"
		MSG_STORE_IS_NIL = "store es nulo"
	}
}
