// SPDX-License-Identifier: MPL-2.0
package cmd

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/alfred-intelligence/shy/internal/cache"
	"github.com/alfred-intelligence/shy/internal/install"
	"github.com/alfred-intelligence/shy/internal/paths"
)

func newUpdateCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "update [<namespace>/<name>]",
		Short: "Refetch and reinstall items that came from a [source]",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := ""
			if len(args) == 1 {
				target = args[0]
			}
			return runUpdate(cmd.OutOrStdout(), target)
		},
	}
	return c
}

func runUpdate(out io.Writer, target string) error {
	home, err := paths.Home()
	if err != nil {
		return err
	}
	c, err := cache.Load(paths.CacheFile(home))
	if err != nil {
		return err
	}

	candidates := []cache.Installed{}
	for _, it := range c.List() {
		if it.Source == "" {
			continue
		}
		if target != "" {
			if it.Namespace+"/"+it.Name != target && it.Name != target {
				continue
			}
		}
		candidates = append(candidates, it)
	}
	if len(candidates) == 0 {
		if target != "" {
			return errors.New("update: nothing to update — target has no [source] (locally created items cannot be updated; see `shy publish`)")
		}
		fmt.Fprintln(out, "shy update: nothing to update.")
		return nil
	}

	updated := 0
	for _, it := range candidates {
		dir, source, ref, cleanup, err := materialise(updateSpec(it), false)
		if err != nil {
			fmt.Fprintf(out, "  ! %s: %v\n", it.Source, err)
			continue
		}
		opts := install.Options{
			Home:   home,
			Source: source,
			Ref:    ref,
			Policy: install.ConflictPreferNew,
		}
		if _, err := install.Bundle(dir, opts, c); err != nil {
			cleanup()
			fmt.Fprintf(out, "  ! %s: %v\n", it.Source, err)
			continue
		}
		cleanup()
		updated++
		fmt.Fprintf(out, "  ✓ %s\n", it.Source)
	}
	if err := c.Save(); err != nil {
		return err
	}
	fmt.Fprintf(out, "shy update: %d/%d source(s) updated\n", updated, len(candidates))
	return nil
}

// updateSpec re-derives an install spec from a cached entry's source.
// Sources of the form "alice/git-autofetch" → "@alice/git-autofetch";
// anything URL-ish or path-ish passes through.
func updateSpec(it cache.Installed) string {
	s := it.Source
	if s == "" {
		return ""
	}
	switch {
	case s[0] == '@', startsWith(s, "github:"), startsWith(s, "http://"),
		startsWith(s, "https://"), startsWith(s, "file://"),
		startsWith(s, "local:"):
		if startsWith(s, "local:") {
			return s[len("local:"):]
		}
		return s
	}
	if hasSlash(s) {
		return "@" + s
	}
	return s
}

func startsWith(s, p string) bool {
	return len(s) >= len(p) && s[:len(p)] == p
}

func hasSlash(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			return true
		}
	}
	return false
}
