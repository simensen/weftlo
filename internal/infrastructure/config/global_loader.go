// Package config provides infrastructure for loading and resolving configuration.
package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/viper"

	domainconfig "github.com/simensen/weftlo/internal/domain/config"
)

// GlobalConfigLoader handles loading of global configuration from the config directory.
// It uses viper to support YAML files with environment variable overrides.
// The filesystem can be injected for testing with in-memory filesystems.
type GlobalConfigLoader struct {
	// envPrefix is the prefix for environment variables (e.g., "AGENT_CONFIG").
	envPrefix string
	// fs is the filesystem used for reading configuration files.
	// This can be afero.OsFs for production or afero.MemMapFs for testing.
	fs afero.Fs
}

// NewGlobalConfigLoader creates a new GlobalConfigLoader with the specified filesystem.
// The filesystem parameter allows for dependency injection of real or in-memory filesystems.
// For production use, pass the result of fs.New() or afero.NewOsFs().
// For testing, pass afero.NewMemMapFs().
func NewGlobalConfigLoader(filesystem afero.Fs) *GlobalConfigLoader {
	return &GlobalConfigLoader{
		envPrefix: "AGENT_CONFIG",
		fs:        filesystem,
	}
}

// Load loads the global configuration from the specified config directory.
// The config directory should contain a config.yaml file.
//
// Environment variables override file values. For example:
//   - AGENT_CONFIG_DEFAULT_PROFILE overrides the default_profile field
//   - AGENT_CONFIG_INSTALL_PREFIX overrides the install_prefix field
//
// Returns a GlobalConfig with zero values if the config file is missing (no error).
// Returns an error only for parse errors or other issues.
func (l *GlobalConfigLoader) Load(configDir string) (*domainconfig.GlobalConfig, error) {
	v := viper.New()

	// Configure viper to use the injected filesystem
	v.SetFs(l.fs)

	// Configure viper for YAML file reading
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configDir)

	// Set up environment variable support with AGENT_CONFIG prefix
	v.SetEnvPrefix(l.envPrefix)
	v.AutomaticEnv()

	// Replace dots and dashes with underscores for environment variable mapping
	// This handles snake_case to SCREAMING_SNAKE_CASE conversion
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	// Attempt to read the config file
	if err := v.ReadInConfig(); err != nil {
		// If the config file is not found, return zero values (no error)
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Still check environment variables even when file is missing
			// Return empty maps/slices, not nil, for new fields
			cfg := &domainconfig.GlobalConfig{
				DefaultProfile:   v.GetString("default_profile"),
				InstallPrefix:    v.GetString("install_prefix"),
				Variables:        make(map[string]interface{}),
				TargetOverrides:  make(map[string]*string),
				ProfileOverrides: make(map[string]domainconfig.ProfileOverrideConfig),
				Ignore:           []string{},
			}
			return cfg, nil
		}

		// For permission errors or file doesn't exist at all, return zero values
		// Note: afero errors are compatible with os.IsNotExist() and os.IsPermission()
		if os.IsNotExist(err) || os.IsPermission(err) {
			cfg := &domainconfig.GlobalConfig{
				DefaultProfile:   v.GetString("default_profile"),
				InstallPrefix:    v.GetString("install_prefix"),
				Variables:        make(map[string]interface{}),
				TargetOverrides:  make(map[string]*string),
				ProfileOverrides: make(map[string]domainconfig.ProfileOverrideConfig),
				Ignore:           []string{},
			}
			return cfg, nil
		}

		// For parse errors, return a typed error
		configPath := filepath.Join(configDir, "config.yaml")
		return nil, &ConfigParseError{
			Path: configPath,
			Err:  err,
		}
	}

	// Load Variables field
	// Use Viper's map interface{} handling
	// Return empty map (not nil) when field missing
	variables := make(map[string]interface{})
	if v.IsSet("variables") {
		variables = v.GetStringMap("variables")
	}

	// Load TargetOverrides field
	// Map values can be:
	//   - null (nil pointer): suppress the target
	//   - empty string: redirect to project root
	//   - non-empty string: redirect to specified path
	targetOverrides := make(map[string]*string)
	if v.IsSet("target_overrides") {
		rawOverrides := v.GetStringMap("target_overrides")
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

	// Load ProfileOverrides field
	// Handle complex YAML: profile_overrides.<profile-name>.target_overrides
	// Return empty map (not nil) when field missing
	profileOverrides := make(map[string]domainconfig.ProfileOverrideConfig)
	if v.IsSet("profile_overrides") {
		rawProfileOverrides := v.GetStringMap("profile_overrides")
		for profileName, profileVal := range rawProfileOverrides {
			profileConfig := domainconfig.ProfileOverrideConfig{
				TargetOverrides: make(map[string]*string),
			}

			// profileVal should be a map containing target_overrides
			if profileMap, ok := profileVal.(map[string]interface{}); ok {
				if targetOverridesVal, exists := profileMap["target_overrides"]; exists {
					if targetOverridesMap, ok := targetOverridesVal.(map[string]interface{}); ok {
						for targetKey, targetVal := range targetOverridesMap {
							if targetVal == nil {
								// null value - suppress the target
								profileConfig.TargetOverrides[targetKey] = nil
							} else {
								// String value - redirect to path
								strVal, ok := targetVal.(string)
								if ok {
									profileConfig.TargetOverrides[targetKey] = &strVal
								}
							}
						}
					}
				}
			}

			profileOverrides[profileName] = profileConfig
		}
	}

	// Load Ignore field
	// Use Viper's slice handling
	// Return empty slice (not nil) when field missing
	ignore := v.GetStringSlice("ignore")
	if ignore == nil {
		ignore = []string{}
	}

	// Unmarshal the configuration into the GlobalConfig struct
	cfg := &domainconfig.GlobalConfig{
		DefaultProfile:   v.GetString("default_profile"),
		InstallPrefix:    v.GetString("install_prefix"),
		Variables:        variables,
		TargetOverrides:  targetOverrides,
		ProfileOverrides: profileOverrides,
		Ignore:           ignore,
	}

	return cfg, nil
}
