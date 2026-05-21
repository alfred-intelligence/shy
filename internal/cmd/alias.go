// SPDX-License-Identifier: MPL-2.0
package cmd

import (
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/alfred-intelligence/shy/internal/cache"
	"github.com/alfred-intelligence/shy/internal/install"
	"github.com/alfred-intelligence/shy/internal/paths"
)

func newAliasCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "alias <name>=<value...>",
		Short: "Add (or overwrite) an alias as a flat file under $SHY_HOME/aliases",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAlias(cmd.OutOrStdout(), args)
		},
	}
	return c
}

func runAlias(out io.Writer, args []string) error {
	// Join all args so `shy alias ll=ls -alh` works without shell quoting.
	spec := strings.Join(args, " ")
	name, value, ok := strings.Cut(spec, "=")
	if !ok || name == "" || value == "" {
		return fmt.Errorf("alias: expected name=value [value...], got %q", spec)
	}
	if !isBashIdentifier(name) {
		return fmt.Errorf("alias: %q is not a valid bash identifier", name)
	}
	home, err := paths.Home()
	if err != nil {
		return err
	}
	c, err := cache.Load(paths.CacheFile(home))
	if err != nil {
		return err
	}
	dst := paths.AliasFile(home, name)
	content := fmt.Sprintf("alias %s=%s\n", name, shellSingleQuote(value))
	if err := writeFileForce(dst, []byte(content)); err != nil {
		return err
	}
	c.Add(cache.Installed{Type: "alias", Name: name, Source: "local"})
	if err := c.Save(); err != nil {
		return err
	}
	fmt.Fprintf(out, "shy alias: wrote %s\n", dst)
	_ = install.ConflictFail
	return nil
}

func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// isBashIdentifier returns true when s is a valid bash alias name:
// starts with a letter or underscore, followed by letters, digits,
// underscores, or hyphens.
func isBashIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !unicode.IsLetter(r) && r != '_' {
				return false
			}
		} else {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' {
				return false
			}
		}
	}
	return true
}
