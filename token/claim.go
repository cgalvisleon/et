package token

import (
	"errors"
	"net/http"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/generic"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
	"github.com/golang-jwt/jwt/v4"
)

type Claim struct {
	Sub    string        `json:"sub"`
	Name   string        `json:"name"`
	Iat    time.Time     `json:"iat"`
	Exp    time.Duration `json:"exp"`
	App    string        `json:"app"`
	Kind   string        `json:"kind"`
	Device string        `json:"device"`
	jwt.StandardClaims
}

func TokenKey(app, device, sub string) string {
	result := strs.Append(app, device, "-")
	result = strs.Append(result, sub, "-")
	return strs.Format(`token:%s`, result)
}

func DelToken(app, device, sub string) error {
	if conn == nil {
		return logs.Log(ERR_NOT_TOKEN_SERVICE)
	}

	key := TokenKey(app, device, sub)
	ok := conn.cache.Del(key)
	if !ok {
		return logs.Alertm(MSG_TOKEN_NOT_FOUND)
	}

	conn.pubsub.Publish("token/delete", et.Json{
		"key": key,
	})

	return nil
}

func DelTokeStrng(tokenString string) error {
	if conn == nil {
		return logs.Log(ERR_NOT_TOKEN_SERVICE)
	}

	token, err := conn.parce(tokenString)
	if err != nil {
		return err
	}

	claim, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return errors.New(ERR_INVALID_CLAIM)
	}

	sub, ok := claim["sub"].(string)
	if !ok {
		return nil
	}

	app, ok := claim["app"].(string)
	if !ok {
		return nil
	}

	device, ok := claim["device"].(string)
	if !ok {
		return nil
	}

	return DelToken(app, device, sub)
}

func ParceToken(tokenString string) (*Claim, error) {
	if conn == nil {
		return nil, logs.Log(ERR_NOT_TOKEN_SERVICE)
	}

	token, err := conn.parce(tokenString)
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, logs.Alertm(MSG_TOKEN_INVALID)
	}

	claim, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, logs.Alertm(MSG_REQUIRED_INVALID)
	}

	sub, ok := claim["sub"].(string)
	if !ok {
		return nil, logs.Alertf(MSG_TOKEN_INVALID_ATRIB, "sub")
	}

	app, ok := claim["app"].(string)
	if !ok {
		return nil, logs.Alertf(MSG_TOKEN_INVALID_ATRIB, "app")
	}

	device, ok := claim["device"].(string)
	if !ok {
		return nil, logs.Alertf(MSG_TOKEN_INVALID_ATRIB, "device")
	}

	name, ok := claim["name"].(string)
	if !ok {
		return nil, logs.Alertf(MSG_TOKEN_INVALID_ATRIB, "name")
	}

	kind, ok := claim["kind"].(string)
	if !ok {
		return nil, logs.Alertf(MSG_TOKEN_INVALID_ATRIB, "kind")
	}

	_exp, ok := claim["exp"].(float64)
	if !ok {
		return nil, logs.Alertf(MSG_TOKEN_INVALID_ATRIB, "duration")
	}

	exp := time.Duration(_exp)

	return &Claim{
		Sub:    sub,
		Name:   name,
		Exp:    exp,
		App:    app,
		Kind:   kind,
		Device: device,
	}, nil
}

func GetFromToken(tokenString string) (*Claim, error) {
	result, err := ParceToken(tokenString)
	if err != nil {
		return nil, err
	}

	key := TokenKey(result.App, result.Device, result.Sub)
	token := conn.cache.Get(key, "")
	if token != tokenString {
		return nil, logs.Alertm(MSG_TOKEN_INVALID)
	}

	conn.cache.Set(key, token, result.Exp)

	return result, nil
}

func GenToken(sub, name, app, kind, device string, expired time.Duration) (string, error) {
	if conn == nil {
		return "", logs.Log(ERR_NOT_TOKEN_SERVICE)
	}

	token, key, err := conn.generate(sub, name, app, kind, device, expired)
	if err != nil {
		return "", err
	}

	conn.cache.Set(key, token, expired)

	conn.pubsub.Publish("token/create", et.Json{
		"key":  key,
		"toke": token,
	})

	return token, nil
}

func GetClient(r *http.Request) et.Json {
	now := utility.Now()
	ctx := r.Context()

	return et.Json{
		"date_of":   now,
		"client_id": generic.New(ctx.Value("clientId")).Str(),
		"name":      generic.New(ctx.Value("name")).Str(),
	}
}
