package jwsp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
)

// MaxMediaDownload: lo más que DownloadMedia lee (100 MB, el límite de un documento en WhatsApp)
const MaxMediaDownload = 100 << 20

// MediaInfo: un medio subido o recibido; Url es temporal (unos 5 minutos) y pide el token para descargarse
type MediaInfo struct {
	Id       string `json:"id"`
	Url      string `json:"url"`
	MimeType string `json:"mime_type"`
	Sha256   string `json:"sha256"`
	FileSize int64  `json:"file_size"`
}

/**
* UploadMedia: POST /{phone_number_id}/media (multipart: messaging_product, type y file); devuelve el id para Media.Id (dura 30 días)
* @param ctx context.Context, filename, mimeType string, content io.Reader
* @return string, error
**/
func (s *Client) UploadMedia(ctx context.Context, filename, mimeType string, content io.Reader) (string, error) {
	path, err := s.phonePath("/media")
	if err != nil {
		return "", err
	}

	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	writer.WriteField("messaging_product", "whatsapp")
	writer.WriteField("type", mimeType)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="file"; filename="`+escapeQuotes(filename)+`"`)
	header.Set("Content-Type", mimeType)
	part, err := writer.CreatePart(header)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, content); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	output, err := s.do(ctx, http.MethodPost, path, nil, writer.FormDataContentType(), buffer.Bytes())
	if err != nil {
		return "", err
	}
	return output.Str("id"), nil
}

/**
* GetMedia: GET /{media_id}?phone_number_id= (la URL temporal y los datos del medio)
* @param ctx context.Context, id string
* @return MediaInfo, error
**/
func (s *Client) GetMedia(ctx context.Context, id string) (MediaInfo, error) {
	if id == "" {
		return MediaInfo{}, errors.New(MSG_MEDIA_ID_REQUIRED)
	}

	output, err := s.Call(ctx, http.MethodGet, "/"+url.PathEscape(id), s.phoneQuery(), nil)
	if err != nil {
		return MediaInfo{}, err
	}
	return MediaInfo{
		Id:       output.Str("id"),
		Url:      output.Str("url"),
		MimeType: output.Str("mime_type"),
		Sha256:   output.Str("sha256"),
		FileSize: int64(output.Num("file_size")),
	}, nil
}

/**
* DeleteMedia: DELETE /{media_id}?phone_number_id=
* @param ctx context.Context, id string
* @return error
**/
func (s *Client) DeleteMedia(ctx context.Context, id string) error {
	if id == "" {
		return errors.New(MSG_MEDIA_ID_REQUIRED)
	}
	_, err := s.Call(ctx, http.MethodDelete, "/"+url.PathEscape(id), s.phoneQuery(), nil)
	return err
}

/**
* DownloadMedia: el contenido de la URL de GetMedia (con el token; hasta MaxMediaDownload)
* @param ctx context.Context, mediaUrl string
* @return []byte, error
**/
func (s *Client) DownloadMedia(ctx context.Context, mediaUrl string) ([]byte, error) {
	if mediaUrl == "" {
		return nil, errors.New(MSG_MEDIA_URL_REQUIRED)
	}
	if s.Token == "" {
		return nil, errors.New(MSG_TOKEN_REQUIRED)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)
	res, err := s.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(io.LimitReader(res.Body, MaxMediaDownload))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, errorOf(res.StatusCode, data)
	}
	return data, nil
}

/**
* phoneQuery: phone_number_id para las llamadas de media (Meta lo usa para comprobar que el medio es del número)
* @return url.Values
**/
func (s *Client) phoneQuery() url.Values {
	if s.PhoneNumberId == "" {
		return nil
	}
	return url.Values{"phone_number_id": {s.PhoneNumberId}}
}

// quoteEscaper: escapa el nombre del archivo como lo hace mime/multipart
var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

/**
* escapeQuotes: el nombre del archivo con comillas y barras invertidas escapadas para el encabezado
* @param value string
* @return string
**/
func escapeQuotes(value string) string {
	return quoteEscaper.Replace(value)
}
