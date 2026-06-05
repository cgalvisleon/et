package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
)

const (
	PROFILE_ADMIN   = "admin"
	PROFILE_APP     = "app"
	PROFILE_DEVELOP = "develop"
	PROFILE_SUPORT  = "suport"
)

/**
* GetTokenKey
* @param app, device, username string
* @return string
**/
func GetKey(app, device, username string) string {
	return fmt.Sprintf("%s:%s:%s", app, device, username)
}

/**
* NewToken
* @param app, device, userId, username string, payload et.Json, duration time.Duration
* @return string, error
**/
func NewToken(app, device, userId, username, tenantId, profileId string, payload et.Json, duration time.Duration) (string, error) {
	if !cache.IsLoad() {
		return "", errors.New(msg.MSG_CACHE_NOT_LOAD)
	}

	result, err := claim.NewToken(app, device, userId, username, tenantId, profileId, payload, duration)
	if err != nil {
		return "", err
	}

	key := GetKey(app, device, username)
	cache.SetWithDuration(key, result, duration)

	return result, nil
}

/**
* NewAuthentication
* @param app, device, userId, username string, duration time.Duration
* @return string, error
**/
func NewAuthentication(app, device, userId, username string, duration time.Duration) (string, error) {
	if app == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "app")
	}
	if device == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "device")
	}
	if username == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "username")
	}

	return NewToken(app, device, userId, username, "", "", et.Json{}, duration)
}

/**
* NewAuthorization
* @param app, device, userId, username, tenantId, profileId string, duration time.Duration
* @return string, error
**/
func NewAuthorization(app, device, userId, username, tenantId, profileId string, duration time.Duration) (string, error) {
	if app == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "app")
	}
	if device == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "device")
	}
	if username == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "username")
	}
	if tenantId == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "tenantId")
	}
	if profileId == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "profileId")
	}

	return NewToken(app, device, userId, username, tenantId, profileId, et.Json{}, duration)
}

/**
* NewAppToken
* @param app, device string, duration time.Duration
* @return string, error
**/
func NewAppToken(app, device string, duration time.Duration) (string, error) {
	if app == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "app")
	}
	if device == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "device")
	}

	return NewToken(app, device, app, app, "", "", et.Json{}, duration)
}

/**
* NewEphemeralToken
* @param app, device, userId, username, tenantId, profileId string, payload et.Json
* @return string, error
**/
func NewEphemeralToken(app, device, userId, username, tenantId, profileId string, payload et.Json, duration time.Duration) (string, error) {
	if app == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "app")
	}
	if device == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "device")
	}
	if username == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "username")
	}

	maxDuration := 15 * time.Minute
	if duration > maxDuration {
		duration = maxDuration
	}

	return NewToken(app, device, userId, username, tenantId, profileId, payload, duration)
}

/**
* GetToken
* @param key string
* @return string, error
**/
func GetToken(key string) (string, error) {
	return cache.Get(key, "")
}

/**
* DeleteToken
* @param app, device, username string
* @return error
**/
func DeleteToken(app, device, username string) error {
	key := GetKey(app, device, username)
	_, err := cache.Delete(key)
	if err != nil {
		return err
	}

	return nil
}

/**
* DeleteTokeByToken
* @param token string
* @return error
**/
func DeleteTokeByToken(token string) error {
	parce, err := claim.ParceToken(token)
	if err != nil {
		return err
	}

	app := parce.App
	device := parce.Device
	username := parce.Username
	return DeleteToken(app, device, username)
}

/**
* Validate
* @param ctx context.Context, token string
* @return *Claim, error
**/
func Validate(token string) (*claim.Claim, error) {
	clm, err := claim.ParceToken(token)
	if err != nil {
		return nil, err
	}

	app := clm.App
	device := clm.Device
	username := clm.Username
	key := GetKey(app, device, username)
	val, err := cache.Get(key, "")
	if err != nil {
		return nil, err
	}

	if val != token {
		cache.Delete(key)
		return nil, err
	}

	return clm, nil
}

/**
* SetToken
* @param app, device, username, token string, duration time.Duration
* @return error
**/
func SetToken(app, device, username, token string, duration time.Duration) error {
	key := GetKey(app, device, username)
	if duration < 0 {
		cache.Delete(key)
		return errors.New(msg.MSG_TOKEN_EXPIRED)
	}

	cache.Set(key, token, duration)

	return nil
}

/**
* RenewToken
* @param token string, duration time.Duration
* @return string, error
**/
func RenewToken(token string, duration time.Duration) (string, error) {
	clm, err := Validate(token)
	if err != nil {
		return "", err
	}

	app := clm.App
	device := clm.Device
	username := clm.Username
	key := GetKey(app, device, username)
	result, err := NewToken(app, device, clm.UserId, username, clm.TenantId, clm.ProfileId, clm.Payload, duration)
	if err != nil {
		return "", err
	}
	cache.Set(key, result, duration)
	return result, nil
}
