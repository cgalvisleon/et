package senders

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/cgalvisleon/et/et"
)

// metaCall: una llamada que recibió la Graph API falsa
type metaCall struct {
	method string
	path   string
	auth   string
	body   et.Json
}

/**
* fakeMeta: Una Graph API falsa: guarda cada llamada y responde lo que answer diga para ella (status y JSON)
* @param t *testing.T, calls *[]metaCall, answer func(call metaCall) (int, string)
* @return *httptest.Server
**/
func fakeMeta(t *testing.T, calls *[]metaCall, answer func(call metaCall) (int, string)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body := et.Json{}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatal(err)
			}
		}
		call := metaCall{method: r.Method, path: r.URL.RequestURI(), auth: r.Header.Get("Authorization"), body: body}
		*calls = append(*calls, call)
		status, output := answer(call)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		io.WriteString(w, output)
	}))
}

/**
* metaOf: Un Meta contra url
* @param t *testing.T, url, waba string
* @return *Meta
**/
func metaOf(t *testing.T, url, waba string) *Meta {
	sender, err := NewMeta(et.Json{"base_url": url, "access_token": "tok", "phone_number_id": "1055", "waba_id": waba})
	if err != nil {
		t.Fatal(err)
	}
	return sender.(*Meta)
}

/**
* TestNewMeta: NewMeta exige phone_number_id y arma la url con la versión
* @param t *testing.T
**/
func TestNewMeta(t *testing.T) {
	if _, err := NewMeta(et.Json{"access_token": "tok"}); err == nil {
		t.Error("sin phone_number_id debe fallar")
	}
	sender, err := NewMeta(et.Json{"access_token": "tok", "phone_number_id": "1055", "api_version": "v26.0"})
	if err != nil || sender.(*Meta).Url != metaUrl+"/v26.0" {
		t.Errorf("url %v %v", sender, err)
	}
}

/**
* TestMetaWhatsApp: El cuerpo de cada tipo de mensaje de WhatsApp
* @param t *testing.T
**/
func TestMetaWhatsApp(t *testing.T) {
	calls := []metaCall{}
	server := fakeMeta(t, &calls, func(metaCall) (int, string) {
		return http.StatusOK, `{"messaging_product":"whatsapp","contacts":[{"input":"573001234567","wa_id":"573001234567"}],"messages":[{"id":"wamid.X"}]}`
	})
	defer server.Close()
	sender := metaOf(t, server.URL, "")

	template := et.Json{"templateName": "cambio_contacto", "language": "es", "templateData": et.Json{
		"body":   et.Json{"placeholders": []any{"Ana", "teléfono"}},
		"header": et.Json{"type": "TEXT", "placeholder": "Aviso"},
	}}
	interactive := et.Json{
		"body":   et.Json{"text": "¿Confirmas?"},
		"header": et.Json{"type": "TEXT", "text": "Pedido"},
		"footer": et.Json{"text": "CoreStudio"},
		"action": et.Json{"buttons": []any{et.Json{"type": "REPLY", "id": "si", "title": "Sí"}}},
	}
	messages := []WhatsApp{
		{To: "+573001234567", Type: "text", Text: "Hola"},
		{To: "573001234567", Type: "template", Template: template},
		{To: "573001234567", Type: "document", Media: et.Json{"mediaUrl": "https://x/f.pdf", "caption": "Factura", "filename": "f.pdf"}},
		{To: "573001234567", Type: "interactive", Media: interactive},
	}
	for _, message := range messages {
		result, err := sender.SendWhatsApp(message)
		if err != nil {
			t.Fatalf("%s: %v", message.Type, err)
		}
		if result.Str("provider") != "meta" {
			t.Errorf("%s: %v", message.Type, result)
		}
	}

	if calls[0].path != "/v25.0/1055/messages" || calls[0].auth != "Bearer tok" || calls[0].method != http.MethodPost {
		t.Errorf("llamada %+v", calls[0])
	}
	text := calls[0].body
	if text.Str("to") != "573001234567" || text.Str("messaging_product") != "whatsapp" || text.Json("text").Str("body") != "Hola" {
		t.Errorf("texto %v", text)
	}
	sent := calls[1].body.Json("template")
	components := sent.ArrayJson("components")
	if sent.Str("name") != "cambio_contacto" || sent.Json("language").Str("code") != "es" || len(components) != 2 {
		t.Fatalf("plantilla %v", sent)
	}
	if components[0].Str("type") != "header" || components[1].ArrayJson("parameters")[1].Str("text") != "teléfono" {
		t.Errorf("componentes %v", components)
	}
	document := calls[2].body.Json("document")
	if document.Str("link") != "https://x/f.pdf" || document.Str("filename") != "f.pdf" || document.Str("caption") != "Factura" {
		t.Errorf("documento %v", document)
	}
	buttons := calls[3].body.Json("interactive")
	reply := buttons.Json("action").ArrayJson("buttons")[0]
	if buttons.Str("type") != "button" || reply.Str("type") != "reply" || reply.Json("reply").Str("title") != "Sí" || buttons.Json("header").Str("text") != "Pedido" {
		t.Errorf("interactivo %v", buttons)
	}

	if _, err := sender.SendWhatsApp(WhatsApp{To: "1", Type: "sticker"}); err == nil {
		t.Error("tipo desconocido sin error")
	}
	if _, err := sender.SendSMS(SMS{To: "1"}); err == nil {
		t.Error("el SMS no lo envía")
	}
}

/**
* TestMetaError: Un error de Meta trae su mensaje y su código
* @param t *testing.T
**/
func TestMetaError(t *testing.T) {
	calls := []metaCall{}
	server := fakeMeta(t, &calls, func(metaCall) (int, string) {
		return http.StatusBadRequest, `{"error":{"message":"(#131030) Recipient phone number not in allowed list","type":"OAuthException","code":131030}}`
	})
	defer server.Close()

	_, err := metaOf(t, server.URL, "").SendWhatsApp(WhatsApp{To: "573001234567", Type: "text", Text: "Hola"})
	if err == nil || !strings.Contains(err.Error(), "not in allowed list") || !strings.Contains(err.Error(), "131030") {
		t.Errorf("error %v", err)
	}
}

/**
* TestMetaTemplates: Publicar, consultar y borrar plantillas en Meta
* @param t *testing.T
**/
func TestMetaTemplates(t *testing.T) {
	calls := []metaCall{}
	server := fakeMeta(t, &calls, func(call metaCall) (int, string) {
		switch {
		case call.method == http.MethodPost && strings.HasSuffix(call.path, "/message_templates"):
			return http.StatusOK, `{"id":"9001","status":"PENDING","category":"UTILITY"}`
		case call.method == http.MethodPost:
			return http.StatusOK, `{"success":true}`
		case call.method == http.MethodGet:
			return http.StatusOK, `{"data":[{"id":"8000","name":"cambio_contacto_v2","language":"es","status":"APPROVED"},{"id":"9001","name":"cambio_contacto","language":"es","status":"APPROVED","category":"UTILITY","quality_score":{"score":"GREEN"}}]}`
		}
		return http.StatusOK, `{"success":true}`
	})
	defer server.Close()

	if _, err := metaOf(t, server.URL, "").PublishWhatsAppTemplate("", WhatsAppTemplate{Name: "x"}); err == nil {
		t.Error("sin waba_id debe fallar")
	}

	sender := metaOf(t, server.URL, "waba1")
	structure := et.Json{
		"header":  et.Json{"format": "TEXT", "text": "Aviso {{1}}", "example": "Cuenta"},
		"body":    et.Json{"text": "Hola {{1}}, se actualizó tu {{2}}.", "examples": []any{"Ana", "teléfono"}},
		"footer":  et.Json{"text": "CoreStudio"},
		"buttons": []any{et.Json{"type": "QUICK_REPLY", "text": "No fui yo"}},
	}
	template := WhatsAppTemplate{Name: "cambio_contacto", Language: "es", Category: "UTILITY", Structure: structure}
	created, err := sender.PublishWhatsAppTemplate("", template)
	if err != nil || created.Id != "9001" || created.Status != "PENDING" {
		t.Fatalf("crear %+v %v", created, err)
	}
	body := calls[len(calls)-1].body
	components := body.ArrayJson("components")
	if calls[len(calls)-1].path != "/v25.0/waba1/message_templates" || body.Str("name") != "cambio_contacto" || len(components) != 4 {
		t.Fatalf("cuerpo %v", body)
	}
	// body_text es una lista con una lista de ejemplos (uno por {{n}})
	example := components[1].Json("example").Array("body_text")
	if components[1].Str("type") != "BODY" || len(example) != 1 {
		t.Errorf("body %v", components[1])
	}
	if len(components[0].Json("example").Array("header_text")) != 1 || components[3].ArrayJson("buttons")[0].Str("text") != "No fui yo" {
		t.Errorf("encabezado o botones %v", components)
	}

	// Editar: POST al id y después su estado (el mismo nombre e idioma, no el parecido)
	template.Id = "9001"
	edited, err := sender.PublishWhatsAppTemplate("", template)
	if err != nil || edited.Status != "APPROVED" || edited.Quality != "HIGH" || edited.Id != "9001" {
		t.Fatalf("editar %+v %v", edited, err)
	}
	if calls[len(calls)-2].path != "/v25.0/9001" || calls[len(calls)-1].method != http.MethodGet {
		t.Errorf("llamadas al editar %+v", calls[len(calls)-2:])
	}

	_, found, err := sender.WhatsAppTemplateState("", "cambio_contacto", "en")
	if err != nil || found {
		t.Errorf("otro idioma no está: %v %v", found, err)
	}

	if err := sender.DeleteWhatsAppTemplate("", "cambio_contacto"); err != nil {
		t.Fatal(err)
	}
	last := calls[len(calls)-1]
	if last.method != http.MethodDelete || last.path != "/v25.0/waba1/message_templates?name=cambio_contacto" {
		t.Errorf("borrar %+v", last)
	}
}

/**
* TestMetaWhatsAppOtp: el código va como parámetro del cuerpo y del botón (button, sub_type url, index "0")
* @param t *testing.T
**/
func TestMetaWhatsAppOtp(t *testing.T) {
	calls := []metaCall{}
	server := fakeMeta(t, &calls, func(metaCall) (int, string) {
		return http.StatusOK, `{"messages":[{"id":"wamid.X"}]}`
	})
	defer server.Close()

	template := et.Json{"templateName": "verification_code", "language": "en_US"}
	if _, err := metaOf(t, server.URL, "").SendWhatsApp(WhatsApp{To: "12015553931", Type: "template", Template: template, Otp: "482913"}); err != nil {
		t.Fatal(err)
	}

	sent := calls[0].body.Json("template")
	components := sent.ArrayJson("components")
	if sent.Str("name") != "verification_code" || sent.Json("language").Str("code") != "en_US" || len(components) != 2 {
		t.Fatalf("plantilla %v", sent)
	}
	body, button := components[0], components[1]
	if body.Str("type") != "body" || body.ArrayJson("parameters")[0].Str("text") != "482913" {
		t.Errorf("cuerpo %v", body)
	}
	if button.Str("type") != "button" || button.Str("sub_type") != "url" || button.Str("index") != "0" || button.ArrayJson("parameters")[0].Str("text") != "482913" {
		t.Errorf("botón %v", button)
	}
}

/**
* TestMetaAuthComponents: una estructura AUTHENTICATION se publica con body add_security_recommendation, footer code_expiration_minutes y el botón OTP COPY_CODE
* @param t *testing.T
**/
func TestMetaAuthComponents(t *testing.T) {
	structure := et.Json{
		"body":    et.Json{"addSecurityRecommendation": true},
		"footer":  et.Json{"codeExpirationMinutes": 10},
		"buttons": []any{et.Json{"otpType": "COPY_CODE", "text": "Copiar código"}},
	}
	components := metaComponentsOf(structure)
	if len(components) != 3 || components[0].Str("type") != "BODY" || !components[0].Bool("add_security_recommendation") || components[0]["text"] != nil {
		t.Fatalf("componentes %v", components)
	}
	button := components[2].ArrayJson("buttons")[0]
	if components[1].Int("code_expiration_minutes") != 10 || button.Str("type") != "OTP" || button.Str("otp_type") != "COPY_CODE" || button.Str("text") != "Copiar código" {
		t.Errorf("footer o botón %v", components)
	}

	plain := metaComponentsOf(et.Json{"body": et.Json{"addSecurityRecommendation": false}, "buttons": []any{et.Json{"otpType": "COPY_CODE"}}})
	if len(plain) != 2 || plain[1].ArrayJson("buttons")[0]["text"] != nil {
		t.Errorf("sin footer ni texto %v", plain)
	}
}

/**
* TestMetaMetadata: Metadata sale como biz_opaque_callback_data (texto JSON); sin Metadata no se manda
* @param t *testing.T
**/
func TestMetaMetadata(t *testing.T) {
	calls := []metaCall{}
	server := fakeMeta(t, &calls, func(metaCall) (int, string) {
		return http.StatusOK, `{"messages":[{"id":"wamid.X"}]}`
	})
	defer server.Close()
	sender := metaOf(t, server.URL, "")

	metadata := et.Json{"order": "A-1", "_origin": et.Json{"flow_id": "flow:x"}}
	if _, err := sender.SendWhatsApp(WhatsApp{To: "573001234567", Type: "text", Text: "Hola", Metadata: metadata}); err != nil {
		t.Fatal(err)
	}
	if _, err := sender.SendWhatsApp(WhatsApp{To: "573001234567", Type: "text", Text: "Hola"}); err != nil {
		t.Fatal(err)
	}

	raw := calls[0].body.Str("biz_opaque_callback_data")
	back := et.Json{}
	if err := json.Unmarshal([]byte(raw), &back); err != nil || back.Str("order") != "A-1" || back.Json("_origin").Str("flow_id") != "flow:x" {
		t.Errorf("biz_opaque_callback_data %q %v", raw, err)
	}
	if _, has := calls[1].body["biz_opaque_callback_data"]; has {
		t.Errorf("sin Metadata no se manda: %v", calls[1].body)
	}
}

/**
* TestMetaMetadataTooLong: Metadata de más de 512 caracteres falla con MSG_META_METADATA_TOO_LONG sin llamar a Meta; de 512 justos sale (cuenta caracteres, no bytes)
* @param t *testing.T
**/
func TestMetaMetadataTooLong(t *testing.T) {
	calls := []metaCall{}
	server := fakeMeta(t, &calls, func(metaCall) (int, string) {
		return http.StatusOK, `{"messages":[{"id":"wamid.X"}]}`
	})
	defer server.Close()
	sender := metaOf(t, server.URL, "")

	// {"n":"…"} son 8 caracteres más el valor
	long := et.Json{"n": strings.Repeat("ñ", metaCallbackMax-7)}
	_, err := sender.SendWhatsApp(WhatsApp{To: "573001234567", Type: "text", Text: "Hola", Metadata: long})
	if err == nil || !strings.Contains(err.Error(), "513") {
		t.Errorf("debe fallar por largo: %v", err)
	}
	if len(calls) != 0 {
		t.Errorf("llamó a Meta: %v", calls)
	}

	exact := et.Json{"n": strings.Repeat("ñ", metaCallbackMax-8)}
	if _, err := sender.SendWhatsApp(WhatsApp{To: "573001234567", Type: "text", Text: "Hola", Metadata: exact}); err != nil {
		t.Fatal(err)
	}
	if raw := calls[0].body.Str("biz_opaque_callback_data"); utf8.RuneCountInString(raw) != metaCallbackMax {
		t.Errorf("largo %d", utf8.RuneCountInString(raw))
	}
}
