package token

import (
	"context"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
	"github.com/golang-jwt/jwt/v4"
)

type contextKey string

func (c contextKey) String(ctx context.Context, def string) string {
	val := ctx.Value(c)
	result, ok := val.(string)
	if !ok {
		return def
	}

	return result
}

const (
	ServiceIdKey contextKey = "serviceId"
	ClientIdKey  contextKey = "clientId"
	NameKey      contextKey = "name"
	IatKey       contextKey = "iat"
	ExpKey       contextKey = "expired"
	AppKey       contextKey = "app"
	KindKey      contextKey = "kind"
	DeviceKey    contextKey = "device"
	TokenKey     contextKey = "token"
)

type Claim struct {
	Salt     string        `json:"salt"`
	ClientId string        `json:"clientId"`
	Name     string        `json:"name"`
	Iat      time.Time     `json:"iat"`
	Expired  time.Duration `json:"expired"`
	App      string        `json:"app"`
	Kind     string        `json:"kind"`
	Device   string        `json:"device"`
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
		"iat":      c.Iat,
		"exp":      c.Expired,
		"app":      c.App,
		"kind":     c.Kind,
		"device":   c.Device,
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
func Generate(clientId, name, app, kind, device string, expired time.Duration) (string, error) {
	c := Claim{
		Salt:     utility.GetOTP(6),
		ClientId: clientId,
		Name:     name,
		Iat:      timezone.NowTime(),
		Expired:  expired,
		App:      app,
		Kind:     kind,
		Device:   device,
	}
	if expired > 0 {
		c.ExpiresAt = timezone.NowTime().Add(expired).Unix()
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
* Validate method to use in token
* @param token string
* @return *Claim
* @return error
**/
func Validate(token string) (*Claim, error) {
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

	clientId, ok := claims["clientId"].(string)
	if !ok {
		return nil, logs.Alertm(ERR_INVALID_CLAIM)
	}

	name, ok := claims["name"].(string)
	if !ok {
		return nil, logs.Alertm(ERR_INVALID_CLAIM)
	}

	iat, ok := claims["iat"].(float64)
	if !ok {
		return nil, logs.Alertm(ERR_INVALID_CLAIM)
	}

	exp, ok := claims["expired"].(float64)
	if !ok {
		return nil, logs.Alertm(ERR_INVALID_CLAIM)
	}

	app, ok := claims["app"].(string)
	if !ok {
		return nil, logs.Alertm(ERR_INVALID_CLAIM)
	}

	kind, ok := claims["kind"].(string)
	if !ok {
		return nil, logs.Alertm(ERR_INVALID_CLAIM)
	}

	device, ok := claims["device"].(string)
	if !ok {
		return nil, logs.Alertm(ERR_INVALID_CLAIM)
	}

	result := &Claim{
		ClientId: clientId,
		Name:     name,
		Iat:      time.Unix(int64(iat), 0),
		Expired:  time.Duration(exp),
		App:      app,
		Kind:     kind,
		Device:   device,
	}
	if result.Expired != 0 {
		result.ExpiresAt = int64(claims["exp"].(float64))
	}

	return result, nil
}
