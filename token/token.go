package token

import (
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/pubsub"
	"github.com/golang-jwt/jwt/v4"
)

type Token struct {
	secret string
	cache  cache.Cache
	pubsub pubsub.PubSub
}

var conn *Token

func Load(cache cache.Cache, pubsub pubsub.PubSub) (*Token, error) {
	if conn != nil {
		return conn, nil
	}

	conn = &Token{
		secret: envar.EnvarStr("", "SECRET"),
		cache:  cache,
		pubsub: pubsub,
	}

	return conn, nil
}

// Parece method to use in token
func (t *Token) parce(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(*jwt.Token) (interface{}, error) {
		return []byte(t.secret), nil
	})
	if err != nil {
		return nil, err
	}

	return token, nil
}

// Generate method to use in token
func (t *Token) Generate(sub, name, app, kind, device string, expired time.Duration) (string, error) {
	c := Claim{
		Sub:    sub,
		Name:   name,
		Iat:    time.Now(),
		Exp:    expired,
		App:    app,
		Kind:   kind,
		Device: device,
	}
	_jwt := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	token, err := _jwt.SignedString([]byte(t.secret))
	if err != nil {
		return "", err
	}

	key := tokenKey(app, device, sub)
	old := t.cache.Get(key, "")
	if t.cache != nil {
		t.cache.Set(key, token, expired)
	}

	if t.pubsub != nil && old != token {
		t.pubsub.Publish(key, et.Json{
			"action": "close",
		})
	}

	return token, nil
}

// Validate method to use in token
func (t *Token) Validate(tokenString string) (bool, error) {
	token, err := t.parce(tokenString)
	if err != nil {
		return false, err
	}

	if token.Valid {
		return true, nil
	}

	err = t.Delete(tokenString)
	if err != nil {
		return false, err
	}

	return false, nil
}

// Delete method to use in token
func (t *Token) Delete(tokenString string) error {
	token, err := t.parce(tokenString)
	if err != nil {
		return err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return logs.Alertm(ERR_INVALID_CLAIM)
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return nil
	}

	app, ok := claims["app"].(string)
	if !ok {
		return nil
	}

	device, ok := claims["device"].(string)
	if !ok {
		return nil
	}

	key := tokenKey(app, device, sub)
	if t.cache != nil {
		t.cache.Del(key)
	}

	if t.pubsub != nil {
		t.pubsub.Publish(key, et.Json{
			"action": "close",
		})
	}

	return nil
}

// Parce method to use in token
func (t *Token) Parce(tokenString string) (*Claim, error) {
	token, err := t.parce(tokenString)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, logs.Alertm(ERR_INVALID_CLAIM)
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return nil, nil
	}

	name, ok := claims["name"].(string)
	if !ok {
		return nil, nil
	}

	iat, ok := claims["iat"].(float64)
	if !ok {
		return nil, nil
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, nil
	}

	app, ok := claims["app"].(string)
	if !ok {
		return nil, nil
	}

	kind, ok := claims["kind"].(string)
	if !ok {
		return nil, nil
	}

	device, ok := claims["device"].(string)
	if !ok {
		return nil, nil
	}

	return &Claim{
		Sub:    sub,
		Name:   name,
		Iat:    time.Unix(int64(iat), 0),
		Exp:    time.Duration(exp),
		App:    app,
		Kind:   kind,
		Device: device,
	}, nil
}
