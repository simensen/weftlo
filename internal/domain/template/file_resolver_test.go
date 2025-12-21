package template

import (
	"sort"
	"testing"
)

// Test 1: Resolve returns correct path for existing entry
func TestMapFileResolver_Resolve_ExistingEntry(t *testing.T) {
	// Arrange
	files := map[string]string{
		"_partials/header.md": "/profiles/vendor/myprofile/templates/_partials/header.md",
		"main.md":             "/profiles/vendor/myprofile/templates/main.md",
	}
	resolver := NewMapFileResolver(files)

	// Act
	absolutePath, found := resolver.Resolve("_partials/header.md")

	// Assert
	if !found {
		t.Fatal("expected found=true for existing entry, got false")
	}
	expected := "/profiles/vendor/myprofile/templates/_partials/header.md"
	if absolutePath != expected {
		t.Errorf("expected absolutePath %q, got %q", expected, absolutePath)
	}
}

// Test 2: Resolve returns false for non-existent entry
func TestMapFileResolver_Resolve_NonExistentEntry(t *testing.T) {
	// Arrange
	files := map[string]string{
		"_partials/header.md": "/profiles/vendor/myprofile/templates/_partials/header.md",
	}
	resolver := NewMapFileResolver(files)

	// Act
	absolutePath, found := resolver.Resolve("_partials/missing.md")

	// Assert
	if found {
		t.Fatal("expected found=false for non-existent entry, got true")
	}
	if absolutePath != "" {
		t.Errorf("expected empty absolutePath for non-existent entry, got %q", absolutePath)
	}
}

// Test 3: Resolution with multiple entries (simulates merged profile)
func TestMapFileResolver_Resolve_MultipleEntries(t *testing.T) {
	// Arrange - Simulate a merged profile where child overrides parent
	// The map represents the final merged state after inheritance resolution
	files := map[string]string{
		// Child profile overrides this partial
		"_partials/header.md": "/profiles/child/templates/_partials/header.md",
		// Parent profile provides this partial (inherited)
		"_partials/footer.md": "/profiles/parent/templates/_partials/footer.md",
		// Grandparent profile provides this partial (inherited through chain)
		"_partials/sidebar.md": "/profiles/grandparent/templates/_partials/sidebar.md",
		// Child profile's main template
		"main.md": "/profiles/child/templates/main.md",
	}
	resolver := NewMapFileResolver(files)

	// Act & Assert - Verify all entries resolve correctly
	testCases := []struct {
		sourcePath   string
		expectedPath string
	}{
		{"_partials/header.md", "/profiles/child/templates/_partials/header.md"},
		{"_partials/footer.md", "/profiles/parent/templates/_partials/footer.md"},
		{"_partials/sidebar.md", "/profiles/grandparent/templates/_partials/sidebar.md"},
		{"main.md", "/profiles/child/templates/main.md"},
	}

	for _, tc := range testCases {
		absolutePath, found := resolver.Resolve(tc.sourcePath)
		if !found {
			t.Errorf("expected found=true for %q, got false", tc.sourcePath)
			continue
		}
		if absolutePath != tc.expectedPath {
			t.Errorf("for %q: expected %q, got %q", tc.sourcePath, tc.expectedPath, absolutePath)
		}
	}
}

// Test 4: Empty resolver returns false for any path
func TestMapFileResolver_Resolve_EmptyResolver(t *testing.T) {
	// Arrange
	resolver := NewMapFileResolver(map[string]string{})

	// Act
	absolutePath, found := resolver.Resolve("any/path.md")

	// Assert
	if found {
		t.Fatal("expected found=false for empty resolver, got true")
	}
	if absolutePath != "" {
		t.Errorf("expected empty absolutePath for empty resolver, got %q", absolutePath)
	}
}

// Test 5: ListSourcePaths returns all keys from MapFileResolver
func TestMapFileResolver_ListSourcePaths_ReturnsAllKeys(t *testing.T) {
	// Arrange
	files := map[string]string{
		"_partials/header.md": "/profiles/vendor/myprofile/templates/_partials/header.md",
		"_partials/footer.md": "/profiles/vendor/myprofile/templates/_partials/footer.md",
		"_skills/coding.md":   "/profiles/vendor/myprofile/templates/_skills/coding.md",
		"main.md":             "/profiles/vendor/myprofile/templates/main.md",
	}
	resolver := NewMapFileResolver(files)

	// Act
	paths := resolver.ListSourcePaths()

	// Assert
	if len(paths) != 4 {
		t.Fatalf("expected 4 paths, got %d", len(paths))
	}

	// Sort both for comparison since map iteration order is not guaranteed
	sort.Strings(paths)
	expected := []string{
		"_partials/footer.md",
		"_partials/header.md",
		"_skills/coding.md",
		"main.md",
	}

	for i, p := range paths {
		if p != expected[i] {
			t.Errorf("at index %d: expected %q, got %q", i, expected[i], p)
		}
	}
}

// Test 6: ListSourcePaths returns empty slice for empty resolver
func TestMapFileResolver_ListSourcePaths_EmptyResolver(t *testing.T) {
	// Arrange
	resolver := NewMapFileResolver(map[string]string{})

	// Act
	paths := resolver.ListSourcePaths()

	// Assert
	if paths == nil {
		t.Fatal("expected non-nil slice, got nil")
	}
	if len(paths) != 0 {
		t.Errorf("expected empty slice, got %d elements", len(paths))
	}
}

// Test 7: ListSourcePaths returns unsorted keys (ordering done by caller)
func TestMapFileResolver_ListSourcePaths_NoOrderingGuarantee(t *testing.T) {
	// Arrange
	files := map[string]string{
		"z.md": "/path/z.md",
		"a.md": "/path/a.md",
		"m.md": "/path/m.md",
	}
	resolver := NewMapFileResolver(files)

	// Act
	paths := resolver.ListSourcePaths()

	// Assert - We just verify all keys are present, not their order
	// The interface makes no ordering guarantee
	if len(paths) != 3 {
		t.Fatalf("expected 3 paths, got %d", len(paths))
	}

	// Check all keys are present (order-independent)
	pathSet := make(map[string]bool)
	for _, p := range paths {
		pathSet[p] = true
	}

	for key := range files {
		if !pathSet[key] {
			t.Errorf("expected key %q to be present in ListSourcePaths result", key)
		}
	}
}
