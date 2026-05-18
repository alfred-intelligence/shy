// SPDX-License-Identifier: MPL-2.0
package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/alfred-intelligence/shy/internal/cache"
	"github.com/alfred-intelligence/shy/internal/collection"
	"github.com/alfred-intelligence/shy/internal/install"
	"github.com/alfred-intelligence/shy/internal/paths"
	"github.com/alfred-intelligence/shy/internal/plugin"
)

func newCollectionCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "collection",
		Aliases: []string{"col"},
		Short:   "Subscribe to, list, update, or unsubscribe from collections",
	}
	c.AddCommand(
		newCollectionSubscribeCmd(),
		newCollectionListCmd(),
		newCollectionUpdateCmd(),
		newCollectionUnsubscribeCmd(),
	)
	return c
}

func newCollectionSubscribeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "subscribe <github:user/name[@ref]|https://...>",
		Short: "Clone a collection and install every item it declares",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := paths.Home()
			if err != nil {
				return err
			}
			c, err := cache.Load(paths.CacheFile(home))
			if err != nil {
				return err
			}
			res, err := collection.Subscribe(collection.SubscribeOptions{
				Home:   home,
				Spec:   args[0],
				Policy: install.PolicyFromEnv(),
			}, c)
			if err != nil {
				return err
			}
			if err := plugin.Rebuild(home, c); err != nil {
				return err
			}
			if err := c.Save(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "shy collection subscribe: %s (%s)\n", res.Name, res.Commit)
			fmt.Fprintf(cmd.OutOrStdout(), "  %d item(s) installed\n", len(res.Installed))
			return nil
		},
	}
}

func newCollectionListCmd() *cobra.Command {
	var asJSON bool
	c := &cobra.Command{
		Use:   "list",
		Short: "Show subscribed collections",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			home, err := paths.Home()
			if err != nil {
				return err
			}
			cc, err := cache.Load(paths.CacheFile(home))
			if err != nil {
				return err
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(struct {
					Collections map[string]cache.Subscription `json:"collections"`
				}{Collections: cc.Collections})
			}
			if len(cc.Collections) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "shy collection list: nothing subscribed yet.")
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Subscribed collections:")
			for _, sub := range cc.Collections {
				ref := sub.Ref
				if ref == "" {
					ref = "default-branch"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  %-30s  %s  %s  %s\n", sub.Name, sub.Repo, ref, shortCommit(sub.Commit))
			}
			return nil
		},
	}
	c.Flags().BoolVar(&asJSON, "json", false, "machine-readable output (plugin API)")
	return c
}

func shortCommit(c string) string {
	if len(c) >= 7 {
		return c[:7]
	}
	return c
}

func newCollectionUpdateCmd() *cobra.Command {
	var apply bool
	var only string
	c := &cobra.Command{
		Use:   "update [<name>]",
		Short: "Re-fetch each subscribed collection (dry-run by default)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				only = args[0]
			}
			home, err := paths.Home()
			if err != nil {
				return err
			}
			c, err := cache.Load(paths.CacheFile(home))
			if err != nil {
				return err
			}
			if !apply {
				return previewUpdate(cmd.OutOrStdout(), home, only, c)
			}
			res, err := collection.Update(home, only, install.PolicyFromEnv(), c)
			if err != nil {
				return err
			}
			if err := plugin.Rebuild(home, c); err != nil {
				return err
			}
			if err := c.Save(); err != nil {
				return err
			}
			for _, r := range res {
				fmt.Fprintf(cmd.OutOrStdout(), "shy collection update: %s → %s (%d item(s))\n", r.Name, shortCommit(r.Commit), len(r.Installed))
			}
			if len(res) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "shy collection update: nothing to update.")
			}
			return nil
		},
	}
	c.Flags().BoolVar(&apply, "apply", false, "apply the update (default is --dry-run)")
	return c
}

func previewUpdate(out io.Writer, home, only string, c *cache.Cache) error {
	if len(c.Collections) == 0 {
		fmt.Fprintln(out, "shy collection update: nothing subscribed.")
		return nil
	}
	fmt.Fprintln(out, "shy collection update (dry-run): re-run with --apply to commit changes.")
	for name, sub := range c.Collections {
		if only != "" && only != name {
			continue
		}
		fmt.Fprintf(out, "  %s — currently at %s; would re-clone from %s\n", name, shortCommit(sub.Commit), sub.Repo)
	}
	return nil
}

func newCollectionUnsubscribeCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "unsubscribe <name>",
		Aliases: []string{"unsub"},
		Short:   "Remove a subscribed collection and the items it brought in",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := paths.Home()
			if err != nil {
				return err
			}
			c, err := cache.Load(paths.CacheFile(home))
			if err != nil {
				return err
			}
			removed, err := collection.Unsubscribe(home, args[0], c)
			if err != nil {
				return err
			}
			if err := plugin.Rebuild(home, c); err != nil {
				return err
			}
			if err := c.Save(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "shy collection unsubscribe: %s (removed %d item(s))\n", args[0], removed)
			return nil
		},
	}
}
