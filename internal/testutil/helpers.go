// Package testutil provides shared test helper functions and utilities.
// This package is intended for use in test files only.
package testutil

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/afero"

	"github.com/simensen/weftlo/internal/domain/config"
	"github.com/simensen/weftlo/internal/domain/profile"
	"github.com/simensen/weftlo/internal/domain/template"
	infraprofile "github.com/simensen/weftlo/internal/infrastructure/profile"
)

// CreateTestFilesystem creates an in-memory filesystem populated with the provided files.
// The files map uses file paths as keys and file contents as values.
// Parent directories are automatically created for each file.
//
// This helper enables fast, isolated tests without touching the real filesystem.
// All errors are fatal test failures with descriptive messages.
//
// Usage:
//
//	fs := testutil.CreateTestFilesystem(t, map[string]string{
//	    "/config/config.yaml": "default_profile: vendor/test",
//	    "/project/.weftlo.yaml": "profile: vendor/proj",
//	})
func CreateTestFilesystem(t *testing.T, files map[string]string) afero.Fs {
	t.Helper()
	fs := afero.NewMemMapFs()
	for path, content := range files {
		dir := filepath.Dir(path)
		if err := fs.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
		if err := afero.WriteFile(fs, path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", path, err)
		}
	}
	return fs
}

// CreateTestProfile creates a Profile with the given name and template files.
// Template files are marked as non-partial by default.
// The variadic templateFiles argument allows convenient creation of profiles
// with any number of source paths.
//
// Usage:
//
//	p := testutil.CreateTestProfile(t, "vendor/myprofile", "file1.md", "file2.md")
func CreateTestProfile(t *testing.T, name string, templateFiles ...string) *profile.Profile {
	t.Helper()
	files := make([]profile.TemplateFile, len(templateFiles))
	for i, f := range templateFiles {
		files[i] = profile.TemplateFile{SourcePath: f, IsPartial: false}
	}
	return &profile.Profile{Name: name, TemplateFiles: files}
}

// CreateTestMergedProfile creates a MergedProfile from a chain of profiles.
// The chain should be ordered root-to-leaf (parent profiles first, child profiles last).
// The profilesBasePath is used for computing absolute paths of template files.
//
// This helper enables testing of merged profile functionality without manual
// setup of the infrastructure layer.
//
// Usage:
//
//	parent := testutil.CreateTestProfile(t, "vendor/parent", "shared.md")
//	child := testutil.CreateTestProfile(t, "vendor/child", "custom.md")
//	merged := testutil.CreateTestMergedProfile(t, fs, "/profiles", parent, child)
func CreateTestMergedProfile(t *testing.T, fs afero.Fs, profilesBasePath string, chain ...*profile.Profile) *infraprofile.MergedProfile {
	t.Helper()
	return infraprofile.NewMergedProfile(chain, fs, profilesBasePath)
}

// CreateTestMergedProfileWithContentDir creates a MergedProfile from a chain of profiles
// with a specified content directory root. This is useful for testing Router integration
// where templates are organized in subdirectories under a content root.
//
// The contentDir parameter sets the ContentConfig.Root on the last profile in the chain,
// which determines where MergedProfile looks for template files.
//
// Usage:
//
//	profile := testutil.CreateTestProfile(t, "vendor/test", "claude/settings.md.tmpl")
//	merged := testutil.CreateTestMergedProfileWithContentDir(t, fs, "/profiles", profile, "content")
func CreateTestMergedProfileWithContentDir(t *testing.T, fs afero.Fs, profilesBasePath string, chain *profile.Profile, contentDir string) *infraprofile.MergedProfile {
	t.Helper()
	// Set the ContentConfig.Root on the profile to specify the content directory
	chain.ContentConfig.Root = contentDir
	return infraprofile.NewMergedProfile([]*profile.Profile{chain}, fs, profilesBasePath)
}

// ContextOption is a functional option for configuring a TemplateContext.
// Use the With* functions to create options for CreateTestTemplateContext.
type ContextOption func(*template.TemplateContext)

// WithProfile sets the Profile on the TemplateContext.
func WithProfile(p *profile.Profile) ContextOption {
	return func(ctx *template.TemplateContext) {
		ctx.Profile = p
	}
}

// WithProjectDir sets the ProjectDir on the TemplateContext.
func WithProjectDir(dir string) ContextOption {
	return func(ctx *template.TemplateContext) {
		ctx.ProjectDir = dir
	}
}

// WithVariables sets the Variables on the TemplateContext.
func WithVariables(vars map[string]interface{}) ContextOption {
	return func(ctx *template.TemplateContext) {
		ctx.Variables = vars
	}
}

// WithGeneratedAt sets the GeneratedAt timestamp on the TemplateContext.
func WithGeneratedAt(t time.Time) ContextOption {
	return func(ctx *template.TemplateContext) {
		ctx.GeneratedAt = t
	}
}

// WithProjectConfig sets the ProjectConfig on the TemplateContext.
func WithProjectConfig(cfg *config.ProjectConfig) ContextOption {
	return func(ctx *template.TemplateContext) {
		ctx.ProjectConfig = cfg
	}
}

// WithGlobalConfig sets the GlobalConfig on the TemplateContext.
func WithGlobalConfig(cfg *config.GlobalConfig) ContextOption {
	return func(ctx *template.TemplateContext) {
		ctx.GlobalConfig = cfg
	}
}

// CreateTestTemplateContext creates a TemplateContext with sensible defaults
// and applies any provided options.
//
// Default values:
//   - Profile: nil
//   - ProjectDir: "/test/project"
//   - GeneratedAt: fixed time (2024-01-15 10:30:00 UTC)
//   - ProjectConfig: nil
//   - GlobalConfig: nil
//   - Variables: nil
//
// Usage:
//
//	ctx := testutil.CreateTestTemplateContext(t,
//	    testutil.WithProfile(myProfile),
//	    testutil.WithVariables(map[string]interface{}{"key": "value"}),
//	)
func CreateTestTemplateContext(t *testing.T, opts ...ContextOption) *template.TemplateContext {
	t.Helper()

	// Create context with sensible defaults
	ctx := &template.TemplateContext{
		Profile:       nil,
		ProjectDir:    "/test/project",
		GeneratedAt:   time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		ProjectConfig: nil,
		GlobalConfig:  nil,
		Variables:     nil,
	}

	// Apply all options
	for _, opt := range opts {
		opt(ctx)
	}

	return ctx
}
