package mistake

import (
	"github.com/cgalvisleon/et/et"
)

type Mistake struct {
	Message string
	Code    string
	Data    et.Json
}

/**
* NewMistake return a new mistake
* @param message, code string, data any
* @return *Mistake
**/
func NewMistake(message, code string, data et.Json) *Mistake {
	return &Mistake{
		Message: message,
		Code:    code,
		Data:    data,
	}
}

/**
* Error return the error message
* @return string
**/
func (m *Mistake) Error() string {
	return m.Message
}
