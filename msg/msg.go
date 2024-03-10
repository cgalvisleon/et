package msg

import (
	"encoding/json"
	"io"
	"os"
)

type Messages map[string]string

type MultilingualService struct {
	MessagesByLanguage map[string]Messages
	Language           string
}

var servie *MultilingualService

func NewMultilingualService(language string) *MultilingualService {
	return &MultilingualService{
		MessagesByLanguage: make(map[string]Messages),
		Language:           language,
	}
}

func (s *MultilingualService) LoadMessages(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	var messages Messages
	err = json.Unmarshal(data, &messages)
	if err != nil {
		return err
	}

	s.MessagesByLanguage[s.Language] = messages
	return nil
}

func (s *MultilingualService) GetMessage(key string) string {
	messages := s.MessagesByLanguage[s.Language]
	return messages[key]
}

func T(msg string) string {
	return servie.GetMessage(msg)
}

func init() {
	servie = NewMultilingualService("es")
	if err := servie.LoadMessages("./es.json"); err != nil {
		return
	}
}
