package service

import (
	"cmp"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/request"
)

// labsMobileUrl: el envío de SMS de LabsMobile (API http/POST JSON, https://www.labsmobile.com/en/sms-api/api-versions/http-rest-post-json)
const labsMobileUrl = "https://api.labsmobile.com/json/send"

// LabsMobile: un proveedor solo de SMS. Sus parámetros (config.fields del kind labsmobile): username (el correo de la
// cuenta), api_token (el token de la API, secret del tenant) y sms_from (el remitente, tpoa: hasta 11 letras o 16 dígitos;
// en Colombia el operador pone su código corto compartido)
type LabsMobile struct {
	Url      string
	Username string
	Token    string
	SmsFrom  string
}

/**
* NewLabsMobile: El sender de LabsMobile con sus parámetros (base_url solo para las pruebas)
* @param params et.Json
* @return Sender, error
**/
func NewLabsMobile(params et.Json) (Sender, error) {
	result := &LabsMobile{
		Url:      cmp.Or(params.TrimString("base_url"), labsMobileUrl),
		Username: params.TrimString("username"),
		Token:    params.TrimString("api_token"),
		SmsFrom:  params.TrimString("sms_from"),
	}
	if result.Username == "" || result.Token == "" {
		return nil, errors.New(MSG_LABSMOBILE_CONFIG)
	}

	return result, nil
}

/**
* SendSMS: POST /json/send con Basic; un texto fuera de GSM 7 bits va como ucs2 y uno largo concatenado
* @param message SMS
* @return et.Json, error
**/
func (s *LabsMobile) SendSMS(message SMS) (et.Json, error) {
	msisdn := strings.TrimPrefix(message.To, "+")
	body := et.Json{
		"message":   message.Text,
		"recipient": []et.Json{{"msisdn": msisdn}},
	}
	if s.SmsFrom != "" {
		body["tpoa"] = s.SmsFrom
	}

	segment := 160
	if !isGsm7(message.Text) {
		body["ucs2"] = 1
		segment = 70
	}
	if utf8.RuneCountInString(message.Text) > segment {
		body["long"] = 1
	}

	credentials := base64.StdEncoding.EncodeToString([]byte(s.Username + ":" + s.Token))
	header := et.Json{
		"Accept":        "application/json",
		"Content-Type":  "application/json",
		"Authorization": "Basic " + credentials,
	}
	res, status := request.Fetch(http.MethodPost, s.Url, header, body, 0, nil)
	if res == nil || len(res.Data) == 0 {
		return et.Json{}, fmt.Errorf(MSG_LABSMOBILE_STATUS, status.Code, status.Message)
	}

	output, err := res.ToJson()
	if err != nil {
		return et.Json{}, fmt.Errorf(MSG_LABSMOBILE_STATUS, status.Code, res.ToString())
	}

	if output.Str("code") != "0" {
		return et.Json{}, fmt.Errorf(MSG_LABSMOBILE_FAILED, output.Str("code"), output.Str("message"))
	}

	return et.Json{"provider": "labsmobile", "response": output}, nil
}

/**
* SendEmail: LabsMobile no envía correos
* @param message Email
* @return et.Json, error
**/
func (s *LabsMobile) SendEmail(message Email) (et.Json, error) {
	return et.Json{}, fmt.Errorf(MSG_CHANNEL_NOT_SUPPORTED, "labsmobile", "email")
}

/**
* SendWhatsApp: LabsMobile no envía WhatsApp
* @param message WhatsApp
* @return et.Json, error
**/
func (s *LabsMobile) SendWhatsApp(message WhatsApp) (et.Json, error) {
	return et.Json{}, fmt.Errorf(MSG_CHANNEL_NOT_SUPPORTED, "labsmobile", "whatsapp")
}

// gsm7: el alfabeto básico GSM 03.38 y su extensión (^{}\[~]|€, que cuentan doble pero no piden Unicode)
const gsm7 = "@£$¥èéùìòÇ\nØø\rÅåΔ_ΦΓΛΩΠΨΣΘΞÆæßÉ !\"#¤%&'()*+,-./0123456789:;<=>?¡ABCDEFGHIJKLMNOPQRSTUVWXYZÄÖÑÜ§¿abcdefghijklmnopqrstuvwxyzäöñüà^{}\\[~]|€"

/**
* isGsm7: Si el texto cabe en el alfabeto GSM 7 bits (si no, el SMS va como Unicode)
* @param text string
* @return bool
**/
func isGsm7(text string) bool {
	for _, char := range text {
		if !strings.ContainsRune(gsm7, char) {
			return false
		}
	}
	return true
}
