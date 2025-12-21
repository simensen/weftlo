package config_test

import (
	"testing"

	"github.com/spf13/afero"

	infraconfig "github.com/simensen/weftlo/internal/infrastructure/config"
)

// Task Group 2 Tests: GlobalConfigLoader Enhancement for Global Target Overrides

// Test 2.1a: GlobalConfigLoader loads TargetOverrides from config.yaml
func TestGlobalConfigLoader_LoadsTargetOverrides(t *testing.T) {
	t.Parallel()
	memFs := afero.NewMemMapFs()

	configDir := "/config"
	err := memFs.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	configContent := []byte(`default_profile: vendor/my-profile
target_overrides:
  agents: .claude/agents/custom
  commands: .claude/commands/custom
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

	// Verify target_overrides is loaded
	if cfg.TargetOverrides == nil {
		t.Fatal("expected TargetOverrides to be non-nil")
	}

	if len(cfg.TargetOverrides) != 2 {
		t.Errorf("expected 2 target overrides, got %d", len(cfg.TargetOverrides))
	}

	// Verify agents override
	agentsOverride, ok := cfg.TargetOverrides["agents"]
	if !ok {
		t.Error("expected 'agents' key in TargetOverrides")
	} else if agentsOverride == nil {
		t.Error("expected 'agents' value to be non-nil pointer")
	} else if *agentsOverride != ".claude/agents/custom" {
		t.Errorf("expected 'agents' value '.claude/agents/custom', got '%s'", *agentsOverride)
	}

	// Verify commands override
	commandsOverride, ok := cfg.TargetOverrides["commands"]
	if !ok {
		t.Error("expected 'commands' key in TargetOverrides")
	} else if commandsOverride == nil {
		t.Error("expected 'commands' value to be non-nil pointer")
	} else if *commandsOverride != ".claude/commands/custom" {
		t.Errorf("expected 'commands' value '.claude/commands/custom', got '%s'", *commandsOverride)
	}
}

// Test 2.1b: GlobalConfigLoader loads ProfileOverrides with nested structure
func TestGlobalConfigLoader_LoadsProfileOverrides(t *testing.T) {
	t.Parallel()
	memFs := afero.NewMemMapFs()

	configDir := "/config"
	err := memFs.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	configContent := []byte(`default_profile: vendor/my-profile
profile_overrides:
  simensen/git-profile:
    target_overrides:
      agents: .claude/agents/simensen
  other-vendor/other-profile:
    target_overrides:
      standards: .project-standards
      skills: .custom-skills
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

	// Verify profile_overrides is loaded
	if cfg.ProfileOverrides == nil {
		t.Fatal("expected ProfileOverrides to be non-nil")
	}

	if len(cfg.ProfileOverrides) != 2 {
		t.Errorf("expected 2 profile overrides, got %d", len(cfg.ProfileOverrides))
	}

	// Verify simensen/git-profile override
	gitProfile, ok := cfg.ProfileOverrides["simensen/git-profile"]
	if !ok {
		t.Error("expected 'simensen/git-profile' key in ProfileOverrides")
	} else {
		if gitProfile.TargetOverrides == nil {
			t.Error("expected TargetOverrides in git-profile to be non-nil")
		} else if len(gitProfile.TargetOverrides) != 1 {
			t.Errorf("expected 1 target override in git-profile, got %d", len(gitProfile.TargetOverrides))
		} else {
			agentsOverride, ok := gitProfile.TargetOverrides["agents"]
			if !ok {
				t.Error("expected 'agents' key in git-profile TargetOverrides")
			} else if agentsOverride == nil || *agentsOverride != ".claude/agents/simensen" {
				t.Errorf("expected 'agents' value '.claude/agents/simensen', got %v", agentsOverride)
			}
		}
	}

	// Verify other-vendor/other-profile override
	otherProfile, ok := cfg.ProfileOverrides["other-vendor/other-profile"]
	if !ok {
		t.Error("expected 'other-vendor/other-profile' key in ProfileOverrides")
	} else {
		if otherProfile.TargetOverrides == nil {
			t.Error("expected TargetOverrides in other-profile to be non-nil")
		} else if len(otherProfile.TargetOverrides) != 2 {
			t.Errorf("expected 2 target overrides in other-profile, got %d", len(otherProfile.TargetOverrides))
		}
	}
}

// Test 2.1c: GlobalConfigLoader loads Ignore patterns from config.yaml
func TestGlobalConfigLoader_LoadsIgnorePatterns(t *testing.T) {
	t.Parallel()
	memFs := afero.NewMemMapFs()

	configDir := "/config"
	err := memFs.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	configContent := []byte(`default_profile: vendor/my-profile
ignore:
  - "**/*.bak"
  - "**/temp/**"
  - "*.log"
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

	// Verify ignore is loaded
	if cfg.Ignore == nil {
		t.Fatal("expected Ignore to be non-nil")
	}

	if len(cfg.Ignore) != 3 {
		t.Errorf("expected 3 ignore patterns, got %d", len(cfg.Ignore))
	}

	expectedPatterns := []string{"**/*.bak", "**/temp/**", "*.log"}
	for i, pattern := range expectedPatterns {
		if i >= len(cfg.Ignore) {
			t.Errorf("missing ignore pattern at index %d", i)
			continue
		}
		if cfg.Ignore[i] != pattern {
			t.Errorf("expected ignore pattern '%s' at index %d, got '%s'", pattern, i, cfg.Ignore[i])
		}
	}
}

// Test 2.1d: GlobalConfigLoader handles null YAML values for suppression semantics
func TestGlobalConfigLoader_NullValuesSuppression(t *testing.T) {
	t.Parallel()
	memFs := afero.NewMemMapFs()

	configDir := "/config"
	err := memFs.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	configContent := []byte(`default_profile: vendor/my-profile
target_overrides:
  agents: .claude/agents/custom
  cursor: null
  prompts: ~
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

	// Verify target_overrides is loaded
	if cfg.TargetOverrides == nil {
		t.Fatal("expected TargetOverrides to be non-nil")
	}

	if len(cfg.TargetOverrides) != 3 {
		t.Errorf("expected 3 target overrides, got %d", len(cfg.TargetOverrides))
	}

	// Verify agents override (non-null value)
	agentsOverride, ok := cfg.TargetOverrides["agents"]
	if !ok {
		t.Error("expected 'agents' key in TargetOverrides")
	} else if agentsOverride == nil {
		t.Error("expected 'agents' value to be non-nil pointer")
	} else if *agentsOverride != ".claude/agents/custom" {
		t.Errorf("expected 'agents' value '.claude/agents/custom', got '%s'", *agentsOverride)
	}

	// Verify cursor override (null value - should be nil pointer for suppression)
	cursorOverride, ok := cfg.TargetOverrides["cursor"]
	if !ok {
		t.Error("expected 'cursor' key in TargetOverrides")
	} else if cursorOverride != nil {
		t.Errorf("expected 'cursor' value to be nil pointer for suppression, got '%s'", *cursorOverride)
	}

	// Verify prompts override (~ is also null in YAML - should be nil pointer for suppression)
	promptsOverride, ok := cfg.TargetOverrides["prompts"]
	if !ok {
		t.Error("expected 'prompts' key in TargetOverrides")
	} else if promptsOverride != nil {
		t.Errorf("expected 'prompts' value to be nil pointer for suppression, got '%s'", *promptsOverride)
	}
}

// Test 2.1e: GlobalConfigLoader returns empty maps/slices (not nil) for missing fields
func TestGlobalConfigLoader_MissingFieldsReturnEmpty(t *testing.T) {
	t.Parallel()
	memFs := afero.NewMemMapFs()

	configDir := "/config"
	err := memFs.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	// Config with only default_profile - no target_overrides, profile_overrides, ignore, or variables
	configContent := []byte(`default_profile: vendor/my-profile
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

	// Verify TargetOverrides is empty map, not nil
	if cfg.TargetOverrides == nil {
		t.Error("expected TargetOverrides to be empty map, got nil")
	} else if len(cfg.TargetOverrides) != 0 {
		t.Errorf("expected TargetOverrides to be empty, got %d entries", len(cfg.TargetOverrides))
	}

	// Verify ProfileOverrides is empty map, not nil
	if cfg.ProfileOverrides == nil {
		t.Error("expected ProfileOverrides to be empty map, got nil")
	} else if len(cfg.ProfileOverrides) != 0 {
		t.Errorf("expected ProfileOverrides to be empty, got %d entries", len(cfg.ProfileOverrides))
	}

	// Verify Ignore is empty slice, not nil
	if cfg.Ignore == nil {
		t.Error("expected Ignore to be empty slice, got nil")
	} else if len(cfg.Ignore) != 0 {
		t.Errorf("expected Ignore to be empty, got %d entries", len(cfg.Ignore))
	}

	// Verify Variables is empty map, not nil
	if cfg.Variables == nil {
		t.Error("expected Variables to be empty map, got nil")
	} else if len(cfg.Variables) != 0 {
		t.Errorf("expected Variables to be empty, got %d entries", len(cfg.Variables))
	}
}

// Test 2.1f: GlobalConfigLoader loads Variables field correctly
func TestGlobalConfigLoader_LoadsVariables(t *testing.T) {
	t.Parallel()
	memFs := afero.NewMemMapFs()

	configDir := "/config"
	err := memFs.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	configContent := []byte(`default_profile: vendor/my-profile
variables:
  namespace: my-namespace
  env: production
  count: 42
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

	// Verify variables is loaded
	if cfg.Variables == nil {
		t.Fatal("expected Variables to be non-nil")
	}

	if len(cfg.Variables) != 3 {
		t.Errorf("expected 3 variables, got %d", len(cfg.Variables))
	}

	// Verify namespace variable
	namespace, ok := cfg.Variables["namespace"]
	if !ok {
		t.Error("expected 'namespace' key in Variables")
	} else if namespace != "my-namespace" {
		t.Errorf("expected 'namespace' value 'my-namespace', got '%v'", namespace)
	}

	// Verify env variable
	env, ok := cfg.Variables["env"]
	if !ok {
		t.Error("expected 'env' key in Variables")
	} else if env != "production" {
		t.Errorf("expected 'env' value 'production', got '%v'", env)
	}
}
