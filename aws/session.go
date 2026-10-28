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
	// Endpoint: another S3-compatible service (MinIO, SeaweedFS...); empty is AWS
	Endpoint string
	// PathStyle: bucket in the path (host/bucket/key), which most S3-compatible services need
	PathStyle bool
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

	region := params.Region
	keyId := params.KeyId
	secret := params.Secret
	token := params.Token

	config := &aws.Config{
		Region: aws.String(region),
		Credentials: credentials.NewStaticCredentials(
			keyId,
			secret,
			token,
		),
	}
	if params.Endpoint != "" {
		config.Endpoint = aws.String(params.Endpoint)
	}
	if params.PathStyle {
		config.S3ForcePathStyle = aws.Bool(true)
	}
	return session.NewSession(config)
}
