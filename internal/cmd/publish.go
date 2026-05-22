// SPDX-License-Identifier: MPL-2.0
package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alfred-intelligence/shy/internal/authoring"
	"github.com/alfred-intelligence/shy/internal/paths"
)

func newPublishCmd() *cobra.Command {
	var toGitHub bool
	var versionOverride string
	c := &cobra.Command{
		Use:   "publish <name>",
		Short: "Initialise the script as a publishable git repo and write its manifest",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPublish(cmd.InOrStdin(), cmd.OutOrStdout(), args[0], versionOverride, toGitHub)
		},
	}
	c.Flags().StringVar(&versionOverride, "version", "", "set the published version explicitly (overrides inference and prompt)")
	c.Flags().BoolVar(&toGitHub, "to-github", false, "create the GitHub repo via `gh` and push (requires gh CLI)")
	return c
}

func runPublish(in io.Reader, out io.Writer, name, versionOverride string, toGitHub bool) error {
	user, err := authoring.GlobalGitUserName()
	if err != nil {
		return err
	}
	if user == "" {
		return errors.New("publish: git user.name is not set.\npublish: run: git config --global user.name \"<your-github-handle>\"")
	}

	home, err := paths.Home()
	if err != nil {
		return err
	}
	dir, found, err := findScriptDir(home, name)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("publish: no script named %q under $SHY_HOME/installed/", name)
	}

	state, gitDir, err := authoring.DetectGit(dir)
	if err != nil {
		return err
	}
	switch state {
	case authoring.InsideParent:
		return inParentRepoError(dir, gitDir)
	case authoring.NoGit:
		fmt.Fprintf(out, "shy publish: initialising git in %s\n", dir)
		if err := authoring.GitInit(dir, "chore: initial publish"); err != nil {
			return err
		}
	case authoring.SelfRoot:
		clean, err := authoring.WorkingTreeClean(dir)
		if err != nil {
			return err
		}
		if clean {
			fmt.Fprintf(out, "shy publish: %s is a git repo (working tree clean)\n", name)
		} else {
			fmt.Fprintf(out, "shy publish: %s is a git repo (WARNING: working tree not clean; uncommitted changes will not be included in version inference)\n", name)
		}
	}

	// Decide the version.
	version, err := decideVersion(dir, versionOverride, in, out)
	if err != nil {
		return err
	}

	// Author the manifest.
	manifestPath := filepath.Join(dir, "manifest.toml")
	descPrompt := promptDefault(in, out, "Description", "")
	license := promptDefault(in, out, "License", "MIT")
	manifestBody := renderManifest(name, version.String(), descPrompt, license, user)
	if err := os.WriteFile(manifestPath, []byte(manifestBody), 0o644); err != nil {
		return fmt.Errorf("publish: write manifest: %w", err)
	}
	fmt.Fprintf(out, "shy publish: wrote manifest %s\n", manifestPath)

	// Stage and commit the manifest if SelfRoot/NoGit-just-init.
	stageAndCommit(dir, fmt.Sprintf("chore: publish %s %s", name, version.String()))

	// Move to user/namespace if currently under host namespace.
	if newDir, moved, err := moveToUserNamespace(home, dir, user, name); err != nil {
		return err
	} else if moved {
		fmt.Fprintf(out, "shy publish: moved to %s\n", newDir)
	}

	if toGitHub {
		if _, err := exec.LookPath("gh"); err != nil {
			return errors.New("publish --to-github: gh CLI not found on PATH")
		}
		c := exec.Command("gh", "repo", "create", user+"/"+name, "--public", "--source", dir, "--push")
		c.Stdout, c.Stderr = out, os.Stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("publish --to-github: %w", err)
		}
	}

	return nil
}

func decideVersion(dir, override string, in io.Reader, out io.Writer) (authoring.SemVer, error) {
	if override != "" {
		return authoring.ParseSemVer(override)
	}
	// Previous manifest version → default current; bump from commits.
	cur := authoring.SemVer{Major: 0, Minor: 1, Patch: 0}
	if data, err := os.ReadFile(filepath.Join(dir, "manifest.toml")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "version") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					v := strings.Trim(strings.TrimSpace(parts[1]), `"`)
					if sv, err := authoring.ParseSemVer(v); err == nil {
						cur = sv
					}
				}
			}
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return authoring.SemVer{}, err
	}

	msgs, err := authoring.CommitsSinceFile(dir, "manifest.toml")
	if err != nil {
		return authoring.SemVer{}, err
	}
	c := authoring.ParseConventional(msgs)
	bump := authoring.InferBump(c)
	suggested := bump.Apply(cur)
	if bump == authoring.BumpNone {
		// No Conventional Commits to drive inference; prompt.
		def := suggested.String()
		got := promptDefault(in, out, fmt.Sprintf("Version (current %s, no conventional commits found)", cur), def)
		return authoring.ParseSemVer(got)
	}
	fmt.Fprintf(out, "shy publish: commits since last publish: %d feat / %d fix / %d breaking → %s bump\n", c.Feat, c.Fix, c.Breaking, bump)
	got := promptDefault(in, out, fmt.Sprintf("Version (suggested %s)", suggested), suggested.String())
	return authoring.ParseSemVer(got)
}

func promptDefault(in io.Reader, out io.Writer, label, def string) string {
	if !isTerminal(in) {
		return def
	}
	if def != "" {
		fmt.Fprintf(out, "%s [%s]: ", label, def)
	} else {
		fmt.Fprintf(out, "%s: ", label)
	}
	r := bufio.NewReader(in)
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

// isTerminal is a cheap heuristic: if stdin is not a regular file we
// assume it's interactive. Wrapped in a function so tests can override
// by piping a string.
func isTerminal(in io.Reader) bool {
	f, ok := in.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func renderManifest(name, version, description, license, author string) string {
	if description == "" {
		description = "A shy snippet."
	}
	return fmt.Sprintf(`name = "%s"
version = "%s"
description = "%s"
license = "%s"
type = "script"
entry = "./%s"

[source]
repo = "%s/%s"

[requires]
bash = ">=4"
`, name, version, description, license, paths.EntryPoint, author, name)
}

func stageAndCommit(dir, msg string) {
	for _, args := range [][]string{
		{"add", "."},
		{"commit", "-q", "-m", msg, "--allow-empty"},
	} {
		exec.Command("git", append([]string{"-C", dir}, args...)...).Run()
	}
}

func moveToUserNamespace(home, dir, author, name string) (string, bool, error) {
	parent := filepath.Dir(dir)
	currentNS := filepath.Base(parent) // e.g. "%hostname"
	authorNS := paths.ScriptPrefix + paths.SafeName(author)
	if currentNS == authorNS {
		return dir, false, nil
	}
	// Only move if currently under installed/%<host>/<name>
	hostNS, _ := paths.HostNamespace()
	if currentNS != paths.ScriptPrefix+hostNS {
		return dir, false, nil
	}
	newParent := filepath.Join(home, "installed", authorNS)
	if err := os.MkdirAll(newParent, 0o755); err != nil {
		return "", false, fmt.Errorf("publish: mkdir: %w", err)
	}
	newDir := filepath.Join(newParent, name)
	if _, err := os.Stat(newDir); err == nil {
		return "", false, fmt.Errorf("publish: destination already exists at %s", newDir)
	}
	if err := os.Rename(dir, newDir); err != nil {
		return "", false, fmt.Errorf("publish: rename: %w", err)
	}
	// Clean up empty old parent.
	_ = os.Remove(parent)
	return newDir, true, nil
}

func inParentRepoError(scriptDir, gitDir string) error {
	return fmt.Errorf(`publish: cannot publish %s: directory is inside another git repository.

  Script directory:  %s
  Parent .git/ at:   %s

  Published scripts must live in their own git repository at the
  script's root. Publishing a subdirectory of another repository
  creates ambiguous source-tracking and version inference.

  To publish this script as its own repository:
    1. Move the script directory outside of any existing repo
    2. Then run: shy publish %s

  To keep it local (no publish required):
    Local scripts work without a manifest. No action needed.`,
		filepath.Base(scriptDir), scriptDir, gitDir, filepath.Base(scriptDir))
}

func findScriptDir(home, name string) (string, bool, error) {
	installed := filepath.Join(home, "installed")
	entries, err := os.ReadDir(installed)
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	for _, ns := range entries {
		if !ns.IsDir() {
			continue
		}
		if !strings.HasPrefix(ns.Name(), paths.ScriptPrefix) {
			continue
		}
		candidate := filepath.Join(installed, ns.Name(), name)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, true, nil
		}
	}
	return "", false, nil
}
