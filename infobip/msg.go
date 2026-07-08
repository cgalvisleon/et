package infobip

import "github.com/cgalvisleon/et/envar"

var (
	MSG_BASE_URL_REQUIRED = "baseUrl is required"
	MSG_API_KEY_REQUIRED  = "apiKey is required"
	MSG_SENDER_REQUIRED   = "sender is required"
)

func init() {
	lang := envar.GetStr("LANG", "en")

	if lang == "es" {
		MSG_BASE_URL_REQUIRED = "la url base es requerida"
		MSG_API_KEY_REQUIRED = "la clave de api es requerida"
		MSG_SENDER_REQUIRED = "el remitente es requerido"
	}
}
