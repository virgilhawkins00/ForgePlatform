// Package main is the entry point for the Forge Platform CLI.
// Forge is a unified engineering platform combining CLI, TUI, TSDB, Wasm plugins, and AI.
package main

import (
	"os"

	"github.com/forge-platform/forge/internal/adapters/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}

