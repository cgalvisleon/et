package ia

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

// embedBatch: how many chunks go to the embedder in one call
const embedBatch = 64

// searchPage: how many chunks Search reads per page (the record limit of the DB may lower it)
const searchPage = 500

/**
* Document: What IngestText indexes. Id is stable: ingesting the same id again replaces its chunks, or does nothing
* when the hash (text, chunking and model) did not change. SourceId groups the documents of one origin, so they can be
* listed and deleted together (DeleteDocumentsOf). ChunkSize and ChunkOverlap at 0 use the Rag's Config (a negative
* ChunkOverlap means no overlap); Model (the embedding model) only enters the hash: a new model re-indexes.
**/
type Document struct {
	TenantId     string
	ProjectId    string
	Id           string
	Name         string
	Source       string
	SourceId     string
	Text         string
	UserId       string
	Model        string
	ChunkSize    int
	ChunkOverlap int
	Metadata     et.Json
}

/**
* documentHash: The fingerprint of what a document indexes: its text, the chunking and the embedding model.
* @param doc Document, size, overlap int
* @return string
**/
func documentHash(doc Document, size, overlap int) string {
	raw := fmt.Sprintf("%d|%d|%s|%s", size, overlap, doc.Model, doc.Text)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

/**
* embedAll: Embeds texts in batches of embedBatch, checking that every text got its vector.
* @param ctx context.Context, texts []string, embed Embedder
* @return [][]float64, error
**/
func embedAll(ctx context.Context, texts []string, embed Embedder) ([][]float64, error) {
	result := make([][]float64, 0, len(texts))
	for start := 0; start < len(texts); start += embedBatch {
		end := min(start+embedBatch, len(texts))
		batch := texts[start:end]
		vectors, err := embed(ctx, batch)
		if err != nil {
			return nil, err
		}

		if len(vectors) != len(batch) {
			return nil, fmt.Errorf(MSG_EMBEDDINGS_COUNT, len(vectors), len(batch))
		}
		result = append(result, vectors...)
	}

	return result, nil
}

/**
* scope: A query on a model of the index limited to a tenant and project.
* @param model *jsql.Model, tenantId, projectId string
* @return *jsql.Query
**/
func scope(model *jsql.Model, tenantId, projectId string) *jsql.Query {
	return model.
		Where(jsql.Eq(jsql.TENANT_ID, tenantId)).
		And(jsql.Eq(jsql.PROJECT_ID, projectId))
}

/**
* IngestText: Indexes doc with embed: chunks its text, embeds every chunk and replaces, in one transaction, the
* document and its chunks. With the same hash as the stored one it does nothing (changed false).
* @param ctx context.Context, doc Document, embed Embedder
* @return et.Json, error
**/
func (s *Rag) IngestText(ctx context.Context, doc Document, embed Embedder) (et.Json, error) {
	if !utility.ValidStr(doc.TenantId, 0, []string{""}) {
		return nil, errors.New(MSG_TENANT_ID_REQUIRED)
	}
	if !utility.ValidStr(doc.ProjectId, 0, []string{""}) {
		return nil, errors.New(MSG_PROJECT_ID_REQUIRED)
	}
	if !utility.ValidStr(doc.Id, 0, []string{""}) {
		return nil, errors.New(MSG_DOCUMENT_ID_REQUIRED)
	}
	if !utility.ValidStr(doc.Name, 0, []string{""}) {
		return nil, errors.New(MSG_NAME_REQUIRED)
	}
	if embed == nil {
		return nil, errors.New(MSG_EMBEDDER_REQUIRED)
	}

	size := cmp.Or(doc.ChunkSize, s.cnf.ChunkSize)
	overlap := cmp.Or(doc.ChunkOverlap, s.cnf.ChunkOverlap)
	hash := documentHash(doc, size, overlap)
	current, err := scope(s.documents, doc.TenantId, doc.ProjectId).And(jsql.Eq(jsql.ID, doc.Id)).One()
	if err != nil {
		return nil, err
	}

	result := et.Json{
		"id":        doc.Id,
		"name":      doc.Name,
		"source":    doc.Source,
		"source_id": doc.SourceId,
		"hash":      hash,
		"changed":   false,
	}
	if current.Ok && current.Result.Str("hash") == hash {
		result["chunk_count"] = current.Result.Int("chunk_count")
		return result, nil
	}

	chunks := chunkText(doc.Text, size, overlap)
	if len(chunks) == 0 {
		return nil, errors.New(MSG_DOCUMENT_EMPTY)
	}

	// Embeddings first: if the provider fails, the previous index stays as it was
	vectors, err := embedAll(ctx, chunks, embed)
	if err != nil {
		return nil, err
	}

	now := timezone.Now()
	createdAt := any(now)
	if current.Ok {
		createdAt = current.Result[jsql.CREATED_AT]
	}

	metadata := doc.Metadata
	if metadata == nil {
		metadata = et.Json{}
	}

	row := et.Json{
		jsql.ID:         doc.Id,
		jsql.TENANT_ID:  doc.TenantId,
		jsql.PROJECT_ID: doc.ProjectId,
		"name":          doc.Name,
		"source":        doc.Source,
		"source_id":     doc.SourceId,
		"hash":          hash,
		"metadata":      metadata,
		"chunk_count":   len(chunks),
		"created_by":    doc.UserId,
		jsql.CREATED_AT: createdAt,
		jsql.UPDATED_AT: now,
	}

	tx := jsql.NewTx()
	if err := s.replaceDocument(tx, row, chunks, vectors); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	result["chunk_count"] = len(chunks)
	result["changed"] = true
	return result, nil
}

/**
* replaceDocument: Inside tx, deletes the document of row (and its chunks) and inserts it again with its chunks.
* @param tx *jsql.Tx, row et.Json, chunks []string, vectors [][]float64
* @return error
**/
func (s *Rag) replaceDocument(tx *jsql.Tx, row et.Json, chunks []string, vectors [][]float64) error {
	id := row.Str(jsql.ID)
	tenantId := row.Str(jsql.TENANT_ID)
	projectId := row.Str(jsql.PROJECT_ID)
	_, err := s.chunks.
		Delete().
		Where(jsql.Eq("document_id", id)).
		And(jsql.Eq(jsql.TENANT_ID, tenantId)).
		And(jsql.Eq(jsql.PROJECT_ID, projectId)).
		ExecTx(tx)
	if err != nil {
		return err
	}

	_, err = s.documents.
		Delete().
		Where(jsql.Eq(jsql.ID, id)).
		And(jsql.Eq(jsql.TENANT_ID, tenantId)).
		And(jsql.Eq(jsql.PROJECT_ID, projectId)).
		ExecTx(tx)
	if err != nil {
		return err
	}

	if _, err := s.documents.Insert(row).ExecTx(tx); err != nil {
		return err
	}

	now := row[jsql.UPDATED_AT]
	for i, content := range chunks {
		chunk := et.Json{
			jsql.ID:         reg.UUID(),
			jsql.TENANT_ID:  tenantId,
			jsql.PROJECT_ID: projectId,
			"document_id":   id,
			"idx":           i,
			"content":       content,
			"embedding":     embeddingValue(vectors[i]),
			jsql.CREATED_AT: now,
			jsql.UPDATED_AT: now,
		}
		if _, err := s.chunks.Insert(chunk).ExecTx(tx); err != nil {
			return err
		}
	}

	return nil
}

/**
* Search: The chunks of a tenant and project closest to query (cosine similarity with embed), best first: at most topK
* (0: the Config's) and only with score >= minScore (and > 0). Each result: chunk_id, document_id, document (its name),
* source, source_id, idx, content and score. It does not call a chat model; a project without chunks gives none without
* embedding the query.
* @param ctx context.Context, tenantId, projectId, query string, topK int, minScore float64, embed Embedder
* @return []et.Json, error
**/
func (s *Rag) Search(ctx context.Context, tenantId, projectId, query string, topK int, minScore float64, embed Embedder) ([]et.Json, error) {
	if !utility.ValidStr(tenantId, 0, []string{""}) {
		return nil, errors.New(MSG_TENANT_ID_REQUIRED)
	}
	if !utility.ValidStr(projectId, 0, []string{""}) {
		return nil, errors.New(MSG_PROJECT_ID_REQUIRED)
	}
	if !utility.ValidStr(query, 1, []string{}) {
		return nil, errors.New(MSG_QUESTION_REQUIRED)
	}
	if embed == nil {
		return nil, errors.New(MSG_EMBEDDER_REQUIRED)
	}

	// Sin fragmentos (una base sin indexar) no hay qué buscar: no se paga el embedding de la pregunta
	indexed, err := scope(s.chunks, tenantId, projectId).Exists()
	if err != nil {
		return nil, err
	}
	if !indexed {
		return []et.Json{}, nil
	}

	topK = cmp.Or(topK, s.cnf.TopK)
	vectors, err := embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}

	if len(vectors) != 1 {
		return nil, fmt.Errorf(MSG_EMBEDDINGS_COUNT, len(vectors), 1)
	}

	// Page by page (ordered by id), so a project with more chunks than the record limit is searched entirely
	queryVector := vectors[0]
	scored := []scoredChunk{}
	for page := 1; ; page++ {
		items, err := scope(s.chunks, tenantId, projectId).
			OrderBy(jsql.ID, true).
			Limit(page, searchPage)
		if err != nil {
			return nil, err
		}

		if len(items.Result) == 0 {
			break
		}

		for _, chunk := range items.Result {
			embedding := chunkEmbedding(chunk)
			score := cosineSimilarity(queryVector, embedding)
			if score <= 0 || score < minScore {
				continue
			}
			scored = append(scored, scoredChunk{chunk: chunk, score: score})
		}
	}

	sort.SliceStable(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})
	if topK > 0 && topK < len(scored) {
		scored = scored[:topK]
	}

	documents, err := s.documentsOf(tenantId, projectId, scored)
	if err != nil {
		return nil, err
	}

	result := make([]et.Json, 0, len(scored))
	for _, item := range scored {
		documentId := item.chunk.Str("document_id")
		document := documents[documentId]
		result = append(result, et.Json{
			"chunk_id":    item.chunk.Str(jsql.ID),
			"document_id": documentId,
			"document":    document.Str("name"),
			"source":      document.Str("source"),
			"source_id":   document.Str("source_id"),
			"idx":         item.chunk.Int("idx"),
			"content":     item.chunk.Str("content"),
			"score":       item.score,
		})
	}

	return result, nil
}

/**
* documentsOf: The documents of the scored chunks, by id.
* @param tenantId, projectId string, scored []scoredChunk
* @return map[string]et.Json, error
**/
func (s *Rag) documentsOf(tenantId, projectId string, scored []scoredChunk) (map[string]et.Json, error) {
	result := map[string]et.Json{}
	if len(scored) == 0 {
		return result, nil
	}

	ids := []any{}
	for _, item := range scored {
		ids = append(ids, item.chunk.Str("document_id"))
	}

	items, err := scope(s.documents, tenantId, projectId).And(jsql.In(jsql.ID, ids)).All()
	if err != nil {
		return nil, err
	}

	for _, row := range items.Result {
		result[row.Str(jsql.ID)] = row
	}

	return result, nil
}

/**
* DeleteDocumentsOf: Deletes, in one transaction, the documents of a source (source_id) and their chunks; returns how
* many documents it deleted.
* @param tenantId, projectId, sourceId string
* @return int, error
**/
func (s *Rag) DeleteDocumentsOf(tenantId, projectId, sourceId string) (int, error) {
	documents, err := scope(s.documents, tenantId, projectId).
		And(jsql.Eq("source_id", sourceId)).
		Select(jsql.ID).
		All()
	if err != nil {
		return 0, err
	}

	return s.deleteDocuments(tenantId, projectId, documents.Result)
}

/**
* DeleteProject: Deletes, in one transaction, every document of a tenant and project and their chunks; returns how
* many documents it deleted.
* @param tenantId, projectId string
* @return int, error
**/
func (s *Rag) DeleteProject(tenantId, projectId string) (int, error) {
	documents, err := scope(s.documents, tenantId, projectId).
		Select(jsql.ID).
		All()
	if err != nil {
		return 0, err
	}

	return s.deleteDocuments(tenantId, projectId, documents.Result)
}

/**
* deleteDocuments: Deletes the given documents and their chunks in one transaction.
* @param tenantId, projectId string, documents []et.Json
* @return int, error
**/
func (s *Rag) deleteDocuments(tenantId, projectId string, documents []et.Json) (int, error) {
	if len(documents) == 0 {
		return 0, nil
	}

	ids := []any{}
	for _, row := range documents {
		ids = append(ids, row.Str(jsql.ID))
	}

	tx := jsql.NewTx()
	_, err := s.chunks.
		Delete().
		Where(jsql.In("document_id", ids)).
		And(jsql.Eq(jsql.TENANT_ID, tenantId)).
		And(jsql.Eq(jsql.PROJECT_ID, projectId)).
		ExecTx(tx)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	_, err = s.documents.
		Delete().
		Where(jsql.In(jsql.ID, ids)).
		And(jsql.Eq(jsql.TENANT_ID, tenantId)).
		And(jsql.Eq(jsql.PROJECT_ID, projectId)).
		ExecTx(tx)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return len(ids), nil
}
