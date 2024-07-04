package token

import (
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/golang-jwt/jwt/v4"
)

type Claim struct {
	ClientId string        `json:"clientId"`
	Name     string        `json:"name"`
	Iat      time.Time     `json:"iat"`
	Exp      time.Duration `json:"exp"`
	App      string        `json:"app"`
	Kind     string        `json:"kind"`
	Device   string        `json:"device"`
	jwt.StandardClaims
}

const (
	ERR_INVALID_CLAIM = "Invalid claim"
	ERR_AUTORIZATION  = "Invalid autorization"
)

// TokenKey generate a key for the token
func Key(app, device, clientId string) string {
	result := strs.Append(app, device, "-")
	result = strs.Append(result, clientId, "-")
	return strs.Format(`token:%s`, result)
}

// Parece method to use in token
func parce(tokenString string) (*jwt.Token, error) {
	secret := envar.GetStr("", "SECRET")
	token, err := jwt.Parse(tokenString, func(*jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	return token, nil
}

// Generate method to use in token
func Generate(clientId, name, app, kind, device string, expired time.Duration) (string, error) {
	c := Claim{
		ClientId: clientId,
		Name:     name,
		Iat:      time.Now(),
		Exp:      expired,
		App:      app,
		Kind:     kind,
		Device:   device,
	}
	jwT := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	secret := envar.GetStr("", "SECRET")
	token, err := jwT.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return token, nil
}

// Validate method to use in token
func Validate(tokenString string) (*Claim, error) {
	token, err := parce(tokenString)
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, logs.Errorm(ERR_AUTORIZATION)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
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

	exp, ok := claims["exp"].(float64)
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

	return &Claim{
		ClientId: clientId,
		Name:     name,
		Iat:      time.Unix(int64(iat), 0),
		Exp:      time.Duration(exp),
		App:      app,
		Kind:     kind,
		Device:   device,
	}, nil
}
