package ia

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/openai/openai-go/v3"
)

/**
* Config: Configuration for the AI provider used by the RAG (embeddings + chat)
* and for the chunking strategy applied to ingested documents.
**/
type Config struct {
	ApiKey         string `json:"-"`
	EmbeddingModel string `json:"embedding_model"`
	ChatModel      string `json:"chat_model"`
	ChunkSize      int    `json:"chunk_size"`
	ChunkOverlap   int    `json:"chunk_overlap"`
	TopK           int    `json:"top_k"`
}

/**
* defaultConfig: Builds a Config from environment variables, filling in any
* zero-valued field of cnf with a sensible default.
* @param cnf Config
* @return Config
**/
func defaultConfig(cnf Config) Config {
	if cnf.ApiKey == "" {
		cnf.ApiKey = envar.GetStr("OPENAI_API_KEY", "")
	}
	if cnf.EmbeddingModel == "" {
		cnf.EmbeddingModel = envar.GetStr("IA_EMBEDDING_MODEL", string(openai.EmbeddingModelTextEmbedding3Small))
	}
	if cnf.ChatModel == "" {
		cnf.ChatModel = envar.GetStr("IA_CHAT_MODEL", string(openai.ChatModelGPT4oMini))
	}
	if cnf.ChunkSize <= 0 {
		cnf.ChunkSize = envar.GetInt("IA_CHUNK_SIZE", 500)
	}
	if cnf.ChunkOverlap < 0 {
		cnf.ChunkOverlap = 0
	}
	if cnf.ChunkOverlap == 0 {
		cnf.ChunkOverlap = envar.GetInt("IA_CHUNK_OVERLAP", 50)
	}
	if cnf.TopK <= 0 {
		cnf.TopK = envar.GetInt("IA_TOP_K", 5)
	}

	return cnf
}
