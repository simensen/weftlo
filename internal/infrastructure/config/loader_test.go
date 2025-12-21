package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"

	domainconfig "github.com/simensen/weftlo/internal/domain/config"
	infraconfig "github.com/simensen/weftlo/internal/infrastructure/config"
)

// ============================================================================
// Test Helper Functions
// ============================================================================

// testConfig represents a test configuration file to be created in the filesystem
type testConfig struct {
	path    string
	content string
}

// setupTestFs creates an in-memory filesystem and populates it with the provided
// test configuration files. This helper enables fast, isolated tests without
// touching the real filesystem.
//
// Usage:
//
//	memFs := setupTestFs(t,
//	    testConfig{path: "/config/config.yaml", content: "default_profile: vendor/test"},
//	    testConfig{path: "/project/.weftlo.yaml", content: "profile: vendor/proj"},
//	)
func setupTestFs(t *testing.T, configs ...testConfig) afero.Fs {
	t.Helper()
	memFs := afero.NewMemMapFs()

	for _, cfg := range configs {
		// Create parent directory
		dir := filepath.Dir(cfg.path)
		if err := memFs.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}

		// Write the config file
		if err := afero.WriteFile(memFs, cfg.path, []byte(cfg.content), 0644); err != nil {
			t.Fatalf("failed to write config file %s: %v", cfg.path, err)
		}
	}

	return memFs
}

// setupEmptyTestFs creates an in-memory filesystem with only the specified
// directories created (no config files). Useful for testing missing file scenarios.
func setupEmptyTestFs(t *testing.T, dirs ...string) afero.Fs {
	t.Helper()
	memFs := afero.NewMemMapFs()

	for _, dir := range dirs {
		if err := memFs.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
	}

	return memFs
}

// ============================================================================
// Test Suite: GlobalConfigLoader
// ============================================================================

// Test 1: LoadGlobalConfig with valid config file
func TestLoadGlobalConfig_ValidConfigFile(t *testing.T) {
	t.Parallel()
	configDir := "/test/config"
	configContent := `default_profile: vendor/my-profile
`
	memFs := setupTestFs(t, testConfig{
		path:    filepath.Join(configDir, "config.yaml"),
		content: configContent,
	})

	loader := infraconfig.NewGlobalConfigLoader(memFs)
	cfg, err := loader.Load(configDir)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if cfg.DefaultProfile != "vendor/my-profile" {
		t.Errorf("expected DefaultProfile 'vendor/my-profile', got '%s'", cfg.DefaultProfile)
	}
}

// Test 2: LoadGlobalConfig with missing file (returns zero values, no error)
func TestLoadGlobalConfig_MissingFile(t *testing.T) {
	t.Parallel()
	configDir := "/test/config"
	memFs := setupEmptyTestFs(t, configDir)

	loader := infraconfig.NewGlobalConfigLoader(memFs)
	cfg, err := loader.Load(configDir)

	if err != nil {
		t.Fatalf("expected no error for missing global config, got: %v", err)
	}

	if cfg == nil {
		t.Fatal("expected non-nil config, got nil")
	}

	// Should return zero values
	if cfg.DefaultProfile != "" {
		t.Errorf("expected empty DefaultProfile, got '%s'", cfg.DefaultProfile)
	}
}

// Test 3: LoadProjectConfig with valid config file
func TestLoadProjectConfig_ValidConfigFile(t *testing.T) {
	t.Parallel()
	projectDir := "/test/project"
	configContent := `profiles:
  - vendor/my-project-profile
`
	memFs := setupTestFs(t, testConfig{
		path:    filepath.Join(projectDir, ".weftlo.yaml"),
		content: configContent,
	})

	loader := infraconfig.NewProjectConfigLoader(memFs)
	cfg, err := loader.Load(projectDir)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(cfg.Profiles) == 0 || cfg.Profiles[0] != "vendor/my-project-profile" {
		t.Errorf("expected Profiles ['vendor/my-project-profile'], got %v", cfg.Profiles)
	}
}

// Test 4: LoadProjectConfig with missing file (returns error)
func TestLoadProjectConfig_MissingFile(t *testing.T) {
	t.Parallel()
	projectDir := "/test/project"
	memFs := setupEmptyTestFs(t, projectDir)

	loader := infraconfig.NewProjectConfigLoader(memFs)
	_, err := loader.Load(projectDir)

	if err == nil {
		t.Fatal("expected error for missing project config, got nil")
	}

	// Verify it's a ConfigNotFoundError
	var notFoundErr *infraconfig.ConfigNotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Errorf("expected ConfigNotFoundError, got %T: %v", err, err)
	}
}

// Test 5: Environment variable override for default_profile
func TestLoadGlobalConfig_EnvVarOverride(t *testing.T) {
	configDir := "/test/config"
	configContent := `default_profile: vendor/file-profile
`
	memFs := setupTestFs(t, testConfig{
		path:    filepath.Join(configDir, "config.yaml"),
		content: configContent,
	})

	// Set environment variable to override the file value
	os.Setenv("AGENT_CONFIG_DEFAULT_PROFILE", "vendor/env-profile")
	defer os.Unsetenv("AGENT_CONFIG_DEFAULT_PROFILE")

	loader := infraconfig.NewGlobalConfigLoader(memFs)
	cfg, err := loader.Load(configDir)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Environment variable should take precedence
	if cfg.DefaultProfile != "vendor/env-profile" {
		t.Errorf("expected DefaultProfile 'vendor/env-profile' (from env var), got '%s'", cfg.DefaultProfile)
	}
}

// Test 6: Invalid YAML syntax returns parsing error
func TestLoadProjectConfig_InvalidYAML(t *testing.T) {
	t.Parallel()
	projectDir := "/test/project"
	invalidContent := `profile: [invalid yaml
  this is not valid:
`
	memFs := setupTestFs(t, testConfig{
		path:    filepath.Join(projectDir, ".weftlo.yaml"),
		content: invalidContent,
	})

	loader := infraconfig.NewProjectConfigLoader(memFs)
	_, err := loader.Load(projectDir)

	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}

	// Verify it's a ConfigParseError
	var parseErr *infraconfig.ConfigParseError
	if !errors.As(err, &parseErr) {
		t.Errorf("expected ConfigParseError, got %T: %v", err, err)
	}
}

// Test 7: Validation failure returns field-specific error
func TestLoadProjectConfig_ValidationFailure(t *testing.T) {
	t.Parallel()
	projectDir := "/test/project"
	configContent := "profiles:\n  - invalid-profile-no-slash\n"
	memFs := setupTestFs(t, testConfig{
		path:    filepath.Join(projectDir, ".weftlo.yaml"),
		content: configContent,
	})

	loader := infraconfig.NewProjectConfigLoader(memFs)
	_, err := loader.Load(projectDir)

	if err == nil {
		t.Fatal("expected error for validation failure, got nil")
	}

	// Verify it's a ConfigValidationError
	var validationErr *infraconfig.ConfigValidationError
	if !errors.As(err, &validationErr) {
		t.Errorf("expected ConfigValidationError, got %T: %v", err, err)
	}

	// Verify field-specific information
	if validationErr.Field != "Profiles[0]" {
		t.Errorf("expected Field 'Profiles[0]', got '%s'", validationErr.Field)
	}
}

// Test 8: Viper precedence (file < environment variables)
func TestLoadGlobalConfig_Precedence(t *testing.T) {
	// Test that env vars override file values even when file doesn't exist
	configDir := "/test/config"
	memFs := setupEmptyTestFs(t, configDir)

	// Set environment variable
	os.Setenv("AGENT_CONFIG_DEFAULT_PROFILE", "vendor/env-only-profile")
	defer os.Unsetenv("AGENT_CONFIG_DEFAULT_PROFILE")

	loader := infraconfig.NewGlobalConfigLoader(memFs)
	cfg, err := loader.Load(configDir)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Environment variable should be picked up even without a file
	if cfg.DefaultProfile != "vendor/env-only-profile" {
		t.Errorf("expected DefaultProfile 'vendor/env-only-profile' (from env var), got '%s'", cfg.DefaultProfile)
	}
}

// Test 9 (Gap Fill): DefaultConfigLoader interface works correctly
func TestDefaultConfigLoader_Interface(t *testing.T) {
	t.Parallel()
	globalDir := "/test/global"
	projectDir := "/test/project"

	memFs := setupTestFs(t,
		testConfig{
			path:    filepath.Join(globalDir, "config.yaml"),
			content: "default_profile: vendor/global-default\n",
		},
		testConfig{
			path:    filepath.Join(projectDir, ".weftlo.yaml"),
			content: "profiles:\n  - vendor/project-specific\n",
		},
	)

	// Use the DefaultConfigLoader via the ConfigLoader interface
	var loader infraconfig.ConfigLoader = infraconfig.NewConfigLoaderWithFs(memFs)

	// Load global config
	globalCfg, err := loader.LoadGlobalConfig(globalDir)
	if err != nil {
		t.Fatalf("expected no error loading global config, got: %v", err)
	}
	if globalCfg.DefaultProfile != "vendor/global-default" {
		t.Errorf("expected global DefaultProfile 'vendor/global-default', got '%s'", globalCfg.DefaultProfile)
	}

	// Load project config
	projectCfg, err := loader.LoadProjectConfig(projectDir)
	if err != nil {
		t.Fatalf("expected no error loading project config, got: %v", err)
	}
	if len(projectCfg.Profiles) == 0 || projectCfg.Profiles[0] != "vendor/project-specific" {
		t.Errorf("expected project Profiles ['vendor/project-specific'], got %v", projectCfg.Profiles)
	}
}

// Test 10 (Updated): Empty profile in project config passes validation (omitempty allows fallback)
// Note: This behavior was changed to support profile resolution chain fallback to global default
func TestLoadProjectConfig_EmptyProfile(t *testing.T) {
	t.Parallel()
	projectDir := "/test/project"
	configContent := "profiles: []\n"
	memFs := setupTestFs(t, testConfig{
		path:    filepath.Join(projectDir, ".weftlo.yaml"),
		content: configContent,
	})

	loader := infraconfig.NewProjectConfigLoader(memFs)
	cfg, err := loader.Load(projectDir)

	// Empty profile should now pass validation (omitempty allows empty string)
	// This enables profile resolution to fall back to global default_profile
	if err != nil {
		t.Fatalf("expected no error for empty profile (omitempty allows fallback), got: %v", err)
	}

	// Verify the profile is empty as expected
	if len(cfg.Profiles) != 0 {
		t.Errorf("expected empty Profiles, got %v", cfg.Profiles)
	}

	// Verify default install prefix is applied
	if cfg.InstallPrefix != domainconfig.DefaultInstallPrefix {
		t.Errorf("expected InstallPrefix '%s', got '%s'", domainconfig.DefaultInstallPrefix, cfg.InstallPrefix)
	}
}

// Test 11 (Gap Fill): Combined global and project config workflow
func TestConfigLoader_CombinedWorkflow(t *testing.T) {
	t.Parallel()
	globalDir := "/test/global"
	projectDir := "/test/project"

	memFs := setupTestFs(t,
		testConfig{
			path:    filepath.Join(globalDir, "config.yaml"),
			content: "default_profile: vendor/fallback-profile\n",
		},
		testConfig{
			path:    filepath.Join(projectDir, ".weftlo.yaml"),
			content: "profiles:\n  - vendor/project-profile\n",
		},
	)

	loader := infraconfig.NewConfigLoaderWithFs(memFs)

	// Load both configs
	globalCfg, err := loader.LoadGlobalConfig(globalDir)
	if err != nil {
		t.Fatalf("failed to load global config: %v", err)
	}

	projectCfg, err := loader.LoadProjectConfig(projectDir)
	if err != nil {
		t.Fatalf("failed to load project config: %v", err)
	}

	// Verify project config takes precedence over global default
	effectiveProfile := ""
	if len(projectCfg.Profiles) > 0 {
		effectiveProfile = projectCfg.Profiles[0]
	}
	if effectiveProfile == "" {
		effectiveProfile = globalCfg.DefaultProfile
	}

	if effectiveProfile != "vendor/project-profile" {
		t.Errorf("expected effective profile 'vendor/project-profile', got '%s'", effectiveProfile)
	}
}

// Test 12 (Gap Fill): Error messages contain useful context
func TestConfigErrors_ContainContext(t *testing.T) {
	t.Parallel()
	projectDir := "/test/project"
	expectedPath := filepath.Join(projectDir, ".weftlo.yaml")

	// First test: ConfigNotFoundError contains path
	memFs := setupEmptyTestFs(t, projectDir)
	loader := infraconfig.NewProjectConfigLoader(memFs)
	_, err := loader.Load(projectDir)

	if err == nil {
		t.Fatal("expected error for missing config")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, expectedPath) {
		t.Errorf("ConfigNotFoundError should contain path '%s', got: %s", expectedPath, errMsg)
	}

	// Second test: ConfigValidationError contains field and path
	configContent := "profiles:\n  - invalid\n"
	err = afero.WriteFile(memFs, expectedPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	_, err = loader.Load(projectDir)
	if err == nil {
		t.Fatal("expected validation error")
	}

	errMsg = err.Error()
	if !strings.Contains(errMsg, expectedPath) {
		t.Errorf("ConfigValidationError should contain path '%s', got: %s", expectedPath, errMsg)
	}
	if !strings.Contains(errMsg, "Profiles") {
		t.Errorf("ConfigValidationError should contain field 'Profiles', got: %s", errMsg)
	}
}

// ============================================================================
// Task Group 4: DefaultConfigLoader Filesystem Coordination Tests
// ============================================================================

// Test 13 (Task Group 4): DefaultConfigLoader passes filesystem to sub-loaders
func TestNewConfigLoaderWithFs_PassesFilesystemToSubLoaders(t *testing.T) {
	t.Parallel()
	globalDir := "/test/global"
	projectDir := "/test/project"

	memFs := setupTestFs(t,
		testConfig{
			path:    filepath.Join(globalDir, "config.yaml"),
			content: "default_profile: vendor/memfs-global\n",
		},
		testConfig{
			path:    filepath.Join(projectDir, ".weftlo.yaml"),
			content: "profiles:\n  - vendor/memfs-project\n",
		},
	)

	// Create loader with injected filesystem
	loader := infraconfig.NewConfigLoaderWithFs(memFs)

	// Verify global config is loaded from in-memory filesystem
	globalCfg, err := loader.LoadGlobalConfig(globalDir)
	if err != nil {
		t.Fatalf("expected no error loading global config, got: %v", err)
	}
	if globalCfg.DefaultProfile != "vendor/memfs-global" {
		t.Errorf("expected DefaultProfile 'vendor/memfs-global', got '%s'", globalCfg.DefaultProfile)
	}

	// Verify project config is loaded from in-memory filesystem
	projectCfg, err := loader.LoadProjectConfig(projectDir)
	if err != nil {
		t.Fatalf("expected no error loading project config, got: %v", err)
	}
	if len(projectCfg.Profiles) == 0 || projectCfg.Profiles[0] != "vendor/memfs-project" {
		t.Errorf("expected Profiles ['vendor/memfs-project'], got %v", projectCfg.Profiles)
	}
}

// Test 14 (Task Group 4): End-to-end config loading with in-memory filesystem
func TestNewConfigLoaderWithFs_EndToEndWithMemFs(t *testing.T) {
	t.Parallel()
	configDir := "/home/testuser/.config/weftlo"
	projectDir := "/home/testuser/projects/my-project"

	memFs := setupTestFs(t,
		testConfig{
			path:    filepath.Join(configDir, "config.yaml"),
			content: "default_profile: vendor/default-profile\n",
		},
		testConfig{
			path:    filepath.Join(projectDir, ".weftlo.yaml"),
			content: "profiles:\n  - vendor/custom-project-profile\n",
		},
	)

	// Create loader with in-memory filesystem
	loader := infraconfig.NewConfigLoaderWithFs(memFs)

	// Load global config
	globalCfg, err := loader.LoadGlobalConfig(configDir)
	if err != nil {
		t.Fatalf("failed to load global config: %v", err)
	}

	// Load project config
	projectCfg, err := loader.LoadProjectConfig(projectDir)
	if err != nil {
		t.Fatalf("failed to load project config: %v", err)
	}

	// Verify the resolution workflow
	if globalCfg.DefaultProfile != "vendor/default-profile" {
		t.Errorf("expected global default profile 'vendor/default-profile', got '%s'", globalCfg.DefaultProfile)
	}

	if len(projectCfg.Profiles) == 0 || projectCfg.Profiles[0] != "vendor/custom-project-profile" {
		t.Errorf("expected project profile 'vendor/custom-project-profile', got '%s'", projectCfg.Profiles[0])
	}

	// Simulate profile resolution: project overrides global
	effectiveProfile := ""
	if len(projectCfg.Profiles) > 0 {
		effectiveProfile = projectCfg.Profiles[0]
	}
	if effectiveProfile == "" {
		effectiveProfile = globalCfg.DefaultProfile
	}

	if effectiveProfile != "vendor/custom-project-profile" {
		t.Errorf("expected effective profile 'vendor/custom-project-profile', got '%s'", effectiveProfile)
	}
}

// Test 15 (Task Group 4): ConfigLoader interface contract remains unchanged
func TestNewConfigLoaderWithFs_InterfaceContractUnchanged(t *testing.T) {
	t.Parallel()
	projectDir := "/test/project"

	memFs := setupTestFs(t, testConfig{
		path:    filepath.Join(projectDir, ".weftlo.yaml"),
		content: "profiles:\n  - vendor/interface-test\n",
	})

	// Verify that NewConfigLoaderWithFs returns a type that satisfies ConfigLoader interface
	var loader infraconfig.ConfigLoader = infraconfig.NewConfigLoaderWithFs(memFs)

	// Verify interface methods work correctly
	// LoadGlobalConfig should work (returns zero values for missing file)
	globalCfg, err := loader.LoadGlobalConfig("/nonexistent")
	if err != nil {
		t.Fatalf("expected no error for missing global config, got: %v", err)
	}
	if globalCfg == nil {
		t.Fatal("expected non-nil global config, got nil")
	}

	// LoadProjectConfig should work
	projectCfg, err := loader.LoadProjectConfig(projectDir)
	if err != nil {
		t.Fatalf("expected no error loading project config, got: %v", err)
	}
	if len(projectCfg.Profiles) == 0 || projectCfg.Profiles[0] != "vendor/interface-test" {
		t.Errorf("expected Profiles ['vendor/interface-test'], got %v", projectCfg.Profiles)
	}

	// Verify the original NewConfigLoader() still works (backward compatibility)
	var defaultLoader infraconfig.ConfigLoader = infraconfig.NewConfigLoader()
	if defaultLoader == nil {
		t.Fatal("NewConfigLoader() should return a valid loader")
	}
}

// ============================================================================
// Task Group 6: Integration Verification Gap Fill Tests
// ============================================================================

// Test 16 (Gap Fill): Empty config file in GlobalConfig returns zero values
func TestLoadGlobalConfig_EmptyFile(t *testing.T) {
	t.Parallel()
	configDir := "/test/config"
	// Empty file (just whitespace or truly empty)
	configContent := ``

	memFs := setupTestFs(t, testConfig{
		path:    filepath.Join(configDir, "config.yaml"),
		content: configContent,
	})

	loader := infraconfig.NewGlobalConfigLoader(memFs)
	cfg, err := loader.Load(configDir)

	// Empty file should not cause an error for global config
	if err != nil {
		t.Fatalf("expected no error for empty global config file, got: %v", err)
	}

	if cfg == nil {
		t.Fatal("expected non-nil config, got nil")
	}

	// Should return zero values since file has no content
	if cfg.DefaultProfile != "" {
		t.Errorf("expected empty DefaultProfile for empty file, got '%s'", cfg.DefaultProfile)
	}
}

// Test 17 (Updated): Empty config file in ProjectConfig passes validation (omitempty allows fallback)
// Note: This behavior was changed to support profile resolution chain fallback to global default
func TestLoadProjectConfig_EmptyFile(t *testing.T) {
	t.Parallel()
	projectDir := "/test/project"
	// Empty file (no profile field)
	configContent := ``

	memFs := setupTestFs(t, testConfig{
		path:    filepath.Join(projectDir, ".weftlo.yaml"),
		content: configContent,
	})

	loader := infraconfig.NewProjectConfigLoader(memFs)
	cfg, err := loader.Load(projectDir)

	// Empty file should now pass validation (omitempty allows empty profile)
	// This enables profile resolution to fall back to global default_profile
	if err != nil {
		t.Fatalf("expected no error for empty project config file (omitempty allows fallback), got: %v", err)
	}

	// Verify the profile is empty as expected
	if len(cfg.Profiles) != 0 {
		t.Errorf("expected empty Profiles, got %v", cfg.Profiles)
	}

	// Verify default install prefix is applied
	if cfg.InstallPrefix != domainconfig.DefaultInstallPrefix {
		t.Errorf("expected InstallPrefix '%s', got '%s'", domainconfig.DefaultInstallPrefix, cfg.InstallPrefix)
	}
}

// Test 18 (Gap Fill): Invalid YAML in GlobalConfig returns ConfigParseError
func TestLoadGlobalConfig_InvalidYAML(t *testing.T) {
	t.Parallel()
	configDir := "/test/config"
	invalidContent := `default_profile: [invalid yaml
  this is not valid:
`
	memFs := setupTestFs(t, testConfig{
		path:    filepath.Join(configDir, "config.yaml"),
		content: invalidContent,
	})

	loader := infraconfig.NewGlobalConfigLoader(memFs)
	_, err := loader.Load(configDir)

	if err == nil {
		t.Fatal("expected error for invalid YAML in global config, got nil")
	}

	// Verify it's a ConfigParseError
	var parseErr *infraconfig.ConfigParseError
	if !errors.As(err, &parseErr) {
		t.Errorf("expected ConfigParseError, got %T: %v", err, err)
	}

	// Verify error contains file path context
	if !strings.Contains(parseErr.Path, "config.yaml") {
		t.Errorf("ConfigParseError should contain path, got: %s", parseErr.Path)
	}
}

// Test 19 (Gap Fill): Deeply nested directory paths work correctly
func TestConfigLoader_DeeplyNestedPaths(t *testing.T) {
	t.Parallel()
	globalDir := "/very/deeply/nested/global/config/directory"
	projectDir := "/another/very/deeply/nested/project/directory"

	memFs := setupTestFs(t,
		testConfig{
			path:    filepath.Join(globalDir, "config.yaml"),
			content: "default_profile: vendor/nested-global\n",
		},
		testConfig{
			path:    filepath.Join(projectDir, ".weftlo.yaml"),
			content: "profiles:\n  - vendor/nested-project\n",
		},
	)

	loader := infraconfig.NewConfigLoaderWithFs(memFs)

	// Verify global config loads from deeply nested path
	globalCfg, err := loader.LoadGlobalConfig(globalDir)
	if err != nil {
		t.Fatalf("expected no error loading global config from nested path, got: %v", err)
	}
	if globalCfg.DefaultProfile != "vendor/nested-global" {
		t.Errorf("expected DefaultProfile 'vendor/nested-global', got '%s'", globalCfg.DefaultProfile)
	}

	// Verify project config loads from deeply nested path
	projectCfg, err := loader.LoadProjectConfig(projectDir)
	if err != nil {
		t.Fatalf("expected no error loading project config from nested path, got: %v", err)
	}
	if len(projectCfg.Profiles) == 0 || projectCfg.Profiles[0] != "vendor/nested-project" {
		t.Errorf("expected Profiles ['vendor/nested-project'], got %v", projectCfg.Profiles)
	}
}

// Test 20 (Gap Fill): Multiple loaders with same filesystem remain isolated
func TestConfigLoader_MultipleLoadersIsolation(t *testing.T) {
	t.Parallel()
	globalDir := "/test/global"
	projectDir := "/test/project"

	memFs := setupTestFs(t,
		testConfig{
			path:    filepath.Join(globalDir, "config.yaml"),
			content: "default_profile: vendor/shared-fs-global\n",
		},
		testConfig{
			path:    filepath.Join(projectDir, ".weftlo.yaml"),
			content: "profiles:\n  - vendor/shared-fs-project\n",
		},
	)

	// Create two separate loaders with the same filesystem
	loader1 := infraconfig.NewConfigLoaderWithFs(memFs)
	loader2 := infraconfig.NewConfigLoaderWithFs(memFs)

	// Both loaders should be able to load the same configs
	cfg1, err := loader1.LoadProjectConfig(projectDir)
	if err != nil {
		t.Fatalf("loader1 failed to load project config: %v", err)
	}

	cfg2, err := loader2.LoadProjectConfig(projectDir)
	if err != nil {
		t.Fatalf("loader2 failed to load project config: %v", err)
	}

	// Verify both got the same data
	if cfg1.Profiles[0] != cfg2.Profiles[0] {
		t.Errorf("expected same profile from both loaders, got '%s' and '%s'", cfg1.Profiles[0], cfg2.Profiles[0])
	}

	if cfg1.Profiles[0] != "vendor/shared-fs-project" {
		t.Errorf("expected Profile 'vendor/shared-fs-project', got '%s'", cfg1.Profiles[0])
	}
}

// ============================================================================
// Task Group 1: Profile Resolution Chain Support Tests
// ============================================================================

// Test 21 (Task Group 1): Profile resolution chain - empty project profile falls back to global
func TestConfigLoader_ProfileResolutionChain_FallbackToGlobal(t *testing.T) {
	t.Parallel()
	globalDir := "/test/global"
	projectDir := "/test/project"

	memFs := setupTestFs(t,
		testConfig{
			path:    filepath.Join(globalDir, "config.yaml"),
			content: "default_profile: vendor/global-fallback\n",
		},
		testConfig{
			path:    filepath.Join(projectDir, ".weftlo.yaml"),
			content: "install_prefix: .custom/path\n", // No profile specified
		},
	)

	loader := infraconfig.NewConfigLoaderWithFs(memFs)

	// Load both configs
	globalCfg, err := loader.LoadGlobalConfig(globalDir)
	if err != nil {
		t.Fatalf("failed to load global config: %v", err)
	}

	projectCfg, err := loader.LoadProjectConfig(projectDir)
	if err != nil {
		t.Fatalf("failed to load project config: %v", err)
	}

	// Verify project profile is empty
	if len(projectCfg.Profiles) != 0 {
		t.Errorf("expected empty Profiles in project config, got %v", projectCfg.Profiles)
	}

	// Simulate profile resolution: empty project profile falls back to global
	effectiveProfile := ""
	if len(projectCfg.Profiles) > 0 {
		effectiveProfile = projectCfg.Profiles[0]
	}
	if effectiveProfile == "" {
		effectiveProfile = globalCfg.DefaultProfile
	}

	if effectiveProfile != "vendor/global-fallback" {
		t.Errorf("expected effective profile 'vendor/global-fallback' (from global), got '%s'", effectiveProfile)
	}

	// Verify the custom install prefix was still loaded
	if projectCfg.InstallPrefix != ".custom/path" {
		t.Errorf("expected InstallPrefix '.custom/path', got '%s'", projectCfg.InstallPrefix)
	}
}
