package cli

import (
	"fmt"
	"io"

	domainprofile "github.com/simensen/weftlo/internal/domain/profile"
)

// displayVariableConflictWarnings writes variable conflict warnings to the provided writer.
//
// Output format:
//
//	Warning: Variable 'key.nested' defined in 'vendor/parent', overridden by 'vendor/child'
//
// Inheritance overrides (a child profile refining a parent's value within a single
// inheritance chain) are the intended behavior of profiles and are noisy in default
// output; they are suppressed unless verbose is true. Cross-profile collisions
// (independent profiles installed into the same project that define the same
// variable, where resolution depends on ordering) are always shown — that case
// is real signal the user wants.
func displayVariableConflictWarnings(w io.Writer, conflicts []domainprofile.VariableConflict, verbose bool) {
	if len(conflicts) == 0 {
		return
	}

	for _, conflict := range conflicts {
		if conflict.Kind == domainprofile.ConflictKindInheritance && !verbose {
			continue
		}
		_, _ = fmt.Fprintf(w, "Warning: Variable '%s' defined in '%s', overridden by '%s'\n",
			conflict.Path,
			conflict.SourceProfile,
			conflict.OverridingProfile,
		)
	}
}
