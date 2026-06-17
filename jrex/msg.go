package jrex

import "github.com/cgalvisleon/et/config"

var (
	MSG_TAG_REQUIRED           = "tag is required, remember that is a unique identifier"
	MSG_INDEX_MODULE_NOT_FOUND = "index module not found"
	MSG_MODULE_NOT_FOUND       = "module not found: %s"
	MSG_STORE_IS_NIL           = "store is nil"
	MSG_ORIGIN_IS_NIL          = "origin store is nil"
	MSG_CODE_NOT_FOUND         = "code not found"
	MSG_JREX_IS_NIL            = "jrex is nil"
)

func init() {
	lang := config.GetStr("LANG", "en")

	if lang == "es" {
		MSG_TAG_REQUIRED = "tag es requerido, recuerda que es un identificador único"
		MSG_INDEX_MODULE_NOT_FOUND = "módulo index no encontrado"
		MSG_MODULE_NOT_FOUND = "módulo no encontrado: %s"
		MSG_STORE_IS_NIL = "store es nulo"
		MSG_ORIGIN_IS_NIL = "store de origen es nulo"
		MSG_CODE_NOT_FOUND = "código no encontrado"
		MSG_JREX_IS_NIL = "jrex es nulo"
	}
}
