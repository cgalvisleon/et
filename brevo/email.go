package brevo

import (
	"slices"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/mistake"
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
		return et.Items{}, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "to")
	}

	if !utility.ValidStr(subject, 0, []string{""}) {
		return et.Items{}, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "subject")
	}

	if !utility.ValidStr(htmlContent, 0, []string{""}) {
		return et.Items{}, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "htmlContent")
	}

	if !slices.Contains([]string{"Transactional", "Promotional"}, tp) {
		return et.Items{}, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "type")
	}

	path := envar.EnvarStr("https://api.brevo.com/v3", "BREVO_SEND_PATH")
	apiKey := envar.EnvarStr("", "BREVO_SEND_KEY")

	if strs.IsEmpty(path) {
		return et.Items{}, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "BREVO_SEND_PATH")
	}

	if strs.IsEmpty(apiKey) {
		return et.Items{}, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "BREVO_SEND_KEY")
	}

	url := strs.Format("%s/smtp/email", path)
	header := et.Json{
		"accept":       "application/json",
		"content-type": "application/json",
		"api-key":      apiKey,
	}

	for k, v := range params {
		k := strs.Format("{{%s}}", k)
		s := strs.Format("%v", v)
		htmlContent = strs.Replace(htmlContent, k, s)
	}

	body := et.Json{
		"sender":      sender,
		"to":          to,
		"subject":     subject,
		"htmlContent": htmlContent,
	}

	result := et.Items{}
	res, status := request.Post(url, header, body)
	if status.Code != 200 {
		return result, mistake.New(status.Message)
	}

	output, _ := res.ToJson()
	result.Add(et.Json{
		"to":     to,
		"type":   tp,
		"sender": "Brevo",
		"status": output,
	})

	return result, nil
}
