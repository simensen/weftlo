package profile

// VariableConflict represents a variable override conflict that occurs during
// deep merge of profile inheritance chain variables. It tracks when a variable
// at a specific path is overridden by a child profile, providing information
// for warning display to users.
type VariableConflict struct {
	// Path is the dot-notation path to the conflicting variable (e.g., "database.connection.host")
	Path string
	// SourceProfile is the name of the profile that originally defined the variable (e.g., "vendor/parent")
	SourceProfile string
	// OverridingProfile is the name of the profile that overrides the variable (e.g., "vendor/child")
	OverridingProfile string
}
