package et

import "github.com/cgalvisleon/et/envar"

var (
	MSG_FIELD_NOT_FOUND                = "field not found"
	MSG_DATA_NOT_FOUND                 = "data not found"
	MSG_INDEX_OUT_OF_RANGE             = "index out of range"
	MSG_FAILED_TO_UNMARSHAL_JSON_VALUE = "failed to unmarshal JSON value:%s"
)

func init() {
	lang := envar.GetStr("LANG", "en")
	if lang == "es" {
		MSG_FIELD_NOT_FOUND = "campo no encontrado"
		MSG_DATA_NOT_FOUND = "datos no encontrados"
		MSG_INDEX_OUT_OF_RANGE = "índice fuera de rango"
		MSG_FAILED_TO_UNMARSHAL_JSON_VALUE = "no se pudo deserializar el JSON value:%s"
	}
}
