package aws

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
)

type Params struct {
	Region string
	KeyId  string
	Secret string
	Token  string
}

/**
* NewSession
* @return *session.Session
**/
func newSession(params Params) (*session.Session, error) {
	if params.Region == "" {
		return nil, errors.New(MSG_REGION_REQUIRED)
	}
	if params.KeyId == "" {
		return nil, errors.New(MSG_KEY_ID_REQUIRED)
	}
	if params.Secret == "" {
		return nil, errors.New(MSG_SECRET_REQUIRED)
	}
	if params.Token == "" {
		return nil, errors.New(MSG_TOKEN_REQUIRED)
	}

	region := params.Region
	keyId := params.KeyId
	secret := params.Secret
	token := params.Token

	return session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
		Credentials: credentials.NewStaticCredentials(
			keyId,
			secret,
			token,
		),
	})), nil
}
