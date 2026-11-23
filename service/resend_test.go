package service

import (
	"encoding/base64"
	"net/http"
	"strings"
	"testing"

	"github.com/cgalvisleon/et/et"
)

/**
* resendOf: Un Resend contra url
* @param t *testing.T, url, from string
* @return Sender
**/
func resendOf(t *testing.T, url, from string) Sender {
	sender, err := NewResend(et.Json{"base_url": url, "api_key": "re_tok", "email_from": from})
	if err != nil {
		t.Fatal(err)
	}
	return sender
}

/**
* TestNewResend: NewResend exige api_key y pone la url por defecto
* @param t *testing.T
**/
func TestNewResend(t *testing.T) {
	if _, err := NewResend(et.Json{"email_from": "a@b.co"}); err == nil {
		t.Error("sin api_key debe fallar")
	}
	sender, err := NewResend(et.Json{"api_key": "re_tok"})
	if err != nil || sender.(*Resend).Url != resendUrl {
		t.Errorf("url por defecto: %v %v", sender, err)
	}
}

/**
* TestResendEmail: El cuerpo y la autenticación de un correo con adjunto
* @param t *testing.T
**/
func TestResendEmail(t *testing.T) {
	var auth string
	var body et.Json
	server := fakeLabsMobile(t, http.StatusOK, `{"id":"49a3999c-0ce1-4ea6-ab68-afcd6dc2e794"}`, &auth, &body)
	defer server.Close()

	message := Email{To: "ana@empresa.co", Subject: "Hola", Html: "<p>Hola</p>", Attachments: []Attachment{{Filename: "a.txt", ContentType: "text/plain", Content: []byte("abc")}}}
	result, err := resendOf(t, server.URL, "CoreStudio <no-reply@empresa.co>").SendEmail(message)
	if err != nil || result.Str("provider") != "resend" || result.Json("response").Str("id") == "" {
		t.Fatalf("%v %v", result, err)
	}
	if auth != "Bearer re_tok" {
		t.Errorf("autorización %q", auth)
	}
	if body.Str("from") != "CoreStudio <no-reply@empresa.co>" || body.ArrayStr("to")[0] != "ana@empresa.co" || body.Str("subject") != "Hola" || body.Str("html") != "<p>Hola</p>" {
		t.Errorf("cuerpo %v", body)
	}
	file := body.ArrayJson("attachments")[0]
	if file.Str("filename") != "a.txt" || file.Str("content") != base64.StdEncoding.EncodeToString([]byte("abc")) || file.Str("content_type") != "text/plain" {
		t.Errorf("adjunto %v", file)
	}

	// El remitente del correo manda sobre el del sender
	message.From = "Otro <otro@empresa.co>"
	message.Attachments = nil
	if _, err := resendOf(t, server.URL, "CoreStudio <no-reply@empresa.co>").SendEmail(message); err != nil || body.Str("from") != "Otro <otro@empresa.co>" || body["attachments"] != nil {
		t.Errorf("remitente del correo: %v %v", body, err)
	}

	if _, err := resendOf(t, server.URL, "").SendEmail(Email{To: "ana@empresa.co"}); err == nil {
		t.Error("sin remitente debe fallar")
	}
	if _, err := resendOf(t, server.URL, "x@y.co").SendSMS(SMS{To: "1"}); err == nil {
		t.Error("el SMS no lo envía")
	}
}

/**
* TestResendError: Un error de Resend trae su nombre y su mensaje
* @param t *testing.T
**/
func TestResendError(t *testing.T) {
	var auth string
	var body et.Json
	server := fakeLabsMobile(t, http.StatusForbidden, `{"statusCode":403,"name":"validation_error","message":"The empresa.co domain is not verified."}`, &auth, &body)
	defer server.Close()

	_, err := resendOf(t, server.URL, "a@empresa.co").SendEmail(Email{To: "ana@empresa.co", Subject: "Hola", Html: "x"})
	if err == nil || !strings.Contains(err.Error(), "validation_error") || !strings.Contains(err.Error(), "not verified") {
		t.Errorf("error %v", err)
	}
}
