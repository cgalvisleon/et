package ia

import "github.com/cgalvisleon/et/envar"

var (
	MSG_TENANT_ID_REQUIRED     = "tenantId is required"
	MSG_PROJECT_ID_REQUIRED    = "projectId is required"
	MSG_NAME_REQUIRED          = "name is required"
	MSG_SOURCE_REQUIRED        = "source is required"
	MSG_FILE_REQUIRED          = "file is required"
	MSG_QUERY_REQUIRED         = "query is required"
	MSG_QUESTION_REQUIRED      = "question is required"
	MSG_DB_REQUIRED            = "db is required"
	MSG_API_KEY_REQUIRED       = "OPENAI_API_KEY is required"
	MSG_UNSUPPORTED_SOURCE     = "unsupported source: %s"
	MSG_DOCUMENT_NOT_FOUND     = "document not found"
	MSG_DOCUMENT_EMPTY         = "document has no extractable text"
	MSG_CONVERSATION_NOT_FOUND = "conversation not found"
	MSG_DOCUMENT_CREATED       = "document created"
	MSG_DOCUMENT_DELETED       = "document deleted"
	MSG_NO_ANSWER              = "No tengo suficiente informacion en el contexto para responder a tu pregunta."
	MSG_EMBEDDER_REQUIRED      = "an embedder is required"
	MSG_DOCUMENT_ID_REQUIRED   = "the document id is required"
	MSG_EMBEDDINGS_FAILED      = "the embeddings provider did not answer: %s"
	MSG_EMBEDDINGS_COUNT       = "the embeddings provider returned %d vectors for %d texts"
)

func init() {
	lang := envar.GetStr("LANG", "en")

	if lang == "es" {
		MSG_TENANT_ID_REQUIRED = "tenantId es requerido"
		MSG_PROJECT_ID_REQUIRED = "projectId es requerido"
		MSG_NAME_REQUIRED = "name es requerido"
		MSG_SOURCE_REQUIRED = "source es requerido"
		MSG_FILE_REQUIRED = "file es requerido"
		MSG_QUERY_REQUIRED = "query es requerido"
		MSG_QUESTION_REQUIRED = "question es requerido"
		MSG_DB_REQUIRED = "db es requerido"
		MSG_API_KEY_REQUIRED = "OPENAI_API_KEY es requerido"
		MSG_UNSUPPORTED_SOURCE = "fuente no soportada: %s"
		MSG_DOCUMENT_NOT_FOUND = "documento no encontrado"
		MSG_DOCUMENT_EMPTY = "el documento no tiene texto extraible"
		MSG_CONVERSATION_NOT_FOUND = "conversacion no encontrada"
		MSG_DOCUMENT_CREATED = "documento creado"
		MSG_DOCUMENT_DELETED = "documento eliminado"
		MSG_EMBEDDER_REQUIRED = "se requiere un generador de embeddings"
		MSG_DOCUMENT_ID_REQUIRED = "el id del documento es requerido"
		MSG_EMBEDDINGS_FAILED = "el proveedor de embeddings no respondió: %s"
		MSG_EMBEDDINGS_COUNT = "el proveedor de embeddings devolvió %d vectores para %d textos"
	}
}
