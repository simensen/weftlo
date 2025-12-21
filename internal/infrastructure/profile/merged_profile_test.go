package profile

import (
	"errors"
	"strconv"
	"testing"

	"github.com/spf13/afero"

	domainprofile "github.com/simensen/weftlo/internal/domain/profile"
	domaintemplate "github.com/simensen/weftlo/internal/domain/template"
)

// =============================================================================
// Task Group 1: MergedProfile Core Tests (6 focused tests)
// =============================================================================

// TestMergedProfile_SingleProfileNoInheritance tests MergedProfile creation with
// a single profile (no inheritance). Verifies that all template files from the
// single profile are correctly merged and accessible.
func TestMergedProfile_SingleProfileNoInheritance(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create a single profile with template files
	profile := &domainprofile.Profile{
		Name:         "acme/base",
		InheritsFrom: "",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "README.md", IsPartial: false},
			{SourcePath: "docs/api.md", IsPartial: false},
			{SourcePath: "_partials/header.md", IsPartial: true},
		},
	}

	chain := []*domainprofile.Profile{profile}
	profilesBasePath := "/home/user/.weftlo/profiles"

	merged := NewMergedProfile(chain, fs, profilesBasePath)

	// Verify leaf profile name is stored
	if merged.LeafProfileName() != "acme/base" {
		t.Errorf("expected LeafProfileName 'acme/base', got '%s'", merged.LeafProfileName())
	}

	// Verify all 3 source paths are available
	sourcePaths := merged.ListSourcePaths()
	if len(sourcePaths) != 3 {
		t.Fatalf("expected 3 source paths, got %d", len(sourcePaths))
	}

	// Verify each source path can be resolved
	expectedPaths := map[string]string{
		"README.md":           "/home/user/.weftlo/profiles/acme/base/content/README.md",
		"docs/api.md":         "/home/user/.weftlo/profiles/acme/base/content/docs/api.md",
		"_partials/header.md": "/home/user/.weftlo/profiles/acme/base/content/_partials/header.md",
	}

	for sourcePath, expectedAbsPath := range expectedPaths {
		absPath, found := merged.Resolve(sourcePath)
		if !found {
			t.Errorf("expected to find source path '%s'", sourcePath)
			continue
		}
		if absPath != expectedAbsPath {
			t.Errorf("expected absolute path '%s' for '%s', got '%s'", expectedAbsPath, sourcePath, absPath)
		}
	}
}

// TestMergedProfile_ChildOverridesParentSemantics tests that child-overrides-parent
// merge semantics work correctly when source paths overlap. Child files should
// take precedence over parent files with the same source path.
func TestMergedProfile_ChildOverridesParentSemantics(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create parent profile
	parent := &domainprofile.Profile{
		Name:         "acme/parent",
		InheritsFrom: "",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "README.md", IsPartial: false},
			{SourcePath: "docs/api.md", IsPartial: false},
			{SourcePath: "config.yaml", IsPartial: false},
		},
	}

	// Create child profile that overrides README.md
	child := &domainprofile.Profile{
		Name:         "acme/child",
		InheritsFrom: "acme/parent",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "README.md", IsPartial: false}, // Overrides parent
			{SourcePath: "extra.md", IsPartial: false},  // New file
		},
	}

	// Chain is ordered root-to-leaf: [parent, child]
	chain := []*domainprofile.Profile{parent, child}
	profilesBasePath := "/home/user/.weftlo/profiles"

	merged := NewMergedProfile(chain, fs, profilesBasePath)

	// Verify README.md resolves to child's path (child overrides parent)
	absPath, found := merged.Resolve("README.md")
	if !found {
		t.Fatal("expected to find README.md")
	}
	expectedChildPath := "/home/user/.weftlo/profiles/acme/child/content/README.md"
	if absPath != expectedChildPath {
		t.Errorf("expected README.md to resolve to child path '%s', got '%s'", expectedChildPath, absPath)
	}

	// Verify docs/api.md resolves to parent's path (not overridden)
	absPath, found = merged.Resolve("docs/api.md")
	if !found {
		t.Fatal("expected to find docs/api.md")
	}
	expectedParentPath := "/home/user/.weftlo/profiles/acme/parent/content/docs/api.md"
	if absPath != expectedParentPath {
		t.Errorf("expected docs/api.md to resolve to parent path '%s', got '%s'", expectedParentPath, absPath)
	}

	// Verify total number of unique source paths
	sourcePaths := merged.ListSourcePaths()
	if len(sourcePaths) != 4 {
		t.Errorf("expected 4 unique source paths, got %d: %v", len(sourcePaths), sourcePaths)
	}
}

// TestMergedProfile_ResolveReturnsCorrectAbsolutePath tests that Resolve() returns
// the correct absolute filesystem path for a known source path.
func TestMergedProfile_ResolveReturnsCorrectAbsolutePath(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	profile := &domainprofile.Profile{
		Name:         "my-vendor/my-profile",
		InheritsFrom: "",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "deeply/nested/file.md", IsPartial: false},
		},
	}

	chain := []*domainprofile.Profile{profile}
	profilesBasePath := "/custom/path/to/profiles"

	merged := NewMergedProfile(chain, fs, profilesBasePath)

	absPath, found := merged.Resolve("deeply/nested/file.md")
	if !found {
		t.Fatal("expected to find deeply/nested/file.md")
	}

	expectedPath := "/custom/path/to/profiles/my-vendor/my-profile/content/deeply/nested/file.md"
	if absPath != expectedPath {
		t.Errorf("expected absolute path '%s', got '%s'", expectedPath, absPath)
	}
}

// TestMergedProfile_ResolveReturnsFalseForUnknownPath tests that Resolve() returns
// found=false for a source path that does not exist in the merged profile.
func TestMergedProfile_ResolveReturnsFalseForUnknownPath(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	profile := &domainprofile.Profile{
		Name:         "acme/base",
		InheritsFrom: "",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "README.md", IsPartial: false},
		},
	}

	chain := []*domainprofile.Profile{profile}
	profilesBasePath := "/home/user/.weftlo/profiles"

	merged := NewMergedProfile(chain, fs, profilesBasePath)

	absPath, found := merged.Resolve("nonexistent.md")
	if found {
		t.Error("expected found=false for nonexistent path")
	}
	if absPath != "" {
		t.Errorf("expected empty string for nonexistent path, got '%s'", absPath)
	}
}

// TestMergedProfile_ListSourcePathsReturnsAllMergedPaths tests that ListSourcePaths()
// returns all merged source paths from the inheritance chain.
func TestMergedProfile_ListSourcePathsReturnsAllMergedPaths(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	parent := &domainprofile.Profile{
		Name:         "acme/parent",
		InheritsFrom: "",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "a.md", IsPartial: false},
			{SourcePath: "b.md", IsPartial: false},
		},
	}

	child := &domainprofile.Profile{
		Name:         "acme/child",
		InheritsFrom: "acme/parent",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "b.md", IsPartial: false}, // Overrides
			{SourcePath: "c.md", IsPartial: false}, // New
		},
	}

	chain := []*domainprofile.Profile{parent, child}
	profilesBasePath := "/home/user/.weftlo/profiles"

	merged := NewMergedProfile(chain, fs, profilesBasePath)

	sourcePaths := merged.ListSourcePaths()

	// Verify we get 3 unique paths (a.md, b.md, c.md)
	if len(sourcePaths) != 3 {
		t.Fatalf("expected 3 source paths, got %d: %v", len(sourcePaths), sourcePaths)
	}

	// Convert to map for easier checking
	pathMap := make(map[string]bool)
	for _, p := range sourcePaths {
		pathMap[p] = true
	}

	for _, expected := range []string{"a.md", "b.md", "c.md"} {
		if !pathMap[expected] {
			t.Errorf("expected source path '%s' to be in list", expected)
		}
	}
}

// TestMergedProfile_IsPartialPreservedFromWinningTemplateFile tests that the IsPartial
// flag is preserved from the winning (child-most) TemplateFile when source paths overlap.
func TestMergedProfile_IsPartialPreservedFromWinningTemplateFile(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Parent has a file marked as partial
	parent := &domainprofile.Profile{
		Name:         "acme/parent",
		InheritsFrom: "",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "shared.md", IsPartial: true},
			{SourcePath: "parent-only.md", IsPartial: true},
		},
	}

	// Child overrides with non-partial version
	child := &domainprofile.Profile{
		Name:         "acme/child",
		InheritsFrom: "acme/parent",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "shared.md", IsPartial: false}, // Overrides, changes IsPartial
			{SourcePath: "child-only.md", IsPartial: true},
		},
	}

	chain := []*domainprofile.Profile{parent, child}
	profilesBasePath := "/home/user/.weftlo/profiles"

	merged := NewMergedProfile(chain, fs, profilesBasePath)

	// Get template files and check IsPartial flags
	templateFiles := merged.TemplateFiles()

	// Convert to map for easier lookup
	fileMap := make(map[string]domainprofile.TemplateFile)
	for _, tf := range templateFiles {
		fileMap[tf.SourcePath] = tf
	}

	// shared.md should have IsPartial=false (child's value)
	if tf, ok := fileMap["shared.md"]; !ok {
		t.Error("expected to find shared.md in template files")
	} else if tf.IsPartial {
		t.Error("expected shared.md to have IsPartial=false (from child)")
	}

	// parent-only.md should have IsPartial=true (parent's value, not overridden)
	if tf, ok := fileMap["parent-only.md"]; !ok {
		t.Error("expected to find parent-only.md in template files")
	} else if !tf.IsPartial {
		t.Error("expected parent-only.md to have IsPartial=true (from parent)")
	}

	// child-only.md should have IsPartial=true (child's value)
	if tf, ok := fileMap["child-only.md"]; !ok {
		t.Error("expected to find child-only.md in template files")
	} else if !tf.IsPartial {
		t.Error("expected child-only.md to have IsPartial=true (from child)")
	}
}

// TestMergedProfile_EmptyChainReturnsEmptyMergedProfile tests that an empty chain
// results in a MergedProfile with no source paths.
func TestMergedProfile_EmptyChainReturnsEmptyMergedProfile(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	chain := []*domainprofile.Profile{}
	profilesBasePath := "/home/user/.weftlo/profiles"

	merged := NewMergedProfile(chain, fs, profilesBasePath)

	// Verify no source paths
	sourcePaths := merged.ListSourcePaths()
	if len(sourcePaths) != 0 {
		t.Errorf("expected 0 source paths for empty chain, got %d", len(sourcePaths))
	}

	// Verify Resolve returns not found
	_, found := merged.Resolve("anything.md")
	if found {
		t.Error("expected found=false for any path in empty merged profile")
	}

	// Verify leaf profile name is empty
	if merged.LeafProfileName() != "" {
		t.Errorf("expected empty LeafProfileName for empty chain, got '%s'", merged.LeafProfileName())
	}
}

// TestMergedProfile_ListSourcePathsReturnsEmptySliceNotNil tests that ListSourcePaths
// returns an empty slice (not nil) when there are no paths registered.
func TestMergedProfile_ListSourcePathsReturnsEmptySliceNotNil(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Profile with no template files
	profile := &domainprofile.Profile{
		Name:          "acme/empty",
		InheritsFrom:  "",
		TemplateFiles: []domainprofile.TemplateFile{},
	}

	chain := []*domainprofile.Profile{profile}
	profilesBasePath := "/home/user/.weftlo/profiles"

	merged := NewMergedProfile(chain, fs, profilesBasePath)

	sourcePaths := merged.ListSourcePaths()

	// Verify it's not nil
	if sourcePaths == nil {
		t.Error("expected ListSourcePaths to return empty slice, not nil")
	}

	// Verify it's empty
	if len(sourcePaths) != 0 {
		t.Errorf("expected 0 source paths, got %d", len(sourcePaths))
	}
}

// =============================================================================
// Task Group 2: LoadMerged Integration Tests (4 focused tests)
// =============================================================================

// TestLoadMerged_ReturnsValidMergedProfileWithInheritance tests that LoadMerged
// returns a properly constructed MergedProfile for a profile with inheritance.
func TestLoadMerged_ReturnsValidMergedProfileWithInheritance(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create parent profile
	parentDir := "/home/user/.weftlo/profiles/acme/parent"
	if err := fs.MkdirAll(parentDir, 0755); err != nil {
		t.Fatalf("failed to create parent directory: %v", err)
	}
	parentYAML := `name: acme/parent
`
	if err := afero.WriteFile(fs, parentDir+"/profile.yaml", []byte(parentYAML), 0644); err != nil {
		t.Fatalf("failed to write parent profile.yaml: %v", err)
	}
	if err := afero.WriteFile(fs, parentDir+"/content/base.md", []byte("# Base"), 0644); err != nil {
		t.Fatalf("failed to write base.md: %v", err)
	}

	// Create child profile
	childDir := "/home/user/.weftlo/profiles/acme/child"
	if err := fs.MkdirAll(childDir, 0755); err != nil {
		t.Fatalf("failed to create child directory: %v", err)
	}
	childYAML := `name: acme/child
inherits_from: acme/parent
`
	if err := afero.WriteFile(fs, childDir+"/profile.yaml", []byte(childYAML), 0644); err != nil {
		t.Fatalf("failed to write child profile.yaml: %v", err)
	}
	if err := afero.WriteFile(fs, childDir+"/content/child.md", []byte("# Child"), 0644); err != nil {
		t.Fatalf("failed to write child.md: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	merged, err := loader.LoadMerged("acme/child")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify leaf profile name
	if merged.LeafProfileName() != "acme/child" {
		t.Errorf("expected LeafProfileName 'acme/child', got '%s'", merged.LeafProfileName())
	}

	// Verify both files are accessible (inherited + own)
	sourcePaths := merged.ListSourcePaths()
	if len(sourcePaths) != 2 {
		t.Errorf("expected 2 source paths, got %d: %v", len(sourcePaths), sourcePaths)
	}

	// Verify parent's file is accessible
	absPath, found := merged.Resolve("base.md")
	if !found {
		t.Error("expected to find inherited base.md")
	}
	expectedParentPath := "/home/user/.weftlo/profiles/acme/parent/content/base.md"
	if absPath != expectedParentPath {
		t.Errorf("expected base.md path '%s', got '%s'", expectedParentPath, absPath)
	}

	// Verify child's file is accessible
	absPath, found = merged.Resolve("child.md")
	if !found {
		t.Error("expected to find child.md")
	}
	expectedChildPath := "/home/user/.weftlo/profiles/acme/child/content/child.md"
	if absPath != expectedChildPath {
		t.Errorf("expected child.md path '%s', got '%s'", expectedChildPath, absPath)
	}
}

// TestLoadMerged_PropagatesProfileNotFoundError tests that LoadMerged propagates
// ProfileNotFoundError when the requested profile does not exist.
func TestLoadMerged_PropagatesProfileNotFoundError(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	_, err := loader.LoadMerged("acme/nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var notFoundErr *ProfileNotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Fatalf("expected ProfileNotFoundError, got %T: %v", err, err)
	}

	if notFoundErr.ProfileName != "acme/nonexistent" {
		t.Errorf("expected ProfileName 'acme/nonexistent', got '%s'", notFoundErr.ProfileName)
	}
}

// TestLoadMerged_PropagatesCircularDependencyError tests that LoadMerged propagates
// CircularDependencyError when a cycle is detected in the inheritance chain.
func TestLoadMerged_PropagatesCircularDependencyError(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile A (inherits from B)
	profileADir := "/home/user/.weftlo/profiles/acme/a"
	if err := fs.MkdirAll(profileADir, 0755); err != nil {
		t.Fatalf("failed to create profile A directory: %v", err)
	}
	profileAYAML := `name: acme/a
inherits_from: acme/b
`
	if err := afero.WriteFile(fs, profileADir+"/profile.yaml", []byte(profileAYAML), 0644); err != nil {
		t.Fatalf("failed to write profile A profile.yaml: %v", err)
	}

	// Create profile B (inherits from A - creates cycle)
	profileBDir := "/home/user/.weftlo/profiles/acme/b"
	if err := fs.MkdirAll(profileBDir, 0755); err != nil {
		t.Fatalf("failed to create profile B directory: %v", err)
	}
	profileBYAML := `name: acme/b
inherits_from: acme/a
`
	if err := afero.WriteFile(fs, profileBDir+"/profile.yaml", []byte(profileBYAML), 0644); err != nil {
		t.Fatalf("failed to write profile B profile.yaml: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	_, err := loader.LoadMerged("acme/a")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var circularErr *CircularDependencyError
	if !errors.As(err, &circularErr) {
		t.Fatalf("expected CircularDependencyError, got %T: %v", err, err)
	}

	if circularErr.ProfileName != "acme/a" {
		t.Errorf("expected ProfileName 'acme/a', got '%s'", circularErr.ProfileName)
	}
}

// TestLoadMerged_PropagatesInheritanceDepthExceededError tests that LoadMerged
// propagates InheritanceDepthExceededError when the inheritance chain is too deep.
func TestLoadMerged_PropagatesInheritanceDepthExceededError(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create a chain that exceeds MaxInheritanceDepth (15)
	for i := 0; i <= MaxInheritanceDepth; i++ {
		profileDir := "/home/user/.weftlo/profiles/acme/level" + strconv.Itoa(i)
		if err := fs.MkdirAll(profileDir, 0755); err != nil {
			t.Fatalf("failed to create profile directory for level %d: %v", i, err)
		}

		var profileYAML string
		if i == 0 {
			profileYAML = "name: acme/level0\n"
		} else {
			profileYAML = "name: acme/level" + strconv.Itoa(i) + "\ninherits_from: acme/level" + strconv.Itoa(i-1) + "\n"
		}

		if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
			t.Fatalf("failed to write profile.yaml for level %d: %v", i, err)
		}
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	// Load the deepest level, which should exceed the limit
	_, err := loader.LoadMerged("acme/level" + strconv.Itoa(MaxInheritanceDepth))
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var depthErr *InheritanceDepthExceededError
	if !errors.As(err, &depthErr) {
		t.Fatalf("expected InheritanceDepthExceededError, got %T: %v", err, err)
	}

	if depthErr.MaxDepth != MaxInheritanceDepth {
		t.Errorf("expected MaxDepth %d, got %d", MaxInheritanceDepth, depthErr.MaxDepth)
	}
}

// =============================================================================
// Task Group 3: Strategic Gap Analysis Tests (up to 6 additional tests)
// =============================================================================

// TestMergedProfile_DeepInheritanceChainWithOverlappingFiles tests a 3-level
// inheritance chain (grandparent -> parent -> child) with files overlapping
// at each level to verify the child-most version is used.
func TestMergedProfile_DeepInheritanceChainWithOverlappingFiles(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	grandparent := &domainprofile.Profile{
		Name:         "acme/grandparent",
		InheritsFrom: "",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "README.md", IsPartial: false},      // Overridden by child
			{SourcePath: "config.yaml", IsPartial: false},    // Overridden by parent
			{SourcePath: "grandparent.md", IsPartial: false}, // Unique to grandparent
		},
	}

	parent := &domainprofile.Profile{
		Name:         "acme/parent",
		InheritsFrom: "acme/grandparent",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "config.yaml", IsPartial: false}, // Overrides grandparent
			{SourcePath: "parent.md", IsPartial: false},   // Unique to parent
		},
	}

	child := &domainprofile.Profile{
		Name:         "acme/child",
		InheritsFrom: "acme/parent",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "README.md", IsPartial: false}, // Overrides grandparent
			{SourcePath: "child.md", IsPartial: false},  // Unique to child
		},
	}

	chain := []*domainprofile.Profile{grandparent, parent, child}
	profilesBasePath := "/home/user/.weftlo/profiles"

	merged := NewMergedProfile(chain, fs, profilesBasePath)

	// Verify total unique paths: grandparent.md, config.yaml, parent.md, README.md, child.md
	sourcePaths := merged.ListSourcePaths()
	if len(sourcePaths) != 5 {
		t.Errorf("expected 5 unique source paths, got %d: %v", len(sourcePaths), sourcePaths)
	}

	// Verify README.md comes from child (most recent override)
	absPath, _ := merged.Resolve("README.md")
	expectedPath := "/home/user/.weftlo/profiles/acme/child/content/README.md"
	if absPath != expectedPath {
		t.Errorf("expected README.md from child '%s', got '%s'", expectedPath, absPath)
	}

	// Verify config.yaml comes from parent (intermediate override)
	absPath, _ = merged.Resolve("config.yaml")
	expectedPath = "/home/user/.weftlo/profiles/acme/parent/content/config.yaml"
	if absPath != expectedPath {
		t.Errorf("expected config.yaml from parent '%s', got '%s'", expectedPath, absPath)
	}

	// Verify grandparent.md comes from grandparent (not overridden)
	absPath, _ = merged.Resolve("grandparent.md")
	expectedPath = "/home/user/.weftlo/profiles/acme/grandparent/content/grandparent.md"
	if absPath != expectedPath {
		t.Errorf("expected grandparent.md from grandparent '%s', got '%s'", expectedPath, absPath)
	}
}

// TestMergedProfile_ProfileWithOnlyPartialFiles tests that a profile containing
// only partial files merges correctly and all partials are accessible.
func TestMergedProfile_ProfileWithOnlyPartialFiles(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	profile := &domainprofile.Profile{
		Name:         "acme/partials-only",
		InheritsFrom: "",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "_header.md", IsPartial: true},
			{SourcePath: "_footer.md", IsPartial: true},
			{SourcePath: "_partials/nav.md", IsPartial: true},
		},
	}

	chain := []*domainprofile.Profile{profile}
	profilesBasePath := "/home/user/.weftlo/profiles"

	merged := NewMergedProfile(chain, fs, profilesBasePath)

	// Verify all 3 partials are accessible
	sourcePaths := merged.ListSourcePaths()
	if len(sourcePaths) != 3 {
		t.Fatalf("expected 3 source paths, got %d", len(sourcePaths))
	}

	// Verify all are marked as partials
	for _, tf := range merged.TemplateFiles() {
		if !tf.IsPartial {
			t.Errorf("expected '%s' to be a partial", tf.SourcePath)
		}
	}

	// Verify resolution works for all partials
	for _, path := range []string{"_header.md", "_footer.md", "_partials/nav.md"} {
		if _, found := merged.Resolve(path); !found {
			t.Errorf("expected to find partial '%s'", path)
		}
	}
}

// TestMergedProfile_EmptyProfileInInheritanceChain tests that an empty profile
// (one with no template files) in the middle of an inheritance chain is handled
// correctly without affecting other profiles' files.
func TestMergedProfile_EmptyProfileInInheritanceChain(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	grandparent := &domainprofile.Profile{
		Name:         "acme/grandparent",
		InheritsFrom: "",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "README.md", IsPartial: false},
		},
	}

	// Parent has no template files
	parent := &domainprofile.Profile{
		Name:          "acme/parent",
		InheritsFrom:  "acme/grandparent",
		TemplateFiles: []domainprofile.TemplateFile{},
	}

	child := &domainprofile.Profile{
		Name:         "acme/child",
		InheritsFrom: "acme/parent",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "child.md", IsPartial: false},
		},
	}

	chain := []*domainprofile.Profile{grandparent, parent, child}
	profilesBasePath := "/home/user/.weftlo/profiles"

	merged := NewMergedProfile(chain, fs, profilesBasePath)

	// Verify we have 2 files (grandparent's README.md and child's child.md)
	sourcePaths := merged.ListSourcePaths()
	if len(sourcePaths) != 2 {
		t.Fatalf("expected 2 source paths, got %d: %v", len(sourcePaths), sourcePaths)
	}

	// Verify grandparent's file is accessible through the empty parent
	absPath, found := merged.Resolve("README.md")
	if !found {
		t.Error("expected to find README.md from grandparent")
	}
	expectedPath := "/home/user/.weftlo/profiles/acme/grandparent/content/README.md"
	if absPath != expectedPath {
		t.Errorf("expected path '%s', got '%s'", expectedPath, absPath)
	}
}

// TestMergedProfile_PartialsFromParentAccessibleAfterMerge tests that partial files
// defined in a parent profile are accessible in the merged profile for template
// include operations.
func TestMergedProfile_PartialsFromParentAccessibleAfterMerge(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	parent := &domainprofile.Profile{
		Name:         "acme/parent",
		InheritsFrom: "",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "_partials/header.md", IsPartial: true},
			{SourcePath: "_partials/footer.md", IsPartial: true},
		},
	}

	child := &domainprofile.Profile{
		Name:         "acme/child",
		InheritsFrom: "acme/parent",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "main.md", IsPartial: false},
		},
	}

	chain := []*domainprofile.Profile{parent, child}
	profilesBasePath := "/home/user/.weftlo/profiles"

	merged := NewMergedProfile(chain, fs, profilesBasePath)

	// Verify parent's partials are accessible
	headerPath, found := merged.Resolve("_partials/header.md")
	if !found {
		t.Error("expected to find parent's _partials/header.md")
	}
	expectedHeaderPath := "/home/user/.weftlo/profiles/acme/parent/content/_partials/header.md"
	if headerPath != expectedHeaderPath {
		t.Errorf("expected path '%s', got '%s'", expectedHeaderPath, headerPath)
	}

	footerPath, found := merged.Resolve("_partials/footer.md")
	if !found {
		t.Error("expected to find parent's _partials/footer.md")
	}
	expectedFooterPath := "/home/user/.weftlo/profiles/acme/parent/content/_partials/footer.md"
	if footerPath != expectedFooterPath {
		t.Errorf("expected path '%s', got '%s'", expectedFooterPath, footerPath)
	}

	// Verify child's file is also accessible
	mainPath, found := merged.Resolve("main.md")
	if !found {
		t.Error("expected to find child's main.md")
	}
	expectedMainPath := "/home/user/.weftlo/profiles/acme/child/content/main.md"
	if mainPath != expectedMainPath {
		t.Errorf("expected path '%s', got '%s'", expectedMainPath, mainPath)
	}
}

// TestMergedProfile_ImplementsFileResolverInterface tests that MergedProfile
// satisfies the FileResolver interface from the template domain package.
// This is a compile-time check that verifies interface satisfaction.
func TestMergedProfile_ImplementsFileResolverInterface(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	profile := &domainprofile.Profile{
		Name:         "acme/test",
		InheritsFrom: "",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "test.md", IsPartial: false},
		},
	}

	chain := []*domainprofile.Profile{profile}
	profilesBasePath := "/home/user/.weftlo/profiles"

	merged := NewMergedProfile(chain, fs, profilesBasePath)

	// This assignment will fail at compile time if MergedProfile does not
	// implement the FileResolver interface
	var resolver domaintemplate.FileResolver = merged

	// Verify the interface methods work correctly
	absPath, found := resolver.Resolve("test.md")
	if !found {
		t.Error("expected FileResolver.Resolve to find test.md")
	}
	if absPath != "/home/user/.weftlo/profiles/acme/test/content/test.md" {
		t.Errorf("unexpected path from FileResolver.Resolve: %s", absPath)
	}

	paths := resolver.ListSourcePaths()
	if len(paths) != 1 {
		t.Errorf("expected 1 path from FileResolver.ListSourcePaths, got %d", len(paths))
	}
}

// TestMergedProfile_ProfileNameSplittingForAbsolutePath tests that the profile name
// is correctly split into vendor and name components for absolute path construction.
// This ensures paths are correctly constructed for various vendor/name formats.
func TestMergedProfile_ProfileNameSplittingForAbsolutePath(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	testCases := []struct {
		profileName  string
		sourcePath   string
		expectedPath string
	}{
		{
			profileName:  "simple/profile",
			sourcePath:   "file.md",
			expectedPath: "/base/simple/profile/content/file.md",
		},
		{
			profileName:  "my-vendor/my-profile",
			sourcePath:   "nested/file.md",
			expectedPath: "/base/my-vendor/my-profile/content/nested/file.md",
		},
		{
			profileName:  "vendor_with_underscore/name_with_underscore",
			sourcePath:   "doc.md",
			expectedPath: "/base/vendor_with_underscore/name_with_underscore/content/doc.md",
		},
		{
			profileName:  "vendor123/profile456",
			sourcePath:   "numbered.md",
			expectedPath: "/base/vendor123/profile456/content/numbered.md",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.profileName, func(t *testing.T) {
			t.Parallel()
			profile := &domainprofile.Profile{
				Name:         tc.profileName,
				InheritsFrom: "",
				TemplateFiles: []domainprofile.TemplateFile{
					{SourcePath: tc.sourcePath, IsPartial: false},
				},
			}

			chain := []*domainprofile.Profile{profile}
			merged := NewMergedProfile(chain, fs, "/base")

			absPath, found := merged.Resolve(tc.sourcePath)
			if !found {
				t.Fatalf("expected to find '%s'", tc.sourcePath)
			}
			if absPath != tc.expectedPath {
				t.Errorf("expected path '%s', got '%s'", tc.expectedPath, absPath)
			}
		})
	}
}

// =============================================================================
// Task Group 3: Profile Chain Variable Merging Tests (5 focused tests)
// =============================================================================

// TestNewMergedProfile_MergesVariablesFromProfileChain tests that NewMergedProfile
// merges variables from the profile inheritance chain using DeepMergeVariables.
// The chain is iterated root-to-leaf, so child variables override parent variables.
func TestNewMergedProfile_MergesVariablesFromProfileChain(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create parent profile with variables
	parent := &domainprofile.Profile{
		Name:          "acme/parent",
		InheritsFrom:  "",
		TemplateFiles: []domainprofile.TemplateFile{},
		Variables: map[string]interface{}{
			"database": map[string]interface{}{
				"host": "localhost",
				"port": 5432,
			},
			"env": "development",
		},
	}

	// Create child profile with overlapping and new variables
	child := &domainprofile.Profile{
		Name:          "acme/child",
		InheritsFrom:  "acme/parent",
		TemplateFiles: []domainprofile.TemplateFile{},
		Variables: map[string]interface{}{
			"database": map[string]interface{}{
				"port":     5433, // Override parent's port
				"username": "admin",
			},
			"version": "1.0.0",
		},
	}

	chain := []*domainprofile.Profile{parent, child}
	profilesBasePath := "/home/user/.weftlo/profiles"

	merged := NewMergedProfile(chain, fs, profilesBasePath)

	// Verify merged variables
	if merged.Variables == nil {
		t.Fatal("expected Variables to be non-nil")
	}

	// Check that parent's env is preserved
	if merged.Variables["env"] != "development" {
		t.Errorf("expected env 'development', got %v", merged.Variables["env"])
	}

	// Check that child's version is added
	if merged.Variables["version"] != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %v", merged.Variables["version"])
	}

	// Check deep merge of database
	db, ok := merged.Variables["database"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected database to be map[string]interface{}, got %T", merged.Variables["database"])
	}

	// Parent's host should be preserved (deep merge)
	if db["host"] != "localhost" {
		t.Errorf("expected database.host 'localhost', got %v", db["host"])
	}

	// Child's port should override parent's port
	if db["port"] != 5433 {
		t.Errorf("expected database.port 5433, got %v", db["port"])
	}

	// Child's username should be added
	if db["username"] != "admin" {
		t.Errorf("expected database.username 'admin', got %v", db["username"])
	}
}

// TestNewMergedProfile_ChildVariablesOverrideParentVariables tests that when both
// parent and child profiles define the same variable key, the child's value wins.
func TestNewMergedProfile_ChildVariablesOverrideParentVariables(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	parent := &domainprofile.Profile{
		Name:          "acme/parent",
		InheritsFrom:  "",
		TemplateFiles: []domainprofile.TemplateFile{},
		Variables: map[string]interface{}{
			"env":       "development",
			"company":   "Acme Corp",
			"log_level": "debug",
		},
	}

	child := &domainprofile.Profile{
		Name:          "acme/child",
		InheritsFrom:  "acme/parent",
		TemplateFiles: []domainprofile.TemplateFile{},
		Variables: map[string]interface{}{
			"env":       "production", // Override
			"log_level": "info",       // Override
		},
	}

	chain := []*domainprofile.Profile{parent, child}
	profilesBasePath := "/home/user/.weftlo/profiles"

	merged := NewMergedProfile(chain, fs, profilesBasePath)

	// Child's env should override parent's
	if merged.Variables["env"] != "production" {
		t.Errorf("expected env 'production' (from child), got %v", merged.Variables["env"])
	}

	// Parent's company should be preserved
	if merged.Variables["company"] != "Acme Corp" {
		t.Errorf("expected company 'Acme Corp' (from parent), got %v", merged.Variables["company"])
	}

	// Child's log_level should override parent's
	if merged.Variables["log_level"] != "info" {
		t.Errorf("expected log_level 'info' (from child), got %v", merged.Variables["log_level"])
	}
}

// TestMergedProfile_MergeWithMergesVariablesBetweenMergedProfiles tests that
// the MergeWith() method correctly merges variables between two MergedProfiles.
func TestMergedProfile_MergeWithMergesVariablesBetweenMergedProfiles(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// First MergedProfile
	profile1 := &domainprofile.Profile{
		Name:          "acme/profile1",
		InheritsFrom:  "",
		TemplateFiles: []domainprofile.TemplateFile{},
		Variables: map[string]interface{}{
			"shared_key":   "from_profile1",
			"profile1_key": "value1",
			"nested": map[string]interface{}{
				"key1": "nested_value1",
				"key2": "nested_value2",
			},
		},
	}

	// Second MergedProfile
	profile2 := &domainprofile.Profile{
		Name:          "acme/profile2",
		InheritsFrom:  "",
		TemplateFiles: []domainprofile.TemplateFile{},
		Variables: map[string]interface{}{
			"shared_key":   "from_profile2", // Override
			"profile2_key": "value2",
			"nested": map[string]interface{}{
				"key2": "nested_value2_override", // Override nested key
				"key3": "nested_value3",
			},
		},
	}

	profilesBasePath := "/home/user/.weftlo/profiles"

	merged1 := NewMergedProfile([]*domainprofile.Profile{profile1}, fs, profilesBasePath)
	merged2 := NewMergedProfile([]*domainprofile.Profile{profile2}, fs, profilesBasePath)

	// Merge profile2 into profile1 (profile2 wins on conflicts)
	combined := merged1.MergeWith(merged2)

	// shared_key should come from profile2
	if combined.Variables["shared_key"] != "from_profile2" {
		t.Errorf("expected shared_key 'from_profile2', got %v", combined.Variables["shared_key"])
	}

	// profile1_key should be preserved
	if combined.Variables["profile1_key"] != "value1" {
		t.Errorf("expected profile1_key 'value1', got %v", combined.Variables["profile1_key"])
	}

	// profile2_key should be added
	if combined.Variables["profile2_key"] != "value2" {
		t.Errorf("expected profile2_key 'value2', got %v", combined.Variables["profile2_key"])
	}

	// Check nested deep merge
	nested, ok := combined.Variables["nested"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected nested to be map[string]interface{}, got %T", combined.Variables["nested"])
	}

	// key1 should be preserved from profile1
	if nested["key1"] != "nested_value1" {
		t.Errorf("expected nested.key1 'nested_value1', got %v", nested["key1"])
	}

	// key2 should be overridden by profile2
	if nested["key2"] != "nested_value2_override" {
		t.Errorf("expected nested.key2 'nested_value2_override', got %v", nested["key2"])
	}

	// key3 should be added from profile2
	if nested["key3"] != "nested_value3" {
		t.Errorf("expected nested.key3 'nested_value3', got %v", nested["key3"])
	}
}

// TestNewMergedProfile_AccumulatesConflictsFromChain tests that conflicts are
// accumulated from all profiles in the inheritance chain.
func TestNewMergedProfile_AccumulatesConflictsFromChain(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	grandparent := &domainprofile.Profile{
		Name:          "acme/grandparent",
		InheritsFrom:  "",
		TemplateFiles: []domainprofile.TemplateFile{},
		Variables: map[string]interface{}{
			"env": "development",
			"database": map[string]interface{}{
				"host": "localhost",
			},
		},
	}

	parent := &domainprofile.Profile{
		Name:          "acme/parent",
		InheritsFrom:  "acme/grandparent",
		TemplateFiles: []domainprofile.TemplateFile{},
		Variables: map[string]interface{}{
			"env": "staging", // Conflict with grandparent
			"database": map[string]interface{}{
				"host": "staging.db", // Conflict with grandparent
			},
		},
	}

	child := &domainprofile.Profile{
		Name:          "acme/child",
		InheritsFrom:  "acme/parent",
		TemplateFiles: []domainprofile.TemplateFile{},
		Variables: map[string]interface{}{
			"env": "production", // Conflict with parent
		},
	}

	chain := []*domainprofile.Profile{grandparent, parent, child}
	profilesBasePath := "/home/user/.weftlo/profiles"

	merged := NewMergedProfile(chain, fs, profilesBasePath)

	// Get conflicts
	conflicts := merged.VariableConflicts()

	// Should have 3 conflicts:
	// 1. grandparent -> parent: env
	// 2. grandparent -> parent: database.host
	// 3. parent -> child: env
	if len(conflicts) != 3 {
		t.Fatalf("expected 3 conflicts, got %d: %+v", len(conflicts), conflicts)
	}

	// Build a map for easier verification
	conflictMap := make(map[string]domainprofile.VariableConflict)
	for _, c := range conflicts {
		key := c.SourceProfile + "->" + c.OverridingProfile + ":" + c.Path
		conflictMap[key] = c
	}

	// Verify grandparent -> parent env conflict
	if _, ok := conflictMap["acme/grandparent->acme/parent:env"]; !ok {
		t.Error("expected conflict for env from grandparent to parent")
	}

	// Verify grandparent -> parent database.host conflict
	if _, ok := conflictMap["acme/grandparent->acme/parent:database.host"]; !ok {
		t.Error("expected conflict for database.host from grandparent to parent")
	}

	// Verify parent -> child env conflict
	if _, ok := conflictMap["acme/parent->acme/child:env"]; !ok {
		t.Error("expected conflict for env from parent to child")
	}
}

// TestMergedProfile_MultiProfileMergeWithVariables tests the LoadMergedMultiple
// pattern where multiple profiles are merged together, each with their own variables.
func TestMergedProfile_MultiProfileMergeWithVariables(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// First profile chain: base -> app
	base := &domainprofile.Profile{
		Name:          "acme/base",
		InheritsFrom:  "",
		TemplateFiles: []domainprofile.TemplateFile{},
		Variables: map[string]interface{}{
			"company": "Acme Corp",
			"settings": map[string]interface{}{
				"theme": "light",
			},
		},
	}
	app := &domainprofile.Profile{
		Name:          "acme/app",
		InheritsFrom:  "acme/base",
		TemplateFiles: []domainprofile.TemplateFile{},
		Variables: map[string]interface{}{
			"app_name": "MyApp",
		},
	}

	// Second profile chain: tools
	tools := &domainprofile.Profile{
		Name:          "acme/tools",
		InheritsFrom:  "",
		TemplateFiles: []domainprofile.TemplateFile{},
		Variables: map[string]interface{}{
			"settings": map[string]interface{}{
				"theme":   "dark", // Override base's theme
				"toolbar": true,
			},
		},
	}

	profilesBasePath := "/home/user/.weftlo/profiles"

	// Create first merged profile from base -> app chain
	merged1 := NewMergedProfile([]*domainprofile.Profile{base, app}, fs, profilesBasePath)

	// Create second merged profile from tools
	merged2 := NewMergedProfile([]*domainprofile.Profile{tools}, fs, profilesBasePath)

	// Merge them together (tools wins on conflicts)
	combined := merged1.MergeWith(merged2)

	// Verify company is preserved from first chain
	if combined.Variables["company"] != "Acme Corp" {
		t.Errorf("expected company 'Acme Corp', got %v", combined.Variables["company"])
	}

	// Verify app_name is preserved from first chain
	if combined.Variables["app_name"] != "MyApp" {
		t.Errorf("expected app_name 'MyApp', got %v", combined.Variables["app_name"])
	}

	// Verify settings is deep merged
	settings, ok := combined.Variables["settings"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected settings to be map[string]interface{}, got %T", combined.Variables["settings"])
	}

	// theme should be overridden by tools
	if settings["theme"] != "dark" {
		t.Errorf("expected settings.theme 'dark', got %v", settings["theme"])
	}

	// toolbar should be added from tools
	if settings["toolbar"] != true {
		t.Errorf("expected settings.toolbar true, got %v", settings["toolbar"])
	}

	// Verify conflicts are accumulated (theme conflict)
	conflicts := combined.VariableConflicts()
	foundThemeConflict := false
	for _, c := range conflicts {
		if c.Path == "settings.theme" {
			foundThemeConflict = true
			break
		}
	}
	if !foundThemeConflict {
		t.Error("expected conflict for settings.theme but none found")
	}
}

// =============================================================================
// Task Group 5: Gap-Filling Tests for Profile Variables Feature
// =============================================================================

// TestLoadMerged_MergesVariablesFromInheritanceChain tests that LoadMerged
// properly loads profiles from disk with variables and merges them through
// the inheritance chain. This is a critical integration test that verifies
// the end-to-end flow from disk to MergedProfile.Variables.
func TestLoadMerged_MergesVariablesFromInheritanceChain(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create parent profile with variables
	parentDir := "/home/user/.weftlo/profiles/acme/parent"
	if err := fs.MkdirAll(parentDir, 0755); err != nil {
		t.Fatalf("failed to create parent directory: %v", err)
	}
	parentYAML := `name: acme/parent
variables:
  database:
    host: localhost
    port: 5432
  env: development
  parent_only: from_parent
`
	if err := afero.WriteFile(fs, parentDir+"/profile.yaml", []byte(parentYAML), 0644); err != nil {
		t.Fatalf("failed to write parent profile.yaml: %v", err)
	}

	// Create child profile with overlapping variables
	childDir := "/home/user/.weftlo/profiles/acme/child"
	if err := fs.MkdirAll(childDir, 0755); err != nil {
		t.Fatalf("failed to create child directory: %v", err)
	}
	childYAML := `name: acme/child
inherits_from: acme/parent
variables:
  database:
    port: 5433
    username: admin
  env: production
  child_only: from_child
`
	if err := afero.WriteFile(fs, childDir+"/profile.yaml", []byte(childYAML), 0644); err != nil {
		t.Fatalf("failed to write child profile.yaml: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	merged, err := loader.LoadMerged("acme/child")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify Variables are populated
	if merged.Variables == nil {
		t.Fatal("expected Variables to be non-nil")
	}

	// Verify parent's env is overridden by child
	if merged.Variables["env"] != "production" {
		t.Errorf("expected env 'production' (from child), got %v", merged.Variables["env"])
	}

	// Verify parent_only is preserved from parent
	if merged.Variables["parent_only"] != "from_parent" {
		t.Errorf("expected parent_only 'from_parent', got %v", merged.Variables["parent_only"])
	}

	// Verify child_only is added from child
	if merged.Variables["child_only"] != "from_child" {
		t.Errorf("expected child_only 'from_child', got %v", merged.Variables["child_only"])
	}

	// Verify deep merge of database
	db, ok := merged.Variables["database"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected database to be map[string]interface{}, got %T", merged.Variables["database"])
	}

	// Parent's host should be preserved
	if db["host"] != "localhost" {
		t.Errorf("expected database.host 'localhost', got %v", db["host"])
	}

	// Child's port should override parent's
	if db["port"] != 5433 {
		t.Errorf("expected database.port 5433, got %v", db["port"])
	}

	// Child's username should be added
	if db["username"] != "admin" {
		t.Errorf("expected database.username 'admin', got %v", db["username"])
	}
}

// TestLoadMerged_TracksConflictsFromDiskProfiles tests that LoadMerged properly
// tracks variable conflicts when loading profiles with conflicting variables from disk.
func TestLoadMerged_TracksConflictsFromDiskProfiles(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create parent profile
	parentDir := "/home/user/.weftlo/profiles/acme/parent"
	if err := fs.MkdirAll(parentDir, 0755); err != nil {
		t.Fatalf("failed to create parent directory: %v", err)
	}
	parentYAML := `name: acme/parent
variables:
  env: development
  database:
    host: localhost
`
	if err := afero.WriteFile(fs, parentDir+"/profile.yaml", []byte(parentYAML), 0644); err != nil {
		t.Fatalf("failed to write parent profile.yaml: %v", err)
	}

	// Create child profile with conflicting variables
	childDir := "/home/user/.weftlo/profiles/acme/child"
	if err := fs.MkdirAll(childDir, 0755); err != nil {
		t.Fatalf("failed to create child directory: %v", err)
	}
	childYAML := `name: acme/child
inherits_from: acme/parent
variables:
  env: production
  database:
    host: prod.db.example.com
`
	if err := afero.WriteFile(fs, childDir+"/profile.yaml", []byte(childYAML), 0644); err != nil {
		t.Fatalf("failed to write child profile.yaml: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	merged, err := loader.LoadMerged("acme/child")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify conflicts are tracked
	conflicts := merged.VariableConflicts()
	if len(conflicts) != 2 {
		t.Fatalf("expected 2 conflicts (env and database.host), got %d: %+v", len(conflicts), conflicts)
	}

	// Build a map for easier verification
	conflictPaths := make(map[string]bool)
	for _, c := range conflicts {
		conflictPaths[c.Path] = true
	}

	if !conflictPaths["env"] {
		t.Error("expected conflict for 'env'")
	}
	if !conflictPaths["database.host"] {
		t.Error("expected conflict for 'database.host'")
	}
}

// TestNewMergedProfile_EmptyVariablesMapTreatedAsNoVariables tests that an empty
// variables map {} is treated the same as nil/no variables.
func TestNewMergedProfile_EmptyVariablesMapTreatedAsNoVariables(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	parent := &domainprofile.Profile{
		Name:          "acme/parent",
		InheritsFrom:  "",
		TemplateFiles: []domainprofile.TemplateFile{},
		Variables: map[string]interface{}{
			"env":     "development",
			"version": "1.0.0",
		},
	}

	// Child has empty variables map (not nil)
	child := &domainprofile.Profile{
		Name:          "acme/child",
		InheritsFrom:  "acme/parent",
		TemplateFiles: []domainprofile.TemplateFile{},
		Variables:     map[string]interface{}{}, // Empty map, not nil
	}

	chain := []*domainprofile.Profile{parent, child}
	profilesBasePath := "/home/user/.weftlo/profiles"

	merged := NewMergedProfile(chain, fs, profilesBasePath)

	// Verify parent's variables are still present (not overwritten by empty map)
	if merged.Variables["env"] != "development" {
		t.Errorf("expected env 'development' from parent, got %v", merged.Variables["env"])
	}
	if merged.Variables["version"] != "1.0.0" {
		t.Errorf("expected version '1.0.0' from parent, got %v", merged.Variables["version"])
	}

	// No conflicts should be reported (empty map doesn't conflict)
	conflicts := merged.VariableConflicts()
	if len(conflicts) != 0 {
		t.Errorf("expected 0 conflicts with empty child variables, got %d: %+v", len(conflicts), conflicts)
	}
}

// TestLoadMerged_ThreeLevelInheritanceWithVariables tests a three-level inheritance
// chain (grandparent -> parent -> child) loading from disk with variables at each level.
func TestLoadMerged_ThreeLevelInheritanceWithVariables(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create grandparent profile
	grandparentDir := "/home/user/.weftlo/profiles/acme/grandparent"
	if err := fs.MkdirAll(grandparentDir, 0755); err != nil {
		t.Fatalf("failed to create grandparent directory: %v", err)
	}
	grandparentYAML := `name: acme/grandparent
variables:
  company: Acme Corp
  env: development
  settings:
    theme: light
    logging: verbose
`
	if err := afero.WriteFile(fs, grandparentDir+"/profile.yaml", []byte(grandparentYAML), 0644); err != nil {
		t.Fatalf("failed to write grandparent profile.yaml: %v", err)
	}

	// Create parent profile
	parentDir := "/home/user/.weftlo/profiles/acme/parent"
	if err := fs.MkdirAll(parentDir, 0755); err != nil {
		t.Fatalf("failed to create parent directory: %v", err)
	}
	parentYAML := `name: acme/parent
inherits_from: acme/grandparent
variables:
  env: staging
  settings:
    logging: standard
    cache: enabled
`
	if err := afero.WriteFile(fs, parentDir+"/profile.yaml", []byte(parentYAML), 0644); err != nil {
		t.Fatalf("failed to write parent profile.yaml: %v", err)
	}

	// Create child profile
	childDir := "/home/user/.weftlo/profiles/acme/child"
	if err := fs.MkdirAll(childDir, 0755); err != nil {
		t.Fatalf("failed to create child directory: %v", err)
	}
	childYAML := `name: acme/child
inherits_from: acme/parent
variables:
  env: production
  app_name: MyApp
`
	if err := afero.WriteFile(fs, childDir+"/profile.yaml", []byte(childYAML), 0644); err != nil {
		t.Fatalf("failed to write child profile.yaml: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	merged, err := loader.LoadMerged("acme/child")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify company from grandparent (not overridden)
	if merged.Variables["company"] != "Acme Corp" {
		t.Errorf("expected company 'Acme Corp' from grandparent, got %v", merged.Variables["company"])
	}

	// Verify env from child (overridden twice)
	if merged.Variables["env"] != "production" {
		t.Errorf("expected env 'production' from child, got %v", merged.Variables["env"])
	}

	// Verify app_name from child
	if merged.Variables["app_name"] != "MyApp" {
		t.Errorf("expected app_name 'MyApp' from child, got %v", merged.Variables["app_name"])
	}

	// Verify settings deep merge
	settings, ok := merged.Variables["settings"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected settings to be map[string]interface{}, got %T", merged.Variables["settings"])
	}

	// theme from grandparent (not overridden)
	if settings["theme"] != "light" {
		t.Errorf("expected settings.theme 'light' from grandparent, got %v", settings["theme"])
	}

	// logging from parent (overrides grandparent)
	if settings["logging"] != "standard" {
		t.Errorf("expected settings.logging 'standard' from parent, got %v", settings["logging"])
	}

	// cache from parent (new key)
	if settings["cache"] != "enabled" {
		t.Errorf("expected settings.cache 'enabled' from parent, got %v", settings["cache"])
	}

	// Verify conflicts are tracked (should have 3: env grandparent->parent, env parent->child, settings.logging grandparent->parent)
	conflicts := merged.VariableConflicts()
	if len(conflicts) != 3 {
		t.Errorf("expected 3 conflicts, got %d: %+v", len(conflicts), conflicts)
	}
}
