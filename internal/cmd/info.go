// SPDX-License-Identifier: MPL-2.0
package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/spf13/cobra"

	"github.com/alfred-intelligence/shy/internal/manifest"
	"github.com/alfred-intelligence/shy/internal/paths"
)

func newInfoCmd() *cobra.Command {
	var raw bool
	var asJSON bool

	c := &cobra.Command{
		Use:   "info <namespace>/<name>",
		Short: "Show an item's README and manifest metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInfo(cmd.OutOrStdout(), args[0], raw, asJSON)
		},
	}
	c.Flags().BoolVar(&raw, "raw", false, "emit raw markdown instead of glamour-rendered output")
	c.Flags().BoolVar(&asJSON, "json", false, "machine-readable manifest output (plugin API)")
	return c
}

func runInfo(out io.Writer, ref string, raw, asJSON bool) error {
	home, err := paths.Home()
	if err != nil {
		return err
	}
	ns, name, ok := splitRef(ref)
	if !ok {
		return fmt.Errorf("info: expected <namespace>/<name>, got %q", ref)
	}

	dir, ok := findItemDir(home, ns, name)
	if !ok {
		return fmt.Errorf("info: no installed item %s/%s", ns, name)
	}

	mPath := filepath.Join(dir, "manifest.toml")
	var m *manifest.Manifest
	if data, err := os.ReadFile(mPath); err == nil {
		m, _ = manifest.Parse(data)
	}

	if asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		payload := map[string]any{
			"namespace": ns,
			"name":      name,
			"dir":       dir,
			"manifest":  m,
		}
		return enc.Encode(payload)
	}

	readme, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if errors.Is(err, fs.ErrNotExist) {
		if m == nil {
			fmt.Fprintf(out, "%s/%s — no manifest, no README\n", ns, name)
			return nil
		}
		fmt.Fprintf(out, "%s/%s — %s (%s)\n", ns, name, m.Description, m.Version)
		return nil
	}
	if err != nil {
		return fmt.Errorf("info: read README: %w", err)
	}

	if raw {
		_, _ = out.Write(readme)
		return nil
	}
	rendered, err := glamour.Render(string(readme), "auto")
	if err != nil {
		// Fall back to raw on render failure.
		_, _ = out.Write(readme)
		return nil
	}
	_, _ = io.WriteString(out, rendered)
	return nil
}

func splitRef(s string) (ns, name string, ok bool) {
	if i := strings.Index(s, "/"); i > 0 && i < len(s)-1 {
		return s[:i], s[i+1:], true
	}
	return "", "", false
}

func findItemDir(home, ns, name string) (string, bool) {
	for _, base := range []string{"scripts", "plugins"} {
		dir := filepath.Join(home, base, ns, name)
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir, true
		}
	}
	return "", false
}
