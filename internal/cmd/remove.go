// SPDX-License-Identifier: MPL-2.0
package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/alfred-intelligence/shy/internal/cache"
	"github.com/alfred-intelligence/shy/internal/install"
	"github.com/alfred-intelligence/shy/internal/paths"
	"github.com/alfred-intelligence/shy/internal/plugin"
)

func newRemoveCmd() *cobra.Command {
	var silent bool
	c := &cobra.Command{
		Use:     "remove <namespace>/<name>|alias:<name>|completion:<tool>",
		Aliases: []string{"rm", "uninstall"},
		Short:   "Remove an installed item",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(cmd.OutOrStdout(), args[0], silent)
		},
	}
	c.Flags().BoolVar(&silent, "silent", false, "suppress output (plugin API)")
	return c
}

func runRemove(out io.Writer, ref string, silent bool) error {
	home, err := paths.Home()
	if err != nil {
		return err
	}
	c, err := cache.Load(paths.CacheFile(home))
	if err != nil {
		return err
	}

	kind, ns, name, err := parseRemoveRef(home, ref)
	if err != nil {
		return err
	}
	removed, err := install.RemoveItem(home, kind, ns, name, c)
	if err != nil {
		return err
	}
	if err := plugin.Rebuild(home, c); err != nil {
		return err
	}
	if err := c.Save(); err != nil {
		return err
	}
	if !silent {
		if removed {
			fmt.Fprintf(out, "shy remove: removed %s %s\n", kind, ref)
		} else {
			fmt.Fprintf(out, "shy remove: %s %s was not in the cache (filesystem cleanup done if found)\n", kind, ref)
		}
	}
	return nil
}

func parseRemoveRef(home, ref string) (kind, ns, name string, err error) {
	switch {
	case len(ref) > 6 && ref[:6] == "alias:":
		return "alias", "", ref[6:], nil
	case len(ref) > 11 && ref[:11] == "completion:":
		return "completion", "", ref[11:], nil
	}
	parts := splitNS(ref)
	if parts == nil {
		return "", "", "", fmt.Errorf("remove: expected <namespace>/<name>, alias:<name>, or completion:<tool>")
	}
	ns, name = parts[0], parts[1]
	// Infer kind from where the item lives on disk.
	if _, statErr := os.Stat(filepath.Join(home, "scripts", ns, name)); statErr == nil {
		return "script", ns, name, nil
	}
	if _, statErr := os.Stat(filepath.Join(home, "plugins", ns, name)); statErr == nil {
		return "plugin", ns, name, nil
	}
	return "", "", "", fmt.Errorf("remove: %s/%s not found under scripts/ or plugins/", ns, name)
}

func splitNS(s string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == '/' && i > 0 && i < len(s)-1 {
			return []string{s[:i], s[i+1:]}
		}
	}
	return nil
}
