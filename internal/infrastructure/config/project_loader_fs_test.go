package config_test

import (
	"errors"
	"testing"

	"github.com/spf13/afero"

	infraconfig "github.com/simensen/weftlo/internal/infrastructure/config"
)

// Test 1: ProjectConfigLoader works with in-memory filesystem
func TestProjectConfigLoader_WithMemMapFs(t *testing.T) {
	t.Parallel()
	memFs := afero.NewMemMapFs()

	// Create project directory and config file
	projectDir := "/project"
	err := memFs.MkdirAll(projectDir, 0755)
	if err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	configContent := []byte("profiles:\n  - vendor/my-profile\n")
	err = afero.WriteFile(memFs, projectDir+"/.weftlo.yaml", configContent, 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Create loader with injected filesystem
	loader := infraconfig.NewProjectConfigLoader(memFs)
	cfg, err := loader.Load(projectDir)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(cfg.Profiles) == 0 || cfg.Profiles[0] != "vendor/my-profile" {
		t.Errorf("expected Profiles ['vendor/my-profile'], got %v", cfg.Profiles)
	}
}

// Test 2: Config file reading through injected filesystem
func TestProjectConfigLoader_ReadsConfigThroughInjectedFs(t *testing.T) {
	t.Parallel()
	memFs := afero.NewMemMapFs()

	// Create project directory with a different profile value
	projectDir := "/test-project"
	err := memFs.MkdirAll(projectDir, 0755)
	if err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	configContent := []byte("profiles:\n  - custom-vendor/custom-profile\n")
	err = afero.WriteFile(memFs, projectDir+"/.weftlo.yaml", configContent, 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	loader := infraconfig.NewProjectConfigLoader(memFs)
	cfg, err := loader.Load(projectDir)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify that the config was read from the in-memory filesystem
	if len(cfg.Profiles) == 0 || cfg.Profiles[0] != "custom-vendor/custom-profile" {
		t.Errorf("expected Profiles ['custom-vendor/custom-profile'], got %v", cfg.Profiles)
	}
}

// Test 3: Error handling with missing file returns correct error type
func TestProjectConfigLoader_MissingFileReturnsConfigNotFoundError(t *testing.T) {
	t.Parallel()
	memFs := afero.NewMemMapFs()

	// Create project directory without config file
	projectDir := "/empty-project"
	err := memFs.MkdirAll(projectDir, 0755)
	if err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	loader := infraconfig.NewProjectConfigLoader(memFs)
	_, err = loader.Load(projectDir)

	if err == nil {
		t.Fatal("expected error for missing config file, got nil")
	}

	// Verify it's a ConfigNotFoundError
	var notFoundErr *infraconfig.ConfigNotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Errorf("expected ConfigNotFoundError, got %T: %v", err, err)
	}
}
