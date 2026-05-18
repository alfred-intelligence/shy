// SPDX-License-Identifier: MPL-2.0
package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/alfred-intelligence/shy/internal/cache"
	"github.com/alfred-intelligence/shy/internal/paths"
)

// newCompletionCmd builds the `shy completion` parent. It hosts both
// `add <tool>` (capture another tool's bash completion) and the
// per-shell emitters (bash/zsh/fish/powershell) that print shy's own
// completion script.
func newCompletionCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "completion",
		Short: "Generate shy's shell completion or capture another tool's",
	}
	c.AddCommand(
		newCompletionAddCmd(),
		newShellCompletionCmd("bash", "Generate bash completion for shy", func(r *cobra.Command, w io.Writer) error { return r.GenBashCompletion(w) }),
		newShellCompletionCmd("zsh", "Generate zsh completion for shy", func(r *cobra.Command, w io.Writer) error { return r.GenZshCompletion(w) }),
		newShellCompletionCmd("fish", "Generate fish completion for shy", func(r *cobra.Command, w io.Writer) error { return r.GenFishCompletion(w, true) }),
		newShellCompletionCmd("powershell", "Generate PowerShell completion for shy", func(r *cobra.Command, w io.Writer) error { return r.GenPowerShellCompletion(w) }),
	)
	return c
}

func newShellCompletionCmd(name, short string, gen func(*cobra.Command, io.Writer) error) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return gen(cmd.Root(), cmd.OutOrStdout())
		},
	}
}

// newCompletionAddCmd implements `shy completion add <tool>`.
func newCompletionAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <tool>",
		Short: "Capture <tool>'s bash completion into $SHY_HOME/completions/<tool>",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCompletionAdd(cmd.OutOrStdout(), args[0])
		},
	}
}

func runCompletionAdd(out io.Writer, tool string) error {
	home, err := paths.Home()
	if err != nil {
		return err
	}
	if _, err := exec.LookPath(tool); err != nil {
		return fmt.Errorf("completion add: %s not on PATH", tool)
	}
	cch, err := cache.Load(paths.CacheFile(home))
	if err != nil {
		return err
	}
	var output []byte
	var lastErr error
	for _, args := range [][]string{
		{tool, "completion", "bash"},
		{tool, "completion", "--shell", "bash"},
		{tool, "bash-completion"},
	} {
		c := exec.Command(args[0], args[1:]...)
		got, err := c.Output()
		if err == nil && len(got) > 0 {
			output = got
			lastErr = nil
			break
		}
		lastErr = err
	}
	if output == nil {
		return fmt.Errorf("completion add: no completion command worked for %s (last error: %v)", tool, lastErr)
	}
	dst := paths.CompletionFile(home, tool)
	if err := writeFileForce(dst, output); err != nil {
		return err
	}
	cch.Add(cache.Installed{Type: "completion", Name: tool, Source: "local"})
	if err := cch.Save(); err != nil {
		return err
	}
	fmt.Fprintf(out, "shy completion add: wrote %s (%d bytes)\n", dst, len(output))
	return nil
}

func writeFileForce(path string, data []byte) error {
	if err := os.MkdirAll(parentDir(path), 0o755); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return os.WriteFile(path, data, 0o644)
}

func parentDir(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[:i]
		}
	}
	return "."
}
