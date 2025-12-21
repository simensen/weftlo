package manifest

// RenderedFile represents a single template file that has been processed
// through the rendering pipeline. It serves as input to the ManifestGenerator.
type RenderedFile struct {
	// TargetPath is the destination path for the rendered file (relative to project root).
	TargetPath string

	// SourceContent is the raw template content before rendering.
	SourceContent []byte

	// OutputContent is the rendered content after template processing.
	OutputContent []byte

	// SourceProfile is the name of the profile that provided this template file.
	// This tracks file provenance through the inheritance chain.
	SourceProfile string

	// TargetCategory is the source directory name that was used to determine
	// the target installation path. This is populated from RoutedFile.TargetCategory
	// during the install process and used for grouping files in CLI output.
	TargetCategory string
}
