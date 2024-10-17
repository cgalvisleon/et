package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/cgalvisleon/et/envar"
)

/**
* AWS Session
**/
func AwsSession() *session.Session {
	region := envar.EnvarStr("", "AWS_REGION")
	id := envar.EnvarStr("", "AWS_ACCESS_KEY_ID")
	secret := envar.EnvarStr("", "AWS_SECRET_ACCESS_KEY")
	token := envar.EnvarStr("", "AWS_SESSION_TOKEN")

	return session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
		Credentials: credentials.NewStaticCredentials(
			id,
			secret,
			token,
		),
	}))
}
