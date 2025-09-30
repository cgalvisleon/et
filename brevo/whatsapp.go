package brevo

import (
	"fmt"
	"slices"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/request"
)

/**
* SendWhatsapp
* @param contactNumbers []string, templateId string, params []et.Json, tp string
* @return et.Items, error
**/
func SendWhatsapp(contactNumbers []string, templateId string, params []et.Json, tp string) (et.Items, error) {
	if len(contactNumbers) == 0 {
		return et.Items{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "contactNumbers")
	}

	if len(templateId) == 0 {
		return et.Items{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "templateId")
	}

	if !slices.Contains([]string{"Transactional", "Promotional"}, tp) {
		return et.Items{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "type")
	}

	if tp == "Promotional" {
		tp = "marketing"
	} else {
		tp = "transactional"
	}

	err := config.Validate([]string{
		"BREVO_SEND_PATH",
		"BREVO_SEND_KEY",
		"BREVO_SENDER",
	})
	if err != nil {
		return et.Items{}, err
	}

	path := config.String("BREVO_SEND_PATH", "")
	apiKey := config.String("BREVO_SEND_KEY", "")
	sender := config.String("BREVO_SENDER", "")
	url := fmt.Sprintf("%s/whatsapp/sendMessage", path)
	header := et.Json{
		"accept":       "application/json",
		"content-type": "application/json",
		"api-key":      apiKey,
	}
	body := et.Json{
		"contactNumbers": []string{},
		"templateId":     templateId,
		"params":         params,
		"senderNumber":   sender,
	}

	result := et.Items{}
	for _, phoneNumber := range contactNumbers {
		body["contactNumbers"] = []string{phoneNumber}
		res, status := request.Fetch("POST", url, header, body)
		if !status.Ok {
			return result, fmt.Errorf(status.Message)
		}

		output, _ := res.ToJson()
		result.Add(et.Json{
			"phoneNumber": phoneNumber,
			"type":        tp,
			"sender":      "Brevo",
			"status":      output,
		})
	}

	return result, nil
}

/**
* SendWhatsappTransactional
* @param contactNumbers []string, templateId string, params []et.Json
* @return et.Items, error
**/
func SendWhatsappTransactional(contactNumbers []string, templateId string, params []et.Json) (et.Items, error) {
	return SendWhatsapp(contactNumbers, templateId, params, "Transactional")
}

/**
* SendWhatsappPromotional
* @param contactNumbers []string, templateId string, params []et.Json
* @return et.Items, error
**/
func SendWhatsappPromotional(contactNumbers []string, templateId string, params []et.Json) (et.Items, error) {
	return SendWhatsapp(contactNumbers, templateId, params, "Promotional")
}
