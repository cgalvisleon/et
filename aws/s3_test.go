package aws

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestUploaderFilePropagatesParseMultipartFormError guards against
// UploaderFile silently discarding the error from r.ParseMultipartForm — a
// malformed request used to fall through to FormFile and surface a less
// specific error (or, if the body happened to satisfy FormFile some other
// way, be missed entirely).
func TestUploaderFilePropagatesParseMultipartFormError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not-a-multipart-body"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=xxx")

	s := &S3AWS{}
	if _, err := s.UploaderFile(req, "bucket", "", ""); err == nil {
		t.Fatal("expected an error from a malformed multipart request")
	}
}
