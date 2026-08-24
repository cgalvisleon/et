package ia

import (
	"context"
	"errors"

	"github.com/cgalvisleon/et/jsql"
)

const (
	tableDocuments     = "ia_documents"
	tableChunks        = "ia_chunks"
	tableConversations = "ia_conversations"
	tableMessages      = "ia_messages"
)

/**
* Rag: Multitenant RAG engine. Owns the jsql models for documents, chunks,
* conversations and messages, and the AI provider configuration used to
* embed and answer questions.
**/
type Rag struct {
	db            *jsql.DB
	schema        string
	cnf           Config
	documents     *jsql.Model
	chunks        *jsql.Model
	conversations *jsql.Model
	messages      *jsql.Model
	embedFn       func(ctx context.Context, text string) ([]float64, error)
	answerFn      func(ctx context.Context, question string, contextChunks []string) (string, error)
}

/**
* defineTenantProjectModel: Defines a standard model (id, created_at, updated_at,
* status, _source) plus indexed tenant_id and project_id columns, so every RAG
* table is queryable per tenant and per project.
* @param db *jsql.DB, schema, name string, version int, userId string
* @return *jsql.Model, error
**/
func defineTenantProjectModel(db *jsql.DB, schema, name string, version int, userId string) (*jsql.Model, error) {
	model, err := db.DefineModel(schema, name, version, userId)
	if err != nil {
		return nil, err
	}

	model.DefineIndex(jsql.TENANT_ID, jsql.KEY, "")
	model.DefineIndex(jsql.PROJECT_ID, jsql.KEY, "")

	return model, nil
}

/**
* Load: Defines and initializes the jsql models backing the RAG (documents,
* chunks, conversations, messages) under the given schema, and returns a
* ready-to-use Rag engine configured with cnf (missing fields are filled from
* environment variables, see Config).
* @param db *jsql.DB, schema string, cnf Config
* @return *Rag, error
**/
func Load(db *jsql.DB, schema string, cnf Config) (*Rag, error) {
	if db == nil {
		return nil, errors.New(MSG_DB_REQUIRED)
	}
	if schema == "" {
		schema = "public"
	}

	cnf = defaultConfig(cnf)
	if cnf.ApiKey == "" {
		return nil, errors.New(MSG_API_KEY_REQUIRED)
	}

	documents, err := defineTenantProjectModel(db, schema, tableDocuments, 1, "ia")
	if err != nil {
		return nil, err
	}
	documents.DefineColumn("name", jsql.TEXT, "")
	documents.DefineColumn("source", jsql.TEXT, "")
	documents.DefineColumn("chunk_count", jsql.INT, 0)
	documents.DefineColumn("created_by", jsql.KEY, "")
	if err := documents.Init(); err != nil {
		return nil, err
	}

	chunks, err := defineTenantProjectModel(db, schema, tableChunks, 1, "ia")
	if err != nil {
		return nil, err
	}
	chunks.DefineIndex("document_id", jsql.KEY, "")
	chunks.DefineColumn("idx", jsql.INT, 0)
	chunks.DefineColumn("content", jsql.MEMO, "")
	chunks.DefineColumn("embedding", jsql.JSON, []any{})
	if err := chunks.Init(); err != nil {
		return nil, err
	}

	conversations, err := defineTenantProjectModel(db, schema, tableConversations, 1, "ia")
	if err != nil {
		return nil, err
	}
	conversations.DefineIndex("user_id", jsql.KEY, "")
	conversations.DefineColumn("title", jsql.TEXT, "")
	if err := conversations.Init(); err != nil {
		return nil, err
	}

	messages, err := defineTenantProjectModel(db, schema, tableMessages, 1, "ia")
	if err != nil {
		return nil, err
	}
	messages.DefineIndex("conversation_id", jsql.KEY, "")
	messages.DefineColumn("role", jsql.TEXT, "")
	messages.DefineColumn("content", jsql.MEMO, "")
	messages.DefineColumn("sources", jsql.JSON, []any{})
	if err := messages.Init(); err != nil {
		return nil, err
	}

	result := &Rag{
		db:            db,
		schema:        schema,
		cnf:           cnf,
		documents:     documents,
		chunks:        chunks,
		conversations: conversations,
		messages:      messages,
	}
	result.embedFn = result.embed
	result.answerFn = result.answer

	return result, nil
}
