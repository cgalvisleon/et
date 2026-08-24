package ia

import "github.com/go-chi/chi/v5"

/**
* LoadRouter: Mounts the RAG's document-ingestion and conversation endpoints
* on mux: POST/GET /documents, POST /documents/sql, DELETE /documents/{id},
* POST /conversation, GET /conversation/{id}.
* @param mux chi.Router
**/
func (s *Rag) LoadRouter(mux chi.Router) {
	mux.Post("/documents", s.httpIngestDocument)
	mux.Get("/documents", s.httpListDocuments)
	mux.Post("/documents/sql", s.httpIngestSQL)
	mux.Delete("/documents/{id}", s.httpDeleteDocument)

	mux.Post("/conversation", s.httpAsk)
	mux.Get("/conversation/{id}", s.httpListMessages)
}
