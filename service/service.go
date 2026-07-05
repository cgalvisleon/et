package service

import "github.com/cgalvisleon/et/et"

const (
	SERVICE_SMS       = "sms"
	SERVICE_WHATSAPP  = "whatsapp"
	SERVICE_EMAIL     = "email"
	SERVICE_OTP_SMS   = "otp_sms"
	SERVICE_OTP_EMAIL = "otp_email"
)

type SenderSMS interface {
	SendSMS(contactNumbers []string, content string, params et.Json, tpMessage string) (et.Item, error)
}

type SenderWhatsapp interface {
	SendWhatsapp(contactNumbers []string, content string, params []et.Json, tpMessage string) (et.Item, error)
	SendWhatsappByTemplateId(contactNumbers []string, templateId string, params []et.Json, tpMessage string) (et.Item, error)
}

type SenderEmail interface {
	SendEmail(from et.Json, to []et.Json, subject string, htmlContent string, params et.Json, tpMessage string) (et.Item, error)
}

type SenderPushNotification interface {
	SendPushNotification(deviceToken string, title string, body string, tpMessage string) (et.Item, error)
}
