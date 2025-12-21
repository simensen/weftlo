package template

import (
	"errors"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/spf13/afero"

	"github.com/simensen/weftlo/internal/domain/profile"
)

// Test 1: Basic glob matching returns concatenated content with newline separator
func TestIncludeGlobFunc_BasicGlobMatchingWithNewlineSeparator(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/templates/_skills/skill1.md", []byte("Skill 1 content"), 0644)
	afero.WriteFile(fs, "/templates/_skills/skill2.md", []byte("Skill 2 content"), 0644)
	afero.WriteFile(fs, "/templates/_skills/skill3.md", []byte("Skill 3 content"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"_skills/skill1.md": "/templates/_skills/skill1.md",
		"_skills/skill2.md": "/templates/_skills/skill2.md",
		"_skills/skill3.md": "/templates/_skills/skill3.md",
	})

	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeGlobFunc := createIncludeGlobFunc(renderCtx, tmplCtx)

	// Act
	result, err := includeGlobFunc("_skills/*.md")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "Skill 1 content\nSkill 2 content\nSkill 3 content"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// Test 2: Alphabetical ordering (case-sensitive: A-Z before a-z)
func TestIncludeGlobFunc_AlphabeticalOrdering(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	// Files with names that test case-sensitive ordering
	afero.WriteFile(fs, "/templates/files/Alpha.md", []byte("Alpha"), 0644)
	afero.WriteFile(fs, "/templates/files/Beta.md", []byte("Beta"), 0644)
	afero.WriteFile(fs, "/templates/files/alpha.md", []byte("alpha"), 0644)
	afero.WriteFile(fs, "/templates/files/beta.md", []byte("beta"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"files/Alpha.md": "/templates/files/Alpha.md",
		"files/Beta.md":  "/templates/files/Beta.md",
		"files/alpha.md": "/templates/files/alpha.md",
		"files/beta.md":  "/templates/files/beta.md",
	})

	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeGlobFunc := createIncludeGlobFunc(renderCtx, tmplCtx)

	// Act
	result, err := includeGlobFunc("files/*.md")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Case-sensitive: A-Z (65-90) comes before a-z (97-122)
	expected := "Alpha\nBeta\nalpha\nbeta"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// Test 3: Empty result when no files match (returns empty string, no error)
func TestIncludeGlobFunc_NoMatchReturnsEmptyString(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/templates/other/file.txt", []byte("content"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"other/file.txt": "/templates/other/file.txt",
	})

	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeGlobFunc := createIncludeGlobFunc(renderCtx, tmplCtx)

	// Act
	result, err := includeGlobFunc("nonexistent/*.md")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

// Test 4: Pattern syntax: * matches single segment, ? matches single char
func TestIncludeGlobFunc_PatternSyntax(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	// Use distinct file names to avoid ambiguity in pattern matching
	// item1.md, item2.md match item?.md (? = single char)
	// items.md does NOT match item?.md (s is single char, but "items" matches)
	// Wait - "items" does match "item?" because s is a single char
	// Use longer suffix to distinguish: item01.md, item02.md (two digits)
	afero.WriteFile(fs, "/templates/docs/item01.md", []byte("item01"), 0644)
	afero.WriteFile(fs, "/templates/docs/item02.md", []byte("item02"), 0644)
	afero.WriteFile(fs, "/templates/docs/item1.md", []byte("item1"), 0644)
	afero.WriteFile(fs, "/templates/docs/items.md", []byte("items"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"docs/item01.md": "/templates/docs/item01.md",
		"docs/item02.md": "/templates/docs/item02.md",
		"docs/item1.md":  "/templates/docs/item1.md",
		"docs/items.md":  "/templates/docs/items.md",
	})

	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeGlobFunc := createIncludeGlobFunc(renderCtx, tmplCtx)

	// Test ? matches exactly one character
	// Pattern "docs/item?.md" matches item1.md and items.md (single char after "item")
	// but NOT item01.md or item02.md (two chars after "item")
	t.Run("question mark matches single char", func(t *testing.T) {
		result, err := includeGlobFunc("docs/item?.md")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		// item1.md and items.md match (sorted alphabetically)
		expected := "item1\nitems"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	// Test * matches any sequence (including multi-char)
	t.Run("asterisk matches any sequence", func(t *testing.T) {
		result, err := includeGlobFunc("docs/item*.md")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		// All four files match (sorted alphabetically)
		expected := "item01\nitem02\nitem1\nitems"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}

// Test 5: Depth tracking integration (files share parent depth level)
func TestIncludeGlobFunc_DepthTrackingIntegration(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	// Create files that themselves use include, to test depth tracking
	afero.WriteFile(fs, "/templates/partials/part1.md", []byte("Part1:{{ include \"leaf.md\" }}"), 0644)
	afero.WriteFile(fs, "/templates/partials/part2.md", []byte("Part2:{{ include \"leaf.md\" }}"), 0644)
	afero.WriteFile(fs, "/templates/leaf.md", []byte("Leaf"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"partials/part1.md": "/templates/partials/part1.md",
		"partials/part2.md": "/templates/partials/part2.md",
		"leaf.md":           "/templates/leaf.md",
	})

	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeGlobFunc := createIncludeGlobFunc(renderCtx, tmplCtx)

	// Act
	result, err := includeGlobFunc("partials/*.md")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "Part1:Leaf\nPart2:Leaf"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// Test 6: Fail-fast on first file error (syntax or execution)
func TestIncludeGlobFunc_FailFastOnError(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/templates/files/a_good.md", []byte("Good content"), 0644)
	afero.WriteFile(fs, "/templates/files/b_bad.md", []byte("{{ .Invalid.Syntax"), 0644) // Invalid template
	afero.WriteFile(fs, "/templates/files/c_good.md", []byte("Should not reach"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"files/a_good.md": "/templates/files/a_good.md",
		"files/b_bad.md":  "/templates/files/b_bad.md",
		"files/c_good.md": "/templates/files/c_good.md",
	})

	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeGlobFunc := createIncludeGlobFunc(renderCtx, tmplCtx)

	// Act
	_, err := includeGlobFunc("files/*.md")

	// Assert - should get a syntax error for b_bad.md
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var syntaxErr *IncludeSyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Fatalf("expected IncludeSyntaxError, got %T: %v", err, err)
	}

	if syntaxErr.SourcePath != "files/b_bad.md" {
		t.Errorf("expected SourcePath %q, got %q", "files/b_bad.md", syntaxErr.SourcePath)
	}
}

// Test 7: Invalid glob pattern returns IncludeGlobPatternError
func TestIncludeGlobFunc_InvalidPatternReturnsError(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	resolver := NewMapFileResolver(map[string]string{})

	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeGlobFunc := createIncludeGlobFunc(renderCtx, tmplCtx)

	// Act - use a malformed character class which is an invalid pattern
	_, err := includeGlobFunc("[invalid")

	// Assert
	if err == nil {
		t.Fatal("expected IncludeGlobPatternError, got nil")
	}

	var patternErr *IncludeGlobPatternError
	if !errors.As(err, &patternErr) {
		t.Fatalf("expected IncludeGlobPatternError, got %T: %v", err, err)
	}

	if patternErr.Pattern != "[invalid" {
		t.Errorf("expected Pattern %q, got %q", "[invalid", patternErr.Pattern)
	}

	// Verify it wraps filepath.ErrBadPattern
	if !errors.Is(patternErr.Err, filepath.ErrBadPattern) {
		t.Errorf("expected wrapped error to be filepath.ErrBadPattern, got %v", patternErr.Err)
	}
}

// Test 8: Depth limit reached triggers IncludeDepthError
func TestIncludeGlobFunc_DepthLimitReturnsError(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/templates/files/file.md", []byte("Content"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"files/file.md": "/templates/files/file.md",
	})

	// Create a render context already at max depth
	renderCtx := &RenderContext{
		Fs:           fs,
		FileResolver: resolver,
		CurrentDepth: 10, // Already at max
		MaxDepth:     10,
		IncludeChain: []string{"level1.md", "level2.md"},
	}
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeGlobFunc := createIncludeGlobFunc(renderCtx, tmplCtx)

	// Act
	_, err := includeGlobFunc("files/*.md")

	// Assert
	if err == nil {
		t.Fatal("expected IncludeDepthError, got nil")
	}

	var depthErr *IncludeDepthError
	if !errors.As(err, &depthErr) {
		t.Fatalf("expected IncludeDepthError, got %T: %v", err, err)
	}

	if depthErr.MaxDepth != 10 {
		t.Errorf("expected MaxDepth 10, got %d", depthErr.MaxDepth)
	}
}

// ============================================================================
// Strategic Gap-Filling Tests (Task Group 4)
// ============================================================================

// Test 9: Character class glob patterns [abc] matching
func TestIncludeGlobFunc_CharacterClassMatching(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/templates/files/file_a.md", []byte("FileA"), 0644)
	afero.WriteFile(fs, "/templates/files/file_b.md", []byte("FileB"), 0644)
	afero.WriteFile(fs, "/templates/files/file_c.md", []byte("FileC"), 0644)
	afero.WriteFile(fs, "/templates/files/file_x.md", []byte("FileX"), 0644)
	afero.WriteFile(fs, "/templates/files/file_y.md", []byte("FileY"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"files/file_a.md": "/templates/files/file_a.md",
		"files/file_b.md": "/templates/files/file_b.md",
		"files/file_c.md": "/templates/files/file_c.md",
		"files/file_x.md": "/templates/files/file_x.md",
		"files/file_y.md": "/templates/files/file_y.md",
	})

	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeGlobFunc := createIncludeGlobFunc(renderCtx, tmplCtx)

	// Act - pattern [abc] matches only a, b, or c
	result, err := includeGlobFunc("files/file_[abc].md")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Should match file_a.md, file_b.md, file_c.md but not file_x.md or file_y.md
	expected := "FileA\nFileB\nFileC"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// Test 10: IncludeChain populated correctly in error scenarios
func TestIncludeGlobFunc_IncludeChainInDepthError(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/templates/files/file.md", []byte("Content"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"files/file.md": "/templates/files/file.md",
	})

	// Create a render context with an existing include chain
	renderCtx := &RenderContext{
		Fs:           fs,
		FileResolver: resolver,
		CurrentDepth: 10,
		MaxDepth:     10,
		IncludeChain: []string{"root.md", "middle.md", "parent.md"},
	}
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeGlobFunc := createIncludeGlobFunc(renderCtx, tmplCtx)

	// Act
	_, err := includeGlobFunc("files/*.md")

	// Assert
	if err == nil {
		t.Fatal("expected IncludeDepthError, got nil")
	}

	var depthErr *IncludeDepthError
	if !errors.As(err, &depthErr) {
		t.Fatalf("expected IncludeDepthError, got %T: %v", err, err)
	}

	// Verify the include chain contains the original chain plus the attempted file
	expectedChain := []string{"root.md", "middle.md", "parent.md", "files/file.md"}
	if len(depthErr.IncludeChain) != len(expectedChain) {
		t.Errorf("expected IncludeChain length %d, got %d: %v", len(expectedChain), len(depthErr.IncludeChain), depthErr.IncludeChain)
	}
	for i, expected := range expectedChain {
		if i < len(depthErr.IncludeChain) && depthErr.IncludeChain[i] != expected {
			t.Errorf("expected IncludeChain[%d] = %q, got %q", i, expected, depthErr.IncludeChain[i])
		}
	}
}

// Test 11: Multiple includeGlob calls in same template
func TestIncludeGlobFunc_MultipleCallsInSameContext(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/templates/_skills/skill1.md", []byte("Skill1"), 0644)
	afero.WriteFile(fs, "/templates/_skills/skill2.md", []byte("Skill2"), 0644)
	afero.WriteFile(fs, "/templates/_docs/doc1.md", []byte("Doc1"), 0644)
	afero.WriteFile(fs, "/templates/_docs/doc2.md", []byte("Doc2"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"_skills/skill1.md": "/templates/_skills/skill1.md",
		"_skills/skill2.md": "/templates/_skills/skill2.md",
		"_docs/doc1.md":     "/templates/_docs/doc1.md",
		"_docs/doc2.md":     "/templates/_docs/doc2.md",
	})

	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeGlobFunc := createIncludeGlobFunc(renderCtx, tmplCtx)

	// Act - call includeGlob twice with different patterns
	result1, err1 := includeGlobFunc("_skills/*.md")
	result2, err2 := includeGlobFunc("_docs/*.md")

	// Assert
	if err1 != nil {
		t.Fatalf("first call: expected no error, got %v", err1)
	}
	if err2 != nil {
		t.Fatalf("second call: expected no error, got %v", err2)
	}

	expected1 := "Skill1\nSkill2"
	expected2 := "Doc1\nDoc2"

	if result1 != expected1 {
		t.Errorf("first call: expected %q, got %q", expected1, result1)
	}
	if result2 != expected2 {
		t.Errorf("second call: expected %q, got %q", expected2, result2)
	}
}

// Test 12: includeGlob with pattern matching zero files followed by include
func TestIncludeGlobFunc_NoMatchFollowedByInclude(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/templates/existing.md", []byte("Existing content"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"existing.md": "/templates/existing.md",
	})

	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeGlobFunc := createIncludeGlobFunc(renderCtx, tmplCtx)
	includeFunc := createIncludeFunc(renderCtx, tmplCtx)

	// Act - first call includeGlob with non-matching pattern
	globResult, globErr := includeGlobFunc("nonexistent/*.md")
	// Then call include for an existing file
	includeResult, includeErr := includeFunc("existing.md")

	// Assert
	if globErr != nil {
		t.Fatalf("includeGlob: expected no error, got %v", globErr)
	}
	if includeErr != nil {
		t.Fatalf("include: expected no error, got %v", includeErr)
	}

	if globResult != "" {
		t.Errorf("includeGlob: expected empty string, got %q", globResult)
	}
	if includeResult != "Existing content" {
		t.Errorf("include: expected %q, got %q", "Existing content", includeResult)
	}
}

// Test 13: Depth limit reached within includeGlob batch (not pre-exhausted)
func TestIncludeGlobFunc_DepthLimitReachedMidBatch(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	// Create files where the second file triggers a deeply nested include that exceeds depth
	// file_a.md is simple, file_b.md creates a chain that exceeds depth
	afero.WriteFile(fs, "/templates/batch/file_a.md", []byte("FileA"), 0644)
	// file_b.md includes level0 which starts a chain
	afero.WriteFile(fs, "/templates/batch/file_b.md", []byte("{{ include \"deep/level0.md\" }}"), 0644)

	// Create a chain of files that will exceed depth when combined with the batch depth
	// Starting depth: 0, includeGlob increments to 1 for each file in batch
	// From file_b.md at depth 1, we need to go 9 more levels to reach depth 10
	for i := 0; i <= 10; i++ {
		var content string
		if i < 10 {
			content = "{{ include \"deep/level" + strconv.Itoa(i+1) + ".md\" }}"
		} else {
			content = "end"
		}
		afero.WriteFile(fs, "/templates/deep/level"+strconv.Itoa(i)+".md", []byte(content), 0644)
	}

	fileMap := map[string]string{
		"batch/file_a.md": "/templates/batch/file_a.md",
		"batch/file_b.md": "/templates/batch/file_b.md",
	}
	for i := 0; i <= 10; i++ {
		fileMap["deep/level"+strconv.Itoa(i)+".md"] = "/templates/deep/level" + strconv.Itoa(i) + ".md"
	}

	resolver := NewMapFileResolver(fileMap)
	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeGlobFunc := createIncludeGlobFunc(renderCtx, tmplCtx)

	// Act - file_a.md should succeed, but file_b.md's nested includes should hit depth limit
	_, err := includeGlobFunc("batch/*.md")

	// Assert - should get a depth error from the nested includes
	if err == nil {
		t.Fatal("expected IncludeDepthError, got nil")
	}

	var depthErr *IncludeDepthError
	if !errors.As(err, &depthErr) {
		t.Fatalf("expected IncludeDepthError, got %T: %v", err, err)
	}

	if depthErr.MaxDepth != 10 {
		t.Errorf("expected MaxDepth 10, got %d", depthErr.MaxDepth)
	}
}

// Test 14: Character class with range [a-c] matching
func TestIncludeGlobFunc_CharacterClassRangeMatching(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/templates/items/item_a.md", []byte("ItemA"), 0644)
	afero.WriteFile(fs, "/templates/items/item_b.md", []byte("ItemB"), 0644)
	afero.WriteFile(fs, "/templates/items/item_c.md", []byte("ItemC"), 0644)
	afero.WriteFile(fs, "/templates/items/item_d.md", []byte("ItemD"), 0644)
	afero.WriteFile(fs, "/templates/items/item_z.md", []byte("ItemZ"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"items/item_a.md": "/templates/items/item_a.md",
		"items/item_b.md": "/templates/items/item_b.md",
		"items/item_c.md": "/templates/items/item_c.md",
		"items/item_d.md": "/templates/items/item_d.md",
		"items/item_z.md": "/templates/items/item_z.md",
	})

	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeGlobFunc := createIncludeGlobFunc(renderCtx, tmplCtx)

	// Act - pattern [a-c] matches characters in range a to c
	result, err := includeGlobFunc("items/item_[a-c].md")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Should match item_a.md, item_b.md, item_c.md but not item_d.md or item_z.md
	expected := "ItemA\nItemB\nItemC"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// Test 15: Error includes context when file not found after glob match
func TestIncludeGlobFunc_FileNotFoundAfterGlobMatch(t *testing.T) {
	// Arrange - create a resolver that lists a file but filesystem doesn't have it
	// This tests the scenario where the file resolver is out of sync with the filesystem
	fs := afero.NewMemMapFs()
	// Only create one file on the filesystem
	afero.WriteFile(fs, "/templates/files/exists.md", []byte("Exists"), 0644)

	// But resolver claims both exist
	resolver := NewMapFileResolver(map[string]string{
		"files/exists.md":  "/templates/files/exists.md",
		"files/missing.md": "/templates/files/missing.md", // File doesn't exist on fs
	})

	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeGlobFunc := createIncludeGlobFunc(renderCtx, tmplCtx)

	// Act
	_, err := includeGlobFunc("files/*.md")

	// Assert - should get a not found error with the source path in context
	if err == nil {
		t.Fatal("expected IncludeNotFoundError, got nil")
	}

	var notFoundErr *IncludeNotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Fatalf("expected IncludeNotFoundError, got %T: %v", err, err)
	}

	// The missing file should be identified in the error
	if notFoundErr.SourcePath != "files/missing.md" {
		t.Errorf("expected SourcePath %q, got %q", "files/missing.md", notFoundErr.SourcePath)
	}
}

// Test 16: Single file match returns content without trailing newline
func TestIncludeGlobFunc_SingleFileMatchNoTrailingNewline(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/templates/single/only.md", []byte("Only content"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"single/only.md": "/templates/single/only.md",
	})

	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeGlobFunc := createIncludeGlobFunc(renderCtx, tmplCtx)

	// Act
	result, err := includeGlobFunc("single/*.md")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Single file should return content without any newline
	expected := "Only content"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
	// Explicitly verify no trailing newline
	if len(result) > 0 && result[len(result)-1] == '\n' {
		t.Error("result should not have trailing newline")
	}
}
