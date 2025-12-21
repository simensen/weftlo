package template

// FileResolver defines the interface for resolving source paths to absolute paths.
// This abstracts the Merged Profile resolution, allowing different implementations
// for production (merged profile lookup) and testing (stub/map-based lookup).
//
// The interface is designed to support the future Merged Profile implementation
// (roadmap item #10) while providing a simple contract for the include function.
type FileResolver interface {
	// Resolve takes a source path (as used in include calls, e.g., "_partials/header.md")
	// and returns the absolute filesystem path where the file can be found.
	//
	// Returns:
	//   - absolutePath: The resolved absolute path to the file
	//   - found: true if the source path was found, false otherwise
	//
	// When found is false, absolutePath will be an empty string.
	Resolve(sourcePath string) (absolutePath string, found bool)

	// ListSourcePaths returns all registered source paths available for resolution.
	// This method is used by includeGlob to enumerate paths for glob pattern matching.
	//
	// Returns:
	//   - A slice containing all source paths (keys) in the resolver
	//   - The order of paths is not guaranteed (caller should sort if needed)
	//   - Returns an empty slice (not nil) if no paths are registered
	ListSourcePaths() []string
}

// MapFileResolver is a simple map-based implementation of FileResolver.
//
// This is a bridge implementation designed to work until the full Merged Profile
// resolution is implemented (see roadmap item #10). It provides a straightforward
// way to map source paths to their absolute filesystem locations.
//
// Integration with Merged Profile:
//
// When the Merged Profile feature is implemented, the map should be populated
// by walking the profile inheritance chain and collecting template file mappings:
//
//  1. Start with the child (most specific) profile
//  2. Add all its template files to the map (sourcePath -> absolutePath)
//  3. Walk up the inheritance chain to parent profiles
//  4. For each parent, add template files that are NOT already in the map
//     (child entries take precedence over parent entries)
//  5. The resulting map represents the fully-merged view of available templates
//
// Example population from a Merged Profile:
//
//	mergedProfile := getMergedProfile(childProfileName)
//	files := make(map[string]string)
//	for _, tf := range mergedProfile.TemplateFiles {
//	    files[tf.SourcePath] = tf.AbsolutePath
//	}
//	resolver := NewMapFileResolver(files)
//
// This pattern allows the include function to resolve paths without knowing
// about the inheritance chain, as all resolution has already been performed
// during the merge phase.
type MapFileResolver struct {
	// files maps source paths to their absolute filesystem paths.
	// Keys are source paths as used in include calls (e.g., "_partials/header.md").
	// Values are absolute filesystem paths where the files are located.
	files map[string]string
}

// NewMapFileResolver creates a new MapFileResolver with the provided file mappings.
//
// The files map should contain entries where:
//   - Keys are source paths (relative paths as used in include calls)
//   - Values are absolute filesystem paths to the actual files
//
// The map is copied internally, so modifications to the original map after
// construction will not affect the resolver.
func NewMapFileResolver(files map[string]string) *MapFileResolver {
	// Create a copy of the map to prevent external modifications
	filesCopy := make(map[string]string, len(files))
	for k, v := range files {
		filesCopy[k] = v
	}
	return &MapFileResolver{files: filesCopy}
}

// Resolve looks up the given source path in the internal map and returns
// the corresponding absolute filesystem path.
//
// Returns:
//   - absolutePath: The absolute path to the file if found, empty string otherwise
//   - found: true if the source path exists in the map, false otherwise
func (r *MapFileResolver) Resolve(sourcePath string) (absolutePath string, found bool) {
	absolutePath, found = r.files[sourcePath]
	if !found {
		return "", false
	}
	return absolutePath, true
}

// ListSourcePaths returns all source paths registered in this resolver.
// The paths are returned in no particular order (map iteration order).
// Callers that require ordering should sort the returned slice.
//
// Returns:
//   - A slice containing all source paths (keys) in the internal map
//   - Returns an empty slice (not nil) if the resolver has no entries
func (r *MapFileResolver) ListSourcePaths() []string {
	paths := make([]string, 0, len(r.files))
	for k := range r.files {
		paths = append(paths, k)
	}
	return paths
}
