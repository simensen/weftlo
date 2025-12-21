package template

import (
	"github.com/spf13/afero"

	"github.com/simensen/weftlo/internal/app/routing"
)

// DefaultMaxIncludeDepth is the default maximum depth for nested includes.
// This prevents infinite recursion from circular includes.
const DefaultMaxIncludeDepth = 10

// RenderContext holds render-time state for template execution with include support.
// It tracks filesystem access, path resolution, and recursion depth during rendering.
//
// A new RenderContext should be created for each top-level render operation.
// The include function will modify CurrentDepth and IncludeChain during nested renders.
type RenderContext struct {
	// Fs is the filesystem interface for reading included files.
	// Use afero.NewOsFs() for production and afero.NewMemMapFs() for testing.
	Fs afero.Fs

	// FileResolver resolves source paths to absolute filesystem paths.
	// This abstracts the Merged Profile resolution logic.
	FileResolver FileResolver

	// Router is the content router for resolving source paths to target paths.
	// This is used by reference() and referenceGlob() template functions to
	// determine where files will be installed. May be nil if routing is not needed.
	Router routing.Router

	// Verbose controls whether warnings are emitted during template rendering.
	// When true, warnings about missing files, suppressed targets, and self-references
	// are output. When false, warnings are suppressed.
	Verbose bool

	// CurrentDepth tracks the current include nesting level.
	// Starts at 0 for the top-level template.
	CurrentDepth int

	// MaxDepth is the maximum allowed include depth (default 10).
	// When CurrentDepth reaches MaxDepth, further includes return IncludeDepthError.
	MaxDepth int

	// IncludeChain tracks the sequence of included files for error messages.
	// Each entry is a source path that was included to reach the current point.
	IncludeChain []string
}

// NewRenderContext creates a new RenderContext with the given filesystem and resolver.
// The MaxDepth is set to DefaultMaxIncludeDepth (10).
// CurrentDepth starts at 0 and IncludeChain starts empty.
// Router is nil and Verbose is false by default.
//
// This constructor is maintained for backward compatibility. Use NewRenderContextWithRouter
// when router access is needed for reference() and referenceGlob() functions.
func NewRenderContext(fs afero.Fs, resolver FileResolver) *RenderContext {
	return &RenderContext{
		Fs:           fs,
		FileResolver: resolver,
		Router:       nil,
		Verbose:      false,
		CurrentDepth: 0,
		MaxDepth:     DefaultMaxIncludeDepth,
		IncludeChain: []string{},
	}
}

// NewRenderContextWithRouter creates a new RenderContext with the given filesystem,
// resolver, router, and verbose flag.
//
// The Router is used by reference() and referenceGlob() template functions to resolve
// source paths to target installation paths. The router should be pre-configured with
// merged variables for dynamic target evaluation.
//
// The Verbose flag controls whether warnings are emitted during template rendering.
// When true, warnings about missing files, suppressed targets, and self-references
// are output to help with debugging.
//
// The MaxDepth is set to DefaultMaxIncludeDepth (10).
// CurrentDepth starts at 0 and IncludeChain starts empty.
func NewRenderContextWithRouter(fs afero.Fs, resolver FileResolver, router routing.Router, verbose bool) *RenderContext {
	return &RenderContext{
		Fs:           fs,
		FileResolver: resolver,
		Router:       router,
		Verbose:      verbose,
		CurrentDepth: 0,
		MaxDepth:     DefaultMaxIncludeDepth,
		IncludeChain: []string{},
	}
}

// clone creates a copy of the RenderContext with an incremented depth
// and the given source path added to the include chain.
// This is used when entering a nested include to track state.
//
// The Router and Verbose fields are propagated to the cloned context,
// ensuring that nested templates have access to routing and can emit warnings.
func (r *RenderContext) clone(sourcePath string) *RenderContext {
	newChain := make([]string, len(r.IncludeChain), len(r.IncludeChain)+1)
	copy(newChain, r.IncludeChain)
	newChain = append(newChain, sourcePath)

	return &RenderContext{
		Fs:           r.Fs,
		FileResolver: r.FileResolver,
		Router:       r.Router,
		Verbose:      r.Verbose,
		CurrentDepth: r.CurrentDepth + 1,
		MaxDepth:     r.MaxDepth,
		IncludeChain: newChain,
	}
}
