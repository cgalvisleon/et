package brevo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cgalvisleon/et/et"
)

// TestSendWhatsappUsesPerRecipientParams guards against SendWhatsapp sending
// the whole params slice (every recipient's template values) to every single
// recipient's request instead of just their own params[i] entry.
func TestSendWhatsappUsesPerRecipientParams(t *testing.T) {
	var received []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		received = append(received, body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"messageId":"1"}`))
	}))
	defer server.Close()

	t.Setenv("BREVO_SEND_PATH", server.URL)
	t.Setenv("BREVO_SEND_KEY", "test-key")
	t.Setenv("BREVO_SENDER", "1234567890")

	contactNumbers := []string{"+111", "+222"}
	params := []et.Json{
		{"1": "Alice"},
		{"1": "Bob"},
	}

	if _, err := SendWhatsapp(contactNumbers, "tmpl", params, "Transactional"); err != nil {
		t.Fatalf("SendWhatsapp failed: %v", err)
	}

	if len(received) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(received))
	}

	p0, _ := received[0]["params"].(map[string]any)
	if p0["1"] != "Alice" {
		t.Fatalf("recipient 0 params = %v, want {1: Alice}", received[0]["params"])
	}
	p1, _ := received[1]["params"].(map[string]any)
	if p1["1"] != "Bob" {
		t.Fatalf("recipient 1 params = %v, want {1: Bob}", received[1]["params"])
	}
}
