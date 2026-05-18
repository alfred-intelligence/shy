// SPDX-License-Identifier: MPL-2.0
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// newGenManCmd writes one man-page per subcommand into the given dir.
// Invoked by GoReleaser before building so packaged .deb/.rpm ship
// `man shy`, `man shy-install`, etc.
func newGenManCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "gen-man <dir>",
		Short:  "Generate cobra man-pages (release tooling)",
		Args:   cobra.ExactArgs(1),
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := args[0]
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("gen-man: mkdir %s: %w", dir, err)
			}
			header := &doc.GenManHeader{
				Title:   "SHY",
				Section: "1",
				Source:  "shy " + Version,
				Manual:  "shy manual",
			}
			return doc.GenManTree(cmd.Root(), header, dir)
		},
	}
}
