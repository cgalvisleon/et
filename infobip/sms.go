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

type Params struct {
	BaseUrl string
	ApiKey  string
	Sender  string
}

/**
* validateParams
* @param params Params
* @return error
**/
func validateParams(params Params) error {
	if params.BaseUrl == "" {
		return errors.New(MSG_BASE_URL_REQUIRED)
	}
	if params.ApiKey == "" {
		return errors.New(MSG_API_KEY_REQUIRED)
	}
	if params.Sender == "" {
		return errors.New(MSG_SENDER_REQUIRED)
	}

	return nil
}

type SenderInfobip struct {
	Params Params
}

/**
* NewSenderInfobip
* @param params Params
* @return *SenderInfobip, error
**/
func NewSenderInfobip(params Params) (*SenderInfobip, error) {
	if err := validateParams(params); err != nil {
		return nil, err
	}

	return &SenderInfobip{
		Params: params,
	}, nil
}

/**
* SendSMS
* @param contactNumbers []string, content string, params et.Json, tpMessage string
* @return et.Item, error
**/
func (s *SenderInfobip) SendSMS(contactNumbers []string, content string, params et.Json, tpMessage string) (et.Item, error) {
	if len(contactNumbers) == 0 {
		return et.Item{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "contactNumbers")
	}

	if content == "" {
		return et.Item{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "content")
	}

	if !slices.Contains([]string{"Transactional", "Promotional"}, tpMessage) {
		return et.Item{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "type")
	}

	message := content
	for k, v := range params {
		k := fmt.Sprintf("{{%s}}", k)
		v := fmt.Sprintf("%v", v)
		message = strs.Replace(message, k, v)
	}

	destinations := make([]et.Json, 0, len(contactNumbers))
	for _, phoneNumber := range contactNumbers {
		destinations = append(destinations, et.Json{"to": phoneNumber})
	}

	url := fmt.Sprintf("%s/sms/3/messages", s.Params.BaseUrl)
	header := et.Json{
		"accept":        "application/json",
		"content-type":  "application/json",
		"authorization": fmt.Sprintf("App %s", s.Params.ApiKey),
	}
	body := et.Json{
		"messages": []et.Json{
			{
				"destinations": destinations,
				"from":         s.Params.Sender,
				"text":         message,
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
			"message":  "SMS sent successfully",
			"result":   output,
		},
	}, nil
}
