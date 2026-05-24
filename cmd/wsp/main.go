package main

import (
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/wsp"
)

func main() {
	sender := wsp.NewSender("token", "phone_number_id")
	message := &wsp.Message{
		To:   "1234567890",
		Text: "Hello World",
	}
	result, err := sender.SendMessage(message)
	if err != nil {
		logs.Error(err)
		return
	}
	logs.Debug(result)
}
