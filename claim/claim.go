package claim

import (
	"context"
	"net/http"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
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
	AppKey       ContextKey = "app"
	NameKey      ContextKey = "name"
	SubjectKey   ContextKey = "subject"
	UsernameKey  ContextKey = "username"
	TokenKey     ContextKey = "token"
	ProjectIdKey ContextKey = "projectId"
	ProfileTpKey ContextKey = "profileTp"
	ModelKey     ContextKey = "model"
	TagKey       ContextKey = "tag"
)

type Claim struct {
	Salt     string        `json:"salt"`
	ID       string        `json:"id"`
	App      string        `json:"app"`
	Name     string        `json:"name"`
	Username string        `json:"username"`
	Device   string        `json:"device"`
	Duration time.Duration `json:"duration"`
	Tag      string        `json:"tag"`
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
		"tag":       c.Tag,
		"expiresAt": time.Unix(c.ExpiresAt, 0).Format("2006-01-02 03:04:05 PM"),
	}
}

/**
* GetTokenKey
* @param app string
* @param device string
* @param id string
* @return string
**/
func GetTokenKey(app, device, id string) string {
	return cache.GenKey("token", app, device, id)
}

/**
* newClaim
* @param id string
* @param app string
* @param name string
* @param username string
* @param device string
* @param duration time.Duration
* @return Claim
**/
func newClaim(id, app, name string, username, device, tag string, duration time.Duration) Claim {
	c := Claim{}
	c.Salt = utility.GetOTP(6)
	c.ID = id
	c.App = app
	c.Name = name
	c.Username = username
	c.Device = device
	c.Duration = duration
	c.Tag = tag
	if c.Duration != 0 {
		c.ExpiresAt = timezone.Add(c.Duration).Unix()
	}

	return c
}

/**
* newToken
* @param c Claim
* @return string
* @return error
**/
func newToken(c Claim) (string, error) {
	secret := envar.GetStr("1977", "SECRET")
	_jwt := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	token, err := _jwt.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	key := GetTokenKey(c.App, c.Device, c.ID)
	err = cache.Set(key, token, c.Duration)
	if err != nil {
		return "", err
	}

	return token, nil
}

/**
* NewToken
* @param id string
* @param app string
* @param name string
* @param username string
* @param device string
* @param duration time.Duration
* @return token string
* @return key string
* @return err error
**/
func NewToken(id, app, name string, username, device string, duration time.Duration) (string, error) {
	c := newClaim(id, app, name, username, device, "", duration)
	return newToken(c)
}

/**
* NewAutorization
* @param id string
* @param app string
* @param name string
* @param username string
* @param device string
* @param tag string
* @param duration time.Duration
* @return token string
* @return err error
**/
func NewAutorization(id, app, name, username, device, tag string, duration time.Duration) (string, error) {
	c := newClaim(id, app, name, username, device, tag, duration)
	return newToken(c)
}

/**
* GetToken
* @param key string
* @return string
* @return error
**/
func GetToken(key string) (string, error) {
	return cache.Get(key, "")
}

/**
* DeleteToken
* @param app string
* @param device string
* @param id string
* @return error
**/
func DeleteToken(app, device, id string) error {
	key := GetTokenKey(app, device, id)
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
* @return *Claim
* @return error
**/
func ParceToken(token string) (*Claim, error) {
	secret := envar.GetStr("1977", "SECRET")
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

	app, ok := claim["app"].(string)
	if !ok {
		return nil, logs.Alertf(MSG_TOKEN_INVALID_ATRIB, "app")
	}

	id, ok := claim["id"].(string)
	if !ok {
		return nil, logs.Alertf(MSG_TOKEN_INVALID_ATRIB, "id")
	}

	name, ok := claim["name"].(string)
	if !ok {
		return nil, logs.Alertf(MSG_TOKEN_INVALID_ATRIB, "name")
	}

	username, ok := claim["username"].(string)
	if !ok {
		return nil, logs.Alertf(MSG_TOKEN_INVALID_ATRIB, "username")
	}

	device, ok := claim["device"].(string)
	if !ok {
		return nil, logs.Alertf(MSG_TOKEN_INVALID_ATRIB, "device")
	}

	second, ok := claim["duration"].(float64)
	if !ok {
		return nil, logs.Alertf(MSG_TOKEN_INVALID_ATRIB, "duration")
	}

	duration := time.Duration(second)

	result := &Claim{
		ID:       id,
		App:      app,
		Name:     name,
		Username: username,
		Device:   device,
		Duration: duration,
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
* @return *Claim
* @return error
**/
func ValidToken(token string) (*Claim, error) {
	result, err := ParceToken(token)
	if err != nil {
		return nil, err
	}

	key := GetTokenKey(result.App, result.Device, result.ID)
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
* @param app string
* @param device string
* @param id string
* @param token string
* @return error
**/
func SetToken(app, device, id, token string, duration time.Duration) error {
	key := GetTokenKey(app, device, id)
	err := cache.Set(key, token, duration)
	if err != nil {
		return err
	}

	return nil
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
* GetClientName
* @param r *http.Request
* @return et.Json
**/
func GetClientName(r *http.Request) string {
	ctx := r.Context()
	return NameKey.String(ctx, "Anonimo")
}

/**
* GetClient
* @param r *http.Request
* @return et.Json
**/
func GetClient(r *http.Request) et.Json {
	now := utility.Now()
	ctx := r.Context()
	username := UsernameKey.String(ctx, "Anonimo")
	fullName := NameKey.String(ctx, "Anonimo")
	clientId := ClientIdKey.String(ctx, "-1")
	tag := TagKey.String(ctx, "")

	return et.Json{
		"date_at":   now,
		"client_id": clientId,
		"username":  username,
		"full_name": fullName,
		"tag":       tag,
	}
}
