package handler

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// newValidator creates a validator that uses JSON tag names in error messages
// instead of Go struct field names (e.g. "refresh_token" instead of "refreshtoken").
func newValidator() *validator.Validate {
	v := validator.New()
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
	return v
}

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
