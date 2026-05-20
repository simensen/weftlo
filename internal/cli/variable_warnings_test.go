package cli

import (
	"bytes"
	"strings"
	"testing"

	domainprofile "github.com/simensen/weftlo/internal/domain/profile"
)

// TestDisplayVariableConflictWarnings_FiltersByKindAndVerbose locks in the
// behavior added in Fix 8: inheritance overrides are noise in default output
// (suppressed unless --verbose), while cross-profile collisions are signal
// (always shown).
func TestDisplayVariableConflictWarnings_FiltersByKindAndVerbose(t *testing.T) {
	t.Parallel()

	inheritance := domainprofile.VariableConflict{
		Path:              "app_name",
		SourceProfile:     "vendor/parent",
		OverridingProfile: "vendor/child",
		Kind:              domainprofile.ConflictKindInheritance,
	}
	collision := domainprofile.VariableConflict{
		Path:              "company",
		SourceProfile:     "vendor/a",
		OverridingProfile: "vendor/b",
		Kind:              domainprofile.ConflictKindCollision,
	}

	cases := []struct {
		name              string
		conflicts         []domainprofile.VariableConflict
		verbose           bool
		expectInheritance bool
		expectCollision   bool
	}{
		{
			name:              "default output suppresses inheritance",
			conflicts:         []domainprofile.VariableConflict{inheritance},
			verbose:           false,
			expectInheritance: false,
		},
		{
			name:              "verbose surfaces inheritance",
			conflicts:         []domainprofile.VariableConflict{inheritance},
			verbose:           true,
			expectInheritance: true,
		},
		{
			name:            "default output keeps collisions loud",
			conflicts:       []domainprofile.VariableConflict{collision},
			verbose:         false,
			expectCollision: true,
		},
		{
			name:              "mixed: only collisions in default",
			conflicts:         []domainprofile.VariableConflict{inheritance, collision},
			verbose:           false,
			expectInheritance: false,
			expectCollision:   true,
		},
		{
			name:              "mixed: both surface under verbose",
			conflicts:         []domainprofile.VariableConflict{inheritance, collision},
			verbose:           true,
			expectInheritance: true,
			expectCollision:   true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			displayVariableConflictWarnings(&buf, tc.conflicts, tc.verbose)
			out := buf.String()

			gotInheritance := strings.Contains(out, "'app_name'")
			gotCollision := strings.Contains(out, "'company'")

			if gotInheritance != tc.expectInheritance {
				t.Errorf("inheritance visibility: got=%v want=%v; output=%q", gotInheritance, tc.expectInheritance, out)
			}
			if gotCollision != tc.expectCollision {
				t.Errorf("collision visibility: got=%v want=%v; output=%q", gotCollision, tc.expectCollision, out)
			}
		})
	}
}
