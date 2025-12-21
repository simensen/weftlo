// Package main provides the entry point for the weftlo CLI.
package main

import (
	"os"

	"github.com/simensen/weftlo/internal/cli"
	"github.com/simensen/weftlo/internal/infrastructure/fs"
)

// version is set at build time via ldflags:
// go build -ldflags "-X main.version=1.0.0" ./cmd/weftlo
var version = "dev"

func main() {
	// Initialize the production filesystem via dependency injection
	filesystem := fs.New()

	// Create and execute the root command with the version
	rootCmd := cli.NewRootCommand(filesystem, version)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
