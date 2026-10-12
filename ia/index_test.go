package ia

import (
	"context"
	"encoding/json"
	"hash/fnv"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cgalvisleon/et/jsql"
	_ "github.com/cgalvisleon/et/jsql/drivers/sqlite"
)

/**
* wordVector: A deterministic embedding for tests: one dimension per hashed word.
* @param text string
* @return []float64
**/
func wordVector(text string) []float64 {
	result := make([]float64, 64)
	for _, word := range strings.Fields(strings.ToLower(text)) {
		h := fnv.New32a()
		h.Write([]byte(strings.Trim(word, ".,;:¿?¡!")))
		result[h.Sum32()%64]++
	}
	return result
}

/**
* countingEmbedder: An Embedder over wordVector that counts the texts it embedded.
* @param calls *int
* @return Embedder
**/
func countingEmbedder(calls *int) Embedder {
	return func(ctx context.Context, texts []string) ([][]float64, error) {
		result := make([][]float64, len(texts))
		for i, text := range texts {
			*calls++
			result[i] = wordVector(text)
		}
		return result, nil
	}
}

/**
* testIndex: A Rag index on a scratch sqlite file.
* @param t *testing.T
* @return *Rag
**/
func testIndex(t *testing.T) *Rag {
	t.Helper()
	file := filepath.Join(t.TempDir(), "rag.db")
	db, err := jsql.ConnectTo(jsql.ConnectParams{
		Driver:      jsql.DriverSqlite,
		Host:        "local",
		Name:        file,
		Connection:  &jsql.SqliteConection{File: file, RecordLimit: 1000},
		RecordLimit: 1000,
	})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	rag, err := LoadIndex(db, "rag", Config{ChunkSize: 20, ChunkOverlap: 5, TopK: 3})
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}
	return rag
}

func TestIngestTextHashAndSearch(t *testing.T) {
	rag := testIndex(t)
	ctx := context.Background()
	calls := 0
	embed := countingEmbedder(&calls)

	doc := Document{
		TenantId:  "t1",
		ProjectId: "p1",
		Id:        "src1",
		Name:      "manual",
		Source:    "text",
		SourceId:  "src1",
		Text:      "Los gatos duermen mucho durante el día. Los perros ladran a los carteros.",
	}
	first, err := rag.IngestText(ctx, doc, embed)
	if err != nil {
		t.Fatalf("IngestText: %v", err)
	}
	if !first.Bool("changed") || first.Int("chunk_count") == 0 {
		t.Fatalf("unexpected first ingest: %v", first)
	}

	before := calls
	again, err := rag.IngestText(ctx, doc, embed)
	if err != nil {
		t.Fatalf("IngestText again: %v", err)
	}
	if again.Bool("changed") || calls != before {
		t.Fatalf("same hash should not re-embed: %v (calls %d → %d)", again, before, calls)
	}

	doc.Text = "Las tortugas viven muchos años en el mar."
	changed, err := rag.IngestText(ctx, doc, embed)
	if err != nil {
		t.Fatalf("IngestText changed: %v", err)
	}
	if !changed.Bool("changed") {
		t.Fatalf("a new text should re-index: %v", changed)
	}

	other := Document{TenantId: "t1", ProjectId: "p1", Id: "src2:notas.txt", Name: "notas.txt", Source: "txt", SourceId: "src2", Text: "Los gatos cazan ratones de noche."}
	if _, err := rag.IngestText(ctx, other, embed); err != nil {
		t.Fatalf("IngestText other: %v", err)
	}

	results, err := rag.Search(ctx, "t1", "p1", "¿qué cazan los gatos?", 2, 0.1, embed)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 || results[0].Str("document_id") != "src2:notas.txt" || results[0].Str("source_id") != "src2" {
		t.Fatalf("unexpected search: %v", results)
	}
	if results[0].Num("score") <= 0 || results[0].Str("document") != "notas.txt" {
		t.Fatalf("missing score or document name: %v", results[0])
	}

	none, err := rag.Search(ctx, "t2", "p1", "gatos", 2, 0, embed)
	if err != nil || len(none) != 0 {
		t.Fatalf("another tenant must not see the chunks: %v %v", none, err)
	}

	deleted, err := rag.DeleteDocumentsOf("t1", "p1", "src2")
	if err != nil || deleted != 1 {
		t.Fatalf("DeleteDocumentsOf: %d %v", deleted, err)
	}

	documents, err := rag.ListDocuments("t1", "p1")
	if err != nil || documents.Count != 1 {
		t.Fatalf("ListDocuments after delete: %v %v", documents, err)
	}

	deleted, err = rag.DeleteProject("t1", "p1")
	if err != nil || deleted != 1 {
		t.Fatalf("DeleteProject: %d %v", deleted, err)
	}
}

func TestHttpEmbedder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" || r.Header.Get("Authorization") != "Bearer key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		var body struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		data := []map[string]any{}
		// In reverse order: the embedder must sort them by index
		for i := len(body.Input) - 1; i >= 0; i-- {
			data = append(data, map[string]any{"index": i, "embedding": []float64{float64(len(body.Input[i]))}})
		}
		json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	defer server.Close()

	embed := HttpEmbedder(server.URL+"/v1/", "key", "test-model")
	vectors, err := embed(context.Background(), []string{"a", "bbb"})
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(vectors) != 2 || vectors[0][0] != 1 || vectors[1][0] != 3 {
		t.Fatalf("unexpected vectors: %v", vectors)
	}

	bad := HttpEmbedder(server.URL+"/v1", "wrong", "test-model")
	if _, err := bad(context.Background(), []string{"a"}); err == nil {
		t.Fatal("expected an error from a 401")
	}
}
