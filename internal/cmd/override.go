// SPDX-License-Identifier: MPL-2.0
package cmd

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/alfred-intelligence/shy/internal/paths"
)

func newOverrideCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "override",
		Aliases: []string{"ov"},
		Short:   "Inspect and manage system-seed overrides",
	}
	c.AddCommand(
		newOverrideListCmd(),
		newOverrideAddCmd(),
		newOverrideRemoveCmd(),
	)
	return c
}

func newOverrideListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show overrides present in the system seed and user directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runOverrideList(cmd.OutOrStdout())
		},
	}
}

type overrideEntry struct {
	Kind   string // script | alias | completion
	Name   string
	System bool
	User   bool
}

func runOverrideList(out io.Writer) error {
	home, err := paths.Home()
	if err != nil {
		return err
	}
	seen := map[string]*overrideEntry{}
	scan := func(root string, isSystem bool) error {
		// Scan scripts: look for %-prefixed namespace dirs under overrides.d/installed/
		scriptsDir := filepath.Join(root, "overrides.d", "installed")
		nsEntries, err := os.ReadDir(scriptsDir)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		for _, ns := range nsEntries {
			if !ns.IsDir() || !strings.HasPrefix(ns.Name(), paths.ScriptPrefix) {
				continue
			}
			nsDir := filepath.Join(scriptsDir, ns.Name())
			nameEntries, err := os.ReadDir(nsDir)
			if err != nil {
				continue
			}
			for _, e := range nameEntries {
				key := "scripts/" + ns.Name() + "/" + e.Name()
				ent, ok := seen[key]
				if !ok {
					ent = &overrideEntry{Kind: "script", Name: ns.Name() + "/" + e.Name()}
					seen[key] = ent
				}
				if isSystem {
					ent.System = true
				} else {
					ent.User = true
				}
			}
		}
		// Scan aliases and completions under helpers/
		for _, pair := range []struct{ kind, subpath string }{
			{"aliases", filepath.Join(root, "overrides.d", "helpers", "aliases")},
			{"completions", filepath.Join(root, "overrides.d", "helpers", "completions")},
		} {
			entries, err := os.ReadDir(pair.subpath)
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			if err != nil {
				return err
			}
			for _, e := range entries {
				key := pair.kind + "/" + e.Name()
				ent, ok := seen[key]
				if !ok {
					ent = &overrideEntry{Kind: kindSingular(pair.kind), Name: e.Name()}
					seen[key] = ent
				}
				if isSystem {
					ent.System = true
				} else {
					ent.User = true
				}
			}
		}
		return nil
	}
	if err := scan(paths.SystemSeed(), true); err != nil {
		return err
	}
	if err := scan(home, false); err != nil {
		return err
	}
	if len(seen) == 0 {
		fmt.Fprintln(out, "shy override list: no overrides present.")
		return nil
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "KIND\tNAME\tSYSTEM\tUSER")
	for _, k := range keys {
		e := seen[k]
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", e.Kind, e.Name, tick(e.System), tick(e.User))
	}
	return tw.Flush()
}

func kindSingular(s string) string {
	switch s {
	case "scripts":
		return "script"
	case "aliases":
		return "alias"
	case "completions":
		return "completion"
	default:
		return s
	}
}

func tick(b bool) string {
	if b {
		return "yes"
	}
	return "-"
}

func newOverrideAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <type>/<name>",
		Short: "Copy an installed item into the system override directory (requires root)",
		Long: `Copy a snippet from the user's $SHY_HOME into /usr/share/shy/overrides.d/.

The override applies to every user on the host on the next ` + "`shy init`" + `.
Requires root because /usr/share/shy/ is system-owned. Run with sudo.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireRoot("override add"); err != nil {
				return err
			}
			return runOverrideAdd(cmd.OutOrStdout(), args[0])
		},
	}
}

func runOverrideAdd(out io.Writer, ref string) error {
	kind, name, err := parseOverrideRef(ref)
	if err != nil {
		return err
	}
	source, err := findUserCopy(kind, name)
	if err != nil {
		return err
	}
	var dstDir string
	switch kind {
	case "alias":
		dstDir = filepath.Join(paths.SystemSeed(), "overrides.d", "helpers", "aliases")
	case "completion":
		dstDir = filepath.Join(paths.SystemSeed(), "overrides.d", "helpers", "completions")
	case "script":
		// Derive the prefixed namespace from the source path.
		srcNSDir := filepath.Dir(source)
		nsEntry := filepath.Base(srcNSDir) // gives %<ns>
		dstDir = filepath.Join(paths.SystemSeed(), "overrides.d", "installed", nsEntry)
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return fmt.Errorf("override add: mkdir %s: %w", dstDir, err)
	}
	dst := filepath.Join(dstDir, name)
	if err := copyFileOrTree(source, dst); err != nil {
		return err
	}
	fmt.Fprintf(out, "shy override add: copied %s to %s\n", source, dst)
	fmt.Fprintln(out, "shy override add: every user must run `shy init` to materialise it under $HOME/.shy/overrides.d/.")
	return nil
}

func newOverrideRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove <type>/<name>",
		Aliases: []string{"rm"},
		Short:   "Remove an override from the system seed (requires root)",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireRoot("override remove"); err != nil {
				return err
			}
			return runOverrideRemove(cmd.OutOrStdout(), args[0])
		},
	}
}

func runOverrideRemove(out io.Writer, ref string) error {
	kind, name, err := parseOverrideRef(ref)
	if err != nil {
		return err
	}
	// Find the source to determine the prefixed namespace for scripts.
	source, err := findUserCopy(kind, name)
	if err != nil {
		return err
	}
	var dst string
	switch kind {
	case "alias":
		dst = filepath.Join(paths.SystemSeed(), "overrides.d", "helpers", "aliases", name)
	case "completion":
		dst = filepath.Join(paths.SystemSeed(), "overrides.d", "helpers", "completions", name)
	case "script":
		srcNSDir := filepath.Dir(source)
		nsEntry := filepath.Base(srcNSDir) // gives %<ns>
		dst = filepath.Join(paths.SystemSeed(), "overrides.d", "installed", nsEntry, name)
	}
	if err := os.RemoveAll(dst); err != nil {
		return fmt.Errorf("override remove: %w", err)
	}
	fmt.Fprintf(out, "shy override remove: removed %s\n", dst)
	return nil
}

func parseOverrideRef(s string) (kind, name string, err error) {
	parts := splitNS(s)
	if parts == nil {
		return "", "", fmt.Errorf("override: expected <type>/<name>, got %q", s)
	}
	switch parts[0] {
	case "script", "alias", "completion":
		return parts[0], parts[1], nil
	}
	return "", "", fmt.Errorf("override: unknown type %q (script|alias|completion)", parts[0])
}

// requireRoot blocks the calling subcommand unless running as root, or
// SHY_TEST_FAKE_ROOT=1 is set (tests).
func requireRoot(op string) error {
	if os.Getenv("SHY_TEST_FAKE_ROOT") == "1" {
		return nil
	}
	if os.Geteuid() == 0 {
		return nil
	}
	return fmt.Errorf("%s: requires root — re-run with sudo", op)
}

func findUserCopy(kind, name string) (string, error) {
	home, err := paths.Home()
	if err != nil {
		return "", err
	}
	var candidate string
	switch kind {
	case "alias":
		candidate = paths.AliasFile(home, name)
	case "completion":
		candidate = paths.CompletionFile(home, name)
	case "script":
		// Search every %-prefixed namespace for the requested name.
		matches, _ := filepath.Glob(filepath.Join(home, "installed", paths.ScriptPrefix+"*", name))
		if len(matches) == 0 {
			return "", fmt.Errorf("override add: no script %q under %s/installed/", name, home)
		}
		if len(matches) > 1 {
			return "", fmt.Errorf("override add: %q is ambiguous (%d namespaces have it); rename one first", name, len(matches))
		}
		candidate = matches[0]
	}
	if _, err := os.Stat(candidate); err != nil {
		return "", fmt.Errorf("override add: %s not found", candidate)
	}
	return candidate, nil
}

func copyFileOrTree(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, data, info.Mode().Perm())
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := copyFileOrTree(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
			return err
		}
	}
	return nil
}
