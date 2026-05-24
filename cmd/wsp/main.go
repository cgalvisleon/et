package main

import (
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/wsp"
)

func main() {
	sender := wsp.NewSender("token", "phone_number_id").IsTest()
	result, err := sender.SendTextMessage("1234567890", "Hello World")
	if err != nil {
		logs.Error(err)
		return
	}
	logs.Debug(result.ToString())
}
