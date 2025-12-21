package config

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// profileNamePattern defines the valid format for profile names.
// Both vendor and name segments must match [0-9a-zA-Z-_]+
var profileNamePattern = regexp.MustCompile(`^[0-9a-zA-Z_-]+$`)

// ValidateProfileName validates that a profile name follows the vendor/name format.
// Both vendor and name must match the pattern [0-9a-zA-Z-_]+.
// Returns nil if valid, or an error with a descriptive message if invalid.
func ValidateProfileName(name string) error {
	if name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}

	parts := strings.Split(name, "/")
	if len(parts) != 2 {
		return fmt.Errorf("profile name must be in vendor/name format, got '%s'", name)
	}

	vendor := parts[0]
	profileName := parts[1]

	if vendor == "" {
		return fmt.Errorf("vendor segment cannot be empty in profile name '%s'", name)
	}

	if profileName == "" {
		return fmt.Errorf("name segment cannot be empty in profile name '%s'", name)
	}

	if !profileNamePattern.MatchString(vendor) {
		return fmt.Errorf("vendor segment '%s' contains invalid characters; must match [0-9a-zA-Z_-]+", vendor)
	}

	if !profileNamePattern.MatchString(profileName) {
		return fmt.Errorf("name segment '%s' contains invalid characters; must match [0-9a-zA-Z_-]+", profileName)
	}

	return nil
}

// profileNameValidator is a custom validator function for go-playground/validator.
// It validates that a field value follows the vendor/name format.
func profileNameValidator(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	return ValidateProfileName(value) == nil
}

// RegisterValidators registers all custom validators with the provided validator instance.
// This should be called once during application initialization.
func RegisterValidators(v *validator.Validate) error {
	return v.RegisterValidation("profile_name", profileNameValidator)
}

// NewValidator creates a new validator instance with all custom validators registered.
// Returns the configured validator and any error that occurred during registration.
func NewValidator() (*validator.Validate, error) {
	v := validator.New()
	if err := RegisterValidators(v); err != nil {
		return nil, fmt.Errorf("failed to register custom validators: %w", err)
	}
	return v, nil
}
