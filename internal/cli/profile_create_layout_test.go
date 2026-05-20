package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/afero"

	"github.com/simensen/weftlo/internal/cli"
)

// TestProfileCreateCommand_CreatesContentDirectory verifies that `profile create`
// scaffolds the `content/` subdirectory that the renderer actually walks.
// Without this, the generated profile produces nothing on install. See Fix 1
// in HANDOFF-UX-FIXES.md.
func TestProfileCreateCommand_CreatesContentDirectory(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	_ = fs.MkdirAll(homeDir+"/.weftlo", 0755)

	cmd := cli.NewProfileCreateCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"acme/base"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	contentDir := homeDir + "/.weftlo/profiles/acme/base/content"
	exists, err := afero.DirExists(fs, contentDir)
	if err != nil {
		t.Fatalf("failed to stat content/: %v", err)
	}
	if !exists {
		t.Fatalf("expected content/ directory at %s", contentDir)
	}
}

// TestProfileCreateCommand_CreatesStarterTemplate verifies that a working
// starter template lands inside content/, so the first install produces a
// rendered file (the value moment from the Quick Start).
func TestProfileCreateCommand_CreatesStarterTemplate(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	_ = fs.MkdirAll(homeDir+"/.weftlo", 0755)

	cmd := cli.NewProfileCreateCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"acme/base"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tmplPath := homeDir + "/.weftlo/profiles/acme/base/content/CLAUDE.md.tmpl"
	content, err := afero.ReadFile(fs, tmplPath)
	if err != nil {
		t.Fatalf("expected starter template at %s: %v", tmplPath, err)
	}

	// Starter must demonstrate variable interpolation (Fix 1's value-moment goal).
	if !strings.Contains(string(content), "{{") {
		t.Errorf("expected starter template to contain a template expression, got:\n%s", content)
	}
}

// TestProfileCreateCommand_ReadmeDoesNotPromiseRootTmpl guards against the
// original README's bad advice that .tmpl files belong at the profile root.
// See Fix 1 / Fix 2.
func TestProfileCreateCommand_ReadmeDoesNotPromiseRootTmpl(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	_ = fs.MkdirAll(homeDir+"/.weftlo", 0755)

	cmd := cli.NewProfileCreateCommandWithHomeDir(fs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"acme/base"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	readme, err := afero.ReadFile(fs, homeDir+"/.weftlo/profiles/acme/base/README.md")
	if err != nil {
		t.Fatalf("failed to read README: %v", err)
	}
	if !strings.Contains(string(readme), "content/") {
		t.Errorf("expected README to direct users to content/, got:\n%s", readme)
	}
	// "to this directory" was the misleading old phrasing pointing at the profile root.
	if strings.Contains(string(readme), "to this directory") {
		t.Errorf("README still suggests templates go at the profile root; got:\n%s", readme)
	}
}
