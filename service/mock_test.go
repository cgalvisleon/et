package service

import (
	"testing"

	"github.com/cgalvisleon/et/cache"

	"github.com/cgalvisleon/et/et"
)

/**
* TestMock: El proveedor mock no envía nada y responde como un proveedor por cada canal, aun sin caché
* @param t *testing.T
**/
func TestMock(t *testing.T) {
	sender, err := New("mock", et.Json{"tenant_id": "tenant:test"})
	if err != nil {
		t.Fatal(err)
	}

	results := []et.Json{}
	sms, err := sender.SendSMS(SMS{To: "573001234567", Text: "Hola"})
	if err != nil {
		t.Fatal(err)
	}
	results = append(results, sms)

	attachment := Attachment{Filename: "r.pdf", ContentType: "application/pdf", Content: []byte("%PDF")}
	email, err := sender.SendEmail(Email{To: "ana@x.co", Subject: "S", Html: "<p>x</p>", Attachments: []Attachment{attachment}})
	if err != nil {
		t.Fatal(err)
	}
	results = append(results, email)

	whatsApp, err := sender.SendWhatsApp(WhatsApp{To: "573001234567", Type: "text", Text: "Hola"})
	if err != nil {
		t.Fatal(err)
	}
	results = append(results, whatsApp)

	for _, result := range results {
		if result.Str("provider") != "mock" || result.Json("response").Str("messageId") == "" {
			t.Errorf("result %v", result)
		}
	}
}

/**
* TestMockTemplates: Publica (aprobada al instante), consulta y borra en la caché del tenant (en memoria, sin REDIS_HOST)
* @param t *testing.T
**/
func TestMockTemplates(t *testing.T) {
	t.Setenv("REDIS_HOST", "")
	if err := cache.Load(); err != nil {
		t.Fatal(err)
	}

	sender, err := New("mock", et.Json{"tenant_id": "tenant:test"})
	if err != nil {
		t.Fatal(err)
	}

	manager, ok := sender.(TemplateManager)
	if !ok {
		t.Fatal("mock is not a TemplateManager")
	}

	template := WhatsAppTemplate{Name: "qa_prueba", Language: "es", Category: "UTILITY", Structure: et.Json{"body": et.Json{"text": "Hola"}}}
	state, err := manager.PublishWhatsAppTemplate("", template)
	if err != nil {
		t.Fatal(err)
	}
	if state.Id == "" || state.Status != "APPROVED" || state.Quality != "HIGH" {
		t.Errorf("state %+v", state)
	}

	found, ok, err := manager.WhatsAppTemplateState("", "qa_prueba", "es")
	if err != nil || !ok || found.Id != state.Id || found.Category != "UTILITY" {
		t.Errorf("found %+v, ok %v, error %v", found, ok, err)
	}

	if err := manager.DeleteWhatsAppTemplate("", "qa_prueba"); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := manager.WhatsAppTemplateState("", "qa_prueba", "es"); err != nil || ok {
		t.Errorf("after delete ok %v, error %v", ok, err)
	}
}
