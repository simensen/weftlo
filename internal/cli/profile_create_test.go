package cli_test

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"

	"github.com/simensen/weftlo/internal/cli"
	infraconfig "github.com/simensen/weftlo/internal/infrastructure/config"
)

// =============================================================================
// Task Group 1: Error Types and Core Command Structure Tests
// =============================================================================

// Test 1: ProfileExistsError.Error() returns descriptive message with profile name
func TestProfileExistsError_Error_ReturnsDescriptiveMessage(t *testing.T) {
	err := &cli.ProfileExistsError{
		ProfileName: "myvendor/myprofile",
	}

	errMsg := err.Error()

	// Verify message contains the profile name
	if !strings.Contains(errMsg, "myvendor/myprofile") {
		t.Errorf("expected error message to contain profile name 'myvendor/myprofile', got: %s", errMsg)
	}

	// Verify message indicates the profile already exists
	if !strings.Contains(errMsg, "already exists") {
		t.Errorf("expected error message to indicate profile 'already exists', got: %s", errMsg)
	}

	// Verify message suggests --force flag
	if !strings.Contains(errMsg, "--force") {
		t.Errorf("expected error message to suggest '--force' flag, got: %s", errMsg)
	}
}

// Test 2: ProfileExistsError.Unwrap() returns underlying error when present
func TestProfileExistsError_Unwrap_ReturnsUnderlyingError(t *testing.T) {
	t.Run("with underlying error", func(t *testing.T) {
		underlyingErr := errors.New("underlying error")
		err := &cli.ProfileExistsError{
			ProfileName: "vendor/name",
			Err:         underlyingErr,
		}

		unwrapped := err.Unwrap()
		if unwrapped != underlyingErr {
			t.Errorf("expected Unwrap() to return underlying error, got: %v", unwrapped)
		}
	})

	t.Run("without underlying error", func(t *testing.T) {
		err := &cli.ProfileExistsError{
			ProfileName: "vendor/name",
		}

		unwrapped := err.Unwrap()
		if unwrapped != nil {
			t.Errorf("expected Unwrap() to return nil when no underlying error, got: %v", unwrapped)
		}
	})
}

// Test 3: Command requires exactly one positional argument
func TestProfileCreateCommand_RequiresExactlyOneArgument(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create required home directory
	_ = fs.MkdirAll(homeDir, 0755)

	t.Run("no arguments returns error", func(t *testing.T) {
		cmd := cli.NewProfileCreateCommandWithHomeDir(fs, func() (string, error) {
			return homeDir, nil
		})

		var stdout bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stdout)
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		if err == nil {
			t.Error("expected error when no arguments provided")
		}
	})

	t.Run("two arguments returns error", func(t *testing.T) {
		cmd := cli.NewProfileCreateCommandWithHomeDir(fs, func() (string, error) {
			return homeDir, nil
		})

		var stdout bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stdout)
		cmd.SetArgs([]string{"vendor/first", "vendor/second"})

		err := cmd.Execute()
		if err == nil {
			t.Error("expected error when two arguments provided")
		}
	})

	t.Run("one argument accepted", func(t *testing.T) {
		cmd := cli.NewProfileCreateCommandWithHomeDir(fs, func() (string, error) {
			return homeDir, nil
		})

		var stdout bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stdout)
		cmd.SetArgs([]string{"vendor/name"})

		// This will proceed to profile creation, but we only care that it doesn't
		// fail on argument count validation. It may fail later (e.g., config not found).
		err := cmd.Execute()
		if err != nil {
			// Check that the error is NOT about argument count
			if strings.Contains(err.Error(), "accepts 1 arg") {
				t.Errorf("unexpected argument count error: %v", err)
			}
		}
	})
}

// Test 4: Command validates profile name format via ValidateProfileName()
func TestProfileCreateCommand_ValidatesProfileNameFormat(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create required directories for successful config resolution
	_ = fs.MkdirAll(homeDir+"/.weftlo", 0755)

	t.Run("valid profile name accepted", func(t *testing.T) {
		cmd := cli.NewProfileCreateCommandWithHomeDir(fs, func() (string, error) {
			return homeDir, nil
		})

		var stdout bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stdout)
		cmd.SetArgs([]string{"myvendor/myprofile"})

		err := cmd.Execute()
		// May fail for other reasons (like directory creation), but NOT for name validation
		if err != nil && strings.Contains(err.Error(), "profile name") && strings.Contains(err.Error(), "format") {
			t.Errorf("valid profile name should not trigger format validation error: %v", err)
		}
	})

	t.Run("profile name with underscores and dashes accepted", func(t *testing.T) {
		cmd := cli.NewProfileCreateCommandWithHomeDir(fs, func() (string, error) {
			return homeDir, nil
		})

		var stdout bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stdout)
		cmd.SetArgs([]string{"my_vendor-123/my-profile_456"})

		err := cmd.Execute()
		// May fail for other reasons, but NOT for name validation
		if err != nil && strings.Contains(err.Error(), "profile name") && strings.Contains(err.Error(), "format") {
			t.Errorf("valid profile name should not trigger format validation error: %v", err)
		}
	})
}

// Test 5: Invalid profile name formats return appropriate error messages
func TestProfileCreateCommand_InvalidProfileNameFormats(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create required home directory
	_ = fs.MkdirAll(homeDir+"/.weftlo", 0755)

	testCases := []struct {
		name        string
		profileName string
		expectError string
	}{
		{
			name:        "missing slash",
			profileName: "noslash",
			expectError: "vendor/name format",
		},
		{
			name:        "empty vendor",
			profileName: "/name",
			expectError: "vendor segment cannot be empty",
		},
		{
			name:        "empty name",
			profileName: "vendor/",
			expectError: "name segment cannot be empty",
		},
		{
			name:        "special characters in vendor",
			profileName: "ven@dor/name",
			expectError: "invalid characters",
		},
		{
			name:        "special characters in name",
			profileName: "vendor/na!me",
			expectError: "invalid characters",
		},
		{
			name:        "multiple slashes",
			profileName: "vendor/name/extra",
			expectError: "vendor/name format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := cli.NewProfileCreateCommandWithHomeDir(fs, func() (string, error) {
				return homeDir, nil
			})

			var stdout bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stdout)
			cmd.SetArgs([]string{tc.profileName})

			err := cmd.Execute()
			if err == nil {
				t.Errorf("expected error for invalid profile name '%s'", tc.profileName)
				return
			}

			if !strings.Contains(err.Error(), tc.expectError) {
				t.Errorf("expected error to contain '%s' for profile name '%s', got: %s",
					tc.expectError, tc.profileName, err.Error())
			}
		})
	}
}

// Test 6: ProfileExistsError satisfies error interface and has expected format
func TestProfileExistsError_SatisfiesErrorInterface(t *testing.T) {
	var err error = &cli.ProfileExistsError{
		ProfileName: "test/profile",
	}

	if err == nil {
		t.Error("ProfileExistsError should not be nil")
	}

	// Verify the error message format is user-friendly
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("expected Error() to return non-empty string")
	}
}

// =============================================================================
// Task Group 2: Directory and File Creation Tests
// =============================================================================

// Test 1: Creates nested directory at profiles/{vendor}/{name}/ with 0755 permissions
func TestProfileCreateCommand_CreatesNestedDirectory(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create required config directory
	_ = fs.MkdirAll(homeDir+"/.weftlo", 0755)

	cmd := cli.NewProfileCreateCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"myvendor/myprofile"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify directory structure was created
	expectedDir := homeDir + "/.weftlo/profiles/myvendor/myprofile"
	exists, err := afero.DirExists(fs, expectedDir)
	if err != nil {
		t.Fatalf("failed to check directory existence: %v", err)
	}
	if !exists {
		t.Errorf("expected directory %s to exist", expectedDir)
	}

	// Verify permissions (afero.MemMapFs doesn't fully support permission checks,
	// but we verify the directory was created)
	info, err := fs.Stat(expectedDir)
	if err != nil {
		t.Fatalf("failed to stat directory: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("expected %s to be a directory", expectedDir)
	}
}

// Test 2: Creates profile.yaml with correct content (name, inherits_from: "")
func TestProfileCreateCommand_CreatesProfileYamlWithCorrectContent(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create required config directory
	_ = fs.MkdirAll(homeDir+"/.weftlo", 0755)

	cmd := cli.NewProfileCreateCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"myvendor/myprofile"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify profile.yaml was created with correct content
	profilePath := homeDir + "/.weftlo/profiles/myvendor/myprofile/profile.yaml"
	content, err := afero.ReadFile(fs, profilePath)
	if err != nil {
		t.Fatalf("failed to read profile.yaml: %v", err)
	}

	contentStr := string(content)

	// Verify name field
	if !strings.Contains(contentStr, "name: myvendor/myprofile") {
		t.Errorf("expected profile.yaml to contain 'name: myvendor/myprofile', got:\n%s", contentStr)
	}

	// Verify inherits_from field is present (explicitly empty)
	if !strings.Contains(contentStr, "inherits_from:") {
		t.Errorf("expected profile.yaml to contain 'inherits_from:', got:\n%s", contentStr)
	}
}

// Test 3: Creates README.md with profile name in heading
func TestProfileCreateCommand_CreatesReadmeWithProfileName(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create required config directory
	_ = fs.MkdirAll(homeDir+"/.weftlo", 0755)

	cmd := cli.NewProfileCreateCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"myvendor/myprofile"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify README.md was created with profile name in heading
	readmePath := homeDir + "/.weftlo/profiles/myvendor/myprofile/README.md"
	content, err := afero.ReadFile(fs, readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	contentStr := string(content)

	// Verify heading contains profile name
	if !strings.Contains(contentStr, "# myvendor/myprofile") {
		t.Errorf("expected README.md to contain '# myvendor/myprofile' heading, got:\n%s", contentStr)
	}

	// Verify it has some placeholder description
	if len(contentStr) < 50 {
		t.Errorf("expected README.md to have placeholder content, got:\n%s", contentStr)
	}
}

// Test 4: Returns PermissionError when directory creation fails
func TestProfileCreateCommand_ReturnsPermissionErrorOnDirectoryFailure(t *testing.T) {
	// Use a read-only filesystem to simulate permission failure
	fs := afero.NewReadOnlyFs(afero.NewMemMapFs())
	homeDir := "/home/testuser"

	cmd := cli.NewProfileCreateCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"myvendor/myprofile"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when directory creation fails")
	}

	// Verify the error is a PermissionError
	var permErr *cli.PermissionError
	if !errors.As(err, &permErr) {
		t.Errorf("expected PermissionError, got: %T - %v", err, err)
	}

	// Verify error contains path information
	if permErr != nil && permErr.Path == "" {
		t.Error("expected PermissionError to include path")
	}
}

// Test 5: Returns PermissionError when file write fails
func TestProfileCreateCommand_ReturnsPermissionErrorOnFileWriteFailure(t *testing.T) {
	// Create a filesystem that allows directory creation but fails on file writes
	baseFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create the config directory so the resolver works
	_ = baseFs.MkdirAll(homeDir+"/.weftlo", 0755)

	// Wrap in a custom filesystem that fails on file creation
	fs := &failOnFileWriteFs{Fs: baseFs}

	cmd := cli.NewProfileCreateCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"myvendor/myprofile"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when file write fails")
	}

	// Verify the error is a PermissionError
	var permErr *cli.PermissionError
	if !errors.As(err, &permErr) {
		t.Errorf("expected PermissionError, got: %T - %v", err, err)
	}

	// Verify error contains path information
	if permErr != nil && permErr.Path == "" {
		t.Error("expected PermissionError to include path")
	}
}

// failOnFileWriteFs is a filesystem that allows directory operations but fails on file writes
type failOnFileWriteFs struct {
	afero.Fs
}

func (f *failOnFileWriteFs) Create(name string) (afero.File, error) {
	return nil, errors.New("permission denied: simulated file write failure")
}

func (f *failOnFileWriteFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	// Allow read-only opens
	if flag == 0 {
		return f.Fs.OpenFile(name, flag, perm)
	}
	return nil, errors.New("permission denied: simulated file write failure")
}

// Test 6: Verifies complete directory structure and file creation
func TestProfileCreateCommand_CreatesCompleteStructure(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create required config directory
	_ = fs.MkdirAll(homeDir+"/.weftlo", 0755)

	cmd := cli.NewProfileCreateCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"testvendor/testprofile"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all paths exist
	expectedPaths := []string{
		homeDir + "/.weftlo/profiles/testvendor/testprofile",
		homeDir + "/.weftlo/profiles/testvendor/testprofile/profile.yaml",
		homeDir + "/.weftlo/profiles/testvendor/testprofile/README.md",
	}

	for _, path := range expectedPaths {
		exists, err := afero.Exists(fs, path)
		if err != nil {
			t.Fatalf("failed to check existence of %s: %v", path, err)
		}
		if !exists {
			t.Errorf("expected %s to exist", path)
		}
	}
}

// =============================================================================
// Task Group 3: Profile Exists Check and Force Flag Tests
// =============================================================================

// Test 1: Returns ProfileExistsError when profile directory exists without --force
func TestProfileCreateCommand_ReturnsProfileExistsErrorWithoutForce(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create required config directory and existing profile directory
	_ = fs.MkdirAll(homeDir+"/.weftlo", 0755)
	existingProfileDir := homeDir + "/.weftlo/profiles/existing/profile"
	_ = fs.MkdirAll(existingProfileDir, 0755)

	cmd := cli.NewProfileCreateCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"existing/profile"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when profile directory already exists without --force")
	}

	// Verify the error is a ProfileExistsError
	var existsErr *cli.ProfileExistsError
	if !errors.As(err, &existsErr) {
		t.Errorf("expected ProfileExistsError, got: %T - %v", err, err)
	}

	// Verify error contains profile name
	if existsErr != nil && existsErr.ProfileName != "existing/profile" {
		t.Errorf("expected ProfileName to be 'existing/profile', got: %s", existsErr.ProfileName)
	}
}

// Test 2: With --force overwrites existing profile.yaml and README.md
func TestProfileCreateCommand_ForceOverwritesExistingFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create required config directory and existing profile with old content
	_ = fs.MkdirAll(homeDir+"/.weftlo", 0755)
	existingProfileDir := homeDir + "/.weftlo/profiles/existing/profile"
	_ = fs.MkdirAll(existingProfileDir, 0755)

	// Create existing files with old content
	oldProfileYaml := "name: old/name\ninherits_from: old/parent\n"
	oldReadme := "# Old Profile\n\nOld content.\n"
	_ = afero.WriteFile(fs, existingProfileDir+"/profile.yaml", []byte(oldProfileYaml), 0644)
	_ = afero.WriteFile(fs, existingProfileDir+"/README.md", []byte(oldReadme), 0644)

	cmd := cli.NewProfileCreateCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"--force", "existing/profile"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error with --force: %v", err)
	}

	// Verify profile.yaml was overwritten with new content
	profileYamlContent, err := afero.ReadFile(fs, existingProfileDir+"/profile.yaml")
	if err != nil {
		t.Fatalf("failed to read profile.yaml: %v", err)
	}
	if !strings.Contains(string(profileYamlContent), "name: existing/profile") {
		t.Errorf("expected profile.yaml to contain new name 'existing/profile', got:\n%s", string(profileYamlContent))
	}
	if strings.Contains(string(profileYamlContent), "old/name") {
		t.Errorf("expected profile.yaml to not contain old name 'old/name', got:\n%s", string(profileYamlContent))
	}

	// Verify README.md was overwritten with new content
	readmeContent, err := afero.ReadFile(fs, existingProfileDir+"/README.md")
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}
	if !strings.Contains(string(readmeContent), "# existing/profile Profile") {
		t.Errorf("expected README.md to contain new heading '# existing/profile Profile', got:\n%s", string(readmeContent))
	}
	if strings.Contains(string(readmeContent), "Old Profile") {
		t.Errorf("expected README.md to not contain old heading 'Old Profile', got:\n%s", string(readmeContent))
	}
}

// Test 3: With --force does not delete extra files in existing profile directory
func TestProfileCreateCommand_ForcePreservesExtraFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create required config directory and existing profile with extra files
	_ = fs.MkdirAll(homeDir+"/.weftlo", 0755)
	existingProfileDir := homeDir + "/.weftlo/profiles/existing/profile"
	_ = fs.MkdirAll(existingProfileDir, 0755)

	// Create existing profile files
	_ = afero.WriteFile(fs, existingProfileDir+"/profile.yaml", []byte("name: old\n"), 0644)
	_ = afero.WriteFile(fs, existingProfileDir+"/README.md", []byte("# Old\n"), 0644)

	// Create extra files that should be preserved
	extraContent := "This is custom template content"
	_ = afero.WriteFile(fs, existingProfileDir+"/custom-template.tmpl", []byte(extraContent), 0644)
	_ = afero.WriteFile(fs, existingProfileDir+"/notes.txt", []byte("important notes"), 0644)

	cmd := cli.NewProfileCreateCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"--force", "existing/profile"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error with --force: %v", err)
	}

	// Verify extra files still exist and have original content
	customTemplateContent, err := afero.ReadFile(fs, existingProfileDir+"/custom-template.tmpl")
	if err != nil {
		t.Fatalf("expected custom-template.tmpl to be preserved, got error: %v", err)
	}
	if string(customTemplateContent) != extraContent {
		t.Errorf("expected custom-template.tmpl content to be preserved, got: %s", string(customTemplateContent))
	}

	notesContent, err := afero.ReadFile(fs, existingProfileDir+"/notes.txt")
	if err != nil {
		t.Fatalf("expected notes.txt to be preserved, got error: %v", err)
	}
	if string(notesContent) != "important notes" {
		t.Errorf("expected notes.txt content to be preserved, got: %s", string(notesContent))
	}
}

// Test 4: Creates new profile when directory does not exist
func TestProfileCreateCommand_CreatesNewProfileWhenNotExists(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create required config directory but NOT the profile directory
	_ = fs.MkdirAll(homeDir+"/.weftlo", 0755)

	profileDir := homeDir + "/.weftlo/profiles/newvendor/newprofile"

	// Verify profile directory does not exist yet
	exists, _ := afero.DirExists(fs, profileDir)
	if exists {
		t.Fatal("test setup error: profile directory should not exist yet")
	}

	cmd := cli.NewProfileCreateCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"newvendor/newprofile"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify profile directory was created
	exists, err = afero.DirExists(fs, profileDir)
	if err != nil {
		t.Fatalf("failed to check directory existence: %v", err)
	}
	if !exists {
		t.Error("expected profile directory to be created")
	}

	// Verify profile.yaml was created
	exists, err = afero.Exists(fs, profileDir+"/profile.yaml")
	if err != nil {
		t.Fatalf("failed to check profile.yaml existence: %v", err)
	}
	if !exists {
		t.Error("expected profile.yaml to be created")
	}

	// Verify README.md was created
	exists, err = afero.Exists(fs, profileDir+"/README.md")
	if err != nil {
		t.Fatalf("failed to check README.md existence: %v", err)
	}
	if !exists {
		t.Error("expected README.md to be created")
	}
}

// Test 5: Short flag -f works the same as --force
func TestProfileCreateCommand_ShortForceFlagWorks(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create required config directory and existing profile directory
	_ = fs.MkdirAll(homeDir+"/.weftlo", 0755)
	existingProfileDir := homeDir + "/.weftlo/profiles/existing/profile"
	_ = fs.MkdirAll(existingProfileDir, 0755)
	_ = afero.WriteFile(fs, existingProfileDir+"/profile.yaml", []byte("name: old\n"), 0644)

	cmd := cli.NewProfileCreateCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"-f", "existing/profile"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error with -f flag: %v", err)
	}

	// Verify profile.yaml was overwritten
	profileYamlContent, err := afero.ReadFile(fs, existingProfileDir+"/profile.yaml")
	if err != nil {
		t.Fatalf("failed to read profile.yaml: %v", err)
	}
	if !strings.Contains(string(profileYamlContent), "name: existing/profile") {
		t.Errorf("expected profile.yaml to contain new name, got:\n%s", string(profileYamlContent))
	}
}

// =============================================================================
// Task Group 4: Command Wrapper and Root Integration Tests
// =============================================================================

// Test 1: Profile create command accessible via "weftlo profile create"
func TestProfileCreateCommand_AccessibleViaRootCommand(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create required config directory
	_ = fs.MkdirAll(homeDir+"/.weftlo", 0755)

	rootCmd := cli.NewRootCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)

	// Execute "weftlo profile create vendor/name"
	rootCmd.SetArgs([]string{"profile", "create", "testvendor/testprofile"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error executing 'profile create': %v", err)
	}

	// Verify profile was created
	profileDir := homeDir + "/.weftlo/profiles/testvendor/testprofile"
	exists, err := afero.DirExists(fs, profileDir)
	if err != nil {
		t.Fatalf("failed to check directory existence: %v", err)
	}
	if !exists {
		t.Error("expected profile directory to be created via root command")
	}
}

// Test 2: --quiet flag from root command suppresses success output
func TestProfileCreateCommand_QuietFlagSuppressesOutput(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create required config directory
	_ = fs.MkdirAll(homeDir+"/.weftlo", 0755)

	rootCmd := cli.NewRootCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)

	// Execute with --quiet flag
	rootCmd.SetArgs([]string{"--quiet", "profile", "create", "testvendor/quiettest"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify profile was created
	profileDir := homeDir + "/.weftlo/profiles/testvendor/quiettest"
	exists, _ := afero.DirExists(fs, profileDir)
	if !exists {
		t.Error("expected profile directory to be created")
	}

	// Verify output is suppressed (should not contain success message)
	output := stdout.String()
	if strings.Contains(output, "Created profile:") {
		t.Errorf("expected success output to be suppressed with --quiet flag, got: %s", output)
	}
}

// Test 3: --verbose flag from root command enables verbose output
func TestProfileCreateCommand_VerboseFlagEnablesOutput(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create required config directory
	_ = fs.MkdirAll(homeDir+"/.weftlo", 0755)

	rootCmd := cli.NewRootCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)

	// Execute with --verbose flag
	rootCmd.SetArgs([]string{"--verbose", "profile", "create", "testvendor/verbosetest"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify profile was created
	profileDir := homeDir + "/.weftlo/profiles/testvendor/verbosetest"
	exists, _ := afero.DirExists(fs, profileDir)
	if !exists {
		t.Error("expected profile directory to be created")
	}

	// Verify verbose output is present (should contain "Creating" messages)
	output := stdout.String()
	if !strings.Contains(output, "Creating") {
		t.Errorf("expected verbose output to contain 'Creating' messages, got: %s", output)
	}
}

// Test 4: Success output displays "Created profile: {vendor/name}" and file list
func TestProfileCreateCommand_SuccessOutputDisplaysProfileAndFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create required config directory
	_ = fs.MkdirAll(homeDir+"/.weftlo", 0755)

	rootCmd := cli.NewRootCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)

	// Execute without quiet flag
	rootCmd.SetArgs([]string{"profile", "create", "testvendor/successtest"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()

	// Verify "Created profile: {vendor/name}" message
	if !strings.Contains(output, "Created profile: testvendor/successtest") {
		t.Errorf("expected output to contain 'Created profile: testvendor/successtest', got: %s", output)
	}

	// Verify file list includes profile.yaml
	if !strings.Contains(output, "profile.yaml") {
		t.Errorf("expected output to list 'profile.yaml', got: %s", output)
	}

	// Verify file list includes README.md
	if !strings.Contains(output, "README.md") {
		t.Errorf("expected output to list 'README.md', got: %s", output)
	}
}

// =============================================================================
// Task Group 5: Test Review and Gap Analysis - Additional Strategic Tests
// =============================================================================

// mockFilesystem implements infraconfig.Filesystem for testing XDG resolution
type mockFilesystem struct {
	homeDir   string
	envVars   map[string]string
	existDirs map[string]bool
}

func (m *mockFilesystem) UserHomeDir() (string, error) {
	return m.homeDir, nil
}

func (m *mockFilesystem) GetEnv(key string) string {
	if m.envVars == nil {
		return ""
	}
	return m.envVars[key]
}

func (m *mockFilesystem) DirExists(path string) (bool, error) {
	if m.existDirs == nil {
		return false, nil
	}
	return m.existDirs[path], nil
}

// Test 1: XDG config directory resolution in profile path
func TestProfileCreateCommand_UsesXDGConfigDirectory(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create XDG config directory structure
	xdgConfigDir := "/custom/xdg/config"
	_ = fs.MkdirAll(xdgConfigDir+"/weftlo", 0755)

	// Create a mock filesystem for the resolver that returns XDG_CONFIG_HOME
	mockFs := &mockFilesystem{
		homeDir: homeDir,
		envVars: map[string]string{
			"XDG_CONFIG_HOME": xdgConfigDir,
		},
	}

	// Create a resolver with the mock filesystem
	resolver := infraconfig.NewResolver(infraconfig.WithFilesystem(mockFs))

	cmd := cli.NewProfileCreateCommandWithResolver(fs, func() (string, error) {
		return homeDir, nil
	}, os.Stdin, os.Stdout, resolver)

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"xdgvendor/xdgprofile"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify profile was created in the XDG config directory, not ~/.weftlo
	expectedProfileDir := xdgConfigDir + "/weftlo/profiles/xdgvendor/xdgprofile"
	exists, err := afero.DirExists(fs, expectedProfileDir)
	if err != nil {
		t.Fatalf("failed to check directory existence: %v", err)
	}
	if !exists {
		t.Errorf("expected profile directory at XDG path %s to exist", expectedProfileDir)
	}

	// Verify profile was NOT created in the fallback location
	fallbackProfileDir := homeDir + "/.weftlo/profiles/xdgvendor/xdgprofile"
	exists, _ = afero.DirExists(fs, fallbackProfileDir)
	if exists {
		t.Errorf("profile should NOT be created at fallback path %s when XDG_CONFIG_HOME is set", fallbackProfileDir)
	}
}

// Test 2: Profile.yaml content is valid YAML and can be unmarshaled
func TestProfileCreateCommand_ProfileYamlIsValidYAML(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create required config directory
	_ = fs.MkdirAll(homeDir+"/.weftlo", 0755)

	cmd := cli.NewProfileCreateCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"yamltest/profile"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read the profile.yaml file
	profilePath := homeDir + "/.weftlo/profiles/yamltest/profile/profile.yaml"
	content, err := afero.ReadFile(fs, profilePath)
	if err != nil {
		t.Fatalf("failed to read profile.yaml: %v", err)
	}

	// Verify content can be parsed as valid YAML
	var parsed struct {
		Name         string `yaml:"name"`
		InheritsFrom string `yaml:"inherits_from"`
	}
	err = yaml.Unmarshal(content, &parsed)
	if err != nil {
		t.Errorf("profile.yaml is not valid YAML: %v\nContent:\n%s", err, string(content))
	}

	// Verify parsed values are correct
	if parsed.Name != "yamltest/profile" {
		t.Errorf("expected name to be 'yamltest/profile', got '%s'", parsed.Name)
	}

	// inherits_from should be empty string (explicitly set, not omitted)
	if parsed.InheritsFrom != "" {
		t.Errorf("expected inherits_from to be empty string, got '%s'", parsed.InheritsFrom)
	}
}

// Test 3: Verbose mode output includes directory and file creation messages
func TestProfileCreateCommand_VerboseOutputIncludesCreationDetails(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create required config directory
	_ = fs.MkdirAll(homeDir+"/.weftlo", 0755)

	rootCmd := cli.NewRootCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)

	// Execute with --verbose flag
	rootCmd.SetArgs([]string{"--verbose", "profile", "create", "verbose/detailed"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()

	// Verify verbose output includes directory creation message
	if !strings.Contains(output, "Creating directory:") {
		t.Errorf("expected verbose output to contain 'Creating directory:', got:\n%s", output)
	}

	// Verify verbose output includes file creation messages
	if !strings.Contains(output, "Creating file:") {
		t.Errorf("expected verbose output to contain 'Creating file:', got:\n%s", output)
	}

	// Verify profile.yaml is mentioned in file creation
	if !strings.Contains(output, "profile.yaml") {
		t.Errorf("expected verbose output to mention 'profile.yaml', got:\n%s", output)
	}

	// Verify README.md is mentioned in file creation
	if !strings.Contains(output, "README.md") {
		t.Errorf("expected verbose output to mention 'README.md', got:\n%s", output)
	}
}

// Test 4: Quiet mode with --force still works correctly
func TestProfileCreateCommand_QuietModeWithForceWorks(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create required config directory and existing profile
	_ = fs.MkdirAll(homeDir+"/.weftlo", 0755)
	existingProfileDir := homeDir + "/.weftlo/profiles/quietforce/profile"
	_ = fs.MkdirAll(existingProfileDir, 0755)
	_ = afero.WriteFile(fs, existingProfileDir+"/profile.yaml", []byte("name: old\n"), 0644)
	_ = afero.WriteFile(fs, existingProfileDir+"/README.md", []byte("# Old\n"), 0644)

	rootCmd := cli.NewRootCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)

	// Execute with both --quiet and --force flags
	rootCmd.SetArgs([]string{"--quiet", "profile", "create", "--force", "quietforce/profile"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error with --quiet --force: %v", err)
	}

	// Verify profile was overwritten
	profileYamlContent, err := afero.ReadFile(fs, existingProfileDir+"/profile.yaml")
	if err != nil {
		t.Fatalf("failed to read profile.yaml: %v", err)
	}
	if !strings.Contains(string(profileYamlContent), "name: quietforce/profile") {
		t.Errorf("expected profile.yaml to be overwritten with new name, got:\n%s", string(profileYamlContent))
	}

	// Verify output is suppressed
	output := stdout.String()
	if strings.Contains(output, "Created profile:") {
		t.Errorf("expected output to be suppressed with --quiet flag, got:\n%s", output)
	}
}

// Test 5: Full end-to-end workflow from argument to files on disk
func TestProfileCreateCommand_EndToEndWorkflow(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Start with only the config directory
	_ = fs.MkdirAll(homeDir+"/.weftlo", 0755)

	// Execute the full command
	rootCmd := cli.NewRootCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{"profile", "create", "e2e-test/my-profile"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	profileDir := homeDir + "/.weftlo/profiles/e2e-test/my-profile"

	// Verify directory structure
	dirExists, err := afero.DirExists(fs, profileDir)
	if err != nil {
		t.Fatalf("failed to check directory: %v", err)
	}
	if !dirExists {
		t.Error("profile directory should exist")
	}

	// Verify profile.yaml exists and has correct content
	profileYaml, err := afero.ReadFile(fs, profileDir+"/profile.yaml")
	if err != nil {
		t.Fatalf("failed to read profile.yaml: %v", err)
	}
	if !strings.Contains(string(profileYaml), "name: e2e-test/my-profile") {
		t.Errorf("profile.yaml should contain correct name, got:\n%s", string(profileYaml))
	}
	if !strings.Contains(string(profileYaml), "inherits_from:") {
		t.Errorf("profile.yaml should contain inherits_from field, got:\n%s", string(profileYaml))
	}

	// Verify README.md exists and has correct content
	readme, err := afero.ReadFile(fs, profileDir+"/README.md")
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}
	if !strings.Contains(string(readme), "# e2e-test/my-profile") {
		t.Errorf("README.md should contain profile name in heading, got:\n%s", string(readme))
	}

	// Verify success output
	output := stdout.String()
	if !strings.Contains(output, "Created profile: e2e-test/my-profile") {
		t.Errorf("expected success message, got:\n%s", output)
	}
	if !strings.Contains(output, "Profile directory:") {
		t.Errorf("expected profile directory in output, got:\n%s", output)
	}
}

// Test 6: Error message quality for common mistakes
func TestProfileCreateCommand_ErrorMessageQuality(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create required config directory
	_ = fs.MkdirAll(homeDir+"/.weftlo", 0755)

	t.Run("no slash in name gives helpful message", func(t *testing.T) {
		cmd := cli.NewProfileCreateCommandWithHomeDir(fs, func() (string, error) {
			return homeDir, nil
		})

		var stdout bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stdout)
		cmd.SetArgs([]string{"myprofile"})

		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for profile name without slash")
		}

		errMsg := err.Error()
		// Verify error message is user-friendly
		if !strings.Contains(errMsg, "vendor/name") {
			t.Errorf("error message should mention vendor/name format, got: %s", errMsg)
		}
	})

	t.Run("existing profile without force gives actionable message", func(t *testing.T) {
		// Create existing profile
		existingDir := homeDir + "/.weftlo/profiles/existing/test"
		_ = fs.MkdirAll(existingDir, 0755)

		cmd := cli.NewProfileCreateCommandWithHomeDir(fs, func() (string, error) {
			return homeDir, nil
		})

		var stdout bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stdout)
		cmd.SetArgs([]string{"existing/test"})

		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for existing profile")
		}

		errMsg := err.Error()
		// Verify error message suggests --force
		if !strings.Contains(errMsg, "--force") {
			t.Errorf("error message should suggest --force flag, got: %s", errMsg)
		}
		// Verify error message mentions the profile name
		if !strings.Contains(errMsg, "existing/test") {
			t.Errorf("error message should mention profile name, got: %s", errMsg)
		}
	})
}
