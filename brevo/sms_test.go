package brevo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cgalvisleon/et/et"
)

// TestSendSmsUsesPerRecipientParams guards against sendSms applying every
// entry of params to every recipient instead of correlating params[i] with
// contactNumbers[i]. Before the fix, string replacement removed each
// placeholder after its first substitution, so every recipient silently
// received params[0]'s values regardless of their own entry in params.
func TestSendSmsUsesPerRecipientParams(t *testing.T) {
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

	contactNumbers := []string{"+111", "+222"}
	content := "Hello {{name}}"
	params := []et.Json{
		{"name": "Alice"},
		{"name": "Bob"},
	}

	if _, err := sendSms("Sender", "org", contactNumbers, content, params, "Transactional"); err != nil {
		t.Fatalf("sendSms failed: %v", err)
	}

	if len(received) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(received))
	}
	if received[0]["content"] != "Hello Alice" {
		t.Fatalf("recipient 0 content = %v, want %q", received[0]["content"], "Hello Alice")
	}
	if received[1]["content"] != "Hello Bob" {
		t.Fatalf("recipient 1 content = %v, want %q", received[1]["content"], "Hello Bob")
	}
}
