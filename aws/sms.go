package aws

import (
	"fmt"
	"slices"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/strs"
)

type SenderAWS struct {
	Params Params
	sess   *session.Session
}

/**
* NewSenderAWS
* @param params Params
* @return *SenderAWS, error
**/
func NewSenderAWS(params Params) (*SenderAWS, error) {
	result := &SenderAWS{
		Params: params,
	}

	sess, err := newSession(params)
	if err != nil {
		return nil, err
	}

	result.sess = sess
	return result, nil
}

/**
* SendSMS
* @param contactNumbers []string, content string, params et.Json, tp string
* @return et.Items, error
**/
func (s *SenderAWS) SendSMS(contactNumbers []string, content string, params et.Json, tpMessage string) (et.Item, error) {
	if len(contactNumbers) == 0 {
		return et.Item{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "contactNumbers")
	}

	if content == "" {
		return et.Item{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "content")
	}

	if !slices.Contains([]string{"Transactional", "Promotional"}, tpMessage) {
		return et.Item{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "type")
	}

	result := []et.Json{}
	for _, phoneNumber := range contactNumbers {
		message := content
		for k, v := range params {
			k := fmt.Sprintf("{{%s}}", k)
			s := fmt.Sprintf("%v", v)
			message = strs.Replace(message, k, s)
		}

		svc := sns.New(s.sess)
		params := &sns.PublishInput{
			Message:     aws.String(message),
			PhoneNumber: aws.String(phoneNumber),
			MessageAttributes: map[string]*sns.MessageAttributeValue{
				"AWS.SNS.SMS.SMSType": {
					DataType:    aws.String("String"),
					StringValue: aws.String(tpMessage),
				},
			},
		}

		output, err := svc.Publish(params)
		if err != nil {
			result = append(result, et.Json{
				"phoneNumber": phoneNumber,
				"error":       err.Error(),
				"result":      output,
			})

			return et.Item{
				Ok: false,
				Result: et.Json{
					"provider": "AWS SNS",
					"type":     tpMessage,
					"message":  err.Error(),
					"result":   output,
				},
			}, err
		}

		result = append(result, et.Json{
			"phoneNumber": phoneNumber,
			"result":      output,
		})
	}

	return et.Item{
		Ok: true,
		Result: et.Json{
			"provider": "AWS SNS",
			"type":     tpMessage,
			"message":  "SMS sent successfully",
			"result":   result,
		},
	}, nil
}
