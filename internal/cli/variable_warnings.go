package cli

import (
	"fmt"
	"io"

	domainprofile "github.com/simensen/weftlo/internal/domain/profile"
)

// displayVariableConflictWarnings writes variable conflict warnings to the provided writer.
// It displays warnings in the format:
//
//	Warning: Variable 'key.nested' defined in 'vendor/parent', overridden by 'vendor/child'
//
// This function only displays output when there are actual conflicts.
// The warnings are displayed before main operation output.
func displayVariableConflictWarnings(w io.Writer, conflicts []domainprofile.VariableConflict) {
	if len(conflicts) == 0 {
		return
	}

	for _, conflict := range conflicts {
		_, _ = fmt.Fprintf(w, "Warning: Variable '%s' defined in '%s', overridden by '%s'\n",
			conflict.Path,
			conflict.SourceProfile,
			conflict.OverridingProfile,
		)
	}
}
