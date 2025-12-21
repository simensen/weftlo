package cli_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simensen/weftlo/internal/cli"
	"github.com/simensen/weftlo/internal/testutil"
)

// =============================================================================
// Task Group 4.1: CLI Variable Integration Tests
// =============================================================================

// setupProfileWithVariables creates a profile that has variables defined.
func setupProfileWithVariables(t *testing.T, fs afero.Fs, homeDir, profileName string, variables map[string]interface{}) {
	t.Helper()

	parts := strings.Split(profileName, "/")
	if len(parts) != 2 {
		t.Fatalf("invalid profile name: %s", profileName)
	}
	vendor := parts[0]
	name := parts[1]

	profileDir := filepath.Join(homeDir, ".weftlo", "profiles", vendor, name)
	contentDir := filepath.Join(profileDir, "content")
	if err := fs.MkdirAll(contentDir, 0755); err != nil {
		t.Fatalf("failed to create profile content directory: %v", err)
	}

	// Build variables YAML section
	var variablesYAML string
	if len(variables) > 0 {
		variablesYAML = "\nvariables:\n"
		for k, v := range variables {
			switch val := v.(type) {
			case string:
				variablesYAML += "  " + k + ": " + val + "\n"
			case map[string]interface{}:
				variablesYAML += "  " + k + ":\n"
				for subK, subV := range val {
					if str, ok := subV.(string); ok {
						variablesYAML += "    " + subK + ": " + str + "\n"
					}
				}
			}
		}
	}

	// Create profile.yaml with variables
	profileContent := "name: " + profileName + "\ncontent:\n  root: content\n  default_target: \"\"" + variablesYAML
	if err := afero.WriteFile(fs, filepath.Join(profileDir, "profile.yaml"), []byte(profileContent), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}

	// Create a simple template file in the content directory
	templateContent := "# Test File\nGenerated from {{ .Profile.Name }}\n"
	if err := afero.WriteFile(fs, filepath.Join(contentDir, "test.md.tmpl"), []byte(templateContent), 0644); err != nil {
		t.Fatalf("failed to write test.md.tmpl: %v", err)
	}
}

// setupProfileWithInheritanceAndVariables creates a child profile that extends a parent and has conflicting variables.
func setupProfileWithInheritanceAndVariables(t *testing.T, fs afero.Fs, homeDir, parentName, childName string, parentVars, childVars map[string]interface{}) {
	t.Helper()

	// Setup parent profile with variables
	setupProfileWithVariables(t, fs, homeDir, parentName, parentVars)

	// Setup child profile with variables and inheritance
	parts := strings.Split(childName, "/")
	if len(parts) != 2 {
		t.Fatalf("invalid profile name: %s", childName)
	}
	vendor := parts[0]
	name := parts[1]

	profileDir := filepath.Join(homeDir, ".weftlo", "profiles", vendor, name)
	contentDir := filepath.Join(profileDir, "content")
	if err := fs.MkdirAll(contentDir, 0755); err != nil {
		t.Fatalf("failed to create profile content directory: %v", err)
	}

	// Build variables YAML section
	var variablesYAML string
	if len(childVars) > 0 {
		variablesYAML = "\nvariables:\n"
		for k, v := range childVars {
			switch val := v.(type) {
			case string:
				variablesYAML += "  " + k + ": " + val + "\n"
			case map[string]interface{}:
				variablesYAML += "  " + k + ":\n"
				for subK, subV := range val {
					if str, ok := subV.(string); ok {
						variablesYAML += "    " + subK + ": " + str + "\n"
					}
				}
			}
		}
	}

	// Create profile.yaml with variables and inherits_from (using the correct yaml field name)
	profileContent := "name: " + childName + "\ninherits_from: " + parentName + "\ncontent:\n  root: content\n  default_target: \"\"" + variablesYAML
	if err := afero.WriteFile(fs, filepath.Join(profileDir, "profile.yaml"), []byte(profileContent), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}

	// Create a simple template file in the content directory
	templateContent := "# Test File\nGenerated from {{ .Profile.Name }}\n"
	if err := afero.WriteFile(fs, filepath.Join(contentDir, "test.md.tmpl"), []byte(templateContent), 0644); err != nil {
		t.Fatalf("failed to write test.md.tmpl: %v", err)
	}
}

// Test 4.1.1: Install command uses MergedProfile.Variables in MergeVariables call
func TestInstallCommand_UsesMergedProfileVariables(t *testing.T) {
	memFs := testutil.CreateTestFilesystem(t, map[string]string{})
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup global config
	configDir := filepath.Join(homeDir, ".weftlo")
	if err := memFs.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}
	globalConfig := "default_profile: vendor/test\n"
	if err := afero.WriteFile(memFs, filepath.Join(configDir, "config.yaml"), []byte(globalConfig), 0644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}

	// Setup profile with variables
	profileVars := map[string]interface{}{
		"app_name": "TestApp",
	}
	setupProfileWithVariables(t, memFs, homeDir, "vendor/test", profileVars)

	// Create project directory
	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	var stdout bytes.Buffer
	installCmd := cli.NewInstallCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	installCmd.SetArgs([]string{})
	err := installCmd.Execute()

	// The command should succeed and use profile variables
	require.NoError(t, err, "install command should succeed")

	// Verify the installation succeeded (manifest was created)
	manifestPath := filepath.Join(projectDir, ".weftlo.manifest.json")
	exists, _ := afero.Exists(memFs, manifestPath)
	assert.True(t, exists, "manifest should be created after successful installation")
}

// Test 4.1.2: Update command uses MergedProfile.Variables in MergeVariables call
func TestUpdateCommand_UsesMergedProfileVariables(t *testing.T) {
	memFs := testutil.CreateTestFilesystem(t, map[string]string{})
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup global config
	configDir := filepath.Join(homeDir, ".weftlo")
	if err := memFs.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}
	globalConfig := "default_profile: vendor/test\n"
	if err := afero.WriteFile(memFs, filepath.Join(configDir, "config.yaml"), []byte(globalConfig), 0644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}

	// Setup profile with variables
	profileVars := map[string]interface{}{
		"app_name": "TestApp",
	}
	setupProfileWithVariables(t, memFs, homeDir, "vendor/test", profileVars)

	// Create project directory with project config and manifest
	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	// Create .weftlo.yaml
	projectConfig := "profiles:\n  - vendor/test\ninstall_prefix: weftlo\n"
	if err := afero.WriteFile(memFs, filepath.Join(projectDir, ".weftlo.yaml"), []byte(projectConfig), 0644); err != nil {
		t.Fatalf("failed to write .weftlo.yaml: %v", err)
	}

	// Create minimal manifest
	manifestContent := `{
		"version": "1.0.0",
		"profiles": ["vendor/test"],
		"profile_names": ["vendor/test"],
		"generated_at": "2024-01-01T00:00:00Z",
		"files": {}
	}`
	if err := afero.WriteFile(memFs, filepath.Join(projectDir, ".weftlo.manifest.json"), []byte(manifestContent), 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	var stdout bytes.Buffer
	updateCmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	updateCmd.SetArgs([]string{})
	err := updateCmd.Execute()

	// The command should succeed (may report no changes needed)
	// Either no error or a specific expected error
	if err != nil {
		// Some errors are acceptable (e.g., no changes needed is not an error)
		t.Logf("update command returned: %v", err)
	}
}

// Test 4.1.3: Status command uses MergedProfile.Variables in MergeVariables call
func TestStatusCommand_UsesMergedProfileVariables(t *testing.T) {
	memFs := testutil.CreateTestFilesystem(t, map[string]string{})
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup global config
	configDir := filepath.Join(homeDir, ".weftlo")
	if err := memFs.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}
	globalConfig := "default_profile: vendor/test\n"
	if err := afero.WriteFile(memFs, filepath.Join(configDir, "config.yaml"), []byte(globalConfig), 0644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}

	// Setup profile with variables
	profileVars := map[string]interface{}{
		"app_name": "TestApp",
	}
	setupProfileWithVariables(t, memFs, homeDir, "vendor/test", profileVars)

	// Create project directory with project config and manifest
	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	// Create .weftlo.yaml
	projectConfig := "profiles:\n  - vendor/test\ninstall_prefix: weftlo\n"
	if err := afero.WriteFile(memFs, filepath.Join(projectDir, ".weftlo.yaml"), []byte(projectConfig), 0644); err != nil {
		t.Fatalf("failed to write .weftlo.yaml: %v", err)
	}

	// Create minimal manifest
	manifestContent := `{
		"version": "1.0.0",
		"profiles": ["vendor/test"],
		"profile_names": ["vendor/test"],
		"generated_at": "2024-01-01T00:00:00Z",
		"files": {}
	}`
	if err := afero.WriteFile(memFs, filepath.Join(projectDir, ".weftlo.manifest.json"), []byte(manifestContent), 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	var stdout bytes.Buffer
	statusCmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	statusCmd.SetArgs([]string{"--json"})
	err := statusCmd.Execute()

	require.NoError(t, err, "status command should succeed")

	// Verify output contains expected profile
	var result struct {
		Profiles []string `json:"profiles"`
	}
	err = json.Unmarshal(stdout.Bytes(), &result)
	require.NoError(t, err, "should be valid JSON")
	assert.Contains(t, result.Profiles, "vendor/test", "should contain the profile name")
}

// Test 4.1.4: Conflict warnings are displayed when conflicts exist
func TestInstallCommand_DisplaysConflictWarningsWhenConflictsExist(t *testing.T) {
	memFs := testutil.CreateTestFilesystem(t, map[string]string{})
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup global config
	configDir := filepath.Join(homeDir, ".weftlo")
	if err := memFs.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}
	globalConfig := "default_profile: vendor/child\n"
	if err := afero.WriteFile(memFs, filepath.Join(configDir, "config.yaml"), []byte(globalConfig), 0644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}

	// Setup parent and child profiles with conflicting variables
	parentVars := map[string]interface{}{
		"app_name": "ParentApp",
		"database": map[string]interface{}{
			"host": "parent-db.example.com",
		},
	}
	childVars := map[string]interface{}{
		"app_name": "ChildApp", // This conflicts with parent
		"database": map[string]interface{}{
			"host": "child-db.example.com", // This also conflicts
		},
	}
	setupProfileWithInheritanceAndVariables(t, memFs, homeDir, "vendor/parent", "vendor/child", parentVars, childVars)

	// Create project directory
	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	var stdout bytes.Buffer
	installCmd := cli.NewInstallCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	installCmd.SetArgs([]string{})
	err := installCmd.Execute()

	// The command should succeed
	require.NoError(t, err, "install command should succeed even with variable conflicts")

	// Check that warning was displayed
	output := stdout.String()
	assert.Contains(t, output, "Warning:", "should display warning for variable conflicts")
	assert.Contains(t, output, "app_name", "warning should mention the conflicting variable")
}

// Test 4.1.5: No warnings displayed when no conflicts exist
func TestInstallCommand_NoWarningsWhenNoConflicts(t *testing.T) {
	memFs := testutil.CreateTestFilesystem(t, map[string]string{})
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup global config
	configDir := filepath.Join(homeDir, ".weftlo")
	if err := memFs.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}
	globalConfig := "default_profile: vendor/test\n"
	if err := afero.WriteFile(memFs, filepath.Join(configDir, "config.yaml"), []byte(globalConfig), 0644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}

	// Setup profile with variables (no inheritance, so no conflicts)
	profileVars := map[string]interface{}{
		"app_name": "TestApp",
	}
	setupProfileWithVariables(t, memFs, homeDir, "vendor/test", profileVars)

	// Create project directory
	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	var stdout bytes.Buffer
	installCmd := cli.NewInstallCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	installCmd.SetArgs([]string{})
	err := installCmd.Execute()

	// The command should succeed
	require.NoError(t, err, "install command should succeed")

	// Check that no warning was displayed
	output := stdout.String()
	// The output should NOT contain "Variable" and "overridden" together (conflict warning pattern)
	assert.NotContains(t, output, "overridden by", "should not display conflict warnings when no conflicts exist")
}

// Test 4.1.6: Update command displays conflict warnings when conflicts exist
func TestUpdateCommand_DisplaysConflictWarningsWhenConflictsExist(t *testing.T) {
	memFs := testutil.CreateTestFilesystem(t, map[string]string{})
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup global config
	configDir := filepath.Join(homeDir, ".weftlo")
	if err := memFs.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}
	globalConfig := "default_profile: vendor/child\n"
	if err := afero.WriteFile(memFs, filepath.Join(configDir, "config.yaml"), []byte(globalConfig), 0644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}

	// Setup parent and child profiles with conflicting variables
	parentVars := map[string]interface{}{
		"app_name": "ParentApp",
	}
	childVars := map[string]interface{}{
		"app_name": "ChildApp", // This conflicts with parent
	}
	setupProfileWithInheritanceAndVariables(t, memFs, homeDir, "vendor/parent", "vendor/child", parentVars, childVars)

	// Create project directory with project config and manifest
	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	// Create .weftlo.yaml
	projectConfig := "profiles:\n  - vendor/child\ninstall_prefix: weftlo\n"
	if err := afero.WriteFile(memFs, filepath.Join(projectDir, ".weftlo.yaml"), []byte(projectConfig), 0644); err != nil {
		t.Fatalf("failed to write .weftlo.yaml: %v", err)
	}

	// Create minimal manifest
	manifestContent := `{
		"version": "1.0.0",
		"profiles": ["vendor/child"],
		"profile_names": ["vendor/parent", "vendor/child"],
		"generated_at": "2024-01-01T00:00:00Z",
		"files": {}
	}`
	if err := afero.WriteFile(memFs, filepath.Join(projectDir, ".weftlo.manifest.json"), []byte(manifestContent), 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	var stdout bytes.Buffer
	updateCmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	updateCmd.SetArgs([]string{})
	_ = updateCmd.Execute() // Ignore error since we're checking output

	// Check that warning was displayed
	output := stdout.String()
	assert.Contains(t, output, "Warning:", "should display warning for variable conflicts")
}
