package install_test

import (
	"strings"
	"testing"

	"github.com/simensen/weftlo/internal/app/dryrun"
	"github.com/simensen/weftlo/internal/app/install"
	"github.com/simensen/weftlo/internal/app/rendering"
)

// =============================================================================
// Operation Builder Tests (Task Group 3)
// =============================================================================

// Test: BuildFromRenderResults converts RenderResult to FileOperation correctly
func TestOperationBuilder_BuildFromRenderResults(t *testing.T) {
	t.Parallel()
	builder := install.NewOperationBuilder()

	renderResults := []rendering.RenderResult{
		{
			SourcePath: "template1.md.tmpl",
			TargetPath: "weftlo/template1.md",
			Content:    "Content of template 1",
		},
		{
			SourcePath: "nested/template2.md.tmpl",
			TargetPath: "weftlo/nested/template2.md",
			Content:    "Content of template 2",
		},
	}

	profileName := "vendor/my-profile"
	operations := builder.BuildFromRenderResults(renderResults, profileName)

	// Verify correct number of operations
	if len(operations) != 2 {
		t.Fatalf("expected 2 operations, got %d", len(operations))
	}

	// Verify first operation
	op1 := operations[0]
	if op1.SourceProfile != profileName {
		t.Errorf("op1 SourceProfile: expected '%s', got '%s'", profileName, op1.SourceProfile)
	}
	if op1.TargetPath != "weftlo/template1.md" {
		t.Errorf("op1 TargetPath: expected 'weftlo/template1.md', got '%s'", op1.TargetPath)
	}
	if string(op1.Content) != "Content of template 1" {
		t.Errorf("op1 Content: expected 'Content of template 1', got '%s'", string(op1.Content))
	}
	if op1.OperationType != dryrun.OperationTypeCreate {
		t.Errorf("op1 OperationType: expected OperationTypeCreate, got '%s'", op1.OperationType)
	}

	// Verify second operation
	op2 := operations[1]
	if op2.SourceProfile != profileName {
		t.Errorf("op2 SourceProfile: expected '%s', got '%s'", profileName, op2.SourceProfile)
	}
	if op2.TargetPath != "weftlo/nested/template2.md" {
		t.Errorf("op2 TargetPath: expected 'weftlo/nested/template2.md', got '%s'", op2.TargetPath)
	}
}

// Test: BuildFromRenderResults handles empty slice
func TestOperationBuilder_EmptySlice(t *testing.T) {
	t.Parallel()
	builder := install.NewOperationBuilder()

	renderResults := []rendering.RenderResult{}
	operations := builder.BuildFromRenderResults(renderResults, "profile")

	if len(operations) != 0 {
		t.Errorf("expected 0 operations for empty input, got %d", len(operations))
	}
}

// =============================================================================
// Task Group 4: Additional Strategic Tests for Operation Builder
// =============================================================================

// Test: BuildFromRenderResultsWithProjectDir creates absolute paths
func TestOperationBuilder_BuildFromRenderResultsWithProjectDir(t *testing.T) {
	t.Parallel()
	builder := install.NewOperationBuilder()

	renderResults := []rendering.RenderResult{
		{
			SourcePath: "template.md.tmpl",
			TargetPath: "weftlo/template.md",
			Content:    "Template content",
		},
		{
			SourcePath: "nested/deep/file.md.tmpl",
			TargetPath: "weftlo/nested/deep/file.md",
			Content:    "Nested content",
		},
	}

	projectDir := "/home/user/project"
	profileName := "vendor/test-profile"
	operations := builder.BuildFromRenderResultsWithProjectDir(renderResults, profileName, projectDir)

	// Verify correct number of operations
	if len(operations) != 2 {
		t.Fatalf("expected 2 operations, got %d", len(operations))
	}

	// Verify first operation has absolute path
	op1 := operations[0]
	if !strings.HasPrefix(op1.TargetPath, "/home/user/project") {
		t.Errorf("op1 TargetPath should start with project directory, got '%s'", op1.TargetPath)
	}
	if !strings.Contains(op1.TargetPath, "weftlo/template.md") {
		t.Errorf("op1 TargetPath should contain relative path, got '%s'", op1.TargetPath)
	}

	// Verify second operation has absolute path with nested structure
	op2 := operations[1]
	if !strings.HasPrefix(op2.TargetPath, "/home/user/project") {
		t.Errorf("op2 TargetPath should start with project directory, got '%s'", op2.TargetPath)
	}
	if !strings.Contains(op2.TargetPath, "nested/deep/file.md") {
		t.Errorf("op2 TargetPath should contain nested path, got '%s'", op2.TargetPath)
	}
}

// Test: BuildFromRenderResultsWithProjectDir normalizes paths to forward slashes
func TestOperationBuilder_BuildFromRenderResultsWithProjectDir_PathNormalization(t *testing.T) {
	t.Parallel()
	builder := install.NewOperationBuilder()

	renderResults := []rendering.RenderResult{
		{
			SourcePath: "file.md.tmpl",
			TargetPath: "weftlo/file.md",
			Content:    "content",
		},
	}

	// Even on systems that use backslashes, paths should be normalized
	projectDir := "/project/path"
	operations := builder.BuildFromRenderResultsWithProjectDir(renderResults, "profile", projectDir)

	if len(operations) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(operations))
	}

	// Verify path uses forward slashes
	if strings.Contains(operations[0].TargetPath, "\\") {
		t.Errorf("expected forward slashes in path, got: %s", operations[0].TargetPath)
	}
}
