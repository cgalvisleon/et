package service

import (
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
)

/**
* VerifyOTP
* @param channel string, otp string
* @response bool, error
**/
func VerifyOTP(key string, otp string) (bool, error) {
	otpCache, err := cache.Get(key, "")
	if err != nil {
		return false, err
	}

	if otpCache != otp {
		return false, nil
	}

	cache.Delete(key)
	return true, nil
}

/**
* SendOTPSMS
* @param key, sender, countryCode, phoneNumber string, length int, duration time.Duration, content string
* @response et.Item, error
**/
func (s *Send) SendOTPSMS(key, sender, countryCode, phoneNumber string, length int, duration time.Duration, content string) (et.Item, error) {
	if duration == 0 {
		duration = 5 * time.Minute
	}
	otp := utility.GetOTP(length)

	if content == "" {
		content = "{{sender}}: Hola, tu código de verificación es {{otp}}. Recuerda que es válido por {{duration}} minutos"
	}
	params := et.Json{
		"sender":   sender,
		"otp":      otp,
		"duration": duration.Minutes(),
	}

	result, err := s.SendSMS([]string{countryCode + phoneNumber}, content, params, TypeAutentication)
	if err != nil {
		return et.Item{}, err
	}

	cache.Set(key, otp, duration)

	return result, nil
}

/**
* SendOTPEmail
* @param key, sender string, from et.Json, name, email string, length int, duration time.Duration, htmlContent string
* @response et.Items, error
**/
func SendOTPEmail(key, sender string, name, email string, length int, duration time.Duration, htmlContent string) (et.Items, error) {
	if duration == 0 {
		duration = 5 * time.Minute
	}
	otp := utility.GetOTP(length)

	if htmlContent == "" {
		htmlContent = "<h1>{{sender}}</h1>: <p>Tu código de verificación es {{otp}}. Recuerda que es válido por {{duration}} minutos</p>"
	}
	params := et.Json{
		"sender":   sender,
		"otp":      otp,
		"duration": duration.Minutes(),
	}

	from := et.Json{
		"name":  sender,
		"email": email,
	}
	to := []et.Json{{
		"name":  name,
		"email": email,
	}}
	result, err := SendEmail(from, to, "OTP", htmlContent, params, TypeAutentication)
	if err != nil {
		return et.Items{}, err
	}

	cache.Set(key, otp, duration)

	return result, nil
}
