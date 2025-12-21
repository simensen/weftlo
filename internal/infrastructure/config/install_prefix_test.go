package config_test

import (
	"testing"

	"github.com/spf13/afero"

	domainconfig "github.com/simensen/weftlo/internal/domain/config"
	infraconfig "github.com/simensen/weftlo/internal/infrastructure/config"
)

// Test 1: Load config with install_prefix field specified
func TestProjectConfigLoader_InstallPrefix_Specified(t *testing.T) {
	t.Parallel()
	memFs := afero.NewMemMapFs()

	// Create project directory and config file with custom install_prefix
	projectDir := "/project"
	err := memFs.MkdirAll(projectDir, 0755)
	if err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	configContent := []byte(`profile: vendor/my-profile
install_prefix: custom-prefix
`)
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

	if cfg.InstallPrefix != "custom-prefix" {
		t.Errorf("expected InstallPrefix 'custom-prefix', got '%s'", cfg.InstallPrefix)
	}
}

// Test 2: Load config without install_prefix field (defaults to "weftlo")
func TestProjectConfigLoader_InstallPrefix_NotSpecified_DefaultsToAgentConfig(t *testing.T) {
	t.Parallel()
	memFs := afero.NewMemMapFs()

	// Create project directory and config file without install_prefix
	projectDir := "/project"
	err := memFs.MkdirAll(projectDir, 0755)
	if err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	configContent := []byte(`profile: vendor/my-profile
`)
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

	if cfg.InstallPrefix != domainconfig.DefaultInstallPrefix {
		t.Errorf("expected InstallPrefix '%s', got '%s'", domainconfig.DefaultInstallPrefix, cfg.InstallPrefix)
	}
}

// Test 3: Load config with empty install_prefix (preserves empty - no prefix)
// When explicitly set to empty string, the install prefix should be empty (no prefix)
func TestProjectConfigLoader_InstallPrefix_Empty_PreservesEmpty(t *testing.T) {
	t.Parallel()
	memFs := afero.NewMemMapFs()

	// Create project directory and config file with empty install_prefix
	projectDir := "/project"
	err := memFs.MkdirAll(projectDir, 0755)
	if err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	configContent := []byte(`profile: vendor/my-profile
install_prefix: ""
`)
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

	// When explicitly set to empty, the install prefix should be empty (no prefix)
	if cfg.InstallPrefix != "" {
		t.Errorf("expected empty InstallPrefix when explicitly set to empty, got '%s'", cfg.InstallPrefix)
	}
}

// Test 4: Load config with multi-level path install_prefix
func TestProjectConfigLoader_InstallPrefix_MultiLevelPath(t *testing.T) {
	t.Parallel()
	memFs := afero.NewMemMapFs()

	// Create project directory and config file with multi-level install_prefix
	projectDir := "/project"
	err := memFs.MkdirAll(projectDir, 0755)
	if err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	configContent := []byte(`profile: vendor/my-profile
install_prefix: .claude/my-project
`)
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

	if cfg.InstallPrefix != ".claude/my-project" {
		t.Errorf("expected InstallPrefix '.claude/my-project', got '%s'", cfg.InstallPrefix)
	}
}
