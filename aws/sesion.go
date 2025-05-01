package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/cgalvisleon/et/config"
)

/**
* NewSession
* @return *session.Session
**/
func newSession() (*session.Session, error) {
	err := config.Validate([]string{
		"AWS_REGION",
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_SESSION_TOKEN",
	})
	if err != nil {
		return nil, err
	}

	region := config.String("AWS_REGION", "")
	keyId := config.String("AWS_ACCESS_KEY_ID", "")
	secret := config.String("AWS_SECRET_ACCESS_KEY", "")
	token := config.String("AWS_SESSION_TOKEN", "")

	return session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
		Credentials: credentials.NewStaticCredentials(
			keyId,
			secret,
			token,
		),
	})), nil
}
