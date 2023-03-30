package app

import (
	"errors"
	"fmt"
)

var ErrInvalidArgument = errors.New("invalid argument")

func NewErrInvalidArgument(arg string) error {
	return fmt.Errorf("%w: %s", ErrInvalidArgument, arg)
}
