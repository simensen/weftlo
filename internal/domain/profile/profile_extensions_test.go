package profile_test

import (
	"testing"

	"github.com/simensen/weftlo/internal/domain/ignore"
	"github.com/simensen/weftlo/internal/domain/profile"
)

// =============================================================================
// Profile Struct Extension Tests (Task Group 1 - Profile Loader Updates)
// =============================================================================
// These tests verify that the Profile struct correctly handles new ContentConfig
// and Matcher fields as part of the Profile Loader Updates spec.

// Test 1: Profile struct ContentConfig field is populated and accessible
func TestProfile_ContentConfigField_IsPopulated(t *testing.T) {
	// Create Profile with ContentConfig field populated
	p := profile.Profile{
		Name:         "test-profile",
		InheritsFrom: "base-profile",
		ContentConfig: profile.ContentConfig{
			Root: "templates",
			Targets: map[string]string{
				"skills":    ".claude/skills",
				"workflows": ".claude/workflows",
			},
			DefaultTarget: "custom-target",
		},
	}

	// Verify ContentConfig field is populated
	if p.ContentConfig.Root != "templates" {
		t.Errorf("expected ContentConfig.Root to be 'templates', got '%s'", p.ContentConfig.Root)
	}
	if p.ContentConfig.DefaultTarget != "custom-target" {
		t.Errorf("expected ContentConfig.DefaultTarget to be 'custom-target', got '%s'", p.ContentConfig.DefaultTarget)
	}
	if len(p.ContentConfig.Targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(p.ContentConfig.Targets))
	}
	if p.ContentConfig.Targets["skills"] != ".claude/skills" {
		t.Errorf("expected Targets[skills] to be '.claude/skills', got '%s'", p.ContentConfig.Targets["skills"])
	}
}

// Test 2: Profile struct ContentConfig defaults are applied when not specified
func TestProfile_ContentConfigDefaults_AreApplied(t *testing.T) {
	// Create Profile with zero-value ContentConfig
	p := profile.Profile{
		Name: "test-profile",
	}

	// Apply defaults using the helper function
	p.ApplyContentConfigDefaults()

	// Verify defaults are applied
	if p.ContentConfig.Root != profile.DefaultContentRoot {
		t.Errorf("expected ContentConfig.Root to be '%s', got '%s'", profile.DefaultContentRoot, p.ContentConfig.Root)
	}
	if p.ContentConfig.DefaultTarget != profile.DefaultTargetPath {
		t.Errorf("expected ContentConfig.DefaultTarget to be '%s', got '%s'", profile.DefaultTargetPath, p.ContentConfig.DefaultTarget)
	}
}

// Test 3: Profile struct ContentConfig defaults preserve explicit values
func TestProfile_ContentConfigDefaults_PreserveExplicitValues(t *testing.T) {
	// Create Profile with partial ContentConfig (only Root set)
	p := profile.Profile{
		Name: "test-profile",
		ContentConfig: profile.ContentConfig{
			Root: "custom-content",
			// DefaultTarget is not set (empty string)
		},
	}

	// Apply defaults
	p.ApplyContentConfigDefaults()

	// Verify explicit Root is preserved
	if p.ContentConfig.Root != "custom-content" {
		t.Errorf("expected ContentConfig.Root to be 'custom-content' (preserved), got '%s'", p.ContentConfig.Root)
	}

	// Verify default is applied only to DefaultTarget
	if p.ContentConfig.DefaultTarget != profile.DefaultTargetPath {
		t.Errorf("expected ContentConfig.DefaultTarget to be '%s', got '%s'", profile.DefaultTargetPath, p.ContentConfig.DefaultTarget)
	}

	// Test reverse: only DefaultTarget set
	p2 := profile.Profile{
		Name: "test-profile-2",
		ContentConfig: profile.ContentConfig{
			DefaultTarget: "my-target",
			// Root is not set (empty string)
		},
	}

	p2.ApplyContentConfigDefaults()

	// Verify default is applied only to Root
	if p2.ContentConfig.Root != profile.DefaultContentRoot {
		t.Errorf("expected ContentConfig.Root to be '%s', got '%s'", profile.DefaultContentRoot, p2.ContentConfig.Root)
	}

	// Verify explicit DefaultTarget is preserved
	if p2.ContentConfig.DefaultTarget != "my-target" {
		t.Errorf("expected ContentConfig.DefaultTarget to be 'my-target' (preserved), got '%s'", p2.ContentConfig.DefaultTarget)
	}
}

// Test 4: Profile struct Matcher field is populated when patterns exist
func TestProfile_MatcherField_IsPopulated(t *testing.T) {
	// Create a matcher with some patterns
	matcher := ignore.NewPatternMatcher([]string{
		"*.tmp",
		"*.bak",
		"node_modules/",
	})

	// Create Profile with Matcher field populated
	p := profile.Profile{
		Name:    "test-profile",
		Matcher: matcher,
	}

	// Verify Matcher field is populated and functional
	if p.Matcher == nil {
		t.Fatal("expected Matcher to be non-nil")
	}

	// Test that the matcher actually matches patterns
	if !p.Matcher.Match("file.tmp", false) {
		t.Error("expected Matcher to match 'file.tmp'")
	}
	if !p.Matcher.Match("backup.bak", false) {
		t.Error("expected Matcher to match 'backup.bak'")
	}
	if !p.Matcher.Match("node_modules", true) {
		t.Error("expected Matcher to match 'node_modules' directory")
	}
	if p.Matcher.Match("main.go", false) {
		t.Error("expected Matcher to NOT match 'main.go'")
	}
}

// Test 5: Profile struct Matcher field with empty/no-op matcher when no patterns
func TestProfile_MatcherField_EmptyMatcherWhenNoPatterns(t *testing.T) {
	// Create an empty matcher (no patterns)
	emptyMatcher := ignore.EmptyMatcher()

	// Create Profile with empty Matcher
	p := profile.Profile{
		Name:    "test-profile",
		Matcher: emptyMatcher,
	}

	// Verify Matcher field is non-nil but matches nothing
	if p.Matcher == nil {
		t.Fatal("expected Matcher to be non-nil even when empty")
	}

	// Empty matcher should not match any file
	if p.Matcher.Match("any-file.txt", false) {
		t.Error("expected empty Matcher to NOT match 'any-file.txt'")
	}
	if p.Matcher.Match("any-dir", true) {
		t.Error("expected empty Matcher to NOT match 'any-dir' directory")
	}
	if p.Matcher.Match("*.tmp", false) {
		t.Error("expected empty Matcher to NOT match '*.tmp'")
	}
}

// Test 6: Profile struct with both ContentConfig and Matcher fields
func TestProfile_BothContentConfigAndMatcher_AreAccessible(t *testing.T) {
	// Create a matcher
	matcher := ignore.NewPatternMatcher([]string{"*.draft"})

	// Create Profile with both new fields populated
	p := profile.Profile{
		Name:         "full-profile",
		InheritsFrom: "base-profile",
		ContentConfig: profile.ContentConfig{
			Root:          "templates",
			DefaultTarget: "weftlo",
			Targets: map[string]string{
				"skills": ".claude/skills",
			},
		},
		Matcher: matcher,
		TemplateFiles: []profile.TemplateFile{
			{SourcePath: "readme.md", TargetPath: "README.md", IsPartial: false},
		},
	}

	// Verify all fields are accessible together
	if p.Name != "full-profile" {
		t.Errorf("expected Name to be 'full-profile', got '%s'", p.Name)
	}
	if p.InheritsFrom != "base-profile" {
		t.Errorf("expected InheritsFrom to be 'base-profile', got '%s'", p.InheritsFrom)
	}
	if p.ContentConfig.Root != "templates" {
		t.Errorf("expected ContentConfig.Root to be 'templates', got '%s'", p.ContentConfig.Root)
	}
	if p.Matcher == nil {
		t.Fatal("expected Matcher to be non-nil")
	}
	if !p.Matcher.Match("file.draft", false) {
		t.Error("expected Matcher to match 'file.draft'")
	}
	if len(p.TemplateFiles) != 1 {
		t.Errorf("expected 1 template file, got %d", len(p.TemplateFiles))
	}
}
