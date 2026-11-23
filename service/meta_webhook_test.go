package service

import (
	"testing"
)

// metaWebhookSample: un POST de Meta con un texto, un botón de plantilla, una respuesta de botones, un estado failed con
// biz_opaque_callback_data y un error de la cuenta (formas de la documentación de Meta, 2026-11-16)
const metaWebhookSample = `{
  "object": "whatsapp_business_account",
  "entry": [{
    "id": "102290129340398",
    "changes": [{
      "field": "messages",
      "value": {
        "messaging_product": "whatsapp",
        "metadata": {"display_phone_number": "15550783881", "phone_number_id": "106540352242922"},
        "contacts": [{"profile": {"name": "Sheena Nelson"}, "wa_id": "16505551234", "user_id": "US.1349"}],
        "messages": [
          {"from": "16505551234", "from_user_id": "US.1349", "id": "wamid.T", "timestamp": "1749416383", "type": "text", "text": {"body": "Hola"}},
          {"context": {"from": "15550783881", "id": "wamid.CTX"}, "from": "16505551234", "id": "wamid.B", "timestamp": "1749416384", "type": "button", "button": {"payload": "SI", "text": "Sí"}},
          {"from": "16505551234", "id": "wamid.I", "timestamp": "1749416385", "type": "interactive", "interactive": {"type": "list_reply", "list_reply": {"id": "row-2", "title": "Martes", "description": "Tarde"}}},
          {"from": "16505551234", "id": "wamid.L", "timestamp": "1749416386", "type": "location", "location": {"latitude": 4.6, "longitude": -74.08}}
        ],
        "statuses": [{
          "id": "wamid.S", "status": "failed", "timestamp": "1750263773", "recipient_id": "16505551234",
          "biz_opaque_callback_data": "{\"order\":\"A-1\"}",
          "errors": [{"code": 131047, "title": "Re-engagement message", "message": "Re-engagement message", "error_data": {"details": "more than 24 hours"}, "href": "/docs"}]
        }],
        "errors": [{"code": 131000, "title": "Something went wrong", "message": "Something went wrong", "error_data": {"details": "x"}}]
      }
    }, {
      "field": "account_update",
      "value": {}
    }]
  }]
}`

/**
* TestMetaSignature: MetaSignatureOk acepta la firma válida y rechaza las demás
* @param t *testing.T
**/
func TestMetaSignature(t *testing.T) {
	raw := []byte(metaWebhookSample)
	signature := MetaSignature(raw, "secreto")
	if !MetaSignatureOk(raw, signature, "secreto") {
		t.Error("la firma válida no pasa")
	}
	bad := []string{"", "sha256=", "sha256=zz", "sha1=" + signature[7:], MetaSignature(raw, "otro")}
	for _, item := range bad {
		if MetaSignatureOk(raw, item, "secreto") {
			t.Errorf("firma %q pasa", item)
		}
	}
	if MetaSignatureOk(append(raw, ' '), signature, "secreto") {
		t.Error("un cuerpo cambiado pasa")
	}
	if MetaSignatureOk(raw, signature, "") {
		t.Error("sin app secret pasa")
	}
	if !MetaVerifyTokenOk("tok", "tok") || MetaVerifyTokenOk("tok", "") || MetaVerifyTokenOk("", "") || MetaVerifyTokenOk("a", "b") {
		t.Error("verify token")
	}
}

/**
* TestParseMetaWebhook: ParseMetaWebhook saca los eventos de un POST de ejemplo
* @param t *testing.T
**/
func TestParseMetaWebhook(t *testing.T) {
	events, err := ParseMetaWebhook([]byte(metaWebhookSample))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 6 {
		t.Fatalf("eventos %d: %+v", len(events), events)
	}
	for _, item := range events {
		if item.PhoneNumberId != "106540352242922" || item.WabaId != "102290129340398" || item.DisplayPhone != "15550783881" {
			t.Errorf("número %+v", item)
		}
	}

	text := events[0].Message
	if events[0].Kind != MetaEventMessage || text.Text != "Hola" || text.From != "16505551234" || text.FromUserId != "US.1349" ||
		text.Name != "Sheena Nelson" || text.MessageId != "wamid.T" || text.Timestamp != 1749416383 {
		t.Errorf("texto %+v", text)
	}
	button := events[1].Message
	if button.Text != "Sí" || button.Reply == nil || button.Reply.Id != "SI" || button.Context.Str("id") != "wamid.CTX" {
		t.Errorf("botón %+v", button)
	}
	list := events[2].Message
	if list.Text != "Martes" || list.Reply.Type != "list_reply" || list.Reply.Id != "row-2" || list.Reply.Description != "Tarde" {
		t.Errorf("lista %+v", list)
	}
	location := events[3].Message
	if location.Type != "location" || location.Text != "" || location.Content.Num("latitude") != 4.6 {
		t.Errorf("ubicación %+v", location)
	}

	status := events[4].Status
	if events[4].Kind != MetaEventStatus || status.Status != "failed" || status.MessageId != "wamid.S" || status.Recipient != "16505551234" ||
		status.CallbackData != `{"order":"A-1"}` || len(status.Errors) != 1 || status.Errors[0].Code != 131047 || status.Errors[0].Details != "more than 24 hours" {
		t.Errorf("estado %+v", status)
	}
	if events[5].Kind != MetaEventError || events[5].Error.Code != 131000 {
		t.Errorf("error %+v", events[5])
	}

	if _, err := ParseMetaWebhook([]byte(`{"object":"page","entry":[]}`)); err == nil {
		t.Error("otro objeto sin error")
	}
	if _, err := ParseMetaWebhook([]byte(`{`)); err == nil {
		t.Error("JSON inválido sin error")
	}
}
