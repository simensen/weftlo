package profile

// ConflictKind classifies how a VariableConflict arose, which determines whether
// it is normal-but-noteworthy (inheritance) or genuinely ambiguous (collision).
//
// This distinction drives output behavior: inheritance overrides are the intended,
// designed-for behavior of child profiles refining parent values, and are noise
// in default output. Cross-profile collisions, where two unrelated profiles
// installed into the same project define the same variable, are real signal:
// the resolution is ordering-dependent and the user typically wants to know.
type ConflictKind int

const (
	// ConflictKindInheritance indicates a conflict that occurred while merging
	// variables within a single inheritance chain (parent -> child). This is the
	// expected, intended behavior of profile inheritance.
	ConflictKindInheritance ConflictKind = iota

	// ConflictKindCollision indicates a conflict that occurred while merging
	// variables across independent profiles (e.g., via LoadMergedMultiple). The
	// resolution depends on profile ordering and is typically a signal the user
	// wants to see.
	ConflictKindCollision
)

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
	// Kind classifies the conflict's origin (inheritance chain vs cross-profile collision).
	// See ConflictKind for details. Defaults to ConflictKindInheritance (zero value),
	// which matches the historical behavior of the domain DeepMergeVariables function.
	Kind ConflictKind
}
