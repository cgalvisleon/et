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
* SendSMS
* @param contactNumbers []string, content string, params []et.Json, tp string
* @return et.Items, error
**/
func SendSMS(contactNumbers []string, content string, params []et.Json, tp string) (et.Items, error) {
	if len(contactNumbers) == 0 {
		return et.Items{}, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "contactNumbers")
	}

	if strs.IsEmpty(content) {
		return et.Items{}, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "content")
	}

	if !slices.Contains([]string{"Transactional", "Promotional"}, tp) {
		return et.Items{}, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "type")
	}

	if tp == "Promotional" {
		tp = "marketing"
	} else {
		tp = "transactional"
	}

	apiKey := envar.EnvarStr("", "BREVO_API_KEY")
	sender := envar.EnvarStr("MyCompany", "BREVO_SENDER")
	organisationPrefix := envar.EnvarStr("", "BREVO_ORGANISATION_PREFIX")
	path := envar.EnvarStr("https://api.brevo.com/v3", "BREVO_API")

	if strs.IsEmpty(apiKey) {
		return et.Items{}, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "BREVO_API_KEY")
	}

	if strs.IsEmpty(sender) {
		return et.Items{}, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "BREVO_SENDER")
	}

	if strs.IsEmpty(organisationPrefix) {
		return et.Items{}, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "BREVO_ORGANISATION_PREFIX")
	}

	if strs.IsEmpty(path) {
		return et.Items{}, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "BREVO_API")
	}

	url := strs.Format("%s/transactionalSMS/sms", path)
	header := et.Json{
		"accept":       "application/json",
		"content-type": "application/json",
		"api-key":      apiKey,
	}
	body := et.Json{
		"type":               tp,
		"unicodeEnabled":     false,
		"sender":             sender,
		"tag":                "t1",
		"organisationPrefix": organisationPrefix,
	}

	result := et.Items{}
	for _, phoneNumber := range contactNumbers {
		message := content
		for _, param := range params {
			for k, v := range param {
				k := strs.Format("{{%s}}", k)
				s := strs.Format("%v", v)
				message = strs.Replace(message, k, s)
			}
		}

		body["recipient"] = phoneNumber
		body["content"] = message
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
