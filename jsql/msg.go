package jsql

import "github.com/cgalvisleon/et/envar"

var (
	MSG_INVALID_FROM    = "Invalid from: %s"
	MSG_MODEL_NOT_FOUND = "Model not found: %s"
)

func init() {
	lang := envar.GetStr("LANG", "en")

	if lang == "es" {
		MSG_INVALID_FROM = "From inválido: %s"
		MSG_MODEL_NOT_FOUND = "Modelo no encontrado: %s"
	}

}
