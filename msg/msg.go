package msg

import "github.com/cgalvisleon/et/envar"

var (
	MSG_ATRIB_REQUIRED                 = "required attribute (%s)"
	MSG_FAILED_TO_UNMARSHAL_JSON_VALUE = "failed to unmarshal JSON value:%s"
	MSG_NOT_CACHE_SERVICE              = "caching service not available"
	MSG_RECORD_NOT_FOUND               = "record not found"
	MSG_TOKEN_INVALID                  = "token invalid"
	MSG_TOKEN_INVALID_ATRIB            = "token invalid, attribute (%s)"
	MSG_TOKEN_EXPIRED                  = "token expired"
	MSG_REQUIRED_INVALID               = "request invalid"
	MSG_ERR_INVALID_CLAIM              = "formato token invalid"
	MSG_ERR_AUTORIZATION               = "invalid authorization"
	MSG_ERR_NOT_CONNECT                = "not connect"
	MSG_ERR_CHANNEL_REQUIRED           = "channel required"
	MSG_PACKAGE_NOT_FOUND              = "package not found"
	MSG_METHOD_NOT_FOUND               = "method not found"
	MSG_ERR_PACKAGE_NOT_FOUND          = "package not found"
	MSG_ERR_METHOD_NOT_FOUND           = "method not found"
	MSG_KEY_REQUIRED                   = "key required"
)

func init() {
	lang := envar.GetStr("LANG", "en")

	if lang == "es" {
		MSG_ATRIB_REQUIRED = "atributo requerido (%s)"
		MSG_FAILED_TO_UNMARSHAL_JSON_VALUE = "no se pudo deserializar el JSON value:%s"
		MSG_NOT_CACHE_SERVICE = "no hay servicio de caching"
		MSG_RECORD_NOT_FOUND = "registro no encontrado"
		MSG_TOKEN_INVALID = "token invalido"
		MSG_TOKEN_INVALID_ATRIB = "token invalido, atributo (%s)"
		MSG_TOKEN_EXPIRED = "token expirado"
		MSG_REQUIRED_INVALID = "solicitud invalida"
		MSG_ERR_INVALID_CLAIM = "formato token invalido"
		MSG_ERR_AUTORIZATION = "invalid autorization"
		MSG_ERR_NOT_CONNECT = "no se pudo conectar"
		MSG_ERR_CHANNEL_REQUIRED = "canal requerido"
		MSG_PACKAGE_NOT_FOUND = "paquete no encontrado"
		MSG_METHOD_NOT_FOUND = "metodo no encontrado"
		MSG_ERR_PACKAGE_NOT_FOUND = "paquete no encontrado"
		MSG_KEY_REQUIRED = "key requerida"
	}
}
