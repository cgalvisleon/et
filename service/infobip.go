package service

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/request"
)

// emailTimeout: lo que se espera a Infobip al enviar un correo con adjuntos
const emailTimeout = 60 * time.Second

// Infobip: el proveedor por defecto (https://www.infobip.com/docs/api). Sus parámetros (config.fields del kind infobip):
// base_url (la url de la cuenta, https://xxxx.api.infobip.com), api_key, sms_from (el remitente de los SMS) y
// email_from / whatsapp_from (los remitentes por defecto de un correo o un WhatsApp que no traen el suyo)
type Infobip struct {
	BaseUrl      string
	ApiKey       string
	SmsFrom      string
	EmailFrom    string
	WhatsAppFrom string
}

/**
* NewInfobip: El sender de Infobip con sus parámetros
* @param params et.Json
* @return Sender, error
**/
func NewInfobip(params et.Json) (Sender, error) {
	result := &Infobip{
		BaseUrl:      params.TrimString("base_url"),
		ApiKey:       params.TrimString("api_key"),
		SmsFrom:      params.TrimString("sms_from"),
		EmailFrom:    params.TrimString("email_from"),
		WhatsAppFrom: params.TrimString("whatsapp_from"),
	}
	if result.BaseUrl == "" || result.ApiKey == "" {
		return nil, errors.New(MSG_INFOBIP_CONFIG)
	}

	return result, nil
}

/**
* fetch: Una llamada a la API de Infobip; devuelve su respuesta (vacía si no trae cuerpo, como un 204)
* @param method, path string, body et.Json
* @return et.Json, error
**/
func (s *Infobip) fetch(method, path string, body et.Json) (et.Json, error) {
	address := s.BaseUrl + path
	header := et.Json{
		// Con mayúsculas: request.Http lee "Content-Type" tal cual
		"Accept":        "application/json",
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("App %s", s.ApiKey),
	}
	res, status := request.Fetch(method, address, header, body, 0, nil)
	// Con respuesta, el error lleva lo que dijo Infobip (su requestError); sin ella (conexión), el mensaje del cliente
	if !status.Ok && res != nil && len(res.Data) > 0 {
		return et.Json{}, fmt.Errorf(MSG_INFOBIP_STATUS, status.Code, res.ToString())
	}
	if !status.Ok {
		return et.Json{}, errors.New(status.Message)
	}

	if res == nil || len(bytes.TrimSpace(res.Data)) == 0 {
		return et.Json{}, nil
	}

	return res.ToJson()
}

/**
* post: Un POST a la API de Infobip; devuelve su respuesta como {provider, response}
* @param path string, body et.Json
* @return et.Json, error
**/
func (s *Infobip) post(path string, body et.Json) (et.Json, error) {
	output, err := s.fetch(http.MethodPost, path, body)
	if err != nil {
		return et.Json{}, err
	}

	return et.Json{"provider": "infobip", "response": output}, nil
}

/**
* SendSMS: POST /sms/3/messages
* @param message SMS
* @return et.Json, error
**/
func (s *Infobip) SendSMS(message SMS) (et.Json, error) {
	// El remitente es opcional: sin sms_from Infobip usa el que tenga la cuenta por defecto
	destinations := []et.Json{{"to": message.To}}
	sms := et.Json{"destinations": destinations, "content": et.Json{"text": message.Text}}
	if s.SmsFrom != "" {
		sms["sender"] = s.SmsFrom
	}
	body := et.Json{"messages": []et.Json{sms}}
	return s.post("/sms/3/messages", body)
}

/**
* SendEmail: POST /email/4/messages; con adjuntos, POST /email/3/send (multipart), que los recibe como archivos
* @param message Email
* @return et.Json, error
**/
func (s *Infobip) SendEmail(message Email) (et.Json, error) {
	if len(message.Attachments) > 0 {
		return s.sendEmailFiles(message)
	}

	destinations := []et.Json{{"to": []et.Json{{"destination": message.To}}}}
	content := et.Json{"subject": message.Subject, "html": message.Html}
	from := cmp.Or(message.From, s.EmailFrom)
	body := et.Json{
		"messages": []et.Json{{"sender": from, "destinations": destinations, "content": content}},
	}
	return s.post("/email/4/messages", body)
}

/**
* sendEmailFiles: Un correo con adjuntos por /email/3/send: los campos y cada adjunto como attachment
* @param message Email
* @return et.Json, error
**/
func (s *Infobip) sendEmailFiles(message Email) (et.Json, error) {
	body := &bytes.Buffer{}
	form := multipart.NewWriter(body)
	from := cmp.Or(message.From, s.EmailFrom)
	fields := [][2]string{{"from", from}, {"to", message.To}, {"subject", message.Subject}, {"html", message.Html}}
	for _, field := range fields {
		if err := form.WriteField(field[0], field[1]); err != nil {
			return et.Json{}, err
		}
	}
	for _, attachment := range message.Attachments {
		header := textproto.MIMEHeader{}
		header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="attachment"; filename=%q`, attachment.Filename))
		header.Set("Content-Type", attachment.ContentType)
		part, err := form.CreatePart(header)
		if err != nil {
			return et.Json{}, err
		}
		if _, err := part.Write(attachment.Content); err != nil {
			return et.Json{}, err
		}
	}
	if err := form.Close(); err != nil {
		return et.Json{}, err
	}

	req, err := http.NewRequest(http.MethodPost, s.BaseUrl+"/email/3/send", body)
	if err != nil {
		return et.Json{}, err
	}
	req.Header.Set("Content-Type", form.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("App %s", s.ApiKey))

	client := &http.Client{Timeout: emailTimeout}
	res, err := client.Do(req)
	if err != nil {
		return et.Json{}, err
	}
	defer res.Body.Close()

	output := et.Json{}
	if err := json.NewDecoder(res.Body).Decode(&output); err != nil {
		return et.Json{}, err
	}
	if res.StatusCode >= http.StatusBadRequest {
		return et.Json{}, fmt.Errorf(MSG_INFOBIP_STATUS, res.StatusCode, output.ToString())
	}

	return et.Json{"provider": "infobip", "response": output}, nil
}

/**
* SendWhatsApp: Un mensaje por tipo: text, image, document, audio y video van a /whatsapp/1/message/<type>, interactive a
* /whatsapp/1/message/interactive/buttons y template a /whatsapp/1/message/template (este, dentro de messages)
* @param message WhatsApp
* @return et.Json, error
**/
func (s *Infobip) SendWhatsApp(message WhatsApp) (et.Json, error) {
	callbackData, err := json.Marshal(message.Metadata)
	if err != nil {
		return et.Json{}, err
	}

	one := et.Json{
		"from":         cmp.Or(message.From, s.WhatsAppFrom),
		"to":           message.To,
		"messageId":    reg.UUID(),
		"callbackData": string(callbackData),
	}
	switch message.Type {
	case "text":
		one["content"] = et.Json{"text": message.Text}
		return s.post("/whatsapp/1/message/text", one)
	case "image", "document", "audio", "video":
		one["content"] = message.Media
		return s.post("/whatsapp/1/message/"+message.Type, one)
	case "interactive":
		one["content"] = message.Media
		return s.post("/whatsapp/1/message/interactive/buttons", one)
	case "template":
		one["content"] = infobipTemplateOf(message.Template, message.Otp)
		body := et.Json{"messages": []et.Json{one}}
		return s.post("/whatsapp/1/message/template", body)
	}

	return et.Json{}, fmt.Errorf(MSG_WHATSAPP_TYPE, message.Type)
}

/**
* infobipTemplateOf: El contenido de una plantilla aprobada; con otp (una plantilla AUTHENTICATION) el código va como placeholder del cuerpo y como parámetro del
* botón, que Meta guarda como URL (templateData.buttons [{type: URL, parameter}])
* @param template et.Json, otp string
* @return et.Json
**/
func infobipTemplateOf(template et.Json, otp string) et.Json {
	if otp == "" {
		return template
	}

	result := template.Clone()
	data := template.Json("templateData").Clone()
	data["body"] = et.Json{"placeholders": []string{otp}}
	data["buttons"] = []et.Json{{"type": "URL", "parameter": otp}}
	result["templateData"] = data
	return result
}

/**
* templatesPath: La ruta de las plantillas de WhatsApp de un remitente (sin él, el whatsapp_from del sender)
* @param sender string
* @return string, error
**/
func (s *Infobip) templatesPath(sender string) (string, error) {
	from := cmp.Or(sender, s.WhatsAppFrom)
	if from == "" {
		return "", errors.New(MSG_WHATSAPP_FROM_REQUIRED)
	}

	return "/whatsapp/2/senders/" + url.PathEscape(from) + "/templates", nil
}

/**
* PublishWhatsAppTemplate: POST /whatsapp/2/senders/{sender}/templates; con el id del proveedor, PATCH …/templates/{id} (solo category, si viene, y structure)
* @param sender string, template WhatsAppTemplate
* @return TemplateState, error
**/
func (s *Infobip) PublishWhatsAppTemplate(sender string, template WhatsAppTemplate) (TemplateState, error) {
	path, err := s.templatesPath(sender)
	if err != nil {
		return TemplateState{}, err
	}

	if template.Id != "" {
		body := et.Json{"structure": template.Structure}
		if template.Category != "" {
			body["category"] = template.Category
		}
		output, err := s.fetch(http.MethodPatch, path+"/"+url.PathEscape(template.Id), body)
		if err != nil {
			return TemplateState{}, err
		}

		return stateOf(output, template), nil
	}

	body := et.Json{
		"name":      template.Name,
		"language":  template.Language,
		"category":  template.Category,
		"structure": template.Structure,
	}
	output, err := s.fetch(http.MethodPost, path, body)
	if err != nil {
		return TemplateState{}, err
	}

	return stateOf(output, template), nil
}

/**
* WhatsAppTemplateState: GET /whatsapp/2/senders/{sender}/templates y la de ese nombre e idioma; false si no está
* @param sender, name, language string
* @return TemplateState, bool, error
**/
func (s *Infobip) WhatsAppTemplateState(sender, name, language string) (TemplateState, bool, error) {
	path, err := s.templatesPath(sender)
	if err != nil {
		return TemplateState{}, false, err
	}

	output, err := s.fetch(http.MethodGet, path, nil)
	if err != nil {
		return TemplateState{}, false, err
	}

	for _, item := range output.ArrayJson("templates") {
		if item.Str("name") == name && item.Str("language") == language {
			template := WhatsAppTemplate{Name: name, Language: language}
			return stateOf(item, template), true, nil
		}
	}

	return TemplateState{}, false, nil
}

/**
* DeleteWhatsAppTemplate: DELETE /whatsapp/2/senders/{sender}/templates/{name} (todas las de ese nombre, en cada idioma)
* @param sender, name string
* @return error
**/
func (s *Infobip) DeleteWhatsAppTemplate(sender, name string) error {
	path, err := s.templatesPath(sender)
	if err != nil {
		return err
	}

	_, err = s.fetch(http.MethodDelete, path+"/"+url.PathEscape(name), nil)
	return err
}
