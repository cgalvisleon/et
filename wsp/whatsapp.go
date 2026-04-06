package wsp

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/request"
)

type Whatsapp struct {
	path            string `json:"-"`
	token           string `json:"-"`
	phone_number_id string `json:"-"`
}

/**
* NewWhatsapp
* @param token string, phone_number_id string
* @return *Whatsapp
**/
func NewWhatsapp(token, phone_number_id string) *Whatsapp {
	return &Whatsapp{
		path:            envar.GetStr("WHATSAPP_API_URL", "https://graph.facebook.com/v22.0"),
		token:           token,
		phone_number_id: phone_number_id,
	}
}

/**
* SendMessage
* @param to string, message *Message
* @return et.Json, error
**/
func (w *Whatsapp) SendMessage(to string, message *Message) (et.Json, error) {
	url := fmt.Sprintf("%s/%s/messages", w.path, w.phone_number_id)
	header := et.Json{
		"Authorization": "Bearer " + w.token,
		"Content-Type":  "application/json",
	}
	message.setTo(to)
	body := message.body()
	res, status := request.Post(url, header, body)
	if status.Code != 200 {
		return et.Json{}, errors.New(status.Message)
	}

	result, err := res.ToJson()
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}
