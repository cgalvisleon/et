package router

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/request"
)

// TestItemMultipart: Item no lee un multipart/form-data como JSON: el handler recibe un cuerpo vacío y lee el archivo
func TestItemMultipart(t *testing.T) {
	api := NewApi("test", "/api/test", "localhost", 0, 0, 1)
	upload := func(r *http.Request, body et.Json) (et.Item, error) {
		file, err := request.GetFile(r, "file", 1<<20)
		if err != nil {
			return et.Item{}, err
		}
		result := et.Json{"name": file.Name, "content": string(file.Content), "owner": r.FormValue("owner"), "body": len(body)}
		return et.Item{Ok: true, Result: result}, nil
	}
	api.Protected("files", POST, "/files", "Upload a file", api.Item(http.StatusOK, upload))

	body := &bytes.Buffer{}
	form := multipart.NewWriter(body)
	if err := form.WriteField("owner", "a"); err != nil {
		t.Fatal(err)
	}
	part, err := form.CreateFormFile("file", "a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("hola")); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(POST, "/api/test/files", body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	rec := httptest.NewRecorder()
	api.Router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	for _, want := range []string{`"name":"a.txt"`, `"content":"hola"`, `"owner":"a"`, `"body":0`} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("%s does not contain %s", rec.Body.String(), want)
		}
	}
}
