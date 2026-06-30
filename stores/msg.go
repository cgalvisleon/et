package stores

import "github.com/cgalvisleon/et/envar"

var (
	MSG_RECORD_EXISTS      = "record already exists"
	MSG_RECORD_NOT_CREATED = "record not created"
	MSG_RECORD_IS_NOT_ITEM = "record is not an item"
)

func init() {
	lang := envar.GetStr("LANG", "en")

	if lang == "es" {
		MSG_RECORD_EXISTS = "registro ya existe"
		MSG_RECORD_NOT_CREATED = "registro no creado"
		MSG_RECORD_IS_NOT_ITEM = "registro no es un item"
	}
}
