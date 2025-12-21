package config_test

import (
	"testing"

	"github.com/spf13/afero"

	infraconfig "github.com/simensen/weftlo/internal/infrastructure/config"
)

// Test 1: GlobalConfigLoader works with in-memory filesystem
func TestGlobalConfigLoader_WithMemMapFs(t *testing.T) {
	t.Parallel()
	// Create an in-memory filesystem
	memFs := afero.NewMemMapFs()

	// Create config directory and file
	configDir := "/config"
	err := memFs.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	configContent := []byte("default_profile: vendor/test-profile\n")
	err = afero.WriteFile(memFs, configDir+"/config.yaml", configContent, 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Create loader with in-memory filesystem
	loader := infraconfig.NewGlobalConfigLoader(memFs)
	cfg, err := loader.Load(configDir)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if cfg.DefaultProfile != "vendor/test-profile" {
		t.Errorf("expected DefaultProfile 'vendor/test-profile', got '%s'", cfg.DefaultProfile)
	}
}

// Test 2: Config file reading goes through injected filesystem
func TestGlobalConfigLoader_ReadsFromInjectedFilesystem(t *testing.T) {
	t.Parallel()
	// Create an in-memory filesystem with specific content
	memFs := afero.NewMemMapFs()

	configDir := "/custom/config/path"
	err := memFs.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	// Write a config file with a specific profile value
	configContent := []byte("default_profile: vendor/injected-fs-profile\n")
	err = afero.WriteFile(memFs, configDir+"/config.yaml", configContent, 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Create loader with the in-memory filesystem
	loader := infraconfig.NewGlobalConfigLoader(memFs)
	cfg, err := loader.Load(configDir)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify the config was read from the in-memory filesystem
	if cfg.DefaultProfile != "vendor/injected-fs-profile" {
		t.Errorf("expected DefaultProfile 'vendor/injected-fs-profile', got '%s'", cfg.DefaultProfile)
	}
}

// Test 3: Missing file returns zero values without error (correct error handling)
func TestGlobalConfigLoader_MissingFileReturnsZeroValues(t *testing.T) {
	t.Parallel()
	// Create an empty in-memory filesystem (no config file)
	memFs := afero.NewMemMapFs()

	// Create only the directory, no config file
	configDir := "/empty/config"
	err := memFs.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	// Create loader with the in-memory filesystem
	loader := infraconfig.NewGlobalConfigLoader(memFs)
	cfg, err := loader.Load(configDir)

	// Global config should return zero values when file is missing, not an error
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

// Task Group 1 Tests: Configuration Updates for CLI Install Command

// Test 4 (Task 1.1c): GlobalConfigLoader reading install_prefix from YAML
func TestGlobalConfigLoader_ReadsInstallPrefix(t *testing.T) {
	t.Parallel()
	memFs := afero.NewMemMapFs()

	configDir := "/config"
	err := memFs.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	// Write a config file with both default_profile and install_prefix
	configContent := []byte(`default_profile: vendor/my-profile
install_prefix: .claude/global-prefix
`)
	err = afero.WriteFile(memFs, configDir+"/config.yaml", configContent, 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	loader := infraconfig.NewGlobalConfigLoader(memFs)
	cfg, err := loader.Load(configDir)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify default_profile is read correctly
	if cfg.DefaultProfile != "vendor/my-profile" {
		t.Errorf("expected DefaultProfile 'vendor/my-profile', got '%s'", cfg.DefaultProfile)
	}

	// Verify install_prefix is read correctly
	if cfg.InstallPrefix != ".claude/global-prefix" {
		t.Errorf("expected InstallPrefix '.claude/global-prefix', got '%s'", cfg.InstallPrefix)
	}
}

// Test 5: GlobalConfigLoader returns empty InstallPrefix when not specified in YAML
func TestGlobalConfigLoader_EmptyInstallPrefix(t *testing.T) {
	t.Parallel()
	memFs := afero.NewMemMapFs()

	configDir := "/config"
	err := memFs.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	// Write a config file with only default_profile (no install_prefix)
	configContent := []byte("default_profile: vendor/my-profile\n")
	err = afero.WriteFile(memFs, configDir+"/config.yaml", configContent, 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	loader := infraconfig.NewGlobalConfigLoader(memFs)
	cfg, err := loader.Load(configDir)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify install_prefix is empty when not specified
	if cfg.InstallPrefix != "" {
		t.Errorf("expected empty InstallPrefix when not specified, got '%s'", cfg.InstallPrefix)
	}
}
