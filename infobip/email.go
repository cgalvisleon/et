package infobip

import (
	"errors"
	"fmt"
	"slices"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/strs"
)

/**
* SendEmail sends an email via the Infobip Email API (POST /email/4/messages).
* @param to []string, subject, htmlContent string, params et.Json, tpMessage string
* @return et.Item, error
**/
func (s *SenderInfobip) SendEmail(to []string, subject, htmlContent string, params et.Json, tpMessage string) (et.Item, error) {
	if len(to) == 0 {
		return et.Item{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "to")
	}

	if subject == "" {
		return et.Item{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "subject")
	}

	if htmlContent == "" {
		return et.Item{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "htmlContent")
	}

	if !slices.Contains([]string{"Transactional", "Promotional"}, tpMessage) {
		return et.Item{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "type")
	}

	content := htmlContent
	for k, v := range params {
		k := fmt.Sprintf("{{%s}}", k)
		v := fmt.Sprintf("%v", v)
		content = strs.Replace(content, k, v)
	}

	destinations := make([]et.Json, 0, len(to))
	for _, address := range to {
		destinations = append(destinations, et.Json{
			"to": []et.Json{{"destination": address}},
		})
	}

	url := fmt.Sprintf("%s/email/4/messages", s.Params.BaseUrl)
	header := et.Json{
		"accept":        "application/json",
		"content-type":  "application/json",
		"authorization": fmt.Sprintf("App %s", s.Params.ApiKey),
	}
	body := et.Json{
		"messages": []et.Json{
			{
				"sender":       s.Params.Sender,
				"destinations": destinations,
				"content": et.Json{
					"subject": subject,
					"html":    content,
				},
			},
		},
	}

	res, status := request.Fetch("POST", url, header, body)
	if !status.Ok {
		return et.Item{
			Ok: false,
			Result: et.Json{
				"provider": "Infobip",
				"type":     tpMessage,
				"message":  status.Message,
			},
		}, errors.New(status.Message)
	}

	output, _ := res.ToJson()

	return et.Item{
		Ok: true,
		Result: et.Json{
			"provider": "Infobip",
			"type":     tpMessage,
			"message":  "Email sent successfully",
			"result":   output,
		},
	}, nil
}
