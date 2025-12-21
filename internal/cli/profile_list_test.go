package cli_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/afero"

	"github.com/simensen/weftlo/internal/cli"
	infraconfig "github.com/simensen/weftlo/internal/infrastructure/config"
)

// =============================================================================
// Task Group 1: Command Structure and Registration Tests
// =============================================================================

// Test 1.1.1: Test `list` subcommand is accessible via `weftlo profile list`
func TestProfileListCommand_AccessibleViaRootCommand(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create required config directory
	_ = fs.MkdirAll(homeDir+"/.weftlo/profiles", 0755)

	rootCmd := cli.NewRootCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)

	// Execute "weftlo profile list"
	rootCmd.SetArgs([]string{"profile", "list"})

	err := rootCmd.Execute()
	// The command should execute without error (empty profile list is valid)
	if err != nil {
		t.Fatalf("unexpected error executing 'profile list': %v", err)
	}
}

// Test 1.1.2: Test `ls` alias works (`weftlo profile ls`)
func TestProfileListCommand_LsAliasWorks(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create required config directory
	_ = fs.MkdirAll(homeDir+"/.weftlo/profiles", 0755)

	rootCmd := cli.NewRootCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)

	// Execute "weftlo profile ls" (alias)
	rootCmd.SetArgs([]string{"profile", "ls"})

	err := rootCmd.Execute()
	// The command should execute without error
	if err != nil {
		t.Fatalf("unexpected error executing 'profile ls' alias: %v", err)
	}
}

// Test 1.1.3: Test command has `--json` flag defined
func TestProfileListCommand_HasJsonFlag(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create required config directory
	_ = fs.MkdirAll(homeDir+"/.weftlo/profiles", 0755)

	rootCmd := cli.NewRootCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)

	// Execute "weftlo profile list --json"
	rootCmd.SetArgs([]string{"profile", "list", "--json"})

	err := rootCmd.Execute()
	// The command should execute without error and output should be JSON
	if err != nil {
		t.Fatalf("unexpected error executing 'profile list --json': %v", err)
	}

	// Verify the output is JSON format (should contain { and })
	output := stdout.String()
	if len(output) == 0 {
		t.Error("expected JSON output, got empty string")
	}
	// JSON output should start with { for an object
	if output[0] != '{' {
		t.Errorf("expected JSON output to start with '{', got: %s", output)
	}
}

// Test 1.1.4: Test command has `--verbose` flag inherited from root
func TestProfileListCommand_HasVerboseFlag(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create required config directory and a profile to list
	_ = fs.MkdirAll(homeDir+"/.weftlo/profiles/test/profile", 0755)
	_ = afero.WriteFile(fs, homeDir+"/.weftlo/profiles/test/profile/profile.yaml", []byte("name: test/profile\n"), 0644)

	rootCmd := cli.NewRootCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)

	// Execute "weftlo --verbose profile list"
	rootCmd.SetArgs([]string{"--verbose", "profile", "list"})

	err := rootCmd.Execute()
	// The command should execute without error with --verbose flag
	if err != nil {
		t.Fatalf("unexpected error executing 'profile list' with --verbose: %v", err)
	}
}

// =============================================================================
// Task Group 2: Profile Directory Enumeration Tests
// =============================================================================

// Test 2.1.1: Test enumerates profiles from `{config-dir}/profiles/{vendor}/{name}/` structure
func TestProfileEnumerator_EnumeratesProfilesFromDirectoryStructure(t *testing.T) {
	fs := afero.NewMemMapFs()
	configDir := "/home/testuser/.weftlo"

	// Create profile directory structure with valid profiles
	_ = fs.MkdirAll(configDir+"/profiles/vendor1/profile1", 0755)
	_ = afero.WriteFile(fs, configDir+"/profiles/vendor1/profile1/profile.yaml", []byte("name: vendor1/profile1\n"), 0644)

	_ = fs.MkdirAll(configDir+"/profiles/vendor2/profile2", 0755)
	_ = afero.WriteFile(fs, configDir+"/profiles/vendor2/profile2/profile.yaml", []byte("name: vendor2/profile2\n"), 0644)

	// Create the enumerator and enumerate profiles
	enumerator := cli.NewProfileEnumerator(fs, configDir)
	validProfiles, brokenProfiles, err := enumerator.EnumerateProfiles()

	if err != nil {
		t.Fatalf("unexpected error enumerating profiles: %v", err)
	}

	// Verify we found both profiles
	if len(validProfiles) != 2 {
		t.Errorf("expected 2 valid profiles, got %d", len(validProfiles))
	}

	// Verify no broken profiles
	if len(brokenProfiles) != 0 {
		t.Errorf("expected 0 broken profiles, got %d", len(brokenProfiles))
	}

	// Verify profile names are correct (sorted alphabetically)
	expectedNames := []string{"vendor1/profile1", "vendor2/profile2"}
	for i, expected := range expectedNames {
		if i >= len(validProfiles) {
			t.Errorf("missing profile at index %d", i)
			continue
		}
		if validProfiles[i].Name != expected {
			t.Errorf("expected profile name %q at index %d, got %q", expected, i, validProfiles[i].Name)
		}
	}
}

// Test 2.1.2: Test returns empty list when no profiles exist
func TestProfileEnumerator_ReturnsEmptyListWhenNoProfiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	configDir := "/home/testuser/.weftlo"

	// Create empty profiles directory
	_ = fs.MkdirAll(configDir+"/profiles", 0755)

	// Create the enumerator and enumerate profiles
	enumerator := cli.NewProfileEnumerator(fs, configDir)
	validProfiles, brokenProfiles, err := enumerator.EnumerateProfiles()

	if err != nil {
		t.Fatalf("unexpected error enumerating profiles: %v", err)
	}

	// Verify empty lists
	if len(validProfiles) != 0 {
		t.Errorf("expected 0 valid profiles, got %d", len(validProfiles))
	}

	if len(brokenProfiles) != 0 {
		t.Errorf("expected 0 broken profiles, got %d", len(brokenProfiles))
	}
}

// Test 2.1.3: Test sorts profiles alphabetically by full name
func TestProfileEnumerator_SortsProfilesAlphabetically(t *testing.T) {
	fs := afero.NewMemMapFs()
	configDir := "/home/testuser/.weftlo"

	// Create profiles in non-alphabetical order
	_ = fs.MkdirAll(configDir+"/profiles/zebra/last", 0755)
	_ = afero.WriteFile(fs, configDir+"/profiles/zebra/last/profile.yaml", []byte("name: zebra/last\n"), 0644)

	_ = fs.MkdirAll(configDir+"/profiles/alpha/first", 0755)
	_ = afero.WriteFile(fs, configDir+"/profiles/alpha/first/profile.yaml", []byte("name: alpha/first\n"), 0644)

	_ = fs.MkdirAll(configDir+"/profiles/middle/second", 0755)
	_ = afero.WriteFile(fs, configDir+"/profiles/middle/second/profile.yaml", []byte("name: middle/second\n"), 0644)

	// Create the enumerator and enumerate profiles
	enumerator := cli.NewProfileEnumerator(fs, configDir)
	validProfiles, _, err := enumerator.EnumerateProfiles()

	if err != nil {
		t.Fatalf("unexpected error enumerating profiles: %v", err)
	}

	// Verify profiles are sorted alphabetically
	expectedOrder := []string{"alpha/first", "middle/second", "zebra/last"}
	if len(validProfiles) != len(expectedOrder) {
		t.Fatalf("expected %d profiles, got %d", len(expectedOrder), len(validProfiles))
	}

	for i, expected := range expectedOrder {
		if validProfiles[i].Name != expected {
			t.Errorf("expected profile %q at position %d, got %q", expected, i, validProfiles[i].Name)
		}
	}
}

// Test 2.1.4: Test identifies valid profiles (profile.yaml exists)
func TestProfileEnumerator_IdentifiesValidProfiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	configDir := "/home/testuser/.weftlo"

	// Create a valid profile with profile.yaml
	_ = fs.MkdirAll(configDir+"/profiles/myvendor/myprofile", 0755)
	_ = afero.WriteFile(fs, configDir+"/profiles/myvendor/myprofile/profile.yaml", []byte("name: myvendor/myprofile\n"), 0644)

	// Create the enumerator and enumerate profiles
	enumerator := cli.NewProfileEnumerator(fs, configDir)
	validProfiles, brokenProfiles, err := enumerator.EnumerateProfiles()

	if err != nil {
		t.Fatalf("unexpected error enumerating profiles: %v", err)
	}

	// Verify we found the valid profile
	if len(validProfiles) != 1 {
		t.Fatalf("expected 1 valid profile, got %d", len(validProfiles))
	}

	if validProfiles[0].Name != "myvendor/myprofile" {
		t.Errorf("expected profile name 'myvendor/myprofile', got %q", validProfiles[0].Name)
	}

	// Verify no broken profiles
	if len(brokenProfiles) != 0 {
		t.Errorf("expected 0 broken profiles, got %d", len(brokenProfiles))
	}
}

// Test 2.1.5: Test identifies broken profiles (directory exists but no profile.yaml)
func TestProfileEnumerator_IdentifiesBrokenProfiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	configDir := "/home/testuser/.weftlo"

	// Create a valid profile
	_ = fs.MkdirAll(configDir+"/profiles/valid/profile", 0755)
	_ = afero.WriteFile(fs, configDir+"/profiles/valid/profile/profile.yaml", []byte("name: valid/profile\n"), 0644)

	// Create a broken profile (directory exists but no profile.yaml)
	_ = fs.MkdirAll(configDir+"/profiles/broken/profile", 0755)

	// Create the enumerator and enumerate profiles
	enumerator := cli.NewProfileEnumerator(fs, configDir)
	validProfiles, brokenProfiles, err := enumerator.EnumerateProfiles()

	if err != nil {
		t.Fatalf("unexpected error enumerating profiles: %v", err)
	}

	// Verify we found 1 valid profile
	if len(validProfiles) != 1 {
		t.Fatalf("expected 1 valid profile, got %d", len(validProfiles))
	}

	if validProfiles[0].Name != "valid/profile" {
		t.Errorf("expected valid profile name 'valid/profile', got %q", validProfiles[0].Name)
	}

	// Verify we found 1 broken profile
	if len(brokenProfiles) != 1 {
		t.Fatalf("expected 1 broken profile, got %d", len(brokenProfiles))
	}

	if brokenProfiles[0].Name != "broken/profile" {
		t.Errorf("expected broken profile name 'broken/profile', got %q", brokenProfiles[0].Name)
	}

	if brokenProfiles[0].Reason != "missing profile.yaml" {
		t.Errorf("expected reason 'missing profile.yaml', got %q", brokenProfiles[0].Reason)
	}
}

// =============================================================================
// Task Group 3: Inheritance Information Extraction Tests
// =============================================================================

// Test 3.1.1: Test parses `inherits_from` field from valid profile.yaml
func TestProfileEnumerator_ParsesInheritsFromField(t *testing.T) {
	fs := afero.NewMemMapFs()
	configDir := "/home/testuser/.weftlo"

	// Create a profile with inherits_from field
	_ = fs.MkdirAll(configDir+"/profiles/child/profile", 0755)
	profileYAML := `name: child/profile
inherits_from: parent/profile
`
	_ = afero.WriteFile(fs, configDir+"/profiles/child/profile/profile.yaml", []byte(profileYAML), 0644)

	// Create the enumerator and enumerate profiles with verbose mode
	enumerator := cli.NewProfileEnumerator(fs, configDir)
	validProfiles, _, err := enumerator.EnumerateProfilesVerbose()

	if err != nil {
		t.Fatalf("unexpected error enumerating profiles: %v", err)
	}

	// Verify we found the profile
	if len(validProfiles) != 1 {
		t.Fatalf("expected 1 valid profile, got %d", len(validProfiles))
	}

	// Verify profile name
	if validProfiles[0].Name != "child/profile" {
		t.Errorf("expected profile name 'child/profile', got %q", validProfiles[0].Name)
	}

	// Verify inherits_from field was parsed
	if validProfiles[0].InheritsFrom != "parent/profile" {
		t.Errorf("expected inherits_from 'parent/profile', got %q", validProfiles[0].InheritsFrom)
	}
}

// Test 3.1.2: Test handles missing `inherits_from` field (no inheritance)
func TestProfileEnumerator_HandlesMissingInheritsFromField(t *testing.T) {
	fs := afero.NewMemMapFs()
	configDir := "/home/testuser/.weftlo"

	// Create a profile without inherits_from field
	_ = fs.MkdirAll(configDir+"/profiles/standalone/profile", 0755)
	profileYAML := `name: standalone/profile
`
	_ = afero.WriteFile(fs, configDir+"/profiles/standalone/profile/profile.yaml", []byte(profileYAML), 0644)

	// Create the enumerator and enumerate profiles with verbose mode
	enumerator := cli.NewProfileEnumerator(fs, configDir)
	validProfiles, _, err := enumerator.EnumerateProfilesVerbose()

	if err != nil {
		t.Fatalf("unexpected error enumerating profiles: %v", err)
	}

	// Verify we found the profile
	if len(validProfiles) != 1 {
		t.Fatalf("expected 1 valid profile, got %d", len(validProfiles))
	}

	// Verify profile name
	if validProfiles[0].Name != "standalone/profile" {
		t.Errorf("expected profile name 'standalone/profile', got %q", validProfiles[0].Name)
	}

	// Verify inherits_from is empty (no inheritance)
	if validProfiles[0].InheritsFrom != "" {
		t.Errorf("expected empty inherits_from, got %q", validProfiles[0].InheritsFrom)
	}
}

// Test 3.1.3: Test handles empty `inherits_from` value
func TestProfileEnumerator_HandlesEmptyInheritsFromValue(t *testing.T) {
	fs := afero.NewMemMapFs()
	configDir := "/home/testuser/.weftlo"

	// Create a profile with empty inherits_from field
	_ = fs.MkdirAll(configDir+"/profiles/empty/profile", 0755)
	profileYAML := `name: empty/profile
inherits_from: ""
`
	_ = afero.WriteFile(fs, configDir+"/profiles/empty/profile/profile.yaml", []byte(profileYAML), 0644)

	// Create the enumerator and enumerate profiles with verbose mode
	enumerator := cli.NewProfileEnumerator(fs, configDir)
	validProfiles, _, err := enumerator.EnumerateProfilesVerbose()

	if err != nil {
		t.Fatalf("unexpected error enumerating profiles: %v", err)
	}

	// Verify we found the profile
	if len(validProfiles) != 1 {
		t.Fatalf("expected 1 valid profile, got %d", len(validProfiles))
	}

	// Verify profile name
	if validProfiles[0].Name != "empty/profile" {
		t.Errorf("expected profile name 'empty/profile', got %q", validProfiles[0].Name)
	}

	// Verify inherits_from is empty
	if validProfiles[0].InheritsFrom != "" {
		t.Errorf("expected empty inherits_from, got %q", validProfiles[0].InheritsFrom)
	}
}

// =============================================================================
// Task Group 4: Output Modes Implementation Tests
// =============================================================================

// Test 4.1.1: Test default output shows "Available profiles:" header with indented names
func TestProfileListCommand_DefaultOutputShowsHeaderWithIndentedNames(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create profiles
	_ = fs.MkdirAll(homeDir+"/.weftlo/profiles/vendor1/profile1", 0755)
	_ = afero.WriteFile(fs, homeDir+"/.weftlo/profiles/vendor1/profile1/profile.yaml", []byte("name: vendor1/profile1\n"), 0644)

	_ = fs.MkdirAll(homeDir+"/.weftlo/profiles/vendor2/profile2", 0755)
	_ = afero.WriteFile(fs, homeDir+"/.weftlo/profiles/vendor2/profile2/profile.yaml", []byte("name: vendor2/profile2\n"), 0644)

	rootCmd := cli.NewRootCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)

	rootCmd.SetArgs([]string{"profile", "list"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()

	// Check for header
	if !strings.Contains(output, "Available profiles:") {
		t.Errorf("expected output to contain 'Available profiles:' header, got: %s", output)
	}

	// Check for two-space indented profile names
	if !strings.Contains(output, "  vendor1/profile1") {
		t.Errorf("expected output to contain '  vendor1/profile1' (two-space indent), got: %s", output)
	}
	if !strings.Contains(output, "  vendor2/profile2") {
		t.Errorf("expected output to contain '  vendor2/profile2' (two-space indent), got: %s", output)
	}
}

// Test 4.1.2: Test default output skips broken profiles silently
func TestProfileListCommand_DefaultOutputSkipsBrokenProfiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create a valid profile
	_ = fs.MkdirAll(homeDir+"/.weftlo/profiles/valid/profile", 0755)
	_ = afero.WriteFile(fs, homeDir+"/.weftlo/profiles/valid/profile/profile.yaml", []byte("name: valid/profile\n"), 0644)

	// Create a broken profile (directory exists but no profile.yaml)
	_ = fs.MkdirAll(homeDir+"/.weftlo/profiles/broken/profile", 0755)

	rootCmd := cli.NewRootCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)

	rootCmd.SetArgs([]string{"profile", "list"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()

	// Verify valid profile is shown
	if !strings.Contains(output, "valid/profile") {
		t.Errorf("expected output to contain 'valid/profile', got: %s", output)
	}

	// Verify broken profile is NOT shown (skipped silently)
	if strings.Contains(output, "broken/profile") {
		t.Errorf("expected output to NOT contain 'broken/profile' (broken profiles should be skipped), got: %s", output)
	}

	// Verify no "Broken profiles:" section in default output
	if strings.Contains(output, "Broken profiles:") {
		t.Errorf("expected output to NOT contain 'Broken profiles:' section in default mode, got: %s", output)
	}
}

// Test 4.1.3: Test verbose output includes "(inherits from ...)" suffix
func TestProfileListCommand_VerboseOutputIncludesInheritance(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create a profile with inheritance
	_ = fs.MkdirAll(homeDir+"/.weftlo/profiles/child/profile", 0755)
	profileYAML := `name: child/profile
inherits_from: parent/profile
`
	_ = afero.WriteFile(fs, homeDir+"/.weftlo/profiles/child/profile/profile.yaml", []byte(profileYAML), 0644)

	// Create a profile without inheritance
	_ = fs.MkdirAll(homeDir+"/.weftlo/profiles/standalone/profile", 0755)
	standaloneYAML := `name: standalone/profile
`
	_ = afero.WriteFile(fs, homeDir+"/.weftlo/profiles/standalone/profile/profile.yaml", []byte(standaloneYAML), 0644)

	rootCmd := cli.NewRootCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)

	rootCmd.SetArgs([]string{"--verbose", "profile", "list"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()

	// Verify inheritance info is shown for child profile
	if !strings.Contains(output, "(inherits from parent/profile)") {
		t.Errorf("expected output to contain '(inherits from parent/profile)', got: %s", output)
	}

	// Verify standalone profile is listed (without inheritance suffix)
	if !strings.Contains(output, "standalone/profile") {
		t.Errorf("expected output to contain 'standalone/profile', got: %s", output)
	}
}

// Test 4.1.4: Test verbose output groups broken profiles at bottom with header
func TestProfileListCommand_VerboseOutputGroupsBrokenProfiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create a valid profile
	_ = fs.MkdirAll(homeDir+"/.weftlo/profiles/valid/profile", 0755)
	_ = afero.WriteFile(fs, homeDir+"/.weftlo/profiles/valid/profile/profile.yaml", []byte("name: valid/profile\n"), 0644)

	// Create a broken profile (directory exists but no profile.yaml)
	_ = fs.MkdirAll(homeDir+"/.weftlo/profiles/broken/profile", 0755)

	rootCmd := cli.NewRootCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)

	rootCmd.SetArgs([]string{"--verbose", "profile", "list"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()

	// Verify "Broken profiles:" header is present
	if !strings.Contains(output, "Broken profiles:") {
		t.Errorf("expected output to contain 'Broken profiles:' header, got: %s", output)
	}

	// Verify broken profile is listed with reason
	if !strings.Contains(output, "broken/profile") {
		t.Errorf("expected output to contain 'broken/profile', got: %s", output)
	}

	// Verify reason is shown
	if !strings.Contains(output, "missing profile.yaml") {
		t.Errorf("expected output to contain 'missing profile.yaml' reason, got: %s", output)
	}

	// Verify broken profiles section comes after valid profiles section
	availableIndex := strings.Index(output, "Available profiles:")
	brokenIndex := strings.Index(output, "Broken profiles:")
	if availableIndex >= brokenIndex {
		t.Errorf("expected 'Broken profiles:' to come after 'Available profiles:', got: %s", output)
	}
}

// Test 4.1.5: Test JSON output returns object with `profiles` array
func TestProfileListCommand_JSONOutputReturnsProfilesArray(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create profiles
	_ = fs.MkdirAll(homeDir+"/.weftlo/profiles/vendor1/profile1", 0755)
	_ = afero.WriteFile(fs, homeDir+"/.weftlo/profiles/vendor1/profile1/profile.yaml", []byte("name: vendor1/profile1\n"), 0644)

	_ = fs.MkdirAll(homeDir+"/.weftlo/profiles/vendor2/profile2", 0755)
	_ = afero.WriteFile(fs, homeDir+"/.weftlo/profiles/vendor2/profile2/profile.yaml", []byte("name: vendor2/profile2\n"), 0644)

	rootCmd := cli.NewRootCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)

	rootCmd.SetArgs([]string{"profile", "list", "--json"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()

	// Parse JSON output
	var result struct {
		Profiles []struct {
			Name string `json:"name"`
		} `json:"profiles"`
	}

	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	// Verify profiles array exists and has correct count
	if len(result.Profiles) != 2 {
		t.Errorf("expected 2 profiles in JSON output, got %d", len(result.Profiles))
	}

	// Verify profile names
	expectedNames := []string{"vendor1/profile1", "vendor2/profile2"}
	for i, expected := range expectedNames {
		if i >= len(result.Profiles) {
			t.Errorf("missing profile at index %d", i)
			continue
		}
		if result.Profiles[i].Name != expected {
			t.Errorf("expected profile name %q at index %d, got %q", expected, i, result.Profiles[i].Name)
		}
	}
}

// Test 4.1.6: Test empty state shows "No profiles found." message
func TestProfileListCommand_EmptyStateShowsNoProfilesMessage(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create empty profiles directory
	_ = fs.MkdirAll(homeDir+"/.weftlo/profiles", 0755)

	rootCmd := cli.NewRootCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)

	// Test default output
	rootCmd.SetArgs([]string{"profile", "list"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "No profiles found.") {
		t.Errorf("expected output to contain 'No profiles found.', got: %s", output)
	}

	// Test JSON output for empty state
	stdout.Reset()
	rootCmd2 := cli.NewRootCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})
	rootCmd2.SetOut(&stdout)
	rootCmd2.SetErr(&stdout)
	rootCmd2.SetArgs([]string{"profile", "list", "--json"})

	err = rootCmd2.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jsonOutput := stdout.String()

	// Parse JSON and verify empty profiles array
	var result struct {
		Profiles []struct {
			Name string `json:"name"`
		} `json:"profiles"`
	}

	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, jsonOutput)
	}

	if len(result.Profiles) != 0 {
		t.Errorf("expected 0 profiles in JSON output for empty state, got %d", len(result.Profiles))
	}
}

// =============================================================================
// Task Group 5: Test Review and Gap Analysis - Additional Strategic Tests
// =============================================================================

// Note: mockFilesystem is already declared in profile_create_test.go
// Since both test files are in the same package (cli_test), we reuse it here.

// Test 5.3.1: XDG resolution test - verify correct config directory is used
// This test verifies that the profile list command respects XDG_CONFIG_HOME
// when resolving the config directory, per spec requirement:
// "Use ConfigDirResolver interface (injected via constructor) to resolve config directory"
func TestProfileListCommand_UsesXDGConfigDirectory(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create XDG config directory structure with profiles
	xdgConfigDir := "/custom/xdg/config"
	_ = fs.MkdirAll(xdgConfigDir+"/weftlo/profiles/xdgvendor/xdgprofile", 0755)
	_ = afero.WriteFile(fs, xdgConfigDir+"/weftlo/profiles/xdgvendor/xdgprofile/profile.yaml", []byte("name: xdgvendor/xdgprofile\n"), 0644)

	// Also create a profile in the fallback location to verify it's NOT used
	_ = fs.MkdirAll(homeDir+"/.weftlo/profiles/fallback/profile", 0755)
	_ = afero.WriteFile(fs, homeDir+"/.weftlo/profiles/fallback/profile/profile.yaml", []byte("name: fallback/profile\n"), 0644)

	// Create a mock filesystem for the resolver that returns XDG_CONFIG_HOME
	mockFs := &mockFilesystem{
		homeDir: homeDir,
		envVars: map[string]string{
			"XDG_CONFIG_HOME": xdgConfigDir,
		},
	}

	// Create a resolver with the mock filesystem
	resolver := infraconfig.NewResolver(infraconfig.WithFilesystem(mockFs))

	var stdout bytes.Buffer
	cmd := cli.NewProfileListCommandWithResolver(fs, func() (string, error) {
		return homeDir, nil
	}, nil, &stdout, resolver)

	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()

	// Verify profile from XDG config directory is listed
	if !strings.Contains(output, "xdgvendor/xdgprofile") {
		t.Errorf("expected output to contain 'xdgvendor/xdgprofile' from XDG config directory, got: %s", output)
	}

	// Verify profile from fallback location is NOT listed (because XDG takes precedence)
	if strings.Contains(output, "fallback/profile") {
		t.Errorf("expected output to NOT contain 'fallback/profile' (XDG_CONFIG_HOME should take precedence), got: %s", output)
	}
}

// Test 5.3.2: JSON output excludes broken profiles
// This test verifies that broken profiles are not included in JSON output,
// per spec requirement: "Only include valid profiles in JSON output (skip broken profiles)"
func TestProfileListCommand_JSONOutputExcludesBrokenProfiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create a valid profile
	_ = fs.MkdirAll(homeDir+"/.weftlo/profiles/valid/profile", 0755)
	_ = afero.WriteFile(fs, homeDir+"/.weftlo/profiles/valid/profile/profile.yaml", []byte("name: valid/profile\n"), 0644)

	// Create a broken profile (directory exists but no profile.yaml)
	_ = fs.MkdirAll(homeDir+"/.weftlo/profiles/broken/profile", 0755)

	rootCmd := cli.NewRootCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)

	rootCmd.SetArgs([]string{"profile", "list", "--json"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()

	// Parse JSON output
	var result struct {
		Profiles []struct {
			Name string `json:"name"`
		} `json:"profiles"`
	}

	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	// Verify only 1 profile in JSON output (the valid one)
	if len(result.Profiles) != 1 {
		t.Errorf("expected 1 profile in JSON output (broken profiles should be excluded), got %d", len(result.Profiles))
	}

	// Verify the profile is the valid one
	if len(result.Profiles) > 0 && result.Profiles[0].Name != "valid/profile" {
		t.Errorf("expected profile name 'valid/profile', got %q", result.Profiles[0].Name)
	}

	// Verify broken profile is not in JSON output by checking raw string
	if strings.Contains(output, "broken/profile") {
		t.Errorf("expected JSON output to NOT contain 'broken/profile', got: %s", output)
	}
}
