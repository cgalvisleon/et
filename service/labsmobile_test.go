package service

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cgalvisleon/et/et"
)

/**
* fakeLabsMobile: Un LabsMobile falso: guarda la autorización y el cuerpo de cada envío y responde status con answer
* @param t *testing.T, status int, answer string, auth *string, body *et.Json
* @return *httptest.Server
**/
func fakeLabsMobile(t *testing.T, status int, answer string, auth *string, body *et.Json) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("método %s", r.Method)
		}
		*auth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		*body = et.Json{}
		if err := json.Unmarshal(raw, body); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		io.WriteString(w, answer)
	}))
}

/**
* labsMobileOf: Un LabsMobile contra url
* @param t *testing.T, url, from string
* @return Sender
**/
func labsMobileOf(t *testing.T, url, from string) Sender {
	sender, err := NewLabsMobile(et.Json{"base_url": url, "username": "ana@empresa.co", "api_token": "tok", "sms_from": from})
	if err != nil {
		t.Fatal(err)
	}
	return sender
}

/**
* TestNewLabsMobile: NewLabsMobile exige api_token y pone la url por defecto
* @param t *testing.T
**/
func TestNewLabsMobile(t *testing.T) {
	if _, err := NewLabsMobile(et.Json{"username": "ana@empresa.co"}); err == nil {
		t.Error("sin api_token debe fallar")
	}
	sender, err := NewLabsMobile(et.Json{"username": "ana@empresa.co", "api_token": "tok"})
	if err != nil || sender.(*LabsMobile).Url != labsMobileUrl {
		t.Errorf("url por defecto: %v %v", sender, err)
	}
}

/**
* TestLabsMobileSMS: El cuerpo y la autenticación de un SMS, en GSM y en Unicode
* @param t *testing.T
**/
func TestLabsMobileSMS(t *testing.T) {
	var auth string
	var body et.Json
	server := fakeLabsMobile(t, http.StatusOK, `{"subid":"65f33a88ceb3d","code":"0","message":"Message has been successfully sent."}`, &auth, &body)
	defer server.Close()

	// Texto GSM, sin remitente: sin ucs2, sin long ni tpoa; el número sin «+»
	sender := labsMobileOf(t, server.URL, "")
	result, err := sender.SendSMS(SMS{To: "+573001234567", Text: "Tu codigo es 123456"})
	if err != nil {
		t.Fatal(err)
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("ana@empresa.co:tok"))
	if auth != want {
		t.Errorf("autorización %q", auth)
	}
	recipient := body.ArrayJson("recipient")
	if len(recipient) != 1 || recipient[0].Str("msisdn") != "573001234567" || body.Str("message") != "Tu codigo es 123456" {
		t.Errorf("cuerpo %v", body)
	}
	for _, key := range []string{"tpoa", "ucs2", "long"} {
		if _, has := body[key]; has {
			t.Errorf("no debía llevar %s: %v", key, body)
		}
	}
	if result.Str("provider") != "labsmobile" || result.Json("response").Str("subid") != "65f33a88ceb3d" {
		t.Errorf("respuesta %v", result)
	}

	// Con tilde (fuera de GSM 7): ucs2; más de 70 caracteres: long; con remitente: tpoa
	sender = labsMobileOf(t, server.URL, "CoreStudio")
	text := "Tu código de verificación es 123456. Vence en 5 minutos. No lo compartas con nadie."
	if _, err := sender.SendSMS(SMS{To: "573001234567", Text: text}); err != nil {
		t.Fatal(err)
	}
	if body.Int("ucs2") != 1 || body.Int("long") != 1 || body.Str("tpoa") != "CoreStudio" {
		t.Errorf("unicode, largo y remitente: %v", body)
	}
}

/**
* TestLabsMobileErrors: Las respuestas de error de LabsMobile
* @param t *testing.T
**/
func TestLabsMobileErrors(t *testing.T) {
	var auth string
	var body et.Json
	cases := []struct {
		status int
		answer string
		want   string
	}{
		{http.StatusOK, `{"subid":"x","code":"35","message":"The account has no enough credit for this sending"}`, "35"},
		{http.StatusUnauthorized, `{"code":"401","message":"Unauthorized"}`, "Unauthorized"},
	}
	for _, tc := range cases {
		server := fakeLabsMobile(t, tc.status, tc.answer, &auth, &body)
		_, err := labsMobileOf(t, server.URL, "").SendSMS(SMS{To: "573001234567", Text: "hola"})
		server.Close()
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%d: error %v, se esperaba %q", tc.status, err, tc.want)
		}
	}

	sender := labsMobileOf(t, "http://127.0.0.1:1", "")
	if _, err := sender.SendEmail(Email{To: "a@b.co"}); err == nil {
		t.Error("el correo no lo envía")
	}
	if _, err := sender.SendWhatsApp(WhatsApp{To: "573001234567"}); err == nil {
		t.Error("el WhatsApp no lo envía")
	}
}

/**
* TestIsGsm7: isGsm7 distingue el alfabeto GSM 7 bits
* @param t *testing.T
**/
func TestIsGsm7(t *testing.T) {
	for text, want := range map[string]bool{
		"Tu codigo es 123456": true,
		"Año, niño, à, ü, €":  true,
		"código":              false,
		"hola 👋":              false,
	} {
		if got := isGsm7(text); got != want {
			t.Errorf("%q: %v", text, got)
		}
	}
}
