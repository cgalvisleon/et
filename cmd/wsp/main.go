package main

import (
	"fmt"

	"github.com/cgalvisleon/et/wsp"
)

func main() {
	sender := wsp.NewSender("token", "phone_number_id")
	message := &wsp.Message{
		To:   "1234567890",
		Type: "text",
		Text: "Hello World",
	}
	result, err := sender.SendMessage(message)
	fmt.Println(result)
	fmt.Println(err)
}
