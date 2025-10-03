package service

import (
	"fmt"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
)

/**
* VerifyOTP
* @param tag string, code string, createdBy string
* @response bool, error
**/
func VerifyOTP(tag string, code, createdBy string) (bool, error) {
	key := fmt.Sprintf("otp:%s:%s", tag, createdBy)
	otp, err := cache.Get(key, "")
	if err != nil {
		return false, err
	}

	if otp == "" {
		return false, nil
	}

	cache.Delete(key)

	return otp == code, nil
}

/**
* SendOTPEmail
* @param tenantId string, from et.Json, email, name string, length int, expiresAt time.Duration, createdBy string
* @response et.Item, error
**/
func SendOTPEmail(tenantId string, from et.Json, email, name string, length int, expiresAt time.Duration, createdBy string) (et.Item, error) {
	code := utility.GetOTP(length)
	key := fmt.Sprintf("otp:%s:%s", email, createdBy)
	cache.Set(key, code, expiresAt)
	result, err := SendEmail(tenantId, from, []et.Json{{email: name}}, MSG_OTP, templateOTPEmail, et.Json{"name": name, "code": code}, TypeAutentication, createdBy)
	if err != nil {
		return et.Item{}, err
	}

	return result.First(), nil
}

/**
* SendOTPSms
* @param tenantId string, countyCode, phone string, createdBy string
* @response et.Item, error
**/
func SendOTPSms(tenantId string, countyCode, phone string, length int, expiresAt time.Duration, createdBy string) (et.Item, error) {
	code := utility.GetOTP(length)
	contactNumber := countyCode + phone
	key := fmt.Sprintf("otp:%s:%s", contactNumber, createdBy)
	cache.Set(key, code, expiresAt)
	result, err := SendSms(tenantId, []string{contactNumber}, templateOTPSMS, et.Json{"code": code}, TypeAutentication, createdBy)
	if err != nil {
		return et.Item{}, err
	}

	return result.First(), nil
}
