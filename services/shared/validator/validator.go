package validator

import "github.com/go-playground/validator/v10"

// StructValidator integrates go-playground/validator with Fiber v3 request binding.
type StructValidator struct {
	validate *validator.Validate
}

// NewStructValidator constructs a Fiber v3 StructValidator instance.
func NewStructValidator() *StructValidator {
	return &StructValidator{
		validate: validator.New(),
	}
}

// Validate validates struct fields against `validate:"..."` tags.
func (v *StructValidator) Validate(out any) error {
	return v.validate.Struct(out)
}
