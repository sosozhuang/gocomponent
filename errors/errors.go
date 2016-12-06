package errors

import (
	"fmt"
	"errors"
)

func Errorf(format string, args ...interface{}) error {
	err := errors.New(fmt.Sprintf(format, args...))
	return err
}
