package dryrun

import (
	"fmt"
	"io"

	"github.com/spf13/afero"
)

// Formatter formats dry-run output for display.
// It accepts a slice of FileOperation results and formats them for human consumption,
// outputting to the provided io.Writer (typically stdout).
type Formatter struct {
	// writer is the output destination for formatted results.
	writer io.Writer

	// fs is the filesystem for reading existing files during diff generation.
	fs afero.Fs
}

// NewFormatter creates a new Formatter with the provided output writer and filesystem.
//
// Parameters:
//   - w: The io.Writer to output formatted results (typically os.Stdout)
//   - fs: The filesystem for reading existing files during diff generation
//
// Example:
//
//	formatter := dryrun.NewFormatter(os.Stdout, afero.NewOsFs())
//	err := formatter.Format(operations)
func NewFormatter(w io.Writer, fs afero.Fs) *Formatter {
	return &Formatter{
		writer: w,
		fs:     fs,
	}
}

// Format formats and outputs the provided file operations.
// For each operation, it displays:
//   - The operation type (e.g., "would create")
//   - The source profile that provided the template
//   - The full target path after install prefix is applied
//   - The content size in human-readable format
//   - A unified diff if the file already exists and differs
//
// At the end, a summary line is printed showing the total count.
//
// Returns an error if writing to the output fails.
func (f *Formatter) Format(operations []FileOperation) error {
	for _, op := range operations {
		if err := f.formatOperation(op); err != nil {
			return err
		}
	}

	// Write summary line
	if err := f.writeSummary(operations); err != nil {
		return err
	}

	return nil
}

// formatOperation formats a single file operation.
func (f *Formatter) formatOperation(op FileOperation) error {
	// Write operation header
	// Format: "would create: target/path (size) [source-profile]"
	_, err := fmt.Fprintf(f.writer, "%s: %s (%s) [%s]\n",
		op.OperationType.String(),
		op.TargetPath,
		FormatSize(op.ContentSize()),
		op.SourceProfile,
	)
	if err != nil {
		return err
	}

	// Check if existing file exists and generate diff if so
	if err := f.writeDiffIfExists(op); err != nil {
		return err
	}

	// Write separator line for readability
	_, err = fmt.Fprintln(f.writer)
	return err
}

// writeDiffIfExists checks if a file exists at the target path and generates
// a unified diff if the content differs.
func (f *Formatter) writeDiffIfExists(op FileOperation) error {
	exists, err := afero.Exists(f.fs, op.TargetPath)
	if err != nil {
		// If we can't check existence, skip diff generation silently
		return nil
	}

	if !exists {
		return nil
	}

	// Read existing file content
	existingContent, err := afero.ReadFile(f.fs, op.TargetPath)
	if err != nil {
		// If we can't read the file, skip diff generation silently
		return nil
	}

	// Generate unified diff
	diff := GenerateDiff(existingContent, op.Content, op.TargetPath)
	if diff == "" {
		// No differences
		return nil
	}

	// Write the diff
	_, err = fmt.Fprint(f.writer, diff)
	return err
}

// writeSummary writes the summary line at the end of output.
// It shows the total count of operations grouped by type.
func (f *Formatter) writeSummary(operations []FileOperation) error {
	// Count operations by type
	createCount := 0
	for _, op := range operations {
		if op.OperationType == OperationTypeCreate {
			createCount++
		}
	}

	// Write summary with proper grammar
	var summary string
	if createCount == 1 {
		summary = "Would create 1 file"
	} else {
		summary = fmt.Sprintf("Would create %d files", createCount)
	}

	_, err := fmt.Fprintln(f.writer, summary)
	return err
}
