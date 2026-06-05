package stores

import (
	"github.com/cgalvisleon/et/envar"
)

var (
	MSG_RECORD_EXISTS = "record already exists"
)

func init() {
	lang := envar.GetStr("LANG", "en")

	if lang == "es" {
		MSG_RECORD_EXISTS = "registro ya existe"
	}
}
