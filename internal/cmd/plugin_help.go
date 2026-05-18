// SPDX-License-Identifier: MPL-2.0
package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/alfred-intelligence/shy/internal/cache"
	"github.com/alfred-intelligence/shy/internal/paths"
	"github.com/alfred-intelligence/shy/internal/plugin"
)

// installPluginHelp wraps cobra's default help so the root command
// lists discovered plugins as a separate section after the native
// commands. Subcommand help is left untouched.
func installPluginHelp(root *cobra.Command) {
	defaultHelp := root.HelpFunc()
	root.SetHelpFunc(func(c *cobra.Command, args []string) {
		defaultHelp(c, args)
		if c != root {
			return
		}
		entries := loadPluginEntries()
		if len(entries) == 0 {
			return
		}
		out := c.OutOrStdout()
		fmt.Fprintln(out, "\nPlugins:")
		tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
		for _, e := range entries {
			desc := e.Description
			if desc == "" {
				desc = "(no description)"
			}
			fmt.Fprintf(tw, "  %s\t%s/%s\t%s\n", e.Command, e.Namespace, e.Name, desc)
		}
		_ = tw.Flush()
	})
}

func loadPluginEntries() []cache.PluginEntry {
	home, err := paths.Home()
	if err != nil {
		return nil
	}
	c, err := cache.Load(paths.CacheFile(home))
	if err != nil {
		return nil
	}
	_ = plugin.EnsureFresh(home, c)
	return c.Plugins
}
