// SPDX-License-Identifier: MPL-2.0
package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/alfred-intelligence/shy/internal/paths"
)

func newCreateCmd() *cobra.Command {
	var noEditor bool
	c := &cobra.Command{
		Use:   "create <name>",
		Short: "Scaffold a new script under the host namespace and open it in $EDITOR",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd.OutOrStdout(), args[0], noEditor)
		},
	}
	c.Flags().BoolVar(&noEditor, "no-editor", false, "skip opening $EDITOR (useful in scripts/CI)")
	return c
}

func runCreate(out interface{ Write([]byte) (int, error) }, name string, noEditor bool) error {
	home, err := paths.Home()
	if err != nil {
		return err
	}
	ns, err := paths.HostNamespace()
	if err != nil {
		return err
	}
	dir := paths.ScriptDir(home, ns, name)
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("create: %s already exists at %s", name, dir)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("create: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create: mkdir: %w", err)
	}
	scriptPath := filepath.Join(dir, name+".sh")
	readmePath := filepath.Join(dir, "README.md")
	scriptBody := fmt.Sprintf("#!/usr/bin/env bash\n# %s — sourced by shy at shell start.\n\n%s() {\n    echo \"OK from %s\"\n}\n", name, sanitiseFnName(name), name)
	readmeBody := fmt.Sprintf("# %s\n\nA shy snippet. Replace this paragraph with a one- or two-sentence\nintroduction.\n\n## Usage\n\nDescribe how %s is invoked.\n", name, sanitiseFnName(name))
	if err := os.WriteFile(scriptPath, []byte(scriptBody), 0o755); err != nil {
		return fmt.Errorf("create: write %s: %w", scriptPath, err)
	}
	if err := os.WriteFile(readmePath, []byte(readmeBody), 0o644); err != nil {
		return fmt.Errorf("create: write %s: %w", readmePath, err)
	}
	fmt.Fprintf(out, "shy create: scaffolded %s at %s\n", name, dir)
	fmt.Fprintf(out, "  script:  %s\n", scriptPath)
	fmt.Fprintf(out, "  readme:  %s\n", readmePath)
	if noEditor {
		return nil
	}
	return openEditor(scriptPath)
}

func openEditor(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		fmt.Fprintln(os.Stderr, "shy create: $EDITOR is unset; skipping editor open.")
		return nil
	}
	c := exec.Command(editor, path)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	return c.Run()
}

func sanitiseFnName(s string) string {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			out = append(out, byte(r))
		case r == '-' || r == '_':
			out = append(out, '_')
		}
	}
	if len(out) == 0 {
		return "run"
	}
	return string(out)
}
