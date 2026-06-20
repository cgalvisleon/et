package jtenant

import "github.com/cgalvisleon/et/config"

var (
	MSG_TENANT_NOT_FOUND = "tenant not found"
)

func init() {
	lang := config.GetStr("LANG", "en")
	if lang == "es" {
		MSG_TENANT_NOT_FOUND = "tenant no encontrado"
	}
}
