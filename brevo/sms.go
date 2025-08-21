package brevo

import (
	"errors"
	"fmt"
	"slices"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/strs"
)

/**
* sendSms
* @param sender, organisation string, contactNumbers []string, content string, params []et.Json, tp string
* @return et.Items, error
**/
func sendSms(sender, organisation string, contactNumbers []string, content string, params []et.Json, tp string) (et.Items, error) {
	if len(contactNumbers) == 0 {
		return et.Items{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "contactNumbers")
	}

	if strs.IsEmpty(content) {
		return et.Items{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "content")
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
	})
	if err != nil {
		return et.Items{}, err
	}

	apiKey := config.String("BREVO_SEND_KEY", "")
	path := config.String("BREVO_SEND_PATH", "")
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
		"organisationPrefix": organisation,
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
		res, err := request.Fetch("POST", url, header, body)
		if err != nil {
			return result, err
		}

		if !res.Status.Ok {
			return result, errors.New(res.Status.Message)
		}

		output, _ := res.Body.ToJson()
		result.Add(et.Json{
			"phoneNumber":  phoneNumber,
			"type":         tp,
			"agent":        "Brevo",
			"sender":       sender,
			"organisation": organisation,
			"status":       output,
		})
	}

	return result, nil
}

/**
* SendSmsTransactional
* @param sender, organisation string, contactNumbers []string, content string, params []et.Json
* @return et.Items, error
**/
func SendSmsTransactional(sender, organisation string, contactNumbers []string, content string, params []et.Json) (et.Items, error) {
	return sendSms(sender, organisation, contactNumbers, content, params, "Transactional")
}

/**
* SendSmsPromotional
* @param sender, organisation string, contactNumbers []string, content string, params []et.Json
* @return et.Items, error
**/
func SendSmsPromotional(sender, organisation string, contactNumbers []string, content string, params []et.Json) (et.Items, error) {
	return sendSms(sender, organisation, contactNumbers, content, params, "Promotional")
}
