package jrex

import "github.com/cgalvisleon/et/config"

var (
	MSG_TAG_REQUIRED           = "tag is required, remember that is a unique identifier"
	MSG_INDEX_MODULE_NOT_FOUND = "index module not found"
)

func init() {
	lang := config.GetStr("LANG", "en")

	if lang == "es" {
		MSG_TAG_REQUIRED = "tag es requerido, recuerda que es un identificador único"
		MSG_INDEX_MODULE_NOT_FOUND = "módulo index no encontrado"
	}
}
