package senders

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cgalvisleon/et/et"
)

// call: una petición que recibió el Infobip falso
type call struct {
	Method      string
	Path        string
	Auth        string
	ContentType string
	Body        []byte
}

/**
* fakeInfobip: Un Infobip falso que guarda cada petición y responde status con answer; devuelve el sender apuntado a él
* @param t *testing.T, status int, answer string
* @return *Infobip, *[]call
**/
func fakeInfobip(t *testing.T, status int, answer string) (*Infobip, *[]call) {
	calls := []call{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		calls = append(calls, call{Method: r.Method, Path: r.URL.Path, Auth: r.Header.Get("Authorization"), ContentType: r.Header.Get("Content-Type"), Body: body})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write([]byte(answer))
	}))
	t.Cleanup(server.Close)

	params := et.Json{"base_url": server.URL, "api_key": "key-1", "sms_from": "coreStudio", "email_from": "no-reply@x.co", "whatsapp_from": "573000000000"}
	sender, err := NewInfobip(params)
	if err != nil {
		t.Fatal(err)
	}

	return sender.(*Infobip), &calls
}

/**
* jsonOf: El cuerpo JSON de una petición
* @param t *testing.T, body []byte
* @return et.Json
**/
func jsonOf(t *testing.T, body []byte) et.Json {
	result := et.Json{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("body is not JSON: %s", body)
	}

	return result
}

/**
* TestNewInfobip: base_url y api_key son obligatorios
* @param t *testing.T
**/
func TestNewInfobip(t *testing.T) {
	for _, params := range []et.Json{{}, {"base_url": "http://x"}, {"api_key": "k"}} {
		if _, err := NewInfobip(params); err == nil {
			t.Errorf("NewInfobip(%v) without error", params)
		}
	}
}

/**
* TestInfobipSMS: La ruta, la llave, el remitente, el destino y el texto; sin sms_from envía sin remitente
* @param t *testing.T
**/
func TestInfobipSMS(t *testing.T) {
	sender, calls := fakeInfobip(t, http.StatusOK, `{"messages":[{"messageId":"m1"}]}`)
	result, err := sender.SendSMS(SMS{To: "573001234567", Text: "Hola", Kind: KindNotification})
	if err != nil {
		t.Fatal(err)
	}

	got := (*calls)[0]
	if got.Path != "/sms/3/messages" || got.Auth != "App key-1" {
		t.Errorf("path %s, auth %s", got.Path, got.Auth)
	}
	message := jsonOf(t, got.Body).ArrayJson("messages")[0]
	if message.Str("sender") != "coreStudio" || message.ArrayJson("destinations")[0].Str("to") != "573001234567" || message.Json("content").Str("text") != "Hola" {
		t.Errorf("message %v", message)
	}
	if result.Str("provider") != "infobip" {
		t.Errorf("result %v", result)
	}

	sender.SmsFrom = ""
	if _, err := sender.SendSMS(SMS{To: "1", Text: "x"}); err != nil {
		t.Fatalf("SMS without sms_from: %v", err)
	}
	if message := jsonOf(t, (*calls)[1].Body).ArrayJson("messages")[0]; message["sender"] != nil {
		t.Errorf("sender without sms_from: %v", message)
	}
}

/**
* TestInfobipEmail: Sin adjuntos va en JSON con el remitente del sender si el mensaje no trae uno; con adjuntos, multipart
* @param t *testing.T
**/
func TestInfobipEmail(t *testing.T) {
	sender, calls := fakeInfobip(t, http.StatusOK, `{"bulkId":"b1"}`)
	if _, err := sender.SendEmail(Email{To: "ana@x.co", Subject: "Asunto", Html: "<p>Hola</p>"}); err != nil {
		t.Fatal(err)
	}

	got := (*calls)[0]
	message := jsonOf(t, got.Body).ArrayJson("messages")[0]
	if got.Path != "/email/4/messages" || message.Str("sender") != "no-reply@x.co" || message.Json("content").Str("subject") != "Asunto" {
		t.Errorf("path %s, message %v", got.Path, message)
	}

	attachment := Attachment{Filename: "r.pdf", ContentType: "application/pdf", Content: []byte("%PDF")}
	email := Email{From: "otro@x.co", To: "ana@x.co", Subject: "S", Html: "<p>x</p>", Attachments: []Attachment{attachment}}
	if _, err := sender.SendEmail(email); err != nil {
		t.Fatal(err)
	}

	got = (*calls)[1]
	body := string(got.Body)
	if got.Path != "/email/3/send" || !strings.HasPrefix(got.ContentType, "multipart/form-data") || got.Auth != "App key-1" {
		t.Errorf("path %s, content type %s, auth %s", got.Path, got.ContentType, got.Auth)
	}
	for _, part := range []string{"otro@x.co", `filename="r.pdf"`, "%PDF"} {
		if !strings.Contains(body, part) {
			t.Errorf("multipart without %q", part)
		}
	}
}

/**
* TestInfobipWhatsApp: La ruta de cada tipo, el remitente del sender y template dentro de messages
* @param t *testing.T
**/
func TestInfobipWhatsApp(t *testing.T) {
	sender, calls := fakeInfobip(t, http.StatusOK, `{"messageId":"w1"}`)
	cases := []struct {
		message WhatsApp
		path    string
	}{
		{WhatsApp{To: "1", Type: "text", Text: "Hola"}, "/whatsapp/1/message/text"},
		{WhatsApp{To: "1", Type: "image", Media: et.Json{"mediaUrl": "https://x.co/a.png"}}, "/whatsapp/1/message/image"},
		{WhatsApp{To: "1", Type: "interactive", Media: et.Json{"body": et.Json{"text": "¿Sí?"}}}, "/whatsapp/1/message/interactive/buttons"},
		{WhatsApp{To: "1", Type: "template", Template: et.Json{"templateName": "otp", "language": "es"}}, "/whatsapp/1/message/template"},
	}
	for i, tc := range cases {
		if _, err := sender.SendWhatsApp(tc.message); err != nil {
			t.Fatalf("%s: %v", tc.message.Type, err)
		}

		got := (*calls)[i]
		body := jsonOf(t, got.Body)
		one := body
		if tc.message.Type == "template" {
			one = body.ArrayJson("messages")[0]
		}
		if got.Path != tc.path || one.Str("from") != "573000000000" || one.Str("to") != "1" {
			t.Errorf("%s: path %s, body %v", tc.message.Type, got.Path, body)
		}
	}

	if _, err := sender.SendWhatsApp(WhatsApp{To: "1", Type: "sticker"}); err == nil {
		t.Error("unknown WhatsApp type sent")
	}
}

/**
* TestInfobipError: Un 4xx devuelve el error con lo que respondió Infobip; sin servidor, el error de la conexión
* @param t *testing.T
**/
func TestInfobipError(t *testing.T) {
	answer := `{"requestError":{"serviceException":{"messageId":"UNAUTHORIZED","text":"Invalid login details"}}}`
	sender, _ := fakeInfobip(t, http.StatusUnauthorized, answer)
	_, err := sender.SendSMS(SMS{To: "1", Text: "x"})
	if err == nil || !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "Invalid login details") {
		t.Errorf("error %v", err)
	}

	sender.BaseUrl = "http://127.0.0.1:1"
	if _, err := sender.SendSMS(SMS{To: "1", Text: "x"}); err == nil {
		t.Error("no error without server")
	}
}

/**
* TestInfobipPublishTemplate: Sin id crea (POST con name, language, category y structure); con id edita (PATCH, sin name)
* @param t *testing.T
**/
func TestInfobipPublishTemplate(t *testing.T) {
	answer := `{"id":"111","name":"qa_prueba","language":"es","status":"PENDING","category":"UTILITY","quality":"UNKNOWN"}`
	sender, calls := fakeInfobip(t, http.StatusCreated, answer)
	structure := et.Json{"body": et.Json{"text": "Hola {{1}}", "examples": []string{"Ana"}}}
	template := WhatsAppTemplate{Name: "qa_prueba", Language: "es", Category: "UTILITY", Structure: structure}
	state, err := sender.PublishWhatsAppTemplate("", template)
	if err != nil {
		t.Fatal(err)
	}

	got := (*calls)[0]
	body := jsonOf(t, got.Body)
	if got.Method != http.MethodPost || got.Path != "/whatsapp/2/senders/573000000000/templates" || got.Auth != "App key-1" {
		t.Errorf("method %s, path %s, auth %s", got.Method, got.Path, got.Auth)
	}
	if body.Str("name") != "qa_prueba" || body.Str("category") != "UTILITY" || body.Str("structure", "body", "text") != "Hola {{1}}" {
		t.Errorf("body %v", body)
	}
	if state.Id != "111" || state.Status != "PENDING" || state.Quality != "UNKNOWN" {
		t.Errorf("state %+v", state)
	}

	template.Id = "111"
	if _, err := sender.PublishWhatsAppTemplate("573111111111", template); err != nil {
		t.Fatal(err)
	}

	got = (*calls)[1]
	body = jsonOf(t, got.Body)
	if got.Method != http.MethodPatch || got.Path != "/whatsapp/2/senders/573111111111/templates/111" || body.Str("name") != "" || len(body.Json("structure")) == 0 {
		t.Errorf("method %s, path %s, body %v", got.Method, got.Path, body)
	}
}

/**
* TestInfobipTemplateState: Lista las plantillas y toma la de ese nombre e idioma
* @param t *testing.T
**/
func TestInfobipTemplateState(t *testing.T) {
	answer := `{"templates":[{"id":1,"name":"qa_prueba","language":"en","status":"PENDING"},{"id":2,"name":"qa_prueba","language":"es","status":"APPROVED","quality":"HIGH","category":"UTILITY"}]}`
	sender, calls := fakeInfobip(t, http.StatusOK, answer)
	state, found, err := sender.WhatsAppTemplateState("", "qa_prueba", "es")
	if err != nil || !found {
		t.Fatalf("found %v, error %v", found, err)
	}
	if (*calls)[0].Method != http.MethodGet || state.Id != "2" || state.Status != "APPROVED" || state.Quality != "HIGH" {
		t.Errorf("method %s, state %+v", (*calls)[0].Method, state)
	}

	if _, found, err := sender.WhatsAppTemplateState("", "otra", "es"); err != nil || found {
		t.Errorf("found %v, error %v", found, err)
	}
}

/**
* TestInfobipDeleteTemplate: DELETE por nombre (204 sin cuerpo); sin número no llama
* @param t *testing.T
**/
func TestInfobipDeleteTemplate(t *testing.T) {
	sender, calls := fakeInfobip(t, http.StatusNoContent, "")
	if err := sender.DeleteWhatsAppTemplate("", "qa_prueba"); err != nil {
		t.Fatal(err)
	}

	got := (*calls)[0]
	if got.Method != http.MethodDelete || got.Path != "/whatsapp/2/senders/573000000000/templates/qa_prueba" {
		t.Errorf("method %s, path %s", got.Method, got.Path)
	}

	sender.WhatsAppFrom = ""
	if err := sender.DeleteWhatsAppTemplate("", "qa_prueba"); err == nil || len(*calls) != 1 {
		t.Error("deleted without a WhatsApp number")
	}
}

/**
* TestInfobipTemplateError: Un 400 devuelve el detalle de Infobip
* @param t *testing.T
**/
func TestInfobipTemplateError(t *testing.T) {
	answer := `{"requestError":{"serviceException":{"messageId":"BAD_REQUEST","text":"Bad request","validationErrors":{"name":["invalid"]}}}}`
	sender, _ := fakeInfobip(t, http.StatusBadRequest, answer)
	template := WhatsAppTemplate{Name: "X", Language: "es", Category: "UTILITY"}
	_, err := sender.PublishWhatsAppTemplate("", template)
	if err == nil || !strings.Contains(err.Error(), "400") || !strings.Contains(err.Error(), "validationErrors") {
		t.Errorf("error %v", err)
	}
}

/**
* TestInfobipWhatsAppOtp: una plantilla AUTHENTICATION lleva el código en el cuerpo y en el botón URL; la plantilla original no cambia
* @param t *testing.T
**/
func TestInfobipWhatsAppOtp(t *testing.T) {
	sender, calls := fakeInfobip(t, http.StatusOK, `{"messages":[{"messageId":"w1"}]}`)
	template := et.Json{"templateName": "qa_otp", "language": "es"}
	if _, err := sender.SendWhatsApp(WhatsApp{To: "573001234567", Type: "template", Template: template, Otp: "482913"}); err != nil {
		t.Fatal(err)
	}

	got := (*calls)[0]
	content := jsonOf(t, got.Body).ArrayJson("messages")[0].Json("content")
	data := content.Json("templateData")
	button := data.ArrayJson("buttons")[0]
	if got.Path != "/whatsapp/1/message/template" || content.Str("templateName") != "qa_otp" || data.Json("body").ArrayStr("placeholders")[0] != "482913" {
		t.Errorf("path %s, content %v", got.Path, content)
	}
	if button.Str("type") != "URL" || button.Str("parameter") != "482913" || len(data.ArrayJson("buttons")) != 1 {
		t.Errorf("button %v", data["buttons"])
	}
	if template["templateData"] != nil {
		t.Errorf("the template changed: %v", template)
	}
}

/**
* TestInfobipPublishAuthTemplate: la estructura AUTHENTICATION va tal cual (addSecurityRecommendation, codeExpirationMinutes y el botón con otpType)
* @param t *testing.T
**/
func TestInfobipPublishAuthTemplate(t *testing.T) {
	sender, calls := fakeInfobip(t, http.StatusCreated, `{"id":"7","status":"PENDING","category":"AUTHENTICATION"}`)
	structure := et.Json{
		"body":    et.Json{"addSecurityRecommendation": true},
		"footer":  et.Json{"codeExpirationMinutes": 5},
		"buttons": []et.Json{{"otpType": "COPY_CODE", "text": "Copiar"}},
	}
	template := WhatsAppTemplate{Name: "qa_otp", Language: "es", Category: "AUTHENTICATION", Structure: structure}
	if _, err := sender.PublishWhatsAppTemplate("", template); err != nil {
		t.Fatal(err)
	}

	body := jsonOf(t, (*calls)[0].Body)
	button := body.Json("structure").ArrayJson("buttons")[0]
	if body.Str("category") != "AUTHENTICATION" || !body.Bool("structure", "body", "addSecurityRecommendation") || body.Int("structure", "footer", "codeExpirationMinutes") != 5 || button.Str("otpType") != "COPY_CODE" {
		t.Errorf("body %v", body)
	}
}
