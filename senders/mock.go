package senders

import (
	"cmp"
	"encoding/json"
	"strings"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

// mockLimit: los mensajes que guarda el proveedor mock por tenant (los más recientes)
const mockLimit = 50

// Mock: un proveedor que no envía nada (para desarrollo y pruebas): cada mensaje va al log y a la lista del tenant en la
// caché (la comparten tenancy, engine y reports), y responde con un id falso. No tiene parámetros; tenant_id lo pone
// send.tenantClients
type Mock struct {
	TenantId string
}

/**
* mockKey: La lista de los mensajes de un tenant en la caché
* @param tenantId string
* @return string
**/
func mockKey(tenantId string) string {
	return "send:mock:" + tenantId
}

/**
* NewMock: El proveedor mock de un tenant
* @param params et.Json
* @return Sender, error
**/
func NewMock(params et.Json) (Sender, error) {
	return &Mock{TenantId: params.Str("tenant_id")}, nil
}

/**
* record: Guarda el mensaje (los últimos mockLimit del tenant) y responde como un proveedor
* @param channel string, message et.Json
* @return et.Json, error
**/
func (s *Mock) record(channel string, message et.Json) (et.Json, error) {
	id := reg.UUID()
	entry := et.Json{"id": id, "channel": channel, "message": message, "sent_at": timezone.Now()}
	logs.Logf("mock", "%s %s", channel, entry.ToString())

	// Sin caché el mensaje queda solo en el log: el envío no falla por eso
	key := mockKey(s.TenantId)
	if err := cache.LPush(key, entry.ToString()); err != nil {
		logs.Alert(err)
	} else if err := cache.LTrim(key, 0, mockLimit-1); err != nil {
		logs.Alert(err)
	}

	return et.Json{"provider": "mock", "response": et.Json{"messageId": id}}, nil
}

/**
* SendSMS: Registra el SMS
* @param message SMS
* @return et.Json, error
**/
func (s *Mock) SendSMS(message SMS) (et.Json, error) {
	data := et.Json{"to": message.To, "text": message.Text, "kind": message.Kind}
	return s.record("sms", data)
}

/**
* SendEmail: Registra el correo; de los adjuntos, solo su nombre, tipo y tamaño
* @param message Email
* @return et.Json, error
**/
func (s *Mock) SendEmail(message Email) (et.Json, error) {
	attachments := []et.Json{}
	for _, attachment := range message.Attachments {
		attachments = append(attachments, et.Json{"filename": attachment.Filename, "contentType": attachment.ContentType, "size": len(attachment.Content)})
	}

	data := et.Json{
		"from":        message.From,
		"to":          message.To,
		"subject":     message.Subject,
		"html":        message.Html,
		"kind":        message.Kind,
		"attachments": attachments,
	}
	return s.record("email", data)
}

/**
* SendWhatsApp: Registra el WhatsApp
* @param message WhatsApp
* @return et.Json, error
**/
func (s *Mock) SendWhatsApp(message WhatsApp) (et.Json, error) {
	data := et.Json{
		"from":     message.From,
		"to":       message.To,
		"type":     message.Type,
		"text":     message.Text,
		"template": message.Template,
		"media":    message.Media,
		"metadata": message.Metadata,
		"kind":     message.Kind,
		"otp":      message.Otp,
	}
	return s.record("whatsapp", data)
}

/**
* MockMessages: Los mensajes que el proveedor mock registró para un tenant, el más reciente primero
* @param tenantId string
* @return []et.Json, error
**/
func MockMessages(tenantId string) ([]et.Json, error) {
	key := mockKey(tenantId)
	items, err := cache.LRange(key, 0, mockLimit-1)
	if err != nil {
		return nil, err
	}

	result := []et.Json{}
	for _, item := range items {
		entry := et.Json{}
		if err := json.Unmarshal([]byte(item), &entry); err != nil {
			continue
		}
		result = append(result, entry)
	}

	return result, nil
}

/**
* ClearMockMessages: Borra los mensajes del proveedor mock de un tenant
* @param tenantId string
* @return error
**/
func ClearMockMessages(tenantId string) error {
	key := mockKey(tenantId)
	_, err := cache.Delete(key)
	return err
}

/**
* mockTemplatesKey: Las plantillas de WhatsApp que el proveedor mock guarda para un tenant (un hash de la caché: nombre:idioma → estado)
* @param tenantId string
* @return string
**/
func mockTemplatesKey(tenantId string) string {
	return "send:mock:templates:" + tenantId
}

/**
* PublishWhatsAppTemplate: Guarda la plantilla aprobada al instante (APPROVED, HIGH); sin id, le da uno; sin categoría, la guardada
* @param sender string, template WhatsAppTemplate
* @return TemplateState, error
**/
func (s *Mock) PublishWhatsAppTemplate(sender string, template WhatsAppTemplate) (TemplateState, error) {
	// Una edición sin categoría conserva la guardada, como en Infobip
	key := mockTemplatesKey(s.TenantId)
	field := template.Name + ":" + template.Language
	saved, _, err := cache.ObjetGet(key, field)
	if err != nil {
		return TemplateState{}, err
	}

	state := TemplateState{
		Id:       cmp.Or(template.Id, reg.UUID()),
		Name:     template.Name,
		Language: template.Language,
		Category: cmp.Or(template.Category, saved.Str("category")),
		Status:   "APPROVED",
		Quality:  "HIGH",
	}
	entry := state.ToJson()
	entry["sender"] = sender
	entry["structure"] = template.Structure
	logs.Logf("mock", "whatsapp template %s", entry.ToString())

	if err := cache.ObjetSet(key, field, entry); err != nil {
		return TemplateState{}, err
	}

	return state, nil
}

/**
* WhatsAppTemplateState: La plantilla guardada de ese nombre e idioma; false si no está
* @param sender, name, language string
* @return TemplateState, bool, error
**/
func (s *Mock) WhatsAppTemplateState(sender, name, language string) (TemplateState, bool, error) {
	key := mockTemplatesKey(s.TenantId)
	field := name + ":" + language
	entry, found, err := cache.ObjetGet(key, field)
	if err != nil || !found {
		return TemplateState{}, false, err
	}

	template := WhatsAppTemplate{Name: name, Language: language}
	return stateOf(entry, template), true, nil
}

/**
* DeleteWhatsAppTemplate: Borra las plantillas guardadas de ese nombre (en cada idioma, como Infobip)
* @param sender, name string
* @return error
**/
func (s *Mock) DeleteWhatsAppTemplate(sender, name string) error {
	key := mockTemplatesKey(s.TenantId)
	fields, err := cache.CollectionGet(key)
	if err != nil {
		return err
	}

	for field := range fields {
		if strings.HasPrefix(field, name+":") {
			if err := cache.CollectionDelete(key, field); err != nil {
				return err
			}
		}
	}

	return nil
}
