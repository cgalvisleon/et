package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/mistake"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/strs"
)

/**
* NewSession
* @return *session.Session
**/
func newSession() (*session.Session, error) {
	region := envar.EnvarStr("", "AWS_REGION")
	keyId := envar.EnvarStr("", "AWS_ACCESS_KEY_ID")
	secret := envar.EnvarStr("", "AWS_SECRET_ACCESS_KEY")
	token := envar.EnvarStr("", "AWS_SESSION_TOKEN")

	if strs.IsEmpty(region) {
		return nil, mistake.Newf(msg.ERR_ENV_REQUIRED, "AWS_REGION")
	}

	if strs.IsEmpty(keyId) {
		return nil, mistake.Newf(msg.ERR_ENV_REQUIRED, "AWS_ACCESS_KEY_ID")
	}

	if strs.IsEmpty(secret) {
		return nil, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "AWS_SECRET_ACCESS_KEY")
	}

	if strs.IsEmpty(token) {
		return nil, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "AWS_SESSION_TOKEN")
	}

	return session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
		Credentials: credentials.NewStaticCredentials(
			keyId,
			secret,
			token,
		),
	})), nil
}
