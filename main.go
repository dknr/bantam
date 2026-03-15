// Package main provides the Bantam CLI entry point using Cobra.
//
// Bantam is a lightweight agent with unified message routing.
package main

import (
	"os"

	"github.com/dknr/bantam/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
