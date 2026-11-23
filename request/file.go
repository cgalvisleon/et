package request

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/cgalvisleon/et/envar"
)

// Messages of the multipart file of a request (GetFile)
var (
	MSG_FILE_REQUIRED  = "the file %s is required"
	MSG_FILE_TOO_LARGE = "the file exceeds the limit of %d bytes"
	MSG_FILE_MULTIPART = "the request must be multipart/form-data"
)

/**
* init: Translates the messages to Spanish when LANG is es
**/
func init() {
	if envar.GetStr("LANG", "en") == "es" {
		MSG_FILE_REQUIRED = "el archivo %s es obligatorio"
		MSG_FILE_TOO_LARGE = "el archivo supera el límite de %d bytes"
		MSG_FILE_MULTIPART = "la petición debe ser multipart/form-data"
	}
}

// multipartMemory: what ParseMultipartForm keeps in memory; the rest of a part goes to a temporary file. multipartSlack:
// what the body may carry besides the file (the other fields and the headers of each part)
const (
	multipartMemory = 8 << 20
	multipartSlack  = 1 << 20
)

/**
* File: A file received in a multipart/form-data request
**/
type File struct {
	Name        string // the file name sent by the client, as is
	ContentType string // the Content-Type of the part; application/octet-stream when it does not say
	Content     []byte
}

/**
* IsMultipart: Whether the request body is multipart/form-data
* @param r *http.Request
* @return bool
**/
func IsMultipart(r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	return err == nil && mediaType == "multipart/form-data"
}

/**
* GetFile: The file of the field of a multipart/form-data request, of at most maxBytes (MSG_FILE_TOO_LARGE; 0 or less, no
* limit; the body is cut at maxBytes plus multipartSlack, so a larger one is not read). The other fields are read
* afterwards with r.FormValue
* @param r *http.Request, field string, maxBytes int64
* @return File, error
**/
func GetFile(r *http.Request, field string, maxBytes int64) (File, error) {
	if !IsMultipart(r) {
		return File{}, errors.New(MSG_FILE_MULTIPART)
	}

	tooLarge := fmt.Errorf(MSG_FILE_TOO_LARGE, maxBytes)
	if maxBytes > 0 {
		limit := maxBytes + multipartSlack
		if r.ContentLength > limit {
			return File{}, tooLarge
		}
		r.Body = http.MaxBytesReader(nil, r.Body, limit)
	}

	if err := r.ParseMultipartForm(multipartMemory); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return File{}, tooLarge
		}
		return File{}, err
	}

	part, header, err := r.FormFile(field)
	if err != nil {
		return File{}, fmt.Errorf(MSG_FILE_REQUIRED, field)
	}
	defer part.Close()

	content, err := io.ReadAll(part)
	if err != nil {
		return File{}, err
	}

	if maxBytes > 0 && int64(len(content)) > maxBytes {
		return File{}, tooLarge
	}

	contentType := strings.TrimSpace(header.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	result := File{Name: header.Filename, ContentType: contentType, Content: content}
	return result, nil
}
