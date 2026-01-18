package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/utility"
)

/**
* NewSession
* @return *session.Session
**/
func newSession() (*session.Session, error) {
	err := utility.Validate([]string{
		"AWS_REGION",
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_SESSION_TOKEN",
	})
	if err != nil {
		return nil, err
	}

	region := envar.GetStr("AWS_REGION", "")
	keyId := envar.GetStr("AWS_ACCESS_KEY_ID", "")
	secret := envar.GetStr("AWS_SECRET_ACCESS_KEY", "")
	token := envar.GetStr("AWS_SESSION_TOKEN", "")

	return session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
		Credentials: credentials.NewStaticCredentials(
			keyId,
			secret,
			token,
		),
	})), nil
}
