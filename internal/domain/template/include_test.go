package template

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/spf13/afero"

	"github.com/simensen/weftlo/internal/domain/config"
	"github.com/simensen/weftlo/internal/domain/profile"
)

// Test 1: Basic include resolves path and renders content
func TestIncludeFunc_BasicInclude(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/templates/_partials/header.md", []byte("# Header Content"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"_partials/header.md": "/templates/_partials/header.md",
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

	includeFunc := createIncludeFunc(renderCtx, tmplCtx)

	// Act
	result, err := includeFunc("_partials/header.md")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "# Header Content"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// Test 2: Included template receives same TemplateContext as parent
func TestIncludeFunc_ReceivesSameTemplateContext(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/templates/partial.md", []byte("Profile: {{ .Profile.Name }}, Project: {{ .ProjectDir }}"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"partial.md": "/templates/partial.md",
	})

	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "vendor/test-profile"},
		"/my/project/dir",
		time.Now(),
		&config.ProjectConfig{Profiles: []string{"vendor/test-profile"}},
		&config.GlobalConfig{DefaultProfile: "vendor/default"},
		nil,
	)

	includeFunc := createIncludeFunc(renderCtx, tmplCtx)

	// Act
	result, err := includeFunc("partial.md")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "Profile: vendor/test-profile, Project: /my/project/dir"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// Test 3: Included template has access to sprig functions
func TestIncludeFunc_SprigFunctionsAvailable(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/templates/partial.md", []byte("Upper: {{ .Profile.Name | upper }}, Trimmed: {{ .ProjectDir | trim }}"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"partial.md": "/templates/partial.md",
	})

	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"  /project  ",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeFunc := createIncludeFunc(renderCtx, tmplCtx)

	// Act
	result, err := includeFunc("partial.md")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "Upper: TEST/PROFILE, Trimmed: /project"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// Test 4: Recursive include works (depth 2-3 levels)
func TestIncludeFunc_RecursiveInclude(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	// Level 1: main includes header
	afero.WriteFile(fs, "/templates/level1.md", []byte("L1:{{ include \"level2.md\" }}"), 0644)
	// Level 2: header includes footer
	afero.WriteFile(fs, "/templates/level2.md", []byte("L2:{{ include \"level3.md\" }}"), 0644)
	// Level 3: footer is the leaf
	afero.WriteFile(fs, "/templates/level3.md", []byte("L3:end"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"level1.md": "/templates/level1.md",
		"level2.md": "/templates/level2.md",
		"level3.md": "/templates/level3.md",
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

	includeFunc := createIncludeFunc(renderCtx, tmplCtx)

	// Act
	result, err := includeFunc("level1.md")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "L1:L2:L3:end"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// Test 5: Recursion depth limit (10 levels) returns IncludeDepthError
func TestIncludeFunc_DepthLimitReturnsIncludeDepthError(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	// Create a chain of files that recursively include each other
	// Each file includes the next level until we exceed 10 levels
	for i := 0; i <= 12; i++ {
		content := []byte("content")
		if i < 12 {
			content = []byte("{{ include \"level" + string(rune('0'+((i+1)%10))) + string(rune('0'+((i+1)/10))) + ".md\" }}")
		}
		afero.WriteFile(fs, "/templates/level"+string(rune('0'+(i%10)))+string(rune('0'+(i/10)))+".md", content, 0644)
	}

	// Simplified approach: create files that form a recursive chain
	fs = afero.NewMemMapFs()
	for i := 0; i <= 11; i++ {
		var content string
		if i < 11 {
			content = "{{ include \"level" + strconv.Itoa(i+1) + ".md\" }}"
		} else {
			content = "end"
		}
		afero.WriteFile(fs, "/templates/level"+strconv.Itoa(i)+".md", []byte(content), 0644)
	}

	fileMap := make(map[string]string)
	for i := 0; i <= 11; i++ {
		fileMap["level"+strconv.Itoa(i)+".md"] = "/templates/level" + strconv.Itoa(i) + ".md"
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

	includeFunc := createIncludeFunc(renderCtx, tmplCtx)

	// Act
	_, err := includeFunc("level0.md")

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

// Test 6: Include of non-existent file returns IncludeNotFoundError
func TestIncludeFunc_NonExistentFileReturnsIncludeNotFoundError(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	resolver := NewMapFileResolver(map[string]string{}) // Empty resolver

	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeFunc := createIncludeFunc(renderCtx, tmplCtx)

	// Act
	_, err := includeFunc("_partials/missing.md")

	// Assert
	if err == nil {
		t.Fatal("expected IncludeNotFoundError, got nil")
	}

	var notFoundErr *IncludeNotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Fatalf("expected IncludeNotFoundError, got %T: %v", err, err)
	}

	if notFoundErr.SourcePath != "_partials/missing.md" {
		t.Errorf("expected SourcePath %q, got %q", "_partials/missing.md", notFoundErr.SourcePath)
	}
}
