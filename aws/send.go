package aws

import (
	"fmt"
	"net/smtp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
)

func SMS(country string, mobile string, message string) (bool, interface{}, error) {
	var result bool

	phoneNumber := country + mobile
	sess := AwsSession()
	svc := sns.New(sess)

	message = strs.RemoveAcents(message)
	params := &sns.PublishInput{
		Message:     aws.String(message),
		PhoneNumber: aws.String(phoneNumber),
	}

	output, err := svc.Publish(params)
	if err != nil {
		return result, output, logs.Error(err)
	}

	return true, output, nil
}

func Email(to string, subject string, message string) (bool, error) {
	from := "email@email.com"
	port := envar.EnvarInt(25, "SMTP_PORT")
	host := envar.EnvarStr("relayappann.email.local", "SMTP_HOST")
	addr := strs.Format(`%s:%d`, host, port)
	c, err := smtp.Dial(addr)
	if err != nil {
		return false, err
	}

	if err := c.Mail(from); err != nil {
		return false, err
	}

	if err := c.Rcpt(to); err != nil {
		return false, err
	}

	wc, err := c.Data()
	if err != nil {
		return false, err
	}

	msg := fmt.Sprintf("To: %s\r\nFrom: %s\r\nSubject: %s\r\n\r\n%s", to, from, subject, message)
	if _, err = wc.Write([]byte(msg)); err != nil {
		return false, err
	}

	if err := wc.Close(); err != nil {
		return false, err
	}

	return true, nil
}
