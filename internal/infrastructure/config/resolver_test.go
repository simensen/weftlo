package config_test

import (
	"os"
	"path/filepath"
	"testing"

	infraconfig "github.com/simensen/weftlo/internal/infrastructure/config"
)

// Test 1: Resolution when $XDG_CONFIG_HOME is set
func TestResolveConfigDir_XDGConfigHomeSet(t *testing.T) {
	t.Parallel()
	// Create a temporary directory to simulate XDG_CONFIG_HOME
	tmpDir := t.TempDir()
	xdgConfigHome := tmpDir

	// Create the resolver with custom filesystem abstraction
	fs := &testFilesystem{
		homeDir:       "/home/testuser",
		xdgConfigHome: xdgConfigHome,
		existingDirs:  map[string]bool{},
	}

	resolver := infraconfig.NewResolver(
		infraconfig.WithFilesystem(fs),
	)

	configDir, err := resolver.ResolveConfigDir()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expected := filepath.Join(xdgConfigHome, "weftlo")
	if configDir != expected {
		t.Errorf("expected config dir '%s', got '%s'", expected, configDir)
	}
}

// Test 2: Resolution when ~/.config/weftlo/ exists (XDG not set)
func TestResolveConfigDir_DotConfigExists(t *testing.T) {
	t.Parallel()
	homeDir := "/home/testuser"
	dotConfigPath := filepath.Join(homeDir, ".config", "weftlo")

	// Create the resolver with custom filesystem abstraction
	fs := &testFilesystem{
		homeDir:       homeDir,
		xdgConfigHome: "", // XDG_CONFIG_HOME not set
		existingDirs: map[string]bool{
			dotConfigPath: true,
		},
	}

	resolver := infraconfig.NewResolver(
		infraconfig.WithFilesystem(fs),
	)

	configDir, err := resolver.ResolveConfigDir()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if configDir != dotConfigPath {
		t.Errorf("expected config dir '%s', got '%s'", dotConfigPath, configDir)
	}
}

// Test 3: Fallback to ~/.weftlo/
func TestResolveConfigDir_FallbackToAgentConfig(t *testing.T) {
	t.Parallel()
	homeDir := "/home/testuser"
	dotConfigPath := filepath.Join(homeDir, ".config", "weftlo")
	agentConfigPath := filepath.Join(homeDir, ".weftlo")

	// Create the resolver with custom filesystem abstraction
	fs := &testFilesystem{
		homeDir:       homeDir,
		xdgConfigHome: "", // XDG_CONFIG_HOME not set
		existingDirs: map[string]bool{
			dotConfigPath: false, // ~/.config/weftlo/ does NOT exist
		},
	}

	resolver := infraconfig.NewResolver(
		infraconfig.WithFilesystem(fs),
	)

	configDir, err := resolver.ResolveConfigDir()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if configDir != agentConfigPath {
		t.Errorf("expected config dir '%s', got '%s'", agentConfigPath, configDir)
	}
}

// Test 4: Home directory expansion (~ to actual path)
func TestResolveConfigDir_HomeDirectoryExpansion(t *testing.T) {
	// Use actual home directory for this test
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot get user home directory: %v", err)
	}

	// Create a real resolver (using actual filesystem)
	// We will test that the resolved path is absolute and doesn't contain ~
	resolver := infraconfig.NewResolver()

	configDir, err := resolver.ResolveConfigDir()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify the path is absolute
	if !filepath.IsAbs(configDir) {
		t.Errorf("expected absolute path, got '%s'", configDir)
	}

	// Verify the path starts with home directory (not ~)
	if configDir[0] == '~' {
		t.Errorf("expected expanded home directory, got path starting with ~: '%s'", configDir)
	}

	// Verify the path is under user's home or XDG_CONFIG_HOME
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigHome != "" {
		expected := filepath.Join(xdgConfigHome, "weftlo")
		if configDir != expected {
			t.Errorf("with XDG_CONFIG_HOME set, expected '%s', got '%s'", expected, configDir)
		}
	} else {
		// Should be either ~/.config/weftlo or ~/.weftlo
		dotConfigPath := filepath.Join(homeDir, ".config", "weftlo")
		agentConfigPath := filepath.Join(homeDir, ".weftlo")

		if configDir != dotConfigPath && configDir != agentConfigPath {
			t.Errorf("expected config dir to be '%s' or '%s', got '%s'", dotConfigPath, agentConfigPath, configDir)
		}
	}
}

// Test 5: Resolution priority order is correct
func TestResolveConfigDir_PriorityOrder(t *testing.T) {
	t.Parallel()
	homeDir := "/home/testuser"
	xdgConfigHome := "/custom/xdg/config"
	dotConfigPath := filepath.Join(homeDir, ".config", "weftlo")
	xdgPath := filepath.Join(xdgConfigHome, "weftlo")

	// Scenario: Both XDG_CONFIG_HOME is set AND ~/.config/weftlo/ exists
	// XDG_CONFIG_HOME should take priority
	fs := &testFilesystem{
		homeDir:       homeDir,
		xdgConfigHome: xdgConfigHome,
		existingDirs: map[string]bool{
			dotConfigPath: true, // ~/.config/weftlo/ exists
		},
	}

	resolver := infraconfig.NewResolver(
		infraconfig.WithFilesystem(fs),
	)

	configDir, err := resolver.ResolveConfigDir()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// XDG_CONFIG_HOME should take priority over ~/.config/weftlo/
	if configDir != xdgPath {
		t.Errorf("expected XDG path '%s' to take priority, got '%s'", xdgPath, configDir)
	}
}

// Test 6: Permission errors are handled gracefully
func TestResolveConfigDir_PermissionErrorHandled(t *testing.T) {
	t.Parallel()
	homeDir := "/home/testuser"
	dotConfigPath := filepath.Join(homeDir, ".config", "weftlo")
	agentConfigPath := filepath.Join(homeDir, ".weftlo")

	// Simulate permission error when checking ~/.config/weftlo/
	fs := &testFilesystem{
		homeDir:         homeDir,
		xdgConfigHome:   "",
		existingDirs:    map[string]bool{},
		permissionError: dotConfigPath,
	}

	resolver := infraconfig.NewResolver(
		infraconfig.WithFilesystem(fs),
	)

	// Should fall back gracefully when permission error occurs
	configDir, err := resolver.ResolveConfigDir()
	if err != nil {
		t.Fatalf("expected no error (graceful fallback), got: %v", err)
	}

	// Should fall back to ~/.weftlo/
	if configDir != agentConfigPath {
		t.Errorf("expected fallback to '%s', got '%s'", agentConfigPath, configDir)
	}
}

// testFilesystem is a mock filesystem for testing
type testFilesystem struct {
	homeDir         string
	xdgConfigHome   string
	existingDirs    map[string]bool
	permissionError string // path that should trigger permission error
}

func (fs *testFilesystem) UserHomeDir() (string, error) {
	return fs.homeDir, nil
}

func (fs *testFilesystem) GetEnv(key string) string {
	if key == "XDG_CONFIG_HOME" {
		return fs.xdgConfigHome
	}
	return ""
}

func (fs *testFilesystem) DirExists(path string) (bool, error) {
	if fs.permissionError != "" && path == fs.permissionError {
		return false, os.ErrPermission
	}
	exists, ok := fs.existingDirs[path]
	if !ok {
		return false, nil
	}
	return exists, nil
}

// Tests for ResolveConfigDirForCreation
// The key difference from ResolveConfigDir is that it checks if ~/.config/ exists (parent)
// rather than ~/.config/weftlo/ exists.

// Test: XDG_CONFIG_HOME takes priority for creation resolution
func TestResolveConfigDirForCreation_XDGConfigHomeSet(t *testing.T) {
	t.Parallel()
	xdgConfigHome := "/custom/xdg/config"

	fs := &testFilesystem{
		homeDir:       "/home/testuser",
		xdgConfigHome: xdgConfigHome,
		existingDirs:  map[string]bool{},
	}

	resolver := infraconfig.NewResolver(
		infraconfig.WithFilesystem(fs),
	)

	configDir, err := resolver.ResolveConfigDirForCreation()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expected := filepath.Join(xdgConfigHome, "weftlo")
	if configDir != expected {
		t.Errorf("expected config dir '%s', got '%s'", expected, configDir)
	}
}

// Test: ~/.config/ exists -> use ~/.config/weftlo/ for creation
func TestResolveConfigDirForCreation_DotConfigParentExists(t *testing.T) {
	t.Parallel()
	homeDir := "/home/testuser"
	dotConfigPath := filepath.Join(homeDir, ".config")
	expectedPath := filepath.Join(dotConfigPath, "weftlo")

	fs := &testFilesystem{
		homeDir:       homeDir,
		xdgConfigHome: "", // XDG_CONFIG_HOME not set
		existingDirs: map[string]bool{
			dotConfigPath: true, // ~/.config/ exists (parent directory)
		},
	}

	resolver := infraconfig.NewResolver(
		infraconfig.WithFilesystem(fs),
	)

	configDir, err := resolver.ResolveConfigDirForCreation()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if configDir != expectedPath {
		t.Errorf("expected config dir '%s', got '%s'", expectedPath, configDir)
	}
}

// Test: ~/.config/ does NOT exist -> fall back to ~/.weftlo/
func TestResolveConfigDirForCreation_FallbackToAgentConfig(t *testing.T) {
	t.Parallel()
	homeDir := "/home/testuser"
	dotConfigPath := filepath.Join(homeDir, ".config")
	agentConfigPath := filepath.Join(homeDir, ".weftlo")

	fs := &testFilesystem{
		homeDir:       homeDir,
		xdgConfigHome: "", // XDG_CONFIG_HOME not set
		existingDirs: map[string]bool{
			dotConfigPath: false, // ~/.config/ does NOT exist
		},
	}

	resolver := infraconfig.NewResolver(
		infraconfig.WithFilesystem(fs),
	)

	configDir, err := resolver.ResolveConfigDirForCreation()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if configDir != agentConfigPath {
		t.Errorf("expected config dir '%s', got '%s'", agentConfigPath, configDir)
	}
}

// Test: XDG takes priority over ~/.config/ for creation
func TestResolveConfigDirForCreation_XDGPriorityOverDotConfig(t *testing.T) {
	t.Parallel()
	homeDir := "/home/testuser"
	xdgConfigHome := "/custom/xdg/config"
	dotConfigPath := filepath.Join(homeDir, ".config")
	xdgPath := filepath.Join(xdgConfigHome, "weftlo")

	// Scenario: Both XDG_CONFIG_HOME is set AND ~/.config/ exists
	// XDG_CONFIG_HOME should take priority
	fs := &testFilesystem{
		homeDir:       homeDir,
		xdgConfigHome: xdgConfigHome,
		existingDirs: map[string]bool{
			dotConfigPath: true, // ~/.config/ exists
		},
	}

	resolver := infraconfig.NewResolver(
		infraconfig.WithFilesystem(fs),
	)

	configDir, err := resolver.ResolveConfigDirForCreation()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if configDir != xdgPath {
		t.Errorf("expected XDG path '%s' to take priority, got '%s'", xdgPath, configDir)
	}
}

// Test: Permission error on ~/.config/ -> fall back gracefully
func TestResolveConfigDirForCreation_PermissionErrorHandled(t *testing.T) {
	t.Parallel()
	homeDir := "/home/testuser"
	dotConfigPath := filepath.Join(homeDir, ".config")
	agentConfigPath := filepath.Join(homeDir, ".weftlo")

	// Simulate permission error when checking ~/.config/
	fs := &testFilesystem{
		homeDir:         homeDir,
		xdgConfigHome:   "",
		existingDirs:    map[string]bool{},
		permissionError: dotConfigPath, // Permission error on ~/.config/
	}

	resolver := infraconfig.NewResolver(
		infraconfig.WithFilesystem(fs),
	)

	// Should fall back gracefully when permission error occurs
	configDir, err := resolver.ResolveConfigDirForCreation()
	if err != nil {
		t.Fatalf("expected no error (graceful fallback), got: %v", err)
	}

	// Should fall back to ~/.weftlo/
	if configDir != agentConfigPath {
		t.Errorf("expected fallback to '%s', got '%s'", agentConfigPath, configDir)
	}
}

// Test: Difference between ResolveConfigDir and ResolveConfigDirForCreation
// When ~/.config/weftlo/ doesn't exist but ~/.config/ does,
// ResolveConfigDir should fall back, but ResolveConfigDirForCreation should use ~/.config/weftlo/
func TestResolveConfigDirForCreation_DifferenceFromResolveConfigDir(t *testing.T) {
	t.Parallel()
	homeDir := "/home/testuser"
	dotConfigPath := filepath.Join(homeDir, ".config")
	dotConfigAgentPath := filepath.Join(dotConfigPath, "weftlo")
	agentConfigPath := filepath.Join(homeDir, ".weftlo")

	// ~/.config/ exists but ~/.config/weftlo/ does not
	fs := &testFilesystem{
		homeDir:       homeDir,
		xdgConfigHome: "",
		existingDirs: map[string]bool{
			dotConfigPath:      true,  // ~/.config/ exists
			dotConfigAgentPath: false, // ~/.config/weftlo/ does NOT exist
		},
	}

	resolver := infraconfig.NewResolver(
		infraconfig.WithFilesystem(fs),
	)

	// ResolveConfigDir should fall back to ~/.weftlo/ because ~/.config/weftlo/ doesn't exist
	readDir, err := resolver.ResolveConfigDir()
	if err != nil {
		t.Fatalf("expected no error for ResolveConfigDir, got: %v", err)
	}
	if readDir != agentConfigPath {
		t.Errorf("ResolveConfigDir: expected '%s', got '%s'", agentConfigPath, readDir)
	}

	// ResolveConfigDirForCreation should use ~/.config/weftlo/ because ~/.config/ exists
	createDir, err := resolver.ResolveConfigDirForCreation()
	if err != nil {
		t.Fatalf("expected no error for ResolveConfigDirForCreation, got: %v", err)
	}
	if createDir != dotConfigAgentPath {
		t.Errorf("ResolveConfigDirForCreation: expected '%s', got '%s'", dotConfigAgentPath, createDir)
	}
}
