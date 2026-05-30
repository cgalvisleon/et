package jsql

import "github.com/cgalvisleon/et/envar"

var (
	MSG_CATALOG_NOT_FOUND    = "Catalog not found: %s"
	MSG_DB_IS_NIL            = "Database is nil"
	MSG_DB_NOT_FOUND         = "Database not found"
	MSG_DRIVER_NOT_FOUND     = "Driver not found"
	MSG_INSTANCE_REQUIRED_ID = "Instance required id"
	MSG_INVALID_FROM         = "Invalid from: %s"
	MSG_KEYS_REQUIRED        = "Keys is required"
	MSG_MODEL_NOT_FOUND      = "Model not found: %s"
	MSG_NAME_REQUIRED        = "Name is required"
	MSG_REQUIRED_FIELD       = "Required field %s"
	MSG_ROLLBACK_ERROR       = "Error rolling back transaction: %v"
	MSG_SCHEMA_NOT_FOUND     = "Schema not found: %s"
	MSG_SCHEMA_REQUIRED      = "Schema is required"
	MSG_SELECTS_REQUIRED     = "Selects is required"
	MSG_TO_MODEL_REQUIRED    = "To model is required"
)

func init() {
	lang := envar.GetStr("LANG", "en")

	if lang == "es" {
		MSG_CATALOG_NOT_FOUND    = "Catálogo no encontrado: %s"
		MSG_DB_IS_NIL            = "Base de datos es nula"
		MSG_DB_NOT_FOUND         = "Base de datos no encontrada"
		MSG_DRIVER_NOT_FOUND     = "Driver no encontrado"
		MSG_INSTANCE_REQUIRED_ID = "Instancia requiere id"
		MSG_INVALID_FROM         = "From inválido: %s"
		MSG_KEYS_REQUIRED        = "Claves son requeridas"
		MSG_MODEL_NOT_FOUND      = "Modelo no encontrado: %s"
		MSG_NAME_REQUIRED        = "Nombre es requerido"
		MSG_REQUIRED_FIELD       = "Campo requerido %s"
		MSG_ROLLBACK_ERROR       = "Error al hacer rollback de la transacción: %v"
		MSG_SCHEMA_NOT_FOUND     = "Esquema no encontrado: %s"
		MSG_SCHEMA_REQUIRED      = "Esquema es requerido"
		MSG_SELECTS_REQUIRED     = "Selecciones son requeridas"
		MSG_TO_MODEL_REQUIRED    = "Modelo destino es requerido"
	}
}
