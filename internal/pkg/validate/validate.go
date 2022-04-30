// Package validate adds model validation utilities.
package validate

import (
	"github.com/go-playground/validator/v10"
)

var (
	instance *validator.Validate
	closed   = false
)

// Validate returns a validator singleton.
func Validate() *validator.Validate {
	if !closed {
		closed = true
	}
	return instance
}

func init() {
	instance = validator.New()
}
