// SPDX-License-Identifier: MPL-2.0
package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alfred-intelligence/shy/internal/paths"
)

const resetConfirmWord = "RESET"

func newSystemResetCmd() *cobra.Command {
	var yes bool
	c := &cobra.Command{
		Use:   "system-reset",
		Short: "Destructive: wipe shy state across the system and every user (requires root)",
		Long: `Remove /usr/share/shy/, /etc/skel/.shy/, and every /home/*/.shy/ on the
machine. This restores shy to a fresh-install state for all users.

This is irreversible. The command requires both --yes-i-know and a
typed "RESET" confirmation. Without those it prints what it would
delete and exits without touching anything.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSystemReset(cmd.InOrStdin(), cmd.OutOrStdout(), yes)
		},
	}
	c.Flags().BoolVar(&yes, "yes-i-know", false, "acknowledge that this is destructive and irreversible")
	return c
}

func runSystemReset(in io.Reader, out io.Writer, yes bool) error {
	if err := requireRoot("system-reset"); err != nil {
		return err
	}

	targets := resetTargets()

	if !yes {
		if len(targets) > 0 {
			fmt.Fprintln(out, "shy system-reset: the following directories will be deleted irreversibly:")
			for _, t := range targets {
				fmt.Fprintf(out, "  %s\n", t)
			}
		}
		fmt.Fprintln(out, "\nshy system-reset: re-run with --yes-i-know to proceed.")
		return errors.New("system-reset: aborted (no --yes-i-know)")
	}

	if len(targets) == 0 {
		fmt.Fprintln(out, "shy system-reset: nothing to delete.")
		return nil
	}

	fmt.Fprintln(out, "shy system-reset: the following directories will be deleted irreversibly:")
	for _, t := range targets {
		fmt.Fprintf(out, "  %s\n", t)
	}

	fmt.Fprintf(out, "\nType '%s' to confirm: ", resetConfirmWord)
	r := bufio.NewReader(in)
	line, _ := r.ReadString('\n')
	if strings.TrimSpace(line) != resetConfirmWord {
		return errors.New("system-reset: confirmation mismatch — aborted, nothing deleted")
	}

	for _, t := range targets {
		if err := os.RemoveAll(t); err != nil {
			return fmt.Errorf("system-reset: remove %s: %w", t, err)
		}
		fmt.Fprintf(out, "  removed %s\n", t)
	}
	fmt.Fprintln(out, "shy system-reset: done.")
	return nil
}

// resetTargets enumerates existing directories that system-reset would
// delete. /home is scanned for any */.shy directory.
func resetTargets() []string {
	candidates := []string{
		paths.SystemSeed(),
		"/etc/skel/.shy",
	}
	if v := os.Getenv("SHY_TEST_HOME_ROOT"); v != "" {
		candidates = append(candidates, scanShyDirs(v)...)
	} else {
		candidates = append(candidates, scanShyDirs("/home")...)
	}

	out := candidates[:0]
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if _, err := os.Stat(c); err == nil {
			out = append(out, c)
		}
	}
	return out
}

func scanShyDirs(root string) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	out := []string{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		out = append(out, filepath.Join(root, e.Name(), ".shy"))
	}
	return out
}
