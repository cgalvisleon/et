package service

import (
	"errors"

	"github.com/cgalvisleon/et/et"
)

type TpMessage int

const (
	TypeNotification TpMessage = iota
	TypeComercial
	TypeAutentication
)

func (tp TpMessage) String() string {
	return [...]string{"Notification", "Comercial", "Autentication"}[tp]
}

func IntToTpMessage(i int) TpMessage {
	return TpMessage(i)
}

type Level int

const (
	LevelPrimary Level = iota
	LevelSecondary
)

type Send struct {
	Name                   string                           `json:"name"`
	Email                  string                           `json:"email"`
	SenderSMS              map[Level]SenderSMS              `json:"-"`
	SenderWhatsapp         map[Level]SenderWhatsapp         `json:"-"`
	SenderEmail            map[Level]SenderEmail            `json:"-"`
	SenderPushNotification map[Level]SenderPushNotification `json:"-"`
}

/**
* NewSend
* @param name string, email string
* @return *Send
**/
func NewSend(name string, email string) *Send {
	return &Send{
		Name:                   name,
		Email:                  email,
		SenderSMS:              make(map[Level]SenderSMS),
		SenderWhatsapp:         make(map[Level]SenderWhatsapp),
		SenderEmail:            make(map[Level]SenderEmail),
		SenderPushNotification: make(map[Level]SenderPushNotification),
	}
}

/**
* AddSenderSMS
* @param level Level, sender SenderSMS
* @return *Send
**/
func (s *Send) AddSenderSMS(level Level, sender SenderSMS) *Send {
	s.SenderSMS[level] = sender
	return s
}

/**
* AddSenderWhatsapp
* @param level Level, sender SenderWhatsapp
* @return *Send
**/
func (s *Send) AddSenderWhatsapp(level Level, sender SenderWhatsapp) *Send {
	s.SenderWhatsapp[level] = sender
	return s
}

/**
* AddSenderEmail
* @param level Level, sender SenderEmail
* @return *Send
**/
func (s *Send) AddSenderEmail(level Level, sender SenderEmail) *Send {
	s.SenderEmail[level] = sender
	return s
}

/**
* AddSenderPushNotification
* @param level Level, sender SenderPushNotification
* @return *Send
**/
func (s *Send) AddSenderPushNotification(level Level, sender SenderPushNotification) *Send {
	s.SenderPushNotification[level] = sender
	return s
}

/**
* SendSMS
* @param contactNumbers []string, content string, params et.Json, tpMessage string
* @response et.Item, error
**/
func (s *Send) SendSMS(contactNumbers []string, content string, params et.Json, tpMessage TpMessage) (et.Item, error) {
	sender, exists := s.SenderSMS[LevelPrimary]
	if !exists {
		return et.Item{}, errors.New(MSG_SEND_SMS_SENDER_REQUIRED)
	}

	result, err := sender.SendSMS(contactNumbers, content, params, tpMessage.String())
	if err != nil {
		secondarySender, exists := s.SenderSMS[LevelSecondary]
		if !exists {
			return et.Item{}, err
		}

		result, err = secondarySender.SendSMS(contactNumbers, content, params, tpMessage.String())
		if err != nil {
			return et.Item{}, err
		}
	}

	return result, nil
}

/**
* SendWhatsapp
* @param templateId string, contactNumbers []string, params []et.Json, tp TpMessage
* @response et.Items, error
**/
func (s *Send) SendWhatsapp(templateId string, contactNumbers []string, params []et.Json, tp TpMessage) (et.Item, error) {
	sender, exists := s.SenderWhatsapp[LevelPrimary]
	if !exists {
		return et.Item{}, errors.New(MSG_SEND_WHATSAPP_SENDER_REQUIRED)
	}

	result, err := sender.SendWhatsapp(contactNumbers, templateId, params, tp.String())
	if err != nil {
		secondarySender, exists := s.SenderWhatsapp[LevelSecondary]
		if !exists {
			return et.Item{}, err
		}

		result, err = secondarySender.SendWhatsapp(contactNumbers, templateId, params, tp.String())
		if err != nil {
			return et.Item{}, err
		}
	}

	return result, nil
}

/**
* SendEmail
* @param from et.Json, to []et.Json, subject string, htmlContent string, params []et.Json, tp TpMessage
* @response et.Items, error
**/
func (s *Send) SendEmail(from et.Json, to []et.Json, subject string, htmlContent string, params et.Json, tp TpMessage) (et.Item, error) {
	sender, exists := s.SenderEmail[LevelPrimary]
	if !exists {
		return et.Item{}, errors.New(MSG_SEND_EMAIL_SENDER_REQUIRED)
	}

	result, err := sender.SendEmail(from, to, subject, htmlContent, params, tp.String())
	if err != nil {
		secondarySender, exists := s.SenderEmail[LevelSecondary]
		if !exists {
			return et.Item{}, err
		}

		result, err = secondarySender.SendEmail(from, to, subject, htmlContent, params, tp.String())
		if err != nil {
			return et.Item{}, err
		}
	}

	return result, nil
}
