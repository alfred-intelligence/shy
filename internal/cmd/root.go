// SPDX-License-Identifier: MPL-2.0

// Package cmd wires cobra subcommands. Each command lives in its own
// file (init.go, install.go, ...). Anything that does real work belongs
// in a sibling internal/ package; cmd/ stays thin so tests can target
// the logic directly.
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is set from main via SetVersion at startup so the binary
// build pipeline owns versioning.
var Version = "dev"

// SetVersion overrides the displayed version string.
func SetVersion(v string) {
	Version = v
}

// New builds the root command tree with every subcommand registered.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:           "shy",
		Short:         "Small Shell Utility — bash snippet, alias, completion manager",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       Version,
	}
	root.SetVersionTemplate("shy {{.Version}}\n")

	// Suppress cobra's default `completion` subcommand; we register our
	// own that hosts both `add <tool>` and the per-shell emitters.
	root.CompletionOptions.DisableDefaultCmd = true

	root.AddCommand(
		newInitCmd(),
		newInstallCmd(),
		newListCmd(),
		newInfoCmd(),
		newRemoveCmd(),
		newUpdateCmd(),
		newAliasCmd(),
		newCompletionCmd(),
		newGenManCmd(),
	)
	return root
}

// Execute runs the CLI with the given context, returning the process
// exit code.
func Execute(ctx context.Context) int {
	root := New()
	if err := root.ExecuteContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "shy: %v\n", err)
		return 1
	}
	return 0
}
