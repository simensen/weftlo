// Package config provides infrastructure for loading and resolving configuration.
package config

import (
	"fmt"
	"strings"
)

// ConfigNotFoundError is returned when a required configuration file is missing.
// This is specifically used for project configuration files that must exist.
type ConfigNotFoundError struct {
	// Path is the path to the configuration file that was not found.
	Path string
}

func (e *ConfigNotFoundError) Error() string {
	return fmt.Sprintf("configuration file not found: %s", e.Path)
}

// ConfigParseError is returned when a configuration file contains invalid YAML syntax.
type ConfigParseError struct {
	// Path is the path to the configuration file that failed to parse.
	Path string
	// Err is the underlying parse error from the YAML parser.
	Err error
}

func (e *ConfigParseError) Error() string {
	return fmt.Sprintf("failed to parse configuration file %s: %v", e.Path, e.Err)
}

func (e *ConfigParseError) Unwrap() error {
	return e.Err
}

// ConfigValidationError is returned when a configuration file fails validation.
// This includes field-specific validation errors.
type ConfigValidationError struct {
	// Path is the path to the configuration file that failed validation.
	Path string
	// Field is the name of the field that failed validation.
	Field string
	// Tag is the validation tag that failed.
	Tag string
	// Value is the value that failed validation.
	Value string
	// Err is the underlying validation error.
	Err error
}

func (e *ConfigValidationError) Error() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "validation failed for configuration file %s", e.Path)
	if e.Field != "" {
		fmt.Fprintf(&sb, ": field '%s'", e.Field)
	}
	if e.Tag != "" {
		fmt.Fprintf(&sb, " failed on '%s' validation", e.Tag)
	}
	if e.Value != "" {
		fmt.Fprintf(&sb, " with value '%s'", e.Value)
	}
	return sb.String()
}

func (e *ConfigValidationError) Unwrap() error {
	return e.Err
}
