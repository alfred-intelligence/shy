// SPDX-License-Identifier: MPL-2.0
package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/alfred-intelligence/shy/internal/cache"
	"github.com/alfred-intelligence/shy/internal/paths"
	"github.com/alfred-intelligence/shy/internal/plugin"
)

// tryDispatchPlugin examines args[0] (the first non-flag token after
// `shy`) and dispatches to a plugin script if it matches the cache
// index and is NOT also a native command. Native commands win — that
// is the rule from docs/01-whitepaper.md "Plugin model".
//
// Returns (exitCode, handled). When handled is false the caller falls
// through to cobra; when true the caller should propagate exitCode.
func tryDispatchPlugin(args []string, root *cobra.Command) (int, bool) {
	if len(args) == 0 {
		return 0, false
	}
	first := args[0]
	// Skip if the first arg looks like a flag.
	if len(first) > 0 && first[0] == '-' {
		return 0, false
	}
	// Native commands always win.
	for _, c := range root.Commands() {
		if c.Name() == first || hasAlias(c, first) {
			return 0, false
		}
	}

	home, err := paths.Home()
	if err != nil {
		return 0, false
	}
	c, err := cache.Load(paths.CacheFile(home))
	if err != nil {
		return 0, false
	}
	// Self-heal if cache is empty but the plugins/ dir has content.
	_ = plugin.EnsureFresh(home, c)

	entry, ok := plugin.Lookup(c, first)
	if !ok {
		return 0, false
	}

	cmd := exec.Command(entry.EntryScript, args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		// Preserve the plugin's own exit code when available.
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode(), true
		}
		fmt.Fprintf(os.Stderr, "shy: plugin %s: %v\n", first, err)
		return 1, true
	}
	return 0, true
}

func hasAlias(c *cobra.Command, name string) bool {
	for _, a := range c.Aliases {
		if a == name {
			return true
		}
	}
	return false
}
