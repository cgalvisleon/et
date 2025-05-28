package mistake

import (
	"errors"
	"fmt"
)

/**
* New
* @param message string
* @return error
**/
func New(message string) error {
	return errors.New(message)
}

/**
* Newf
* @param format string, args ...any
* @return error
**/
func Newf(format string, args ...any) error {
	message := fmt.Sprintf(format, args...)
	return New(message)
}
