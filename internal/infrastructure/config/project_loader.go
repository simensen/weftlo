// Package config provides infrastructure for loading and resolving configuration.
package config

import (
	"os"
	"path/filepath"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/afero"
	"github.com/spf13/viper"

	domainconfig "github.com/simensen/weftlo/internal/domain/config"
)

// ProjectConfigLoader handles loading of project configuration from a project root.
// It uses viper to support YAML files and validates the configuration using go-playground/validator.
// The filesystem is injected to enable testable file operations.
type ProjectConfigLoader struct {
	// validate is the validator instance with custom validators registered.
	validate *validator.Validate
	// fs is the filesystem used for reading configuration files.
	fs afero.Fs
}

// NewProjectConfigLoader creates a new ProjectConfigLoader with the specified filesystem.
// The filesystem parameter allows for dependency injection, enabling tests to use
// an in-memory filesystem (afero.NewMemMapFs()) instead of the real filesystem.
func NewProjectConfigLoader(fs afero.Fs) *ProjectConfigLoader {
	v, err := domainconfig.NewValidator()
	if err != nil {
		// If validator creation fails, create a new one without custom validators
		// This should not happen in normal operation
		v = validator.New()
	}
	return &ProjectConfigLoader{
		validate: v,
		fs:       fs,
	}
}

// Load loads the project configuration from the specified project root.
// The project root should contain a .weftlo.yaml file.
//
// Returns:
//   - ConfigNotFoundError if the config file is missing
//   - ConfigParseError for invalid YAML syntax
//   - ConfigValidationError if validation fails
func (l *ProjectConfigLoader) Load(projectRoot string) (*domainconfig.ProjectConfig, error) {
	v := viper.New()

	// Configure viper to use the injected filesystem
	v.SetFs(l.fs)

	// Configure viper for YAML file reading
	// Note: .weftlo.yaml - the leading dot is part of the filename
	v.SetConfigName(".weftlo")
	v.SetConfigType("yaml")
	v.AddConfigPath(projectRoot)

	configPath := filepath.Join(projectRoot, ".weftlo.yaml")

	// Attempt to read the config file
	if err := v.ReadInConfig(); err != nil {
		// If the config file is not found, return a specific error
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, &ConfigNotFoundError{
				Path: configPath,
			}
		}

		// For permission errors or file doesn't exist at all
		// Note: afero errors are compatible with os.IsNotExist()
		if os.IsNotExist(err) {
			return nil, &ConfigNotFoundError{
				Path: configPath,
			}
		}

		// For parse errors, return a typed error
		return nil, &ConfigParseError{
			Path: configPath,
			Err:  err,
		}
	}

	// Read install_prefix from config, applying default only if not specified
	// If explicitly set to empty string, use empty (no prefix)
	installPrefix := domainconfig.DefaultInstallPrefix
	if v.IsSet("install_prefix") {
		installPrefix = v.GetString("install_prefix")
	}

	// Read profiles array from config
	profiles := v.GetStringSlice("profiles")

	// Read target_overrides from config
	// Map values can be:
	//   - null (nil pointer): suppress the target
	//   - empty string: redirect to project root
	//   - non-empty string: redirect to specified path
	var targetOverrides map[string]*string
	if v.IsSet("target_overrides") {
		rawOverrides := v.GetStringMap("target_overrides")
		if len(rawOverrides) > 0 {
			targetOverrides = make(map[string]*string)
			for key, val := range rawOverrides {
				if val == nil {
					// null value - suppress the target
					targetOverrides[key] = nil
				} else {
					// String value - redirect to path
					strVal, ok := val.(string)
					if ok {
						targetOverrides[key] = &strVal
					}
				}
			}
		}
	}

	// Read ignore patterns from config
	ignorePatterns := v.GetStringSlice("ignore")

	// Read variables from config
	var variables map[string]interface{}
	if v.IsSet("variables") {
		variables = v.GetStringMap("variables")
	}

	// Unmarshal the configuration into the ProjectConfig struct
	cfg := &domainconfig.ProjectConfig{
		Profiles:        profiles,
		InstallPrefix:   installPrefix,
		Variables:       variables,
		TargetOverrides: targetOverrides,
		Ignore:          ignorePatterns,
	}

	// Validate the loaded configuration
	if err := l.validate.Struct(cfg); err != nil {
		// Handle validation errors
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			// Return the first validation error with field-specific details
			for _, fieldErr := range validationErrors {
				return nil, &ConfigValidationError{
					Path:  configPath,
					Field: fieldErr.Field(),
					Tag:   fieldErr.Tag(),
					Value: fieldErr.Param(),
					Err:   err,
				}
			}
		}
		// Fallback for other validation errors
		return nil, &ConfigValidationError{
			Path: configPath,
			Err:  err,
		}
	}

	return cfg, nil
}
