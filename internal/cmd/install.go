// SPDX-License-Identifier: MPL-2.0
package cmd

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alfred-intelligence/shy/internal/cache"
	"github.com/alfred-intelligence/shy/internal/install"
	"github.com/alfred-intelligence/shy/internal/paths"
	"github.com/alfred-intelligence/shy/internal/plugin"
)

func newInstallCmd() *cobra.Command {
	var trackMain bool
	var silent bool

	c := &cobra.Command{
		Use:   "install <path|url|@user/repo>",
		Short: "Install a snippet, alias, completion, or plugin",
		Long: `Install from one of:

  - a local directory containing manifest.toml
  - a tar.gz or http(s) URL containing the same
  - a GitHub reference @user/repo (pinned to current HEAD by default)
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(args[0], trackMain, silent, cmd.OutOrStdout())
		},
	}
	c.Flags().BoolVar(&trackMain, "track-main", false, "follow the default branch instead of pinning to HEAD")
	c.Flags().BoolVar(&silent, "silent", false, "suppress output (plugin API)")
	return c
}

func runInstall(spec string, trackMain, silent bool, out interface{ Write([]byte) (int, error) }) error {
	home, err := paths.Home()
	if err != nil {
		return err
	}
	c, err := cache.Load(paths.CacheFile(home))
	if err != nil {
		return err
	}

	dir, source, ref, cleanup, err := materialise(spec, trackMain)
	if err != nil {
		return err
	}
	defer cleanup()

	opts := install.Options{
		Home:   home,
		Source: source,
		Ref:    ref,
		Policy: install.PolicyFromEnv(),
		Silent: silent,
	}
	res, err := install.Bundle(dir, opts, c)
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
		fmt.Fprintf(out, "shy install: %d item(s) installed from %s\n", len(res.Installed), source)
		for _, it := range res.Installed {
			ns := it.Namespace
			if ns != "" {
				fmt.Fprintf(out, "  %s %s/%s @ %s\n", it.Type, ns, it.Name, it.Version)
			} else {
				fmt.Fprintf(out, "  %s %s\n", it.Type, it.Name)
			}
		}
	}
	return nil
}

// materialise resolves the install spec into a local directory holding
// manifest.toml + payload. Returns (dir, source-descriptor, ref,
// cleanup-fn).
func materialise(spec string, trackMain bool) (string, string, string, func(), error) {
	cleanup := func() {}
	switch {
	case strings.HasPrefix(spec, "@"):
		repo := strings.TrimPrefix(spec, "@")
		dir, ref, err := gitClone("https://github.com/"+repo+".git", "", !trackMain)
		if err != nil {
			return "", "", "", cleanup, err
		}
		return dir, repo, ref, func() { os.RemoveAll(dir) }, nil

	case strings.HasPrefix(spec, "github:"):
		repo := strings.TrimPrefix(spec, "github:")
		var atRef string
		if i := strings.Index(repo, "@"); i > 0 {
			atRef = repo[i+1:]
			repo = repo[:i]
		}
		dir, ref, err := gitClone("https://github.com/"+repo+".git", atRef, !trackMain && atRef == "")
		if err != nil {
			return "", "", "", cleanup, err
		}
		return dir, repo, ref, func() { os.RemoveAll(dir) }, nil

	case strings.HasPrefix(spec, "http://"), strings.HasPrefix(spec, "https://"):
		u, err := url.Parse(spec)
		if err != nil {
			return "", "", "", cleanup, fmt.Errorf("install: parse url: %w", err)
		}
		if strings.HasSuffix(u.Path, ".git") || strings.Contains(u.Host, "github.com") {
			dir, ref, err := gitClone(spec, "", !trackMain)
			if err != nil {
				return "", "", "", cleanup, err
			}
			return dir, spec, ref, func() { os.RemoveAll(dir) }, nil
		}
		return "", "", "", cleanup, fmt.Errorf("install: non-git URLs not yet supported")

	case strings.HasPrefix(spec, "file://"):
		path := strings.TrimPrefix(spec, "file://")
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", "", "", cleanup, err
		}
		return abs, "file://" + abs, "", cleanup, nil

	default:
		abs, err := filepath.Abs(spec)
		if err != nil {
			return "", "", "", cleanup, err
		}
		info, err := os.Stat(abs)
		if err != nil {
			return "", "", "", cleanup, fmt.Errorf("install: %w", err)
		}
		if !info.IsDir() {
			return "", "", "", cleanup, fmt.Errorf("install: %s is not a directory", abs)
		}
		return abs, "local:" + abs, "", cleanup, nil
	}
}

// gitClone clones repo into a fresh tempdir. If ref is non-empty it
// checks out that ref. If pin is true the resolved commit SHA is
// returned in ref; otherwise ref reflects the input.
func gitClone(repo, ref string, pin bool) (string, string, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return "", "", fmt.Errorf("install: git not found on PATH")
	}
	tmp, err := os.MkdirTemp("", "shy-clone-")
	if err != nil {
		return "", "", fmt.Errorf("install: tmpdir: %w", err)
	}
	args := []string{"clone", "--depth", "1"}
	if ref != "" {
		args = append(args, "--branch", ref)
	}
	args = append(args, repo, tmp)
	cmd := exec.Command("git", args...)
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmp)
		return "", "", fmt.Errorf("install: git clone %s: %w", repo, err)
	}
	if pin {
		c := exec.Command("git", "-C", tmp, "rev-parse", "HEAD")
		out, err := c.Output()
		if err == nil {
			ref = strings.TrimSpace(string(out))
		}
	}
	return tmp, ref, nil
}
