package ia

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cgalvisleon/et/jsql"
	_ "github.com/cgalvisleon/et/jsql/drivers/sqlite"
)

/**
* fakeEmbed: Deterministic stand-in for the OpenAI embedding call, so tests
* can exercise storage, retrieval and ranking without live network access.
* Encodes text as a small bag-of-words vector over a fixed vocabulary.
**/
func fakeEmbed(_ context.Context, text string) ([]float64, error) {
	vocab := []string{"gato", "perro", "factura", "pago", "python", "golang"}
	vec := make([]float64, len(vocab))
	lower := strings.ToLower(text)
	for i, word := range vocab {
		if strings.Contains(lower, word) {
			vec[i] = 1
		}
	}
	return vec, nil
}

func newTestRag(t *testing.T) *Rag {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "ia_test.db")
	db, err := jsql.NewDB("tenant:root", "local", dbPath, jsql.DriverSqlite, false)
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	if err := db.Init(); err != nil {
		t.Fatalf("db.Init: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	rag, err := Load(db, "main", Config{ApiKey: "test-key"})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	rag.embedFn = fakeEmbed
	rag.answerFn = func(_ context.Context, _ string, contextChunks []string) (string, error) {
		if len(contextChunks) == 0 {
			return MSG_NO_ANSWER, nil
		}
		return contextChunks[0], nil
	}

	return rag
}

func TestChunkText(t *testing.T) {
	words := "one two three four five six seven eight nine ten"
	chunks := chunkText(words, 4, 1)
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	if chunks[0] != "one two three four" {
		t.Fatalf("unexpected first chunk: %q", chunks[0])
	}

	if len(chunkText("", 10, 2)) != 0 {
		t.Fatal("empty text should yield no chunks")
	}
}

func TestCosineSimilarity(t *testing.T) {
	a := []float64{1, 0, 0}
	b := []float64{1, 0, 0}
	c := []float64{0, 1, 0}

	if got := cosineSimilarity(a, b); got != 1 {
		t.Fatalf("identical vectors: got %v, want 1", got)
	}
	if got := cosineSimilarity(a, c); got != 0 {
		t.Fatalf("orthogonal vectors: got %v, want 0", got)
	}
	if got := cosineSimilarity(a, []float64{1, 0}); got != 0 {
		t.Fatalf("mismatched length: got %v, want 0", got)
	}
}

func TestIngestAndAskIsolatesTenantsAndRanksBySimilarity(t *testing.T) {
	rag := newTestRag(t)
	ctx := context.Background()

	if _, err := rag.IngestFile(ctx, "t1", "p1", "pets.txt", []byte("El gato y el perro son mascotas."), "u1"); err != nil {
		t.Fatalf("IngestFile pets: %v", err)
	}
	if _, err := rag.IngestFile(ctx, "t1", "p1", "billing.txt", []byte("La factura de pago vence pronto."), "u1"); err != nil {
		t.Fatalf("IngestFile billing: %v", err)
	}
	if _, err := rag.IngestFile(ctx, "t2", "p1", "other-tenant.txt", []byte("python golang"), "u2"); err != nil {
		t.Fatalf("IngestFile other tenant: %v", err)
	}

	docs, err := rag.ListDocuments("t1", "p1")
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if docs.Count != 2 {
		t.Fatalf("expected 2 documents for t1/p1, got %d", docs.Count)
	}

	result, err := rag.Ask(ctx, "t1", "p1", "", "u1", "cuentame sobre el gato")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}

	answer, _ := result["answer"].(string)
	if !strings.Contains(answer, "gato") {
		t.Fatalf("expected the pets chunk to be retrieved as top match, got answer: %q", answer)
	}

	convID, _ := result["conversation_id"].(string)
	if convID == "" {
		t.Fatal("expected a conversation id to be created")
	}

	msgs, err := rag.ListMessages("t1", "p1", convID)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if msgs.Count != 2 {
		t.Fatalf("expected 2 messages (question + answer), got %d", msgs.Count)
	}

	// Cross-tenant isolation: t2's chunks must never surface for t1's question.
	other, err := rag.Ask(ctx, "t2", "p1", "", "u2", "cuentame sobre el gato")
	if err != nil {
		t.Fatalf("Ask t2: %v", err)
	}
	otherAnswer, _ := other["answer"].(string)
	if otherAnswer != MSG_NO_ANSWER {
		t.Fatalf("expected no-context answer for unrelated tenant, got: %q", otherAnswer)
	}
}

func TestIngestFileUnsupportedExtension(t *testing.T) {
	rag := newTestRag(t)
	_, err := rag.IngestFile(context.Background(), "t1", "p1", "notes.xyz", []byte("hello"), "u1")
	if err == nil {
		t.Fatal("expected an error for an unsupported extension")
	}
}

func TestExtractCsvText(t *testing.T) {
	text, err := extractCsvText([]byte("a,b\n1,2\n"))
	if err != nil {
		t.Fatalf("extractCsvText: %v", err)
	}
	if !strings.Contains(text, "1, 2") {
		t.Fatalf("expected rendered csv row, got: %q", text)
	}
}
