package fs_test

import (
	"testing"

	"github.com/spf13/afero"

	"github.com/simensen/weftlo/internal/infrastructure/fs"
)

// Test 1: New() returns a valid afero.Fs instance
func TestNew_ReturnsValidAferoFs(t *testing.T) {
	t.Parallel()
	filesystem := fs.New()

	if filesystem == nil {
		t.Fatal("expected New() to return non-nil filesystem")
	}

	// Verify it implements afero.Fs interface
	var _ afero.Fs = filesystem
}

// Test 2: Basic file operations work through wrapper
func TestNew_BasicFileOperationsWork(t *testing.T) {
	t.Parallel()
	filesystem := fs.New()

	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	testFile := tmpDir + "/test.txt"
	testContent := []byte("hello world")

	// Write file
	err := afero.WriteFile(filesystem, testFile, testContent, 0644)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Read file
	content, err := afero.ReadFile(filesystem, testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(content) != string(testContent) {
		t.Errorf("expected content '%s', got '%s'", testContent, content)
	}

	// Stat file
	info, err := filesystem.Stat(testFile)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}
	if info.Size() != int64(len(testContent)) {
		t.Errorf("expected size %d, got %d", len(testContent), info.Size())
	}
}

// Test 3: afero.NewMemMapFs can be used interchangeably
func TestMemMapFs_Interchangeable(t *testing.T) {
	t.Parallel()
	// Create an in-memory filesystem
	memFs := afero.NewMemMapFs()

	testFile := "/virtual/test.txt"
	testContent := []byte("in-memory content")

	// Create directory
	err := memFs.MkdirAll("/virtual", 0755)
	if err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// Write file
	err = afero.WriteFile(memFs, testFile, testContent, 0644)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Read file
	content, err := afero.ReadFile(memFs, testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(content) != string(testContent) {
		t.Errorf("expected content '%s', got '%s'", testContent, content)
	}

	// Verify it satisfies the same interface as New()
	var osFs afero.Fs = fs.New()
	var inMemFs afero.Fs = memFs
	_ = osFs
	_ = inMemFs
}

// Test 4: Verify filesystem type is OS filesystem for production use
func TestNew_ReturnsOsFs(t *testing.T) {
	t.Parallel()
	filesystem := fs.New()

	// Verify filesystem is not nil and functions correctly
	// We test this by checking that the Name() method returns "OsFs"
	// which is the name used by afero.OsFs
	if name := filesystem.Name(); name != "OsFs" {
		t.Errorf("expected filesystem name 'OsFs', got '%s'", name)
	}
}
