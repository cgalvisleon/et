package service

import (
	"fmt"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/utility"
)

/**
* VerifyOTP
* @param channel string, otp, createdBy string
* @response bool, error
**/
func VerifyOTP(channel string, otp, createdBy string) (bool, error) {
	key := fmt.Sprintf("service:otp:%s", channel)
	otpCache, err := cache.Get(key, "")
	if err != nil {
		return false, err
	}

	if otpCache != otp {
		return false, nil
	}

	return true, nil
}

/**
* SendOTPSMS
* @param tenantId, serviceId, countryCode, phoneNumber string, length int, duration time.Duration, createdBy string
* @response et.Items, error
**/
func SendOTPSMS(tenantId, serviceId, sender, countryCode, phoneNumber string, length int, duration time.Duration, createdBy string) (et.Items, error) {
	serviceId = reg.TagULID("service", serviceId)
	otp := utility.GetOTP(length)
	channel := fmt.Sprintf("%s%s", countryCode, phoneNumber)
	key := fmt.Sprintf("service:otp:%s", channel)
	msg := "{{sender}}: Hola, tu código de verificación es {{otp}}. Recuerda que es válido por {{duration}} minutos"
	params := et.Json{
		"sender":   sender,
		"otp":      otp,
		"duration": duration.Minutes(),
	}
	result, err := SendSms(tenantId, serviceId, []string{countryCode + phoneNumber}, msg, params, TypeAutentication, createdBy)
	if err != nil {
		return et.Items{}, err
	}

	cache.Set(key, otp, duration)
	if set != nil {
		set(serviceId, et.Json{
			"tenantId":  tenantId,
			"serviceId": serviceId,
			"service":   SERVICE_OTP_SMS,
			"from":      sender,
			"to":        channel,
			"content":   msg,
			"params":    params,
			"type":      TypeAutentication.String(),
			"createdBy": createdBy,
			"result":    result,
		})
	}
	return result, nil
}

/**
* SendOTPEmail
* @param tenantId, serviceId string, from et.Json, name, email string, length int, duration time.Duration, createdBy string
* @response et.Items, error
**/
func SendOTPEmail(tenantId, serviceId string, from et.Json, name, email string, length int, duration time.Duration, createdBy string) (et.Items, error) {
	serviceId = reg.TagULID("service", serviceId)
	otp := utility.GetOTP(length)
	channel := email
	key := fmt.Sprintf("service:otp:%s", channel)
	msg := "<h1>Hola</h1>: <p>Tu código de verificación es {{otp}}. Recuerda que es válido por {{duration}} minutos</p>"
	params := et.Json{
		"otp":      otp,
		"duration": duration.Minutes(),
	}
	to := []et.Json{{"name": name, "email": email}}
	result, err := SendEmail(tenantId, serviceId, from, to, "OTP", msg, params, TypeAutentication, createdBy)
	if err != nil {
		return et.Items{}, err
	}

	cache.Set(key, otp, duration)
	if set != nil {
		set(serviceId, et.Json{
			"tenantId":  tenantId,
			"serviceId": serviceId,
			"service":   SERVICE_OTP_EMAIL,
			"from":      from,
			"to":        to,
			"content":   msg,
			"params":    params,
			"type":      TypeAutentication.String(),
			"createdBy": createdBy,
			"result":    result,
		})
	}
	return result, nil
}

/**
* SendOTPByTemplateId
* @param tenantId, serviceId string, from et.Json, name, email string, length int, duration time.Duration, templateId string, createdBy string
* @response et.Items, error
**/
func SendOTPByTemplateId(tenantId, serviceId string, from et.Json, name, email string, length int, duration time.Duration, templateId string, createdBy string) (et.Items, error) {
	serviceId = reg.TagULID("service", serviceId)
	otp := utility.GetOTP(length)
	key := fmt.Sprintf("service:otp:%s", email)
	params := et.Json{
		"otp":      otp,
		"duration": duration.Minutes(),
	}
	to := []et.Json{{"name": name, "email": email}}
	result, err := SendEmailByTemplateId(tenantId, serviceId, from, to, "OTP", templateId, params, TypeAutentication, createdBy)
	if err != nil {
		return et.Items{}, err
	}

	cache.Set(key, otp, duration)
	if set != nil {
		set(serviceId, et.Json{
			"tenantId":   tenantId,
			"serviceId":  serviceId,
			"service":    SERVICE_OTP_EMAIL,
			"from":       from,
			"to":         to,
			"templateId": templateId,
			"params":     params,
			"type":       TypeAutentication.String(),
			"createdBy":  createdBy,
			"result":     result,
		})
	}
	return result, nil
}
