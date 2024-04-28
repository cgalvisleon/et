package token

import (
	"time"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
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

// TokenKey generate a key for the token
func tokenKey(app, device, sub string) string {
	result := strs.Append(app, device, "-")
	result = strs.Append(result, sub, "-")
	return strs.Format(`token:%s`, result)
}

// Generate a token
func Generate(sub, name, app, kind, device string, expired time.Duration) (string, error) {
	if conn == nil {
		return "", logs.Log(ERR_NOT_TOKEN_SERVICE)
	}

	token, err := conn.Generate(sub, name, app, kind, device, expired)
	if err != nil {
		return "", err
	}

	return token, nil
}

// Validate a token
func Validate(tokenString string) (bool, error) {
	if conn == nil {
		return false, logs.Log(ERR_NOT_TOKEN_SERVICE)
	}

	return conn.Validate(tokenString)
}

// Delete a token
func Delete(tokenString string) error {
	if conn == nil {
		return logs.Log(ERR_NOT_TOKEN_SERVICE)
	}

	return conn.Delete(tokenString)
}

// Parce a token
func Parce(tokenString string) (*Claim, error) {
	if conn == nil {
		return nil, logs.Log(ERR_NOT_TOKEN_SERVICE)
	}

	return conn.Parce(tokenString)
}
