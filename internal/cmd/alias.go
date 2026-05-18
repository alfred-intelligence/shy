// SPDX-License-Identifier: MPL-2.0
package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alfred-intelligence/shy/internal/cache"
	"github.com/alfred-intelligence/shy/internal/install"
	"github.com/alfred-intelligence/shy/internal/paths"
)

func newAliasCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "alias <name>=<value>",
		Short: "Add (or overwrite) an alias as a flat file under $SHY_HOME/aliases",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAlias(cmd.OutOrStdout(), args[0])
		},
	}
	return c
}

func runAlias(out io.Writer, spec string) error {
	name, value, ok := strings.Cut(spec, "=")
	if !ok || name == "" || value == "" {
		return fmt.Errorf("alias: expected name=value, got %q", spec)
	}
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `'"`)
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
