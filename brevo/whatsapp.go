package brevo

import (
	"slices"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/mistake"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/strs"
)

/**
* SendWhatsapp
* @param contactNumbers []string, templateId string, params []et.Json, tp string
* @return et.Items, error
**/
func SendWhatsapp(contactNumbers []string, templateId string, params []et.Json, tp string) (et.Items, error) {
	if len(contactNumbers) == 0 {
		return et.Items{}, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "contactNumbers")
	}

	if len(templateId) == 0 {
		return et.Items{}, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "templateId")
	}

	if !slices.Contains([]string{"Transactional", "Promotional"}, tp) {
		return et.Items{}, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "type")
	}

	if tp == "Promotional" {
		tp = "marketing"
	} else {
		tp = "transactional"
	}

	path := envar.EnvarStr("https://api.brevo.com/v3", "BREVO_SEND_PATH")
	apiKey := envar.EnvarStr("", "BREVO_SEND_KEY")
	sender := envar.EnvarStr("", "BREVO_SEND_SENDER")

	if strs.IsEmpty(path) {
		return et.Items{}, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "BREVO_SEND_PATH")
	}

	if strs.IsEmpty(apiKey) {
		return et.Items{}, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "BREVO_SEND_KEY")
	}

	if strs.IsEmpty(sender) {
		return et.Items{}, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "BREVO_SEND_SENDER")
	}

	url := strs.Format("%s/whatsapp/sendMessage", path)
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
		res, status := request.Post(url, header, body)
		if status.Code != 200 {
			return result, mistake.New(status.Message)
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
