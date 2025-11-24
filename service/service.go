package service

import "github.com/cgalvisleon/et/et"

const (
	SERVICE_SMS       = "sms"
	SERVICE_WHATSAPP  = "whatsapp"
	SERVICE_EMAIL     = "email"
	SERVICE_OTP_SMS   = "otp_sms"
	SERVICE_OTP_EMAIL = "otp_email"
)

type SetFn func(id string, data et.Json) error
type GetFn func(id string) (string, error)
type DeleteFn func(id string) error
type GetTemplateFn func(id string) (string, error)

var (
	set         SetFn
	get         GetFn
	delete      DeleteFn
	getTemplate GetTemplateFn
)

func OnSet(f SetFn) {
	set = f
}

func OnGet(f GetFn) {
	get = f
}

func OnDelete(f DeleteFn) {
	delete = f
}

func OnGetTemplate(f GetTemplateFn) {
	getTemplate = f
}
