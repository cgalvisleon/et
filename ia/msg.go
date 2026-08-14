package ia

import "github.com/cgalvisleon/et/envar"

var (
	MSG_STORE_IS_NIL      = "store is nil"
	MSG_KB_NOT_FOUND      = "knowledge base not found"
	MSG_FACT_NOT_FOUND    = "fact not found"
	MSG_STATEMENT_EMPTY   = "statement is required"
	MSG_KB_ID_REQUIRED    = "knowledge base id is required"
	MSG_MODEL_NOT_TRAINED = "model has no trained weights"
	MSG_MODEL_NOT_FOUND   = "unknown store collection: %s"
)

func init() {
	lang := envar.GetStr("LANG", "en")

	if lang == "es" {
		MSG_STORE_IS_NIL = "store es nulo"
		MSG_KB_NOT_FOUND = "base de conocimiento no encontrada"
		MSG_FACT_NOT_FOUND = "hecho no encontrado"
		MSG_STATEMENT_EMPTY = "el enunciado es requerido"
		MSG_KB_ID_REQUIRED = "el id de la base de conocimiento es requerido"
		MSG_MODEL_NOT_TRAINED = "el modelo no tiene pesos entrenados"
		MSG_MODEL_NOT_FOUND = "coleccion de almacenamiento desconocida: %s"
	}
}
