package mistake

import (
	"errors"
	"fmt"
)

func New(message string) error {
	return errors.New(message)
}

func Newf(format string, args ...any) error {
	message := fmt.Sprintf(format, args...)
	return New(message)
}
