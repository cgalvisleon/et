package request

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

/**
* multipartRequest: A POST with a multipart/form-data body: the file in the field "file" and the other fields
* @param t *testing.T, name string, content []byte, fields map[string]string
* @return *http.Request
**/
func multipartRequest(t *testing.T, name string, content []byte, fields map[string]string) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	form := multipart.NewWriter(body)
	for key, value := range fields {
		if err := form.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	part, err := form.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/files", body)
	r.Header.Set("Content-Type", form.FormDataContentType())
	return r
}

func TestGetFile(t *testing.T) {
	r := multipartRequest(t, "informe.pdf", []byte("hola"), map[string]string{"owner_id": "PRJ-1"})
	file, err := GetFile(r, "file", 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if file.Name != "informe.pdf" || string(file.Content) != "hola" || file.ContentType != "application/octet-stream" {
		t.Fatalf("file %+v", file)
	}
	if r.FormValue("owner_id") != "PRJ-1" {
		t.Fatalf("owner_id %q", r.FormValue("owner_id"))
	}
}

func TestGetFileTooLarge(t *testing.T) {
	tooLarge := fmt.Errorf(MSG_FILE_TOO_LARGE, 1024).Error()

	// The file itself, even if the body fits in the slack
	r := multipartRequest(t, "big.bin", bytes.Repeat([]byte("x"), 1025), nil)
	if _, err := GetFile(r, "file", 1024); err == nil || err.Error() != tooLarge {
		t.Fatalf("file: %v", err)
	}

	// Exactly the limit
	r = multipartRequest(t, "ok.bin", bytes.Repeat([]byte("x"), 1024), nil)
	if _, err := GetFile(r, "file", 1024); err != nil {
		t.Fatalf("limit: %v", err)
	}

	// A body beyond the slack: by Content-Length, before reading, and without it (chunked) while reading
	big := bytes.Repeat([]byte("x"), 1024+multipartSlack+1)
	r = multipartRequest(t, "big.bin", big, nil)
	if _, err := GetFile(r, "file", 1024); err == nil || err.Error() != tooLarge {
		t.Fatalf("Content-Length: %v", err)
	}
	r = multipartRequest(t, "big.bin", big, nil)
	r.ContentLength = -1
	if _, err := GetFile(r, "file", 1024); err == nil || err.Error() != tooLarge {
		t.Fatalf("chunked: %v", err)
	}

	// No limit
	r = multipartRequest(t, "big.bin", big, nil)
	if _, err := GetFile(r, "file", 0); err != nil {
		t.Fatal(err)
	}
}

func TestGetFileErrors(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/files", strings.NewReader(`{"a":1}`))
	r.Header.Set("Content-Type", "application/json")
	if _, err := GetFile(r, "file", 0); err == nil || err.Error() != MSG_FILE_MULTIPART {
		t.Fatalf("json: %v", err)
	}

	r = multipartRequest(t, "a.txt", []byte("a"), nil)
	if _, err := GetFile(r, "other", 0); err == nil || err.Error() != fmt.Sprintf(MSG_FILE_REQUIRED, "other") {
		t.Fatalf("missing field: %v", err)
	}
}
