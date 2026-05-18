// SPDX-License-Identifier: MPL-2.0
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/alfred-intelligence/shy/internal/cache"
	"github.com/alfred-intelligence/shy/internal/paths"
)

func newListCmd() *cobra.Command {
	var typeFilter string
	var sources bool
	var asJSON bool

	c := &cobra.Command{
		Use:   "list",
		Short: "Show installed snippets, aliases, completions, and plugins",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.OutOrStdout(), typeFilter, sources, asJSON)
		},
	}
	c.Flags().StringVar(&typeFilter, "type", "", "filter by type: script|plugin|alias|completion")
	c.Flags().BoolVar(&sources, "sources", false, "show the source each item came from")
	c.Flags().BoolVar(&asJSON, "json", false, "machine-readable output (plugin API)")
	return c
}

func runList(out io.Writer, typeFilter string, sources, asJSON bool) error {
	home, err := paths.Home()
	if err != nil {
		return err
	}
	c, err := cache.Load(paths.CacheFile(home))
	if err != nil {
		return err
	}
	items := c.List()
	if typeFilter != "" {
		filtered := items[:0]
		for _, it := range items {
			if it.Type == typeFilter {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}

	if asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(struct {
			Items []cache.Installed `json:"items"`
		}{Items: items})
	}

	if len(items) == 0 {
		fmt.Fprintln(out, "shy list: nothing installed yet — try `shy install <path>` or `shy collection subscribe <repo>`")
		return nil
	}

	// Group by type for friendly output.
	byType := map[string][]cache.Installed{}
	for _, it := range items {
		byType[it.Type] = append(byType[it.Type], it)
	}
	order := []string{"script", "plugin", "alias", "completion"}
	tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
	for _, t := range order {
		group := byType[t]
		if len(group) == 0 {
			continue
		}
		sort.SliceStable(group, func(i, j int) bool {
			return key(group[i]) < key(group[j])
		})
		fmt.Fprintf(tw, "\n%s:\n", strings.ToUpper(plural(t)))
		for _, it := range group {
			label := it.Name
			if it.Namespace != "" {
				label = it.Namespace + "/" + it.Name
			}
			ver := it.Version
			if ver == "" {
				ver = "—"
			}
			if sources {
				src := it.Source
				if src == "" {
					src = "local"
				}
				fmt.Fprintf(tw, "  %s\t%s\t%s\n", label, ver, src)
			} else {
				fmt.Fprintf(tw, "  %s\t%s\n", label, ver)
			}
		}
	}
	_ = tw.Flush()
	return nil
}

func key(i cache.Installed) string {
	if i.Namespace != "" {
		return i.Namespace + "/" + i.Name
	}
	return i.Name
}

func plural(t string) string {
	switch t {
	case "alias":
		return "aliases"
	default:
		return t + "s"
	}
}
