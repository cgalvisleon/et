package envar

var (
	MSG_ATRIB_REQUIRED = "required attribute (%s)"
)

func init() {
	lang := GetStr("LANG", "en")

	if lang == "es" {
		MSG_ATRIB_REQUIRED = "atributo requerido (%s)"
	}
}
