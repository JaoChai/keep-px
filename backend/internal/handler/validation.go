package handler

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// FormatValidationErrors converts validator.ValidationErrors into
// user-friendly messages that do not leak internal struct field names.
func FormatValidationErrors(err error) string {
	ve, ok := err.(validator.ValidationErrors)
	if !ok {
		return "invalid request"
	}

	msgs := make([]string, 0, len(ve))
	for _, fe := range ve {
		field := strings.ToLower(fe.Field())
		switch fe.Tag() {
		case "required":
			msgs = append(msgs, fmt.Sprintf("%s is required", field))
		case "min":
			msgs = append(msgs, fmt.Sprintf("%s must be at least %s", field, fe.Param()))
		case "max":
			msgs = append(msgs, fmt.Sprintf("%s must be at most %s", field, fe.Param()))
		case "oneof":
			msgs = append(msgs, fmt.Sprintf("%s must be one of: %s", field, fe.Param()))
		case "email":
			msgs = append(msgs, fmt.Sprintf("%s must be a valid email", field))
		case "url":
			msgs = append(msgs, fmt.Sprintf("%s must be a valid URL", field))
		default:
			msgs = append(msgs, fmt.Sprintf("%s is invalid", field))
		}
	}
	return strings.Join(msgs, "; ")
}
