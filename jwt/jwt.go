package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
)

/**
* GetTokenKey
* @param app, device, userId string
* @return string
**/
func GetKey(app, device, userId string) string {
	return fmt.Sprintf("%s:%s:%s", app, device, userId)
}

/**
* NewToken
* @param app, device, userId, name string, payload et.Json, duration time.Duration
* @return string, error
**/
func NewToken(app, device, userId, name string, payload et.Json, duration time.Duration) (string, error) {
	if err := cache.Load(); err != nil {
		return "", errors.New(msg.MSG_CACHE_NOT_LOAD)
	}

	result, err := claim.NewToken(app, device, userId, "", "", name, payload, duration)
	if err != nil {
		return "", err
	}

	key := GetKey(app, device, userId)
	cache.Set(key, result, duration)

	return result, nil
}

/**
* NewAuthentication
* @param app, device, userId, name string, duration time.Duration
* @return string, error
**/
func NewAuthentication(app, device, userId, name string, duration time.Duration) (string, error) {
	if app == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "app")
	}
	if device == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "device")
	}
	if userId == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "userId")
	}
	if name == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "name")
	}

	return NewToken(app, device, userId, name, et.Json{}, duration)
}

/**
* NewAuthorization
* @param app, device, userId, name, tenantId, roleId string, duration time.Duration
* @return string, error
**/
func NewAuthorization(app, device, userId, name, tenantId, roleId string, duration time.Duration) (string, error) {
	if app == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "app")
	}
	if device == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "device")
	}
	if userId == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "userId")
	}
	if name == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "name")
	}
	if tenantId == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "tenantId")
	}
	if roleId == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "roleId")
	}
	if err := cache.Load(); err != nil {
		return "", errors.New(msg.MSG_CACHE_NOT_LOAD)
	}

	result, err := claim.NewToken(app, device, userId, tenantId, roleId, name, et.Json{}, duration)
	if err != nil {
		return "", err
	}

	key := GetKey(app, device, userId)
	cache.Set(key, result, duration)

	return result, nil
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

	return NewToken(app, device, app, app, et.Json{}, duration)
}

/**
* NewEphemeralToken
* @param app, device, userId, name string, duration time.Duration
* @return string, error
**/
func NewEphemeralToken(app, device, userId, name string, duration time.Duration) (string, error) {
	if app == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "app")
	}
	if device == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "device")
	}
	if userId == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "userId")
	}
	if name == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "name")
	}

	maxDuration := time.Duration(envar.GetInt("JWT_EPHEMERAL_DURATION", 15)) * time.Minute
	if duration > maxDuration {
		duration = maxDuration
	}

	return NewToken(app, device, userId, name, et.Json{}, duration)
}

/**
* GetToken
* @param key string
* @return string, bool, error
**/
func GetToken(key string) (string, bool, error) {
	result, err := cache.Get(key, "")
	if err == cache.ErrNotFound {
		return "", false, nil
	} else if err != nil {
		return "", false, err
	}

	return result, true, nil
}

/**
* DeleteToken
* @param app, device, sessionID string
* @return error
**/
func DeleteToken(app, device, sessionID string) error {
	key := GetKey(app, device, sessionID)
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
	userId := parce.UserID
	return DeleteToken(app, device, userId)
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
	userId := clm.UserID
	key := GetKey(app, device, userId)
	val, exists, err := GetToken(key)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, nil
	}

	if val != token {
		cache.Delete(key)
		return nil, errors.New(msg.MSG_TOKEN_INVALID)
	}

	return clm, nil
}

/**
* SetToken
* @param app, device, sessionID, token string, duration time.Duration
* @return error
**/
func SetToken(app, device, sessionID, token string, duration time.Duration) error {
	key := GetKey(app, device, sessionID)
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
	userId := clm.UserID
	result, err := NewToken(app, device, userId, clm.Name, clm.Payload, duration)
	if err != nil {
		return "", err
	}

	key := GetKey(app, device, userId)
	cache.Set(key, result, duration)
	return result, nil
}
