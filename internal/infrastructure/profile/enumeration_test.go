package profile

import (
	"testing"

	domaintemplate "github.com/simensen/weftlo/internal/domain/template"
)

// =============================================================================
// Task Group 1: Template Enumeration Core Logic Tests
// =============================================================================

// TestEnumerateOutputTemplates_ExcludesPartialFiles tests that enumeration
// correctly excludes partial files (files/directories starting with underscore).
func TestEnumerateOutputTemplates_ExcludesPartialFiles(t *testing.T) {
	t.Parallel()
	// Create a resolver with a mix of partial and non-partial files
	files := map[string]string{
		"README.md":                "/abs/README.md",
		"_partial.md":              "/abs/_partial.md",
		"_partials/header.md":      "/abs/_partials/header.md",
		"docs/_footer.md":          "/abs/docs/_footer.md",
		"docs/index.md":            "/abs/docs/index.md",
		"_components/nested/ui.md": "/abs/_components/nested/ui.md",
	}
	resolver := domaintemplate.NewMapFileResolver(files)

	result := EnumerateOutputTemplates(resolver)

	// Should only include non-partial files
	if len(result) != 2 {
		t.Fatalf("expected 2 output templates, got %d", len(result))
	}

	// Verify only non-partial files are included
	expectedPaths := map[string]bool{
		"README.md":     true,
		"docs/index.md": true,
	}

	for _, tf := range result {
		if !expectedPaths[tf.SourcePath] {
			t.Errorf("unexpected file included: %s", tf.SourcePath)
		}
		delete(expectedPaths, tf.SourcePath)
	}

	for path := range expectedPaths {
		t.Errorf("expected file missing: %s", path)
	}
}

// TestEnumerateOutputTemplates_IncludesDotfiles tests that enumeration
// correctly includes dotfiles (files starting with .) since they are now
// valid content inside the content root.
func TestEnumerateOutputTemplates_IncludesDotfiles(t *testing.T) {
	t.Parallel()
	// Create a resolver with dotfiles and regular files
	files := map[string]string{
		"README.md":             "/abs/README.md",
		".gitignore":            "/abs/.gitignore",
		".env":                  "/abs/.env",
		"config/.hidden":        "/abs/config/.hidden",
		"my.config/settings.md": "/abs/my.config/settings.md",
		"docs/api.md":           "/abs/docs/api.md",
	}
	resolver := domaintemplate.NewMapFileResolver(files)

	result := EnumerateOutputTemplates(resolver)

	// Should include ALL files including dotfiles (since content root only contains installable content)
	if len(result) != 6 {
		t.Fatalf("expected 6 output templates (including dotfiles), got %d", len(result))
	}

	// Verify all files are included
	expectedPaths := map[string]bool{
		"README.md":             true,
		".gitignore":            true,
		".env":                  true,
		"config/.hidden":        true,
		"my.config/settings.md": true,
		"docs/api.md":           true,
	}

	for _, tf := range result {
		if !expectedPaths[tf.SourcePath] {
			t.Errorf("unexpected file included: %s", tf.SourcePath)
		}
		delete(expectedPaths, tf.SourcePath)
	}

	for path := range expectedPaths {
		t.Errorf("expected file missing: %s", path)
	}
}

// TestEnumerateOutputTemplates_IncludesProfileYAML tests that enumeration
// includes profile.yaml when it exists in the content root. Since content
// enumeration only walks the content root directory, profile.yaml at the
// profile root (outside content) is never encountered. However, if someone
// puts a profile.yaml inside content, it IS enumerated.
func TestEnumerateOutputTemplates_IncludesProfileYAML(t *testing.T) {
	t.Parallel()
	// Create a resolver with profile.yaml and regular files
	// In the new architecture, if profile.yaml is in the resolver's view,
	// it means it was inside the content root and should be enumerated.
	files := map[string]string{
		"README.md":    "/abs/README.md",
		"profile.yaml": "/abs/profile.yaml",
		"docs/api.md":  "/abs/docs/api.md",
	}
	resolver := domaintemplate.NewMapFileResolver(files)

	result := EnumerateOutputTemplates(resolver)

	// Should include ALL files including profile.yaml (if in content root)
	if len(result) != 3 {
		t.Fatalf("expected 3 output templates, got %d", len(result))
	}

	// Verify all files are present
	expectedPaths := map[string]bool{
		"README.md":    true,
		"profile.yaml": true,
		"docs/api.md":  true,
	}

	for _, tf := range result {
		if !expectedPaths[tf.SourcePath] {
			t.Errorf("unexpected file included: %s", tf.SourcePath)
		}
	}
}

// TestEnumerateOutputTemplates_SortedAlphabetically tests that results
// are sorted alphabetically by SourcePath.
func TestEnumerateOutputTemplates_SortedAlphabetically(t *testing.T) {
	t.Parallel()
	// Create a resolver with files in non-alphabetical order
	files := map[string]string{
		"zebra.md":    "/abs/zebra.md",
		"apple.md":    "/abs/apple.md",
		"docs/zoo.md": "/abs/docs/zoo.md",
		"docs/ant.md": "/abs/docs/ant.md",
		"banana.md":   "/abs/banana.md",
	}
	resolver := domaintemplate.NewMapFileResolver(files)

	result := EnumerateOutputTemplates(resolver)

	// Verify count
	if len(result) != 5 {
		t.Fatalf("expected 5 output templates, got %d", len(result))
	}

	// Verify alphabetical ordering
	expectedOrder := []string{
		"apple.md",
		"banana.md",
		"docs/ant.md",
		"docs/zoo.md",
		"zebra.md",
	}

	for i, expected := range expectedOrder {
		if result[i].SourcePath != expected {
			t.Errorf("expected result[%d] to be '%s', got '%s'", i, expected, result[i].SourcePath)
		}
	}
}

// TestEnumerateOutputTemplates_EmptyResolverReturnsEmptySlice tests that
// an empty FileResolver returns an empty slice (not nil).
func TestEnumerateOutputTemplates_EmptyResolverReturnsEmptySlice(t *testing.T) {
	t.Parallel()
	// Create an empty resolver
	files := map[string]string{}
	resolver := domaintemplate.NewMapFileResolver(files)

	result := EnumerateOutputTemplates(resolver)

	// Should return empty slice, not nil
	if result == nil {
		t.Fatal("expected empty slice, got nil")
	}

	if len(result) != 0 {
		t.Errorf("expected 0 output templates, got %d", len(result))
	}
}

// TestEnumerateOutputTemplates_AllFilesHaveIsPartialFalse tests that
// all returned TemplateFile structs have IsPartial=false.
func TestEnumerateOutputTemplates_AllFilesHaveIsPartialFalse(t *testing.T) {
	t.Parallel()
	// Create a resolver with various files (only non-partials should be returned)
	files := map[string]string{
		"README.md":           "/abs/README.md",
		"docs/api.md":         "/abs/docs/api.md",
		"standards/code.md":   "/abs/standards/code.md",
		"_partials/header.md": "/abs/_partials/header.md", // Should be excluded
	}
	resolver := domaintemplate.NewMapFileResolver(files)

	result := EnumerateOutputTemplates(resolver)

	// Verify all returned files have IsPartial=false
	for _, tf := range result {
		if tf.IsPartial {
			t.Errorf("expected IsPartial=false for '%s', got true", tf.SourcePath)
		}
	}
}

// =============================================================================
// Task Group 2: Filtering Utilities Tests
// =============================================================================

// TestIsPartialFile_UnderscorePrefixedFile tests that isPartialFile correctly
// identifies files starting with underscore as partials.
func TestIsPartialFile_UnderscorePrefixedFile(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		path     string
		expected bool
	}{
		{"_partial.md", true},
		{"_header.md", true},
		{"__double.md", true},
		{"README.md", false},
		{"my_file.md", false}, // Underscore in middle, not prefix
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			result := isPartialFile(tc.path)
			if result != tc.expected {
				t.Errorf("isPartialFile(%q) = %v, expected %v", tc.path, result, tc.expected)
			}
		})
	}
}

// TestIsPartialFile_UnderscorePrefixedDirectory tests that isPartialFile correctly
// identifies files in underscore-prefixed directories as partials.
func TestIsPartialFile_UnderscorePrefixedDirectory(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		path     string
		expected bool
	}{
		{"_partials/header.md", true},
		{"_components/button.md", true},
		{"docs/_private/secret.md", true},
		{"_a/_b/_c/deep.md", true}, // Multiple underscore directories
		{"normal/regular.md", false},
		{"my_dir/file.md", false}, // Underscore in middle of dir name
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			result := isPartialFile(tc.path)
			if result != tc.expected {
				t.Errorf("isPartialFile(%q) = %v, expected %v", tc.path, result, tc.expected)
			}
		})
	}
}

// TestIsPartialFile_EdgeCases tests edge cases for isPartialFile.
func TestIsPartialFile_EdgeCases(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		path     string
		expected bool
	}{
		{"deeply nested partial dir", "a/b/c/_d/e/file.md", true},
		{"partial file in regular dir", "docs/api/_internal.md", true},
		{"underscore only filename", "_.md", true},
		{"underscore only dirname", "_/file.md", true},
		{"multiple underscores in middle", "my__file.md", false},
		{"trailing underscore", "file_.md", false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isPartialFile(tc.path)
			if result != tc.expected {
				t.Errorf("isPartialFile(%q) = %v, expected %v", tc.path, result, tc.expected)
			}
		})
	}
}

// =============================================================================
// Task Group 3: FileResolver Integration Tests
// =============================================================================

// TestEnumerateOutputTemplates_WorksWithMapFileResolver tests that enumeration
// works correctly with MapFileResolver populated with merged profile data.
func TestEnumerateOutputTemplates_WorksWithMapFileResolver(t *testing.T) {
	t.Parallel()
	// Simulate a merged profile with templates from multiple levels
	files := map[string]string{
		// From parent profile
		"shared/base.md":      "/home/user/.weftlo/profiles/acme/parent/shared/base.md",
		"_partials/common.md": "/home/user/.weftlo/profiles/acme/parent/_partials/common.md",
		// From child profile (overrides/additions)
		"README.md":           "/home/user/.weftlo/profiles/acme/child/README.md",
		"docs/api.md":         "/home/user/.weftlo/profiles/acme/child/docs/api.md",
		"_partials/header.md": "/home/user/.weftlo/profiles/acme/child/_partials/header.md",
	}
	resolver := domaintemplate.NewMapFileResolver(files)

	result := EnumerateOutputTemplates(resolver)

	// Should only include non-partial files
	if len(result) != 3 {
		t.Fatalf("expected 3 output templates, got %d", len(result))
	}

	// Verify expected files (sorted)
	expectedPaths := []string{
		"README.md",
		"docs/api.md",
		"shared/base.md",
	}

	for i, expected := range expectedPaths {
		if result[i].SourcePath != expected {
			t.Errorf("expected result[%d] to be '%s', got '%s'", i, expected, result[i].SourcePath)
		}
	}
}

// TestEnumerateOutputTemplates_ChildOverridesParent tests that the enumeration
// correctly handles child profile templates that override parent templates.
// Note: Override logic is handled by the FileResolver/MergedProfile, not enumeration.
func TestEnumerateOutputTemplates_ChildOverridesParent(t *testing.T) {
	t.Parallel()
	// When child overrides parent, the FileResolver only contains one entry per SourcePath.
	// The "override" already happened during MergedProfile construction.
	// Here we verify enumeration works with the resulting unified view.
	files := map[string]string{
		// This is the merged view - child's version won for README.md
		"README.md":        "/home/user/.weftlo/profiles/acme/child/README.md",
		"shared/common.md": "/home/user/.weftlo/profiles/acme/parent/shared/common.md",
		"docs/api.md":      "/home/user/.weftlo/profiles/acme/child/docs/api.md",
	}
	resolver := domaintemplate.NewMapFileResolver(files)

	result := EnumerateOutputTemplates(resolver)

	// All files should be enumerated (no duplicates since FileResolver handles overrides)
	if len(result) != 3 {
		t.Fatalf("expected 3 output templates, got %d", len(result))
	}

	// Verify sorted order
	expectedPaths := []string{
		"README.md",
		"docs/api.md",
		"shared/common.md",
	}

	for i, expected := range expectedPaths {
		if result[i].SourcePath != expected {
			t.Errorf("expected result[%d] to be '%s', got '%s'", i, expected, result[i].SourcePath)
		}
	}
}

// TestEnumerateOutputTemplates_RealisticMultiProfileScenario tests enumeration
// with a realistic multi-profile scenario including various file types.
// Note: Since content enumeration now walks only content root, the resolver
// only receives files from content root. Dotfiles and profile.yaml inside
// content ARE included.
func TestEnumerateOutputTemplates_RealisticMultiProfileScenario(t *testing.T) {
	t.Parallel()
	// Simulate a realistic merged profile with:
	// - Regular templates from parent and child
	// - Partial files that should be excluded
	// - Dotfiles that ARE included (valid content in content root)
	files := map[string]string{
		// Output templates
		"README.md":               "/profiles/child/README.md",
		"docs/getting-started.md": "/profiles/child/docs/getting-started.md",
		"docs/api/endpoints.md":   "/profiles/parent/docs/api/endpoints.md",
		"standards/code-style.md": "/profiles/parent/standards/code-style.md",
		"CLAUDE.md":               "/profiles/child/CLAUDE.md",
		// Partials (should be excluded)
		"_partials/header.md":       "/profiles/child/_partials/header.md",
		"_partials/footer.md":       "/profiles/parent/_partials/footer.md",
		"_components/navigation.md": "/profiles/child/_components/navigation.md",
		"docs/_internal/draft.md":   "/profiles/child/docs/_internal/draft.md",
		// Dotfiles (now included - valid content in content root)
		".gitignore":       "/profiles/parent/.gitignore",
		".env.example":     "/profiles/child/.env.example",
		"config/.settings": "/profiles/child/config/.settings",
	}
	resolver := domaintemplate.NewMapFileResolver(files)

	result := EnumerateOutputTemplates(resolver)

	// Should include output templates AND dotfiles (8 total), exclude only partials
	expectedCount := 8
	if len(result) != expectedCount {
		t.Fatalf("expected %d output templates, got %d", expectedCount, len(result))
	}

	// Verify sorted alphabetical order (including dotfiles which sort first due to '.')
	expectedPaths := []string{
		".env.example",
		".gitignore",
		"CLAUDE.md",
		"README.md",
		"config/.settings",
		"docs/api/endpoints.md",
		"docs/getting-started.md",
		"standards/code-style.md",
	}

	for i, expected := range expectedPaths {
		if result[i].SourcePath != expected {
			t.Errorf("expected result[%d] to be '%s', got '%s'", i, expected, result[i].SourcePath)
		}
	}

	// Verify all have correct IsPartial=false
	for _, tf := range result {
		if tf.IsPartial {
			t.Errorf("expected IsPartial=false for '%s'", tf.SourcePath)
		}
		if tf.TargetPath != "" {
			t.Errorf("expected TargetPath to be empty for '%s', got '%s'", tf.SourcePath, tf.TargetPath)
		}
		if tf.Checksum != "" {
			t.Errorf("expected Checksum to be empty for '%s', got '%s'", tf.SourcePath, tf.Checksum)
		}
	}
}

// TestEnumerateOutputTemplates_EmptyFileResolverReturnsEmptyResults tests that
// an empty FileResolver returns empty results (not nil).
func TestEnumerateOutputTemplates_EmptyFileResolverReturnsEmptyResults(t *testing.T) {
	t.Parallel()
	// Empty resolver
	resolver := domaintemplate.NewMapFileResolver(map[string]string{})

	result := EnumerateOutputTemplates(resolver)

	// Should return empty slice, not nil
	if result == nil {
		t.Fatal("expected empty slice, got nil")
	}

	if len(result) != 0 {
		t.Errorf("expected 0 output templates, got %d", len(result))
	}
}

// TestEnumerateOutputTemplates_TemplateFileFieldsCorrectlyPopulated tests that
// the returned TemplateFile structs have fields correctly populated/empty.
func TestEnumerateOutputTemplates_TemplateFileFieldsCorrectlyPopulated(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"README.md":   "/abs/README.md",
		"docs/api.md": "/abs/docs/api.md",
	}
	resolver := domaintemplate.NewMapFileResolver(files)

	result := EnumerateOutputTemplates(resolver)

	for _, tf := range result {
		// SourcePath should be populated
		if tf.SourcePath == "" {
			t.Error("expected SourcePath to be populated")
		}

		// TargetPath should be empty (computed in Rendering Pipeline phase)
		if tf.TargetPath != "" {
			t.Errorf("expected TargetPath to be empty for '%s', got '%s'", tf.SourcePath, tf.TargetPath)
		}

		// Checksum should be empty (computed in Manifest Generation phase)
		if tf.Checksum != "" {
			t.Errorf("expected Checksum to be empty for '%s', got '%s'", tf.SourcePath, tf.Checksum)
		}

		// IsPartial should be false for all returned files
		if tf.IsPartial {
			t.Errorf("expected IsPartial=false for '%s'", tf.SourcePath)
		}
	}
}

// =============================================================================
// Task Group 4: Additional Strategic Tests for Gap Analysis
// =============================================================================

// TestEnumerateOutputTemplates_DeeplyNestedDirectories tests enumeration
// with deeply nested directory structures (more than 5 levels).
func TestEnumerateOutputTemplates_DeeplyNestedDirectories(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"a/b/c/d/e/f/deep.md":       "/abs/a/b/c/d/e/f/deep.md",
		"a/b/c/d/e/f/g/h/deeper.md": "/abs/a/b/c/d/e/f/g/h/deeper.md",
		"root.md":                   "/abs/root.md",
		"a/_partial/nested/file.md": "/abs/a/_partial/nested/file.md", // Should be excluded (partial)
		"a/b/c/d/e/f/g/_private.md": "/abs/a/b/c/d/e/f/g/_private.md", // Should be excluded (partial)
	}
	resolver := domaintemplate.NewMapFileResolver(files)

	result := EnumerateOutputTemplates(resolver)

	// Should include only non-partial files
	if len(result) != 3 {
		t.Fatalf("expected 3 output templates, got %d", len(result))
	}

	// Verify sorted order
	expectedPaths := []string{
		"a/b/c/d/e/f/deep.md",
		"a/b/c/d/e/f/g/h/deeper.md",
		"root.md",
	}

	for i, expected := range expectedPaths {
		if result[i].SourcePath != expected {
			t.Errorf("expected result[%d] to be '%s', got '%s'", i, expected, result[i].SourcePath)
		}
	}
}

// TestEnumerateOutputTemplates_SpecialCharactersInPaths tests enumeration
// with special characters in file and directory names.
func TestEnumerateOutputTemplates_SpecialCharactersInPaths(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"file-with-dashes.md":        "/abs/file-with-dashes.md",
		"file_with_underscores.md":   "/abs/file_with_underscores.md", // Not partial (underscore in middle)
		"dir-name/file.md":           "/abs/dir-name/file.md",
		"spaces are weird/but ok.md": "/abs/spaces are weird/but ok.md",
		"123-numeric-prefix.md":      "/abs/123-numeric-prefix.md",
		"CamelCase/MixedCase.MD":     "/abs/CamelCase/MixedCase.MD",
	}
	resolver := domaintemplate.NewMapFileResolver(files)

	result := EnumerateOutputTemplates(resolver)

	// All files should be included (none are partials)
	if len(result) != 6 {
		t.Fatalf("expected 6 output templates, got %d", len(result))
	}

	// Verify all files are present
	expectedPaths := map[string]bool{
		"file-with-dashes.md":        true,
		"file_with_underscores.md":   true,
		"dir-name/file.md":           true,
		"spaces are weird/but ok.md": true,
		"123-numeric-prefix.md":      true,
		"CamelCase/MixedCase.MD":     true,
	}

	for _, tf := range result {
		if !expectedPaths[tf.SourcePath] {
			t.Errorf("unexpected file: %s", tf.SourcePath)
		}
		delete(expectedPaths, tf.SourcePath)
	}

	for path := range expectedPaths {
		t.Errorf("missing expected file: %s", path)
	}
}

// TestEnumerateOutputTemplates_OnlyPartialFiles tests enumeration when
// the resolver only contains partial files that should be excluded.
func TestEnumerateOutputTemplates_OnlyPartialFiles(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"_partial.md":         "/abs/_partial.md",
		"_partials/header.md": "/abs/_partials/header.md",
	}
	resolver := domaintemplate.NewMapFileResolver(files)

	result := EnumerateOutputTemplates(resolver)

	// Should return empty slice (all files are partials)
	if result == nil {
		t.Fatal("expected empty slice, got nil")
	}

	if len(result) != 0 {
		t.Errorf("expected 0 output templates, got %d", len(result))
	}
}

// TestEnumerateOutputTemplates_SortingWithMixedCase tests that sorting
// is case-sensitive and produces consistent results.
func TestEnumerateOutputTemplates_SortingWithMixedCase(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"Zebra.md": "/abs/Zebra.md",
		"apple.md": "/abs/apple.md",
		"Apple.md": "/abs/Apple.md",
		"zebra.md": "/abs/zebra.md",
		"APPLE.md": "/abs/APPLE.md",
	}
	resolver := domaintemplate.NewMapFileResolver(files)

	result := EnumerateOutputTemplates(resolver)

	if len(result) != 5 {
		t.Fatalf("expected 5 output templates, got %d", len(result))
	}

	// Verify sorting is consistent and case-sensitive
	// In ASCII/Unicode order: uppercase letters come before lowercase
	// A-Z (65-90) < a-z (97-122)
	expectedOrder := []string{
		"APPLE.md",
		"Apple.md",
		"Zebra.md",
		"apple.md",
		"zebra.md",
	}

	for i, expected := range expectedOrder {
		if result[i].SourcePath != expected {
			t.Errorf("expected result[%d] to be '%s', got '%s'", i, expected, result[i].SourcePath)
		}
	}
}

// TestEnumerateOutputTemplates_ConsistentResultsOnMultipleCalls tests that
// calling enumeration multiple times produces identical results.
func TestEnumerateOutputTemplates_ConsistentResultsOnMultipleCalls(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"README.md":   "/abs/README.md",
		"docs/api.md": "/abs/docs/api.md",
		"CLAUDE.md":   "/abs/CLAUDE.md",
	}
	resolver := domaintemplate.NewMapFileResolver(files)

	// Call enumeration multiple times
	result1 := EnumerateOutputTemplates(resolver)
	result2 := EnumerateOutputTemplates(resolver)
	result3 := EnumerateOutputTemplates(resolver)

	// All results should be identical
	if len(result1) != len(result2) || len(result2) != len(result3) {
		t.Fatal("results have different lengths across calls")
	}

	for i := range result1 {
		if result1[i].SourcePath != result2[i].SourcePath || result2[i].SourcePath != result3[i].SourcePath {
			t.Errorf("inconsistent result at index %d: %s, %s, %s",
				i, result1[i].SourcePath, result2[i].SourcePath, result3[i].SourcePath)
		}
	}
}
