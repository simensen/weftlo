// Package config provides infrastructure for loading and resolving configuration.
package config

import (
	"github.com/spf13/afero"

	domainconfig "github.com/simensen/weftlo/internal/domain/config"
)

// ConfigLoader defines the interface for loading configuration files.
// This interface enables dependency injection for testing and allows
// different implementations for various configuration sources.
type ConfigLoader interface {
	// LoadGlobalConfig loads the global configuration from the specified config directory.
	// The config directory should contain a config.yaml file.
	// Returns a GlobalConfig with zero values if the config file is missing (no error).
	// Returns an error only for parse errors or other issues.
	LoadGlobalConfig(configDir string) (*domainconfig.GlobalConfig, error)

	// LoadProjectConfig loads the project configuration from the specified project root.
	// The project root should contain a .weftlo.yaml file.
	// Returns a ConfigNotFoundError if the config file is missing.
	// Returns a ConfigParseError for invalid YAML syntax.
	// Returns a ConfigValidationError if validation fails.
	LoadProjectConfig(projectRoot string) (*domainconfig.ProjectConfig, error)
}

// DefaultConfigLoader is the standard implementation of ConfigLoader
// that uses viper to load configuration from YAML files with environment variable support.
type DefaultConfigLoader struct {
	globalLoader  *GlobalConfigLoader
	projectLoader *ProjectConfigLoader
}

// NewConfigLoader creates a new DefaultConfigLoader with default settings.
// It uses the real OS filesystem for production use.
// For testing with an in-memory filesystem, use NewConfigLoaderWithFs instead.
func NewConfigLoader() *DefaultConfigLoader {
	return NewConfigLoaderWithFs(afero.NewOsFs())
}

// NewConfigLoaderWithFs creates a new DefaultConfigLoader with the specified filesystem.
// The filesystem parameter allows for dependency injection, enabling tests to use
// an in-memory filesystem (afero.NewMemMapFs()) instead of the real filesystem.
// Both the global and project loaders receive the same filesystem instance.
func NewConfigLoaderWithFs(fs afero.Fs) *DefaultConfigLoader {
	return &DefaultConfigLoader{
		globalLoader:  NewGlobalConfigLoader(fs),
		projectLoader: NewProjectConfigLoader(fs),
	}
}

// LoadGlobalConfig loads the global configuration from the specified config directory.
func (l *DefaultConfigLoader) LoadGlobalConfig(configDir string) (*domainconfig.GlobalConfig, error) {
	return l.globalLoader.Load(configDir)
}

// LoadProjectConfig loads the project configuration from the specified project root.
func (l *DefaultConfigLoader) LoadProjectConfig(projectRoot string) (*domainconfig.ProjectConfig, error) {
	return l.projectLoader.Load(projectRoot)
}
