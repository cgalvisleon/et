package config

var (
	MSG_ATRIB_REQUIRED      = "required attribute (%s)"
	MSG_CONFIG_STORE_IS_NIL = "config store is nil"
	MSG_CONFIG_NOT_LOADED   = "config not loaded"
)

func init() {
	lang := GetStr("LANG", "en")

	if lang == "es" {
		MSG_ATRIB_REQUIRED = "atributo requerido (%s)"
		MSG_CONFIG_STORE_IS_NIL = "config store es nulo"
		MSG_CONFIG_NOT_LOADED = "config no cargado"
	}
}
