package jwsp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/cgalvisleon/et/et"
)

// Categorías de plantilla
const (
	CategoryAuthentication = "AUTHENTICATION"
	CategoryMarketing      = "MARKETING"
	CategoryUtility        = "UTILITY"
)

// maxTemplatePages: las páginas que Template recorre como mucho buscando un nombre
const maxTemplatePages = 20

// templateFields: los campos que se piden al listar plantillas
const templateFields = "id,name,language,status,category,quality_score,rejected_reason,components,parameter_format"

// TemplateSpec: una plantilla para crear (POST /{waba_id}/message_templates). ParameterFormat es positional ({{1}}) o named
// ({{nombre}}); TTL son los segundos que Meta intenta entregarla (message_send_ttl_seconds; 0 no lo manda)
type TemplateSpec struct {
	Name            string              `json:"name"`
	Language        string              `json:"language"`
	Category        string              `json:"category"`
	ParameterFormat string              `json:"parameter_format,omitempty"`
	TTL             int                 `json:"message_send_ttl_seconds,omitempty"`
	Components      []TemplateComponent `json:"components"`
}

// TemplateUpdate: lo que cambia una edición (POST /{template_id})
type TemplateUpdate struct {
	Category   string              `json:"category,omitempty"`
	TTL        int                 `json:"message_send_ttl_seconds,omitempty"`
	Components []TemplateComponent `json:"components,omitempty"`
}

// TemplateComponent: un componente de la definición: HEADER (Format TEXT, IMAGE, VIDEO, DOCUMENT o LOCATION), BODY, FOOTER o
// BUTTONS. En AUTHENTICATION el BODY no lleva texto (AddSecurityRecommendation), el FOOTER lleva CodeExpirationMinutes y el
// botón es OTP
type TemplateComponent struct {
	Type                      string           `json:"type"`
	Format                    string           `json:"format,omitempty"`
	Text                      string           `json:"text,omitempty"`
	Example                   *TemplateExample `json:"example,omitempty"`
	Buttons                   []TemplateButton `json:"buttons,omitempty"`
	AddSecurityRecommendation *bool            `json:"add_security_recommendation,omitempty"`
	CodeExpirationMinutes     int              `json:"code_expiration_minutes,omitempty"`
}

// TemplateExample: los ejemplos que pide la revisión: HeaderText, BodyText (una lista con un ejemplo por {{n}}), HeaderHandle
// (media subida con la API de subida reanudable) y los de parámetros con nombre
type TemplateExample struct {
	HeaderText      []string       `json:"header_text,omitempty"`
	HeaderHandle    []string       `json:"header_handle,omitempty"`
	BodyText        [][]string     `json:"body_text,omitempty"`
	HeaderTextNamed []NamedExample `json:"header_text_named_params,omitempty"`
	BodyTextNamed   []NamedExample `json:"body_text_named_params,omitempty"`
}

// NamedExample: el ejemplo de un parámetro con nombre
type NamedExample struct {
	ParamName string `json:"param_name"`
	Example   string `json:"example"`
}

// TemplateButton: un botón de la definición: QUICK_REPLY, URL (Url con {{1}} y Example), PHONE_NUMBER, COPY_CODE (Example)
// u OTP (OtpType COPY_CODE, ONE_TAP o ZERO_TAP; SupportedApps para ONE_TAP/ZERO_TAP)
type TemplateButton struct {
	Type                 string         `json:"type"`
	Text                 string         `json:"text,omitempty"`
	Url                  string         `json:"url,omitempty"`
	PhoneNumber          string         `json:"phone_number,omitempty"`
	Example              []string       `json:"example,omitempty"`
	OtpType              string         `json:"otp_type,omitempty"`
	AutofillText         string         `json:"autofill_text,omitempty"`
	ZeroTapTermsAccepted *bool          `json:"zero_tap_terms_accepted,omitempty"`
	SupportedApps        []SupportedApp `json:"supported_apps,omitempty"`
}

// SupportedApp: la app Android que recibe el código de un botón ONE_TAP o ZERO_TAP
type SupportedApp struct {
	PackageName   string `json:"package_name"`
	SignatureHash string `json:"signature_hash"`
}

// TemplateInfo: una plantilla de la cuenta. Quality es quality_score.score (GREEN, YELLOW, RED o UNKNOWN)
type TemplateInfo struct {
	Id              string              `json:"id"`
	Name            string              `json:"name"`
	Language        string              `json:"language"`
	Status          string              `json:"status"`
	Category        string              `json:"category"`
	Quality         string              `json:"quality"`
	RejectedReason  string              `json:"rejected_reason,omitempty"`
	ParameterFormat string              `json:"parameter_format,omitempty"`
	Components      []TemplateComponent `json:"components,omitempty"`
}

// TemplateQuery: el filtro de Templates (vacío no filtra; Name busca por nombre parecido). After es el cursor de la página
// siguiente (TemplatePage.After)
type TemplateQuery struct {
	Name     string
	Language string
	Status   string
	Category string
	Limit    int
	After    string
}

// TemplatePage: una página de plantillas; After vacío si no hay más
type TemplatePage struct {
	Data  []TemplateInfo `json:"data"`
	After string         `json:"after,omitempty"`
}

// TemplateResult: la respuesta al crear (id, estado y categoría que Meta asignó)
type TemplateResult struct {
	Id       string `json:"id"`
	Status   string `json:"status"`
	Category string `json:"category"`
}

/**
* Bool: un puntero a value (para AddSecurityRecommendation y ZeroTapTermsAccepted)
* @param value bool
* @return *bool
**/
func Bool(value bool) *bool {
	return &value
}

/**
* AuthComponents: los componentes de una plantilla AUTHENTICATION con un botón OTP (otpType COPY_CODE si viene vacío; text
* opcional; minutes 0 no pone el pie)
* @param security bool, minutes int, otpType, text string
* @return []TemplateComponent
**/
func AuthComponents(security bool, minutes int, otpType, text string) []TemplateComponent {
	if otpType == "" {
		otpType = "COPY_CODE"
	}
	result := []TemplateComponent{{Type: "BODY", AddSecurityRecommendation: Bool(security)}}
	if minutes > 0 {
		result = append(result, TemplateComponent{Type: "FOOTER", CodeExpirationMinutes: minutes})
	}
	button := TemplateButton{Type: "OTP", OtpType: otpType, Text: text}
	return append(result, TemplateComponent{Type: "BUTTONS", Buttons: []TemplateButton{button}})
}

/**
* Templates: GET /{waba_id}/message_templates con el filtro
* @param ctx context.Context, query TemplateQuery
* @return TemplatePage, error
**/
func (s *Client) Templates(ctx context.Context, query TemplateQuery) (TemplatePage, error) {
	path, err := s.wabaPath("/message_templates")
	if err != nil {
		return TemplatePage{}, err
	}

	values := url.Values{"fields": {templateFields}}
	set := func(key, value string) {
		if value != "" {
			values.Set(key, value)
		}
	}
	set("name", query.Name)
	set("language", query.Language)
	set("status", query.Status)
	set("category", query.Category)
	set("after", query.After)
	if query.Limit > 0 {
		values.Set("limit", strconv.Itoa(query.Limit))
	}

	output, err := s.Call(ctx, http.MethodGet, path, values, nil)
	if err != nil {
		return TemplatePage{}, err
	}

	result := TemplatePage{Data: []TemplateInfo{}}
	for _, item := range output.ArrayJson("data") {
		result.Data = append(result.Data, templateInfoOf(item))
	}
	if output.Json("paging").Str("next") != "" {
		result.After = output.Json("paging").Json("cursors").Str("after")
	}
	return result, nil
}

/**
* Template: la plantilla con ese nombre e idioma exactos (Meta filtra por nombre parecido); false si no está
* @param ctx context.Context, name, language string
* @return TemplateInfo, bool, error
**/
func (s *Client) Template(ctx context.Context, name, language string) (TemplateInfo, bool, error) {
	query := TemplateQuery{Name: name}
	for range maxTemplatePages {
		page, err := s.Templates(ctx, query)
		if err != nil {
			return TemplateInfo{}, false, err
		}
		for _, item := range page.Data {
			if item.Name == name && item.Language == language {
				return item, true, nil
			}
		}
		if page.After == "" {
			return TemplateInfo{}, false, nil
		}
		query.After = page.After
	}
	return TemplateInfo{}, false, nil
}

/**
* templateInfoOf
* @param item et.Json
* @return TemplateInfo
**/
func templateInfoOf(item et.Json) TemplateInfo {
	result := TemplateInfo{
		Id:              item.Str("id"),
		Name:            item.Str("name"),
		Language:        item.Str("language"),
		Status:          item.Str("status"),
		Category:        item.Str("category"),
		Quality:         item.Json("quality_score").Str("score"),
		RejectedReason:  item.Str("rejected_reason"),
		ParameterFormat: item.Str("parameter_format"),
	}
	if result.Quality == "" {
		result.Quality = "UNKNOWN"
	}
	if raw, err := json.Marshal(item["components"]); err == nil {
		json.Unmarshal(raw, &result.Components)
	}
	return result
}

/**
* CreateTemplate: POST /{waba_id}/message_templates
* @param ctx context.Context, spec TemplateSpec
* @return TemplateResult, error
**/
func (s *Client) CreateTemplate(ctx context.Context, spec TemplateSpec) (TemplateResult, error) {
	path, err := s.wabaPath("/message_templates")
	if err != nil {
		return TemplateResult{}, err
	}
	if spec.Name == "" {
		return TemplateResult{}, errors.New(MSG_TEMPLATE_REQUIRED)
	}

	output, err := s.Call(ctx, http.MethodPost, path, nil, spec)
	if err != nil {
		return TemplateResult{}, err
	}
	return TemplateResult{Id: output.Str("id"), Status: output.Str("status"), Category: output.Str("category")}, nil
}

/**
* UpdateTemplate: POST /{template_id} (componentes, categoría o TTL; Meta la vuelve a revisar)
* @param ctx context.Context, id string, update TemplateUpdate
* @return error
**/
func (s *Client) UpdateTemplate(ctx context.Context, id string, update TemplateUpdate) error {
	if id == "" {
		return errors.New(MSG_TEMPLATE_REQUIRED)
	}
	_, err := s.Call(ctx, http.MethodPost, "/"+url.PathEscape(id), nil, update)
	return err
}

/**
* DeleteTemplate: DELETE /{waba_id}/message_templates?name= (todas las de ese nombre, en cada idioma); con id (hsm_id) solo esa
* @param ctx context.Context, name, id string
* @return error
**/
func (s *Client) DeleteTemplate(ctx context.Context, name, id string) error {
	path, err := s.wabaPath("/message_templates")
	if err != nil {
		return err
	}
	if name == "" {
		return errors.New(MSG_TEMPLATE_REQUIRED)
	}

	values := url.Values{"name": {name}}
	if id != "" {
		values.Set("hsm_id", id)
	}
	_, err = s.Call(ctx, http.MethodDelete, path, values, nil)
	return err
}
