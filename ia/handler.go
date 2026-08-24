package ia

import (
	"io"
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/response"
)

/**
* httpIngestDocument: Handles a multipart file upload (field "file") and
* ingests it as a document. Expects tenant_id, project_id and user_id as
* multipart form fields.
* @param w http.ResponseWriter, r *http.Request
**/
func (s *Rag) httpIngestDocument(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, MSG_FILE_REQUIRED)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	tenantId := r.FormValue("tenant_id")
	projectId := r.FormValue("project_id")
	userId := r.FormValue("user_id")

	result, err := s.IngestFile(r.Context(), tenantId, projectId, header.Filename, data, userId)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusCreated, et.Item{Ok: true, Result: result})
}

/**
* httpIngestSQL: Ingests the result of a SQL query as a document.
* Body: tenant_id, project_id, name, query, user_id.
* @param w http.ResponseWriter, r *http.Request
**/
func (s *Rag) httpIngestSQL(w http.ResponseWriter, r *http.Request) {
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	tenantId := body.Str("tenant_id")
	projectId := body.Str("project_id")
	name := body.Str("name")
	query := body.Str("query")
	userId := body.Str("user_id")

	result, err := s.IngestSQL(r.Context(), tenantId, projectId, name, query, nil, userId)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusCreated, et.Item{Ok: true, Result: result})
}

/**
* httpListDocuments: Lists the documents ingested under a tenant and project.
* Query params: tenant_id, project_id.
* @param w http.ResponseWriter, r *http.Request
**/
func (s *Rag) httpListDocuments(w http.ResponseWriter, r *http.Request) {
	tenantId := request.Query(r, "tenant_id").Str()
	projectId := request.Query(r, "project_id").Str()

	items, err := s.ListDocuments(tenantId, projectId)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEMS(w, r, http.StatusOK, items)
}

/**
* httpDeleteDocument: Deletes a document and its chunks.
* Path param: id. Query params: tenant_id, project_id.
* @param w http.ResponseWriter, r *http.Request
**/
func (s *Rag) httpDeleteDocument(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()
	tenantId := request.Query(r, "tenant_id").Str()
	projectId := request.Query(r, "project_id").Str()

	if err := s.DeleteDocument(tenantId, projectId, id); err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{Ok: true, Result: et.Json{"message": MSG_DOCUMENT_DELETED}})
}

/**
* httpAsk: Answers a question via RAG.
* Body: tenant_id, project_id, conversation_id, user_id, question.
* @param w http.ResponseWriter, r *http.Request
**/
func (s *Rag) httpAsk(w http.ResponseWriter, r *http.Request) {
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	tenantId := body.Str("tenant_id")
	projectId := body.Str("project_id")
	conversationId := body.Str("conversation_id")
	userId := body.Str("user_id")
	question := body.Str("question")

	result, err := s.Ask(r.Context(), tenantId, projectId, conversationId, userId, question)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{Ok: true, Result: result})
}

/**
* httpListMessages: Lists the message history of a conversation.
* Path param: id. Query params: tenant_id, project_id.
* @param w http.ResponseWriter, r *http.Request
**/
func (s *Rag) httpListMessages(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()
	tenantId := request.Query(r, "tenant_id").Str()
	projectId := request.Query(r, "project_id").Str()

	items, err := s.ListMessages(tenantId, projectId, id)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEMS(w, r, http.StatusOK, items)
}
