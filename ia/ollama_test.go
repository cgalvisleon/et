package ia

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOllamaLLMComplete(t *testing.T) {
	var gotReq ollamaGenerateRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/generate" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		json.NewEncoder(w).Encode(ollamaGenerateResponse{Response: "Según lo que sé, el cielo es azul.", Done: true})
	}))
	defer server.Close()

	llm := NewOllamaLLM(server.URL, "llama3.2", time.Second)
	answer, err := llm.Complete(context.Background(), "¿de que color es el cielo?")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if answer != "Según lo que sé, el cielo es azul." {
		t.Fatalf("unexpected answer: %q", answer)
	}

	if gotReq.Model != "llama3.2" || gotReq.Stream {
		t.Fatalf("unexpected request body: %+v", gotReq)
	}
}

func TestOllamaLLMCompleteErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("model not found"))
	}))
	defer server.Close()

	llm := NewOllamaLLM(server.URL, "missing-model", time.Second)
	if _, err := llm.Complete(context.Background(), "hola"); err == nil {
		t.Fatalf("expected an error for a non-200 response")
	}
}

func TestOllamaLLMCompleteTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		json.NewEncoder(w).Encode(ollamaGenerateResponse{Response: "too late"})
	}))
	defer server.Close()

	llm := NewOllamaLLM(server.URL, "llama3.2", 5*time.Millisecond)
	if _, err := llm.Complete(context.Background(), "hola"); err == nil {
		t.Fatalf("expected a timeout error")
	}
}
