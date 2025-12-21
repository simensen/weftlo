package template

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simensen/weftlo/internal/app/routing"
	"github.com/simensen/weftlo/internal/domain/profile"
)

// TestRenderContextCreationWithRouter tests that RenderContext can be created with a Router field.
func TestRenderContextCreationWithRouter(t *testing.T) {
	fs := afero.NewMemMapFs()
	resolver := NewMapFileResolver(map[string]string{})

	// Create a minimal router for testing
	config := profile.ContentConfig{
		Targets: map[string]string{
			"claude": ".claude",
		},
	}
	router, err := routing.NewContentRouter(config, nil, nil)
	require.NoError(t, err)

	// Create render context with router using the new constructor
	ctx := NewRenderContextWithRouter(fs, resolver, router, false)

	// Verify all fields are set correctly
	assert.NotNil(t, ctx.Fs)
	assert.NotNil(t, ctx.FileResolver)
	assert.NotNil(t, ctx.Router)
	assert.Same(t, router, ctx.Router)
	assert.Equal(t, 0, ctx.CurrentDepth)
	assert.Equal(t, DefaultMaxIncludeDepth, ctx.MaxDepth)
	assert.Empty(t, ctx.IncludeChain)
	assert.False(t, ctx.Verbose)
}

// TestRenderContextClonePreservesRouter tests that clone() preserves the Router reference.
func TestRenderContextClonePreservesRouter(t *testing.T) {
	fs := afero.NewMemMapFs()
	resolver := NewMapFileResolver(map[string]string{})

	// Create a minimal router for testing
	config := profile.ContentConfig{
		Targets: map[string]string{
			"claude": ".claude",
		},
	}
	router, err := routing.NewContentRouter(config, nil, nil)
	require.NoError(t, err)

	// Create render context with router
	ctx := NewRenderContextWithRouter(fs, resolver, router, true)

	// Clone the context
	clonedCtx := ctx.clone("test/path.md")

	// Verify Router is preserved in the clone
	assert.NotNil(t, clonedCtx.Router)
	assert.Same(t, router, clonedCtx.Router)

	// Verify Verbose is preserved in the clone
	assert.True(t, clonedCtx.Verbose)

	// Verify other clone behavior still works
	assert.Equal(t, 1, clonedCtx.CurrentDepth)
	assert.Equal(t, []string{"test/path.md"}, clonedCtx.IncludeChain)
}

// TestRenderContextVerboseFlag tests that the Verbose flag can be enabled and disabled.
func TestRenderContextVerboseFlag(t *testing.T) {
	fs := afero.NewMemMapFs()
	resolver := NewMapFileResolver(map[string]string{})

	// Test with verbose disabled
	ctxQuiet := NewRenderContextWithRouter(fs, resolver, nil, false)
	assert.False(t, ctxQuiet.Verbose)

	// Test with verbose enabled
	ctxVerbose := NewRenderContextWithRouter(fs, resolver, nil, true)
	assert.True(t, ctxVerbose.Verbose)

	// Verify verbose flag is preserved through clone
	clonedQuiet := ctxQuiet.clone("path1.md")
	assert.False(t, clonedQuiet.Verbose)

	clonedVerbose := ctxVerbose.clone("path2.md")
	assert.True(t, clonedVerbose.Verbose)
}

// TestNewRenderContextWithRouterConstructor tests the NewRenderContextWithRouter constructor.
func TestNewRenderContextWithRouterConstructor(t *testing.T) {
	fs := afero.NewMemMapFs()
	resolver := NewMapFileResolver(map[string]string{})

	// Create a router with variables
	config := profile.ContentConfig{
		Targets: map[string]string{
			"claude":    ".claude",
			"_partials": "_partials",
		},
	}
	router, err := routing.NewContentRouter(config, nil, nil)
	require.NoError(t, err)

	// Test constructor with all parameters
	ctx := NewRenderContextWithRouter(fs, resolver, router, true)

	// Verify all fields
	assert.Same(t, fs, ctx.Fs)
	assert.Same(t, resolver, ctx.FileResolver)
	assert.Same(t, router, ctx.Router)
	assert.True(t, ctx.Verbose)
	assert.Equal(t, 0, ctx.CurrentDepth)
	assert.Equal(t, DefaultMaxIncludeDepth, ctx.MaxDepth)
	assert.Empty(t, ctx.IncludeChain)

	// Test backward compatibility - existing NewRenderContext still works
	ctxOld := NewRenderContext(fs, resolver)
	assert.Same(t, fs, ctxOld.Fs)
	assert.Same(t, ctxOld.FileResolver, resolver)
	assert.Nil(t, ctxOld.Router)
	assert.False(t, ctxOld.Verbose)
}
