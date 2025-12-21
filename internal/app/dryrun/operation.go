package dryrun

// FileOperation represents a single planned file operation during dry-run mode.
// It contains all information needed to display what would happen when the
// install command is executed without the --dry-run flag.
type FileOperation struct {
	// SourceProfile is the name of the profile that provided the template file.
	// This tracks file provenance through the inheritance chain.
	SourceProfile string

	// TargetPath is the full destination path where the file would be written.
	// This includes the install prefix applied by the PathMapper.
	TargetPath string

	// Content is the rendered template content that would be written to disk.
	Content []byte

	// OperationType indicates what kind of operation would be performed
	// (create, update, or delete). Currently only "create" is supported.
	OperationType OperationType
}

// NewFileOperation creates a new FileOperation with the given parameters.
// This constructor provides a clear way to create operations with all fields.
func NewFileOperation(sourceProfile, targetPath string, content []byte, opType OperationType) FileOperation {
	return FileOperation{
		SourceProfile: sourceProfile,
		TargetPath:    targetPath,
		Content:       content,
		OperationType: opType,
	}
}

// ContentSize returns the size of the content in bytes.
func (f FileOperation) ContentSize() int64 {
	return int64(len(f.Content))
}
