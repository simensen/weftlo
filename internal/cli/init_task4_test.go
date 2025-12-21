package cli_test

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"

	"github.com/simensen/weftlo/internal/cli"
)

// =============================================================================
// Task Group 4: Init Command Content Directory Tests
// =============================================================================

// TestInitCommand_CreatesContentSubdirectory tests that the init command
// creates a content/ subdirectory inside the default profile directory.
func TestInitCommand_CreatesContentSubdirectory(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	initCmd := cli.NewInitCommandWithHomeDir(memFs, func() (string, error) {
		return homeDir, nil
	})

	initCmd.SetArgs([]string{})
	err := initCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Check that content/ directory exists inside the default profile
	contentDir := filepath.Join(homeDir, ".weftlo", "profiles", "default", "default", "content")
	exists, err := afero.DirExists(memFs, contentDir)
	if err != nil {
		t.Fatalf("error checking content directory existence: %v", err)
	}
	if !exists {
		t.Errorf("expected content directory %s to exist", contentDir)
	}
}

// TestInitCommand_PlacesReadmeInsideContentDirectory tests that the init command
// places the README.md file inside the content/ subdirectory.
func TestInitCommand_PlacesReadmeInsideContentDirectory(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	initCmd := cli.NewInitCommandWithHomeDir(memFs, func() (string, error) {
		return homeDir, nil
	})

	initCmd.SetArgs([]string{})
	err := initCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Check that README.md exists inside content/
	readmePath := filepath.Join(homeDir, ".weftlo", "profiles", "default", "default", "content", "README.md")
	exists, err := afero.Exists(memFs, readmePath)
	if err != nil {
		t.Fatalf("error checking README.md existence: %v", err)
	}
	if !exists {
		t.Errorf("expected README.md at %s to exist", readmePath)
	}

	// Verify README.md does NOT exist at old location (profile root)
	oldReadmePath := filepath.Join(homeDir, ".weftlo", "profiles", "default", "default", "README.md")
	exists, err = afero.Exists(memFs, oldReadmePath)
	if err != nil {
		t.Fatalf("error checking old README.md location: %v", err)
	}
	if exists {
		t.Errorf("expected README.md to NOT exist at old location %s", oldReadmePath)
	}
}

// TestInitCommand_ProfileYamlRemainsAtProfileRoot tests that the init command
// keeps profile.yaml at the profile root (not inside content/).
func TestInitCommand_ProfileYamlRemainsAtProfileRoot(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	initCmd := cli.NewInitCommandWithHomeDir(memFs, func() (string, error) {
		return homeDir, nil
	})

	initCmd.SetArgs([]string{})
	err := initCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Check that profile.yaml exists at profile root (correct location)
	profileYamlPath := filepath.Join(homeDir, ".weftlo", "profiles", "default", "default", "profile.yaml")
	exists, err := afero.Exists(memFs, profileYamlPath)
	if err != nil {
		t.Fatalf("error checking profile.yaml existence: %v", err)
	}
	if !exists {
		t.Errorf("expected profile.yaml at %s to exist", profileYamlPath)
	}

	// Verify profile.yaml does NOT exist inside content/ (wrong location)
	wrongProfileYamlPath := filepath.Join(homeDir, ".weftlo", "profiles", "default", "default", "content", "profile.yaml")
	exists, err = afero.Exists(memFs, wrongProfileYamlPath)
	if err != nil {
		t.Fatalf("error checking wrong profile.yaml location: %v", err)
	}
	if exists {
		t.Errorf("expected profile.yaml to NOT exist inside content/ at %s", wrongProfileYamlPath)
	}
}

// TestInitCommand_VerboseOutput_IncludesContentDirectory tests that verbose output
// mentions the content/ directory creation.
func TestInitCommand_VerboseOutput_IncludesContentDirectory(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	var stdout captureBuffer
	initCmd := cli.NewInitCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, nil, &stdout)

	initCmd.SetArgs([]string{"--verbose"})
	err := initCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()

	// Check that verbose output mentions the content directory
	contentDirPath := filepath.Join(homeDir, ".weftlo", "profiles", "default", "default", "content")
	if !containsSubstring(output, contentDirPath) && !containsSubstring(output, "content") {
		t.Errorf("expected verbose output to mention content directory, got: %s", output)
	}
}

// TestInitCommand_SuccessOutput_ShowsContentFiles tests that success output
// shows files inside content/ directory with correct relative paths.
func TestInitCommand_SuccessOutput_ShowsContentFiles(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	var stdout captureBuffer
	initCmd := cli.NewInitCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, nil, &stdout)

	initCmd.SetArgs([]string{})
	err := initCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()

	// Check that output shows content/ directory
	if !containsSubstring(output, "content/") {
		t.Errorf("expected success output to show 'content/' directory, got: %s", output)
	}

	// Check that README.md is shown inside content/
	if !containsSubstring(output, "content/README.md") {
		t.Errorf("expected success output to show 'content/README.md', got: %s", output)
	}
}

// captureBuffer is a simple buffer that implements io.Writer for capturing output.
type captureBuffer struct {
	data []byte
}

func (b *captureBuffer) Write(p []byte) (n int, err error) {
	b.data = append(b.data, p...)
	return len(p), nil
}

func (b *captureBuffer) String() string {
	return string(b.data)
}

// containsSubstring checks if str contains substr.
func containsSubstring(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || len(str) > 0 && containsSubstringHelper(str, substr))
}

func containsSubstringHelper(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
