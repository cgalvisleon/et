package claim

import (
	"context"
	"net/http"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
	"github.com/golang-jwt/jwt/v4"
)

type ContextKey string

/**
* String
* @param ctx context.Context, def string
* @return string
**/
func (c ContextKey) String(ctx context.Context, def string) string {
	val := ctx.Value(c)
	result, ok := val.(string)
	if !ok {
		return def
	}

	return result
}

/**
* Json
* @param ctx context.Context, def et.Json
* @return et.Json
**/
func (c ContextKey) Json(ctx context.Context, def et.Json) et.Json {
	val := ctx.Value(c)
	result, ok := val.(et.Json)
	if !ok {
		return def
	}

	return result
}

/**
* Int
* @param ctx context.Context, def int
* @return int
**/
func (c ContextKey) Int(ctx context.Context, def int) int {
	val := ctx.Value(c)
	result, ok := val.(int)
	if !ok {
		return def
	}

	return result
}

/**
* Num
* @param ctx context.Context, def float64
* @return float64
**/
func (c ContextKey) Num(ctx context.Context, def float64) float64 {
	val := ctx.Value(c)
	result, ok := val.(float64)
	if !ok {
		return def
	}

	return result
}

const (
	ServiceIdKey ContextKey = "serviceId"
	ClientIdKey  ContextKey = "clientId"
	AppKey       ContextKey = "app"
	NameKey      ContextKey = "name"
	DeviceKey    ContextKey = "device"
	UsernameKey  ContextKey = "username"
	DataKey      ContextKey = "data"
	DurationKey  ContextKey = "duration"
)

type Claim struct {
	Salt     string        `json:"salt"`
	ID       string        `json:"id"`
	App      string        `json:"app"`
	Name     string        `json:"name"`
	Username string        `json:"username"`
	Device   string        `json:"device"`
	Duration time.Duration `json:"duration"`
	Data     et.Json       `json:"data"`
	jwt.StandardClaims
}

/**
* ToJson
* @return et.Json
**/
func (c *Claim) ToJson() et.Json {
	return et.Json{
		"id":        c.ID,
		"app":       c.App,
		"name":      c.Name,
		"username":  c.Username,
		"device":    c.Device,
		"subject":   c.Subject,
		"duration":  c.Duration,
		"data":      c.Data,
		"expiresAt": time.Unix(c.ExpiresAt, 0).Format("2006-01-02 03:04:05 PM"),
	}
}

/**
* getTokenKey
* @param app, device, id string
* @return string
**/
func getTokenKey(app, device, id string) string {
	return reg.GenKey("token", app, device, id)
}

/**
* newClaim
* @param id, app, name, username, device, data et.Json, duration time.Duration
* @return Claim
**/
func newClaim(id, app, name, username, device string, data et.Json, duration time.Duration) Claim {
	c := Claim{}
	c.Salt = utility.GetOTP(6)
	c.ID = id
	c.App = app
	c.Name = name
	c.Username = username
	c.Device = device
	c.Duration = duration
	c.Data = data
	if c.Duration != 0 {
		c.ExpiresAt = timezone.Add(c.Duration).Unix()
	}

	return c
}

/**
* newToken
* @param c Claim
* @return string, error
**/
func newToken(c Claim) (string, error) {
	secret := config.String("SECRET", "1977")
	_jwt := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	token, err := _jwt.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	key := getTokenKey(c.App, c.Device, c.ID)
	cache.Set(key, token, c.Duration)

	return token, nil
}

/**
* NewToken
* @param id, app, name, username, device string, duration time.Duration
* @return string, error
**/
func NewToken(id, app, name, username, device string, duration time.Duration) (string, error) {
	result := newClaim(id, app, name, username, device, et.Json{}, duration)
	return newToken(result)
}

/**
* NewAutorization
* @param id, app, name, username, device string, data et.Json, duration time.Duration
* @return string, error
**/
func NewTokenData(id, app, name, username, device string, data et.Json, duration time.Duration) (string, error) {
	c := newClaim(id, app, name, username, device, data, duration)
	return newToken(c)
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
* @param app, device, id string
* @return error
**/
func DeleteToken(app, device, id string) error {
	key := getTokenKey(app, device, id)
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
	claim, err := ParceToken(token)
	if err != nil {
		return err
	}

	return DeleteToken(claim.App, claim.Device, claim.ID)
}

/**
* ParceToken
* @param token string
* @return *Claim, error
**/
func ParceToken(token string) (*Claim, error) {
	secret := config.String("SECRET", "1977")
	jToken, err := jwt.Parse(token, func(*jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, logs.Alert(err)
	}

	if !jToken.Valid {
		return nil, logs.Alertm(MSG_TOKEN_INVALID)
	}

	claim, ok := jToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, logs.Alertm(MSG_REQUIRED_INVALID)
	}

	params := et.Json{}
	for k, v := range claim {
		params[k] = v
	}

	app, ok := claim["app"].(string)
	if !ok {
		return nil, logs.Alertf(MSG_TOKEN_INVALID_ATRIB, "app")
	}

	device, ok := claim["device"].(string)
	if !ok {
		return nil, logs.Alertf(MSG_TOKEN_INVALID_ATRIB, "device")
	}

	id, ok := claim["id"].(string)
	if !ok {
		return nil, logs.Alertf(MSG_TOKEN_INVALID_ATRIB, "id")
	}

	second := params.Num("duration")
	duration := time.Duration(second)

	result := &Claim{
		ID:       id,
		App:      app,
		Name:     params.Str("name"),
		Username: params.Str("username"),
		Device:   device,
		Duration: duration,
		Data:     params.Json("data"),
	}
	if result.Duration != 0 {
		exp, ok := claim["exp"].(float64)
		if ok {
			result.ExpiresAt = int64(exp)
		}
	}

	return result, nil
}

/**
* ValidToken
* @param token string
* @return *Claim, error
**/
func ValidToken(token string) (*Claim, error) {
	result, err := ParceToken(token)
	if err != nil {
		return nil, err
	}

	key := getTokenKey(result.App, result.Device, result.ID)
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
* SetToken
* @param app, device, id string, token string, duration time.Duration
* @return string
**/
func SetToken(app, device, id, token string, duration time.Duration) string {
	key := getTokenKey(app, device, id)
	cache.Set(key, token, duration)

	return key
}

/**
* ServiceId
* @param r *http.Request
* @return string
**/
func ServiceId(r *http.Request) string {
	ctx := r.Context()
	return ServiceIdKey.String(ctx, "-1")
}

/**
* ClientId
* @param r *http.Request
* @return et.Json
**/
func ClientId(r *http.Request) string {
	ctx := r.Context()
	return ClientIdKey.String(ctx, "-1")
}

/**
* ClientName
* @param r *http.Request
* @return string
**/
func ClientName(r *http.Request) string {
	ctx := r.Context()
	return NameKey.String(ctx, "Anonimo")
}

/**
* App
* @param r *http.Request
* @return string
**/
func App(r *http.Request) string {
	ctx := r.Context()
	return AppKey.String(ctx, "-1")
}

/**
* Username
* @param r *http.Request
* @return string
**/
func Username(r *http.Request) string {
	ctx := r.Context()
	return UsernameKey.String(ctx, "Anonimo")
}

/**
* Device
* @param r *http.Request
* @return string
**/
func Device(r *http.Request) string {
	ctx := r.Context()
	return DeviceKey.String(ctx, "Anonimo")
}

/**
* Data
* @param r *http.Request
* @return et.Json
**/
func Data(r *http.Request) et.Json {
	ctx := r.Context()
	return DataKey.Json(ctx, et.Json{})
}

/**
* Duration
* @param r *http.Request
* @return time.Duration
**/
func Duration(r *http.Request) time.Duration {
	ctx := r.Context()
	return time.Duration(DurationKey.Num(ctx, 0))
}

/**
* Client
**/
func Client(r *http.Request) et.Json {
	now := utility.Now()
	ctx := r.Context()
	return et.Json{
		"date_at":    now,
		"client_id":  ClientIdKey.String(ctx, "-1"),
		"name":       NameKey.String(ctx, "Anonimo"),
		"username":   UsernameKey.String(ctx, "Anonimo"),
		"device":     DeviceKey.String(ctx, "Anonimo"),
		"data":       DataKey.Json(ctx, et.Json{}),
		"service_id": ServiceIdKey.String(ctx, "-1"),
		"app":        AppKey.String(ctx, "-1"),
		"duration":   DurationKey.Num(ctx, 0),
	}
}
