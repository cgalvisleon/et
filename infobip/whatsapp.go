package infobip

import (
	"errors"
	"fmt"
	"slices"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/request"
)

/**
* SendWhatsApp sends a pre-approved WhatsApp template message per contact
* number via the Infobip WhatsApp API (POST /whatsapp/1/message/template).
* placeholders[i] fills the positional {{n}} placeholders of the template
* body for contactNumbers[i]; a missing entry sends the template with no
* placeholders.
* @param contactNumbers []string, templateName, language string, placeholders [][]string, tpMessage string
* @return et.Items, error
**/
func (s *SenderInfobip) SendWhatsApp(contactNumbers []string, templateName, language string, placeholders [][]string, tpMessage string) (et.Items, error) {
	if len(contactNumbers) == 0 {
		return et.Items{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "contactNumbers")
	}

	if templateName == "" {
		return et.Items{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "templateName")
	}

	if !slices.Contains([]string{"Transactional", "Promotional"}, tpMessage) {
		return et.Items{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "type")
	}

	if language == "" {
		language = "en"
	}

	url := fmt.Sprintf("%s/whatsapp/1/message/template", s.Params.BaseUrl)
	header := et.Json{
		"accept":        "application/json",
		"content-type":  "application/json",
		"authorization": fmt.Sprintf("App %s", s.Params.ApiKey),
	}

	result := et.Items{}
	for i, phoneNumber := range contactNumbers {
		ph := []string{}
		if i < len(placeholders) {
			ph = placeholders[i]
		}

		body := et.Json{
			"from":      s.Params.Sender,
			"to":        phoneNumber,
			"messageId": reg.UUID(),
			"content": et.Json{
				"templateName": templateName,
				"templateData": et.Json{
					"body": et.Json{
						"placeholders": ph,
					},
				},
				"language": language,
			},
		}

		res, status := request.Fetch("POST", url, header, body)
		if !status.Ok {
			return result, errors.New(status.Message)
		}

		output, _ := res.ToJson()
		result.Add(et.Json{
			"provider":    "Infobip",
			"type":        tpMessage,
			"phoneNumber": phoneNumber,
			"message":     "WhatsApp message sent successfully",
			"result":      output,
		})
	}

	return result, nil
}
