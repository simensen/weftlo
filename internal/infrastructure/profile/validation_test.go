package profile_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/afero"

	infraprofile "github.com/simensen/weftlo/internal/infrastructure/profile"
)

// TestValidateProfileLayout_RejectsTemplatesAtRoot verifies Fix 2: *.tmpl
// files at the profile root are a hard error, since the renderer only walks
// content/.
func TestValidateProfileLayout_RejectsTemplatesAtRoot(t *testing.T) {
	fs := afero.NewMemMapFs()
	profileDir := "/cfg/profiles/acme/base"
	_ = fs.MkdirAll(profileDir+"/content", 0755)
	_ = afero.WriteFile(fs, profileDir+"/CLAUDE.md.tmpl", []byte("oops"), 0644)
	_ = afero.WriteFile(fs, profileDir+"/profile.yaml", []byte("name: acme/base\n"), 0644)

	err := infraprofile.ValidateProfileLayout(fs, profileDir, "acme/base", "content")
	if err == nil {
		t.Fatal("expected MisplacedTemplatesError, got nil")
	}

	var misErr *infraprofile.MisplacedTemplatesError
	if !errors.As(err, &misErr) {
		t.Fatalf("expected MisplacedTemplatesError, got %T: %v", err, err)
	}
	if misErr.ProfileName != "acme/base" {
		t.Errorf("expected ProfileName 'acme/base', got %q", misErr.ProfileName)
	}
	if len(misErr.Files) != 1 || misErr.Files[0] != "CLAUDE.md.tmpl" {
		t.Errorf("expected [CLAUDE.md.tmpl], got %v", misErr.Files)
	}
	if !strings.Contains(misErr.Error(), "outside content/") {
		t.Errorf("error message should name the expected location, got: %s", misErr.Error())
	}
}

// TestValidateProfileLayout_AllowsTemplatesInContent verifies that a correct
// layout passes validation.
func TestValidateProfileLayout_AllowsTemplatesInContent(t *testing.T) {
	fs := afero.NewMemMapFs()
	profileDir := "/cfg/profiles/acme/base"
	_ = fs.MkdirAll(profileDir+"/content", 0755)
	_ = afero.WriteFile(fs, profileDir+"/content/CLAUDE.md.tmpl", []byte("ok"), 0644)
	_ = afero.WriteFile(fs, profileDir+"/profile.yaml", []byte("name: acme/base\n"), 0644)
	_ = afero.WriteFile(fs, profileDir+"/README.md", []byte("# acme/base\n"), 0644)

	if err := infraprofile.ValidateProfileLayout(fs, profileDir, "acme/base", "content"); err != nil {
		t.Errorf("expected no error for valid layout, got: %v", err)
	}
}

// TestValidateProfileLayout_NonRecursive verifies that nested .tmpl files
// outside content/ are NOT flagged — only the profile root is checked. The
// content/ walker handles everything deeper.
func TestValidateProfileLayout_NonRecursive(t *testing.T) {
	fs := afero.NewMemMapFs()
	profileDir := "/cfg/profiles/acme/base"
	_ = fs.MkdirAll(profileDir+"/extras", 0755)
	_ = afero.WriteFile(fs, profileDir+"/extras/notes.tmpl", []byte("nested"), 0644)

	if err := infraprofile.ValidateProfileLayout(fs, profileDir, "acme/base", "content"); err != nil {
		t.Errorf("expected non-recursive check to ignore nested .tmpl files, got: %v", err)
	}
}
