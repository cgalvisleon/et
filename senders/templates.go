package senders

import (
	"github.com/cgalvisleon/et/et"
)

// WhatsAppTemplate: una plantilla de WhatsApp para el proveedor: Id es el del proveedor (vacío al crearla), Name en
// minúsculas, dígitos y _, Language (es, es_MX, en…), Category (MARKETING, UTILITY o AUTHENTICATION) y Structure
// {header?, body, footer?, buttons?} con la forma de Infobip
type WhatsAppTemplate struct {
	Id        string
	Name      string
	Language  string
	Category  string
	Structure et.Json
}

// TemplateState: el estado de una plantilla en el proveedor (Status APPROVED, PENDING, REJECTED…; Quality HIGH, MEDIUM,
// LOW o UNKNOWN)
type TemplateState struct {
	Id       string
	Name     string
	Language string
	Category string
	Status   string
	Quality  string
}

// TemplateManager: un proveedor que publica plantillas de WhatsApp (opcional: no todos lo hacen); sender es el número
// de WhatsApp del remitente en formato internacional
type TemplateManager interface {
	PublishWhatsAppTemplate(sender string, template WhatsAppTemplate) (TemplateState, error)
	WhatsAppTemplateState(sender, name, language string) (TemplateState, bool, error)
	DeleteWhatsAppTemplate(sender, name string) error
}

/**
* stateOf: El estado de una respuesta del proveedor {id, name, language, category, status, quality}; lo que falta sale de la plantilla enviada (y
* status PENDING)
* @param output et.Json, template WhatsAppTemplate
* @return TemplateState
**/
func stateOf(output et.Json, template WhatsAppTemplate) TemplateState {
	return TemplateState{
		Id:       output.ValStr(template.Id, "id"),
		Name:     output.ValStr(template.Name, "name"),
		Language: output.ValStr(template.Language, "language"),
		Category: output.ValStr(template.Category, "category"),
		Status:   output.ValStr("PENDING", "status"),
		Quality:  output.ValStr("UNKNOWN", "quality"),
	}
}

/**
* ToJson: El estado como {id, name, language, category, status, quality}
* @return et.Json
**/
func (s TemplateState) ToJson() et.Json {
	return et.Json{
		"id":       s.Id,
		"name":     s.Name,
		"language": s.Language,
		"category": s.Category,
		"status":   s.Status,
		"quality":  s.Quality,
	}
}
