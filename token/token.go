package token

import (
	"context"
	"net/http"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
	"github.com/golang-jwt/jwt/v4"
)

type ContextKey string

func (c ContextKey) String(ctx context.Context, def string) string {
	val := ctx.Value(c)
	result, ok := val.(string)
	if !ok {
		return def
	}

	return result
}

const (
	ServiceIdKey ContextKey = "serviceId"
	ClientIdKey  ContextKey = "clientId"
	NameKey      ContextKey = "name"
	AppKey       ContextKey = "app"
	DeviceKey    ContextKey = "device"
	DuractionKey ContextKey = "duration"
	TokenKey     ContextKey = "token"
)

type Claim struct {
	Salt     string        `json:"salt"`
	ClientId string        `json:"clientId"`
	Name     string        `json:"name"`
	App      string        `json:"app"`
	Device   string        `json:"device"`
	Duration time.Duration `json:"expired"`
	jwt.StandardClaims
}

/**
* ToJson method to use in Claim
* @return et.Json
**/
func (c *Claim) ToJson() et.Json {
	return et.Json{
		"clientId": c.ClientId,
		"name":     c.Name,
		"app":      c.App,
		"device":   c.Device,
		"expired":  c.Duration,
	}
}

/**
* Key return a key
* @param app string
* @param device string
* @param clientId string
* @return string
**/
func Key(app, device, clientId string) string {
	return cache.GenKey("token", app, device, clientId)
}

/**
* parce method to use in token
* @param token string
* @return *jwt.Token
* @return error
**/
func parce(token string) (*jwt.Token, error) {
	secret := envar.GetStr("1977", "SECRET")
	result, err := jwt.Parse(token, func(*jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* Generate method to use in token
* @param clientId string
* @param name string
* @param app string
* @param kind string
* @param device string
* @param expired time.Duration
* @return string
* @return error
**/
func Generate(clientId, name, app, device string, expired time.Duration) (string, error) {
	c := Claim{
		Salt:     utility.GetOTP(6),
		ClientId: clientId,
		Name:     name,
		App:      app,
		Device:   device,
		Duration: expired,
	}
	if c.Duration != 0 {
		c.ExpiresAt = time.Now().Add(c.Duration).Unix()
	}
	jwT := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	secret := envar.GetStr("1977", "SECRET")
	token, err := jwT.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	key := Key(app, device, clientId)
	cache.Set(key, token, expired)

	return token, nil
}

/**
* Parce
* @param token string
* @return *Claim
* @return error
**/
func Parce(token string) (*Claim, error) {
	jwT, err := parce(token)
	if err != nil {
		return nil, err
	}

	if !jwT.Valid {
		return nil, logs.Alertm(ERR_AUTORIZATION)
	}

	claims, ok := jwT.Claims.(jwt.MapClaims)
	if !ok {
		return nil, logs.Alertm(ERR_INVALID_CLAIM)
	}

	clientId, ok := claims[string(ClientIdKey)].(string)
	if !ok {
		return nil, logs.Alertm(ERR_INVALID_CLAIM)
	}

	name, ok := claims[string(NameKey)].(string)
	if !ok {
		return nil, logs.Alertm(ERR_INVALID_CLAIM)
	}

	app, ok := claims[string(AppKey)].(string)
	if !ok {
		return nil, logs.Alertm(ERR_INVALID_CLAIM)
	}

	device, ok := claims[string(DeviceKey)].(string)
	if !ok {
		return nil, logs.Alertm(ERR_INVALID_CLAIM)
	}

	second, ok := claims[string(DuractionKey)].(float64)
	if !ok {
		return nil, logs.Alertf(MSG_TOKEN_INVALID_ATRIB, "duration")
	}

	duration := time.Duration(second)

	result := &Claim{
		ClientId: clientId,
		Name:     name,
		App:      app,
		Device:   device,
		Duration: duration,
	}
	if result.Duration != 0 {
		result.ExpiresAt = int64(claims["exp"].(float64))
	}

	return result, nil
}

/**
* Valid
* @param token string
* @return *Claim
* @return error
**/
func Valid(token string) (*Claim, error) {
	result, err := Parce(token)
	if err != nil {
		return nil, err
	}

	key := Key(result.App, result.Device, result.ClientId)
	val, err := cache.Get(key, "")
	if err != nil {
		return nil, err
	}

	if val != token {
		cache.Delete(key)
		return nil, err
	}

	return result, nil
}

/**
* Set
* @param app string
* @param device string
* @param id string
* @param token string
* @return error
**/
func Set(app, device, id, token string, duration time.Duration) error {
	key := Key(app, device, id)
	err := cache.Set(key, token, duration)
	if err != nil {
		return err
	}

	return nil
}

/**
* GetUser
* @param r *http.Request
* @return et.Json
**/
func ClientId(r *http.Request) string {
	ctx := r.Context()
	return ClientIdKey.String(ctx, "-1")
}

/**
* GetUser
* @param r *http.Request
* @return et.Json
**/
func GetUser(r *http.Request) et.Json {
	now := utility.Now()
	ctx := r.Context()
	username := ClientIdKey.String(ctx, "Anonimo")
	fullName := NameKey.String(ctx, "Anonimo")

	return et.Json{
		"date_at":   now,
		"username":  username,
		"full_name": fullName,
	}
}
