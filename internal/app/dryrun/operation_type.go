// Package dryrun provides application-layer services for dry-run mode output.
// It formats and displays planned file operations without writing any files
// to disk, allowing users to preview what would happen during installation.
package dryrun

// OperationType represents the type of file operation that would be performed.
// Currently only "create" is supported; "update" and "delete" require
// Change Detection (#16) and are reserved for future implementation.
type OperationType string

const (
	// OperationTypeCreate indicates a new file would be created.
	OperationTypeCreate OperationType = "create"

	// OperationTypeUpdate is reserved for future use when Change Detection (#16)
	// is implemented. It will indicate an existing file would be modified.
	OperationTypeUpdate OperationType = "update"

	// OperationTypeDelete is reserved for future use when Change Detection (#16)
	// is implemented. It will indicate an existing file would be removed.
	OperationTypeDelete OperationType = "delete"
)

// String returns a human-readable description of the operation type.
// This is used for display in dry-run output.
func (o OperationType) String() string {
	switch o {
	case OperationTypeCreate:
		return "would create"
	case OperationTypeUpdate:
		return "would update"
	case OperationTypeDelete:
		return "would delete"
	default:
		return "unknown operation"
	}
}
