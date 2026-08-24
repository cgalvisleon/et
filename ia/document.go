package ia

import (
	"context"
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

/**
* IngestFile: Ingests a file (detecting its source by filename extension —
* pdf, docx, xlsx, csv, txt or md), extracting, chunking, embedding and
* persisting its content under the given tenant and project.
* @param ctx context.Context, tenantId, projectId, name string, data []byte, userId string
* @return et.Json, error
**/
func (s *Rag) IngestFile(ctx context.Context, tenantId, projectId, name string, data []byte, userId string) (et.Json, error) {
	source, err := sourceFromFilename(name)
	if err != nil {
		return nil, err
	}

	text, err := extractText(source, data)
	if err != nil {
		return nil, err
	}

	return s.ingest(ctx, tenantId, projectId, name, source, text, userId)
}

/**
* IngestSQL: Ingests the result of a SQL query, run against sourceDB (or the
* Rag's own DB when sourceDB is nil), as a document under the given tenant
* and project.
* @param ctx context.Context, tenantId, projectId, name, query string, sourceDB *jsql.DB, userId string
* @return et.Json, error
**/
func (s *Rag) IngestSQL(ctx context.Context, tenantId, projectId, name, query string, sourceDB *jsql.DB, userId string) (et.Json, error) {
	if !utility.ValidStr(query, 1, []string{}) {
		return nil, errors.New(MSG_QUERY_REQUIRED)
	}

	if sourceDB == nil {
		sourceDB = s.db
	}

	text, err := extractSqlText(sourceDB, query)
	if err != nil {
		return nil, err
	}

	return s.ingest(ctx, tenantId, projectId, name, SourceSQL, text, userId)
}

/**
* ingest: Shared pipeline for every ingestion path: validates the tenant
* scope, chunks text, embeds every chunk and persists the document plus its
* chunks.
* @param ctx context.Context, tenantId, projectId, name, source, text, userId string
* @return et.Json, error
**/
func (s *Rag) ingest(ctx context.Context, tenantId, projectId, name, source, text, userId string) (et.Json, error) {
	if !utility.ValidStr(tenantId, 0, []string{""}) {
		return nil, errors.New(MSG_TENANT_ID_REQUIRED)
	}
	if !utility.ValidStr(projectId, 0, []string{""}) {
		return nil, errors.New(MSG_PROJECT_ID_REQUIRED)
	}
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, errors.New(MSG_NAME_REQUIRED)
	}

	chunks := chunkText(text, s.cnf.ChunkSize, s.cnf.ChunkOverlap)
	if len(chunks) == 0 {
		return nil, errors.New(MSG_DOCUMENT_EMPTY)
	}

	now := timezone.Now()
	documentId := reg.UUID()
	_, err := s.documents.Insert(et.Json{
		"id":          documentId,
		"tenant_id":   tenantId,
		"project_id":  projectId,
		"name":        name,
		"source":      source,
		"chunk_count": len(chunks),
		"created_by":  userId,
		"created_at":  now,
		"updated_at":  now,
	}).ExecTx(nil)
	if err != nil {
		return nil, err
	}

	for i, content := range chunks {
		embedding, err := s.embedFn(ctx, content)
		if err != nil {
			return nil, err
		}

		_, err = s.chunks.Insert(et.Json{
			"id":          reg.UUID(),
			"tenant_id":   tenantId,
			"project_id":  projectId,
			"document_id": documentId,
			"idx":         i,
			"content":     content,
			"embedding":   embeddingValue(embedding),
			"created_at":  now,
			"updated_at":  now,
		}).ExecTx(nil)
		if err != nil {
			return nil, err
		}
	}

	return et.Json{
		"id":          documentId,
		"tenant_id":   tenantId,
		"project_id":  projectId,
		"name":        name,
		"source":      source,
		"chunk_count": len(chunks),
	}, nil
}

/**
* embeddingValue: Converts a []float64 embedding into the []any shape the
* jsql command builder knows how to marshal as a JSON column value.
* @param embedding []float64
* @return []any
**/
func embeddingValue(embedding []float64) []any {
	result := make([]any, len(embedding))
	for i, v := range embedding {
		result[i] = v
	}
	return result
}

/**
* ListDocuments: Lists every document ingested under the given tenant and project.
* @param tenantId, projectId string
* @return et.Items, error
**/
func (s *Rag) ListDocuments(tenantId, projectId string) (et.Items, error) {
	return s.documents.
		Where(jsql.Eq(jsql.TENANT_ID, tenantId)).
		And(jsql.Eq(jsql.PROJECT_ID, projectId)).
		All()
}

/**
* DeleteDocument: Deletes a document and every chunk derived from it, scoped
* to the given tenant and project.
* @param tenantId, projectId, id string
* @return error
**/
func (s *Rag) DeleteDocument(tenantId, projectId, id string) error {
	_, err := s.chunks.
		Delete().
		Where(jsql.Eq(jsql.TENANT_ID, tenantId)).
		And(jsql.Eq(jsql.PROJECT_ID, projectId)).
		And(jsql.Eq("document_id", id)).
		ExecTx(nil)
	if err != nil {
		return err
	}

	_, err = s.documents.
		Delete().
		Where(jsql.Eq(jsql.TENANT_ID, tenantId)).
		And(jsql.Eq(jsql.PROJECT_ID, projectId)).
		And(jsql.Eq("id", id)).
		ExecTx(nil)
	return err
}
