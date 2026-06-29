package dt

import "github.com/cgalvisleon/et/envar"

var (
	MSG_VALUE_IS_NOT_BYTE           = "value is not a byte"
	MSG_VALUE_IS_NOT_STRING         = "value is not a string"
	MSG_VALUE_IS_NOT_INT            = "value is not an int"
	MSG_VALUE_IS_NOT_INT64          = "value is not an int64"
	MSG_VALUE_IS_NOT_FLOAT          = "value is not a float"
	MSG_VALUE_IS_NOT_BOOL           = "value is not a bool"
	MSG_VALUE_IS_NOT_TIME           = "value is not a time"
	MSG_VALUE_IS_NOT_DURATION       = "value is not a duration"
	MSG_VALUE_IS_NOT_ARRAY          = "value is not an array"
	MSG_VALUE_IS_NOT_JSON           = "value is not a json"
	MSG_VALUE_IS_NOT_ITEM           = "value is not an item"
	MSG_VALUE_IS_NOT_ITEMS          = "value is not an items"
	MSG_VALUE_IS_NOT_LIST           = "value is not a list"
	MSG_VALUE_IS_NOT_ARRAY_STR      = "value is not an array of strings"
	MSG_VALUE_IS_NOT_ARRAY_INT      = "value is not an array of ints"
	MSG_VALUE_IS_NOT_ARRAY_INT64    = "value is not an array of int64s"
	MSG_VALUE_IS_NOT_ARRAY_FLOAT    = "value is not an array of floats"
	MSG_VALUE_IS_NOT_ARRAY_TIME     = "value is not an array of times"
	MSG_VALUE_IS_NOT_ARRAY_DURATION = "value is not an array of durations"
	MSG_VALUE_IS_NOT_ARRAY_JSON     = "value is not an array of jsons"
)

func init() {
	lang := envar.GetStr("LANG", "en")
	if lang == "es" {
		MSG_VALUE_IS_NOT_BYTE = "valor no es un byte"
		MSG_VALUE_IS_NOT_STRING = "valor no es un string"
		MSG_VALUE_IS_NOT_INT = "valor no es un int"
		MSG_VALUE_IS_NOT_INT64 = "valor no es un int64"
		MSG_VALUE_IS_NOT_FLOAT = "valor no es un float"
		MSG_VALUE_IS_NOT_BOOL = "valor no es un bool"
		MSG_VALUE_IS_NOT_TIME = "valor no es un time"
		MSG_VALUE_IS_NOT_DURATION = "valor no es un duration"
		MSG_VALUE_IS_NOT_ARRAY = "valor no es un array"
		MSG_VALUE_IS_NOT_JSON = "valor no es un json"
		MSG_VALUE_IS_NOT_ITEM = "valor no es un item"
		MSG_VALUE_IS_NOT_ITEMS = "valor no es un items"
		MSG_VALUE_IS_NOT_LIST = "valor no es un list"
		MSG_VALUE_IS_NOT_ARRAY_STR = "valor no es un array de strings"
		MSG_VALUE_IS_NOT_ARRAY_INT = "valor no es un array de ints"
		MSG_VALUE_IS_NOT_ARRAY_INT64 = "valor no es un array de int64s"
		MSG_VALUE_IS_NOT_ARRAY_FLOAT = "valor no es un array de floats"
		MSG_VALUE_IS_NOT_ARRAY_TIME = "valor no es un array de times"
		MSG_VALUE_IS_NOT_ARRAY_DURATION = "valor no es un array de durations"
		MSG_VALUE_IS_NOT_ARRAY_JSON = "valor no es un array de jsons"
	}
}
