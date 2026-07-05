package aws

import "github.com/cgalvisleon/et/envar"

var (
	MSG_REGION_REQUIRED = "region is required"
	MSG_KEY_ID_REQUIRED = "keyId is required"
	MSG_SECRET_REQUIRED = "secret is required"
	MSG_TOKEN_REQUIRED  = "token is required"
)

func init() {
	lang := envar.GetStr("LANG", "en")

	if lang == "es" {
		MSG_REGION_REQUIRED = "la región es requerida"
		MSG_KEY_ID_REQUIRED = "la clave de acceso es requerida"
		MSG_SECRET_REQUIRED = "la clave de secret es requerida"
		MSG_TOKEN_REQUIRED = "el token es requerido"
	}
}
