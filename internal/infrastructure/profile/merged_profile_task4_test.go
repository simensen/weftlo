package profile

import (
	"testing"

	"github.com/spf13/afero"

	domainignore "github.com/simensen/weftlo/internal/domain/ignore"
	domainprofile "github.com/simensen/weftlo/internal/domain/profile"
)

// =============================================================================
// Task Group 4: MergedProfile ContentConfig and Matcher Merge Tests
// =============================================================================

// TestMergedProfile_ContentConfigDeepMerge_RootOverrideOnlyWhenNonEmpty tests that
// ContentConfig.Root is only overridden by child when child's Root is non-empty.
func TestMergedProfile_ContentConfigDeepMerge_RootOverrideOnlyWhenNonEmpty(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Parent with explicit Root
	parent := &domainprofile.Profile{
		Name:         "acme/parent",
		InheritsFrom: "",
		ContentConfig: domainprofile.ContentConfig{
			Root: "parent-content",
		},
		TemplateFiles: []domainprofile.TemplateFile{},
	}

	// Child with empty Root - should preserve parent's Root
	childEmpty := &domainprofile.Profile{
		Name:         "acme/child-empty",
		InheritsFrom: "acme/parent",
		ContentConfig: domainprofile.ContentConfig{
			Root: "", // Empty - should not override parent
		},
		TemplateFiles: []domainprofile.TemplateFile{},
	}

	chain := []*domainprofile.Profile{parent, childEmpty}
	profilesBasePath := "/home/user/.weftlo/profiles"

	merged := NewMergedProfile(chain, fs, profilesBasePath)

	// Verify parent's Root is preserved when child's Root is empty
	if merged.ContentConfig.Root != "parent-content" {
		t.Errorf("expected Root to be 'parent-content' when child Root is empty, got '%s'", merged.ContentConfig.Root)
	}

	// Now test with non-empty child Root
	childOverride := &domainprofile.Profile{
		Name:         "acme/child-override",
		InheritsFrom: "acme/parent",
		ContentConfig: domainprofile.ContentConfig{
			Root: "child-content", // Non-empty - should override parent
		},
		TemplateFiles: []domainprofile.TemplateFile{},
	}

	chain2 := []*domainprofile.Profile{parent, childOverride}
	merged2 := NewMergedProfile(chain2, fs, profilesBasePath)

	// Verify child's Root overrides parent's Root when non-empty
	if merged2.ContentConfig.Root != "child-content" {
		t.Errorf("expected Root to be 'child-content' when child Root is non-empty, got '%s'", merged2.ContentConfig.Root)
	}
}

// TestMergedProfile_ContentConfigDeepMerge_TargetsShallowMerge tests that
// ContentConfig.Targets are shallow merged (child overrides parent keys, parent keys preserved).
func TestMergedProfile_ContentConfigDeepMerge_TargetsShallowMerge(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Parent with some targets
	parent := &domainprofile.Profile{
		Name:         "acme/parent",
		InheritsFrom: "",
		ContentConfig: domainprofile.ContentConfig{
			Targets: map[string]string{
				"claude": ".claude",
				"shared": "weftlo/shared",
			},
		},
		TemplateFiles: []domainprofile.TemplateFile{},
	}

	// Child with overlapping and new targets
	child := &domainprofile.Profile{
		Name:         "acme/child",
		InheritsFrom: "acme/parent",
		ContentConfig: domainprofile.ContentConfig{
			Targets: map[string]string{
				"claude": ".ai/claude", // Override parent value
				"cursor": ".cursor",    // New key
			},
		},
		TemplateFiles: []domainprofile.TemplateFile{},
	}

	chain := []*domainprofile.Profile{parent, child}
	profilesBasePath := "/home/user/.weftlo/profiles"

	merged := NewMergedProfile(chain, fs, profilesBasePath)

	// Verify child key overrides parent for same key
	if merged.ContentConfig.Targets["claude"] != ".ai/claude" {
		t.Errorf("expected Targets['claude'] to be '.ai/claude' (child override), got '%s'", merged.ContentConfig.Targets["claude"])
	}

	// Verify parent key not in child is preserved
	if merged.ContentConfig.Targets["shared"] != "weftlo/shared" {
		t.Errorf("expected Targets['shared'] to be 'weftlo/shared' (from parent), got '%s'", merged.ContentConfig.Targets["shared"])
	}

	// Verify new child key is present
	if merged.ContentConfig.Targets["cursor"] != ".cursor" {
		t.Errorf("expected Targets['cursor'] to be '.cursor' (from child), got '%s'", merged.ContentConfig.Targets["cursor"])
	}

	// Verify total count (3 unique keys)
	if len(merged.ContentConfig.Targets) != 3 {
		t.Errorf("expected 3 targets in merged result, got %d", len(merged.ContentConfig.Targets))
	}
}

// TestMergedProfile_ContentConfigDeepMerge_DefaultTargetOverrideOnlyWhenNonEmpty tests that
// ContentConfig.DefaultTarget is only overridden by child when child's DefaultTarget is non-empty.
func TestMergedProfile_ContentConfigDeepMerge_DefaultTargetOverrideOnlyWhenNonEmpty(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Parent with explicit DefaultTarget
	parent := &domainprofile.Profile{
		Name:         "acme/parent",
		InheritsFrom: "",
		ContentConfig: domainprofile.ContentConfig{
			DefaultTarget: "parent-default",
		},
		TemplateFiles: []domainprofile.TemplateFile{},
	}

	// Child with empty DefaultTarget - should preserve parent's DefaultTarget
	childEmpty := &domainprofile.Profile{
		Name:         "acme/child-empty",
		InheritsFrom: "acme/parent",
		ContentConfig: domainprofile.ContentConfig{
			DefaultTarget: "", // Empty - should not override parent
		},
		TemplateFiles: []domainprofile.TemplateFile{},
	}

	chain := []*domainprofile.Profile{parent, childEmpty}
	profilesBasePath := "/home/user/.weftlo/profiles"

	merged := NewMergedProfile(chain, fs, profilesBasePath)

	// Verify parent's DefaultTarget is preserved when child's DefaultTarget is empty
	if merged.ContentConfig.DefaultTarget != "parent-default" {
		t.Errorf("expected DefaultTarget to be 'parent-default' when child DefaultTarget is empty, got '%s'", merged.ContentConfig.DefaultTarget)
	}

	// Now test with non-empty child DefaultTarget
	childOverride := &domainprofile.Profile{
		Name:         "acme/child-override",
		InheritsFrom: "acme/parent",
		ContentConfig: domainprofile.ContentConfig{
			DefaultTarget: "child-default", // Non-empty - should override parent
		},
		TemplateFiles: []domainprofile.TemplateFile{},
	}

	chain2 := []*domainprofile.Profile{parent, childOverride}
	merged2 := NewMergedProfile(chain2, fs, profilesBasePath)

	// Verify child's DefaultTarget overrides parent's DefaultTarget when non-empty
	if merged2.ContentConfig.DefaultTarget != "child-default" {
		t.Errorf("expected DefaultTarget to be 'child-default' when child DefaultTarget is non-empty, got '%s'", merged2.ContentConfig.DefaultTarget)
	}
}

// TestMergedProfile_MatcherMerge_ParentPatternsFirstThenChildPatterns tests that
// Matcher patterns are merged with parent patterns first, then child patterns.
func TestMergedProfile_MatcherMerge_ParentPatternsFirstThenChildPatterns(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Parent with ignore pattern for all .draft files
	parentMatcher := domainignore.NewPatternMatcher([]string{"*.draft.md"})

	parent := &domainprofile.Profile{
		Name:          "acme/parent",
		InheritsFrom:  "",
		Matcher:       parentMatcher,
		TemplateFiles: []domainprofile.TemplateFile{},
	}

	// Child with negation pattern to re-include specific file
	childMatcher := domainignore.NewPatternMatcher([]string{"!important.draft.md"})

	child := &domainprofile.Profile{
		Name:          "acme/child",
		InheritsFrom:  "acme/parent",
		Matcher:       childMatcher,
		TemplateFiles: []domainprofile.TemplateFile{},
	}

	chain := []*domainprofile.Profile{parent, child}
	profilesBasePath := "/home/user/.weftlo/profiles"

	merged := NewMergedProfile(chain, fs, profilesBasePath)

	// Verify parent's pattern excludes general draft files
	if !merged.Matcher.Match("random.draft.md", false) {
		t.Error("expected 'random.draft.md' to be ignored (parent pattern *.draft.md)")
	}

	// Verify child's negation pattern re-includes the specific file
	// Parent pattern would match, but child's negation comes after
	if merged.Matcher.Match("important.draft.md", false) {
		t.Error("expected 'important.draft.md' to NOT be ignored (child negation !important.draft.md)")
	}

	// Verify non-matching files are not ignored
	if merged.Matcher.Match("readme.md", false) {
		t.Error("expected 'readme.md' to NOT be ignored (no matching pattern)")
	}
}

// TestMergedProfile_MergeWith_ContentConfigMerge tests that MergeWith properly
// merges ContentConfig from two MergedProfiles.
func TestMergedProfile_MergeWith_ContentConfigMerge(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// First merged profile (acts as "parent" in multi-profile merge)
	first := &domainprofile.Profile{
		Name:         "acme/first",
		InheritsFrom: "",
		ContentConfig: domainprofile.ContentConfig{
			Root:          "first-content",
			DefaultTarget: "first-default",
			Targets: map[string]string{
				"claude": ".claude",
				"shared": "weftlo/shared",
			},
		},
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "first.md", IsPartial: false},
		},
	}

	// Second merged profile (acts as "child" in multi-profile merge)
	second := &domainprofile.Profile{
		Name:         "acme/second",
		InheritsFrom: "",
		ContentConfig: domainprofile.ContentConfig{
			Root:          "second-content",
			DefaultTarget: "", // Empty - should preserve first's DefaultTarget
			Targets: map[string]string{
				"claude": ".ai/claude", // Override first's value
				"cursor": ".cursor",    // New key
			},
		},
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "second.md", IsPartial: false},
		},
	}

	profilesBasePath := "/home/user/.weftlo/profiles"

	// Create two separate MergedProfiles
	mergedFirst := NewMergedProfile([]*domainprofile.Profile{first}, fs, profilesBasePath)
	mergedSecond := NewMergedProfile([]*domainprofile.Profile{second}, fs, profilesBasePath)

	// Merge them
	combined := mergedFirst.MergeWith(mergedSecond)

	// Verify Root is overridden by second (non-empty)
	if combined.ContentConfig.Root != "second-content" {
		t.Errorf("expected Root to be 'second-content' (from second profile), got '%s'", combined.ContentConfig.Root)
	}

	// Verify DefaultTarget is preserved from first (second's is empty)
	if combined.ContentConfig.DefaultTarget != "first-default" {
		t.Errorf("expected DefaultTarget to be 'first-default' (preserved from first), got '%s'", combined.ContentConfig.DefaultTarget)
	}

	// Verify targets are merged correctly
	if combined.ContentConfig.Targets["claude"] != ".ai/claude" {
		t.Errorf("expected Targets['claude'] to be '.ai/claude' (from second), got '%s'", combined.ContentConfig.Targets["claude"])
	}
	if combined.ContentConfig.Targets["shared"] != "weftlo/shared" {
		t.Errorf("expected Targets['shared'] to be 'weftlo/shared' (from first), got '%s'", combined.ContentConfig.Targets["shared"])
	}
	if combined.ContentConfig.Targets["cursor"] != ".cursor" {
		t.Errorf("expected Targets['cursor'] to be '.cursor' (from second), got '%s'", combined.ContentConfig.Targets["cursor"])
	}
}

// TestMergedProfile_MergeWith_MatcherMerge tests that MergeWith properly
// merges Matchers from two MergedProfiles.
func TestMergedProfile_MergeWith_MatcherMerge(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// First merged profile with exclude pattern
	firstMatcher := domainignore.NewPatternMatcher([]string{"*.log"})
	first := &domainprofile.Profile{
		Name:          "acme/first",
		InheritsFrom:  "",
		Matcher:       firstMatcher,
		TemplateFiles: []domainprofile.TemplateFile{},
	}

	// Second merged profile with negation pattern
	secondMatcher := domainignore.NewPatternMatcher([]string{"!important.log"})
	second := &domainprofile.Profile{
		Name:          "acme/second",
		InheritsFrom:  "",
		Matcher:       secondMatcher,
		TemplateFiles: []domainprofile.TemplateFile{},
	}

	profilesBasePath := "/home/user/.weftlo/profiles"

	// Create two separate MergedProfiles
	mergedFirst := NewMergedProfile([]*domainprofile.Profile{first}, fs, profilesBasePath)
	mergedSecond := NewMergedProfile([]*domainprofile.Profile{second}, fs, profilesBasePath)

	// Merge them
	combined := mergedFirst.MergeWith(mergedSecond)

	// Verify first's pattern excludes general .log files
	if !combined.Matcher.Match("debug.log", false) {
		t.Error("expected 'debug.log' to be ignored (first profile pattern *.log)")
	}

	// Verify second's negation pattern re-includes specific file
	if combined.Matcher.Match("important.log", false) {
		t.Error("expected 'important.log' to NOT be ignored (second profile negation !important.log)")
	}

	// Verify non-matching files are not ignored
	if combined.Matcher.Match("readme.md", false) {
		t.Error("expected 'readme.md' to NOT be ignored (no matching pattern)")
	}
}

// TestMergedProfile_ToProfile_IncludesContentConfigAndMatcher tests that
// ToProfile returns a Profile with ContentConfig and Matcher fields populated.
func TestMergedProfile_ToProfile_IncludesContentConfigAndMatcher(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile with ContentConfig and Matcher
	testMatcher := domainignore.NewPatternMatcher([]string{"*.tmp"})
	profile := &domainprofile.Profile{
		Name:         "acme/test",
		InheritsFrom: "",
		ContentConfig: domainprofile.ContentConfig{
			Root:          "test-content",
			DefaultTarget: "test-default",
			Targets: map[string]string{
				"custom": "custom-path",
			},
		},
		Matcher: testMatcher,
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "test.md", IsPartial: false},
		},
	}

	chain := []*domainprofile.Profile{profile}
	profilesBasePath := "/home/user/.weftlo/profiles"

	merged := NewMergedProfile(chain, fs, profilesBasePath)
	result := merged.ToProfile()

	// Verify ContentConfig is populated
	if result.ContentConfig.Root != "test-content" {
		t.Errorf("expected ToProfile().ContentConfig.Root to be 'test-content', got '%s'", result.ContentConfig.Root)
	}
	if result.ContentConfig.DefaultTarget != "test-default" {
		t.Errorf("expected ToProfile().ContentConfig.DefaultTarget to be 'test-default', got '%s'", result.ContentConfig.DefaultTarget)
	}
	if result.ContentConfig.Targets["custom"] != "custom-path" {
		t.Errorf("expected ToProfile().ContentConfig.Targets['custom'] to be 'custom-path', got '%s'", result.ContentConfig.Targets["custom"])
	}

	// Verify Matcher is populated and functional
	if result.Matcher == nil {
		t.Error("expected ToProfile().Matcher to be non-nil")
	} else if !result.Matcher.Match("test.tmp", false) {
		t.Error("expected ToProfile().Matcher to match '*.tmp' pattern")
	}
}
