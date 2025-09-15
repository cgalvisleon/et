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
	"github.com/cgalvisleon/et/utility"
)

/**
* SendEmail
* @param sender et.Json, to []et.Json, subject string, htmlContent string, params et.Json, tp string
* @return et.Items, error
**/
func SendEmail(sender et.Json, to []et.Json, subject string, htmlContent string, params et.Json, tp string) (et.Items, error) {
	if len(to) == 0 {
		return et.Items{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "to")
	}

	if !utility.ValidStr(subject, 0, []string{""}) {
		return et.Items{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "subject")
	}

	if !utility.ValidStr(htmlContent, 0, []string{""}) {
		return et.Items{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "htmlContent")
	}

	if !slices.Contains([]string{"Transactional", "Promotional"}, tp) {
		return et.Items{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "type")
	}

	err := config.Validate([]string{
		"BREVO_SEND_PATH",
		"BREVO_SEND_KEY",
	})
	if err != nil {
		return et.Items{}, err
	}

	path := config.String("BREVO_SEND_PATH", "")
	apiKey := config.String("BREVO_SEND_KEY", "")
	url := fmt.Sprintf("%s/smtp/email", path)
	header := et.Json{
		"accept":       "application/json",
		"content-type": "application/json",
		"api-key":      apiKey,
	}

	for k, v := range params {
		k := fmt.Sprintf("{{%s}}", k)
		s := fmt.Sprintf("%v", v)
		htmlContent = strs.Replace(htmlContent, k, s)
	}

	body := et.Json{
		"sender":      sender,
		"to":          to,
		"subject":     subject,
		"htmlContent": htmlContent,
	}

	result := et.Items{}
	res, err := request.Fetch("POST", url, header, body)
	if err != nil {
		return result, err
	}

	if !res.Status.Ok {
		return result, errors.New(res.Status.Message)
	}

	output, _ := res.Body.ToJson()
	result.Add(et.Json{
		"to":     to,
		"type":   tp,
		"sender": "Brevo",
		"status": output,
	})

	return result, nil
}

/**
* SendEmailTransactional
* @param sender et.Json, to []et.Json, subject string, htmlContent string, params et.Json
* @return et.Items, error
**/
func SendEmailTransactional(sender et.Json, to []et.Json, subject string, htmlContent string, params et.Json) (et.Items, error) {
	return SendEmail(sender, to, subject, htmlContent, params, "Transactional")
}

/**
* SendEmailPromotional
* @param sender et.Json, to []et.Json, subject string, htmlContent string, params et.Json
* @return et.Items, error
**/
func SendEmailPromotional(sender et.Json, to []et.Json, subject string, htmlContent string, params et.Json) (et.Items, error) {
	return SendEmail(sender, to, subject, htmlContent, params, "Promotional")
}
