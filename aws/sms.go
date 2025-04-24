package aws

import (
	"slices"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/mistake"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/strs"
)

/**
* SendSMS
* @param contactNumbers []string, content string, params []et.Json, tp string
* @return et.Items, error
**/
func SendSMS(contactNumbers []string, content string, params []et.Json, tp string) (et.Items, error) {
	if len(contactNumbers) == 0 {
		return et.Items{}, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "contactNumbers")
	}

	if content == "" {
		return et.Items{}, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "content")
	}

	if !slices.Contains([]string{"Transactional", "Promotional"}, tp) {
		return et.Items{}, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "type")
	}

	sess, err := newSession()
	if err != nil {
		return et.Items{}, err
	}

	result := et.Items{}
	for _, phoneNumber := range contactNumbers {
		message := content
		for _, param := range params {
			for k, v := range param {
				k := strs.Format("{{%s}}", k)
				s := strs.Format("%v", v)
				message = strs.Replace(message, k, s)
			}
		}

		svc := sns.New(sess)
		params := &sns.PublishInput{
			Message:     aws.String(message),
			PhoneNumber: aws.String(phoneNumber),
			MessageAttributes: map[string]*sns.MessageAttributeValue{
				"AWS.SNS.SMS.SMSType": {
					DataType:    aws.String("String"),
					StringValue: aws.String(tp),
				},
			},
		}

		output, err := svc.Publish(params)
		if err != nil {
			return result, err
		}

		result.Add(et.Json{
			"phoneNumber": phoneNumber,
			"type":        tp,
			"sender":      "AWS SNS",
			"status": et.Json{
				"sequence": output.SequenceNumber,
				"status":   output.MessageId,
			},
		})
	}

	return result, nil
}
