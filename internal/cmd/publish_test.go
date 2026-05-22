// SPDX-License-Identifier: MPL-2.0
package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alfred-intelligence/shy/internal/paths"
)

func TestCreateScaffolds(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SHY_HOME", home)
	t.Setenv("EDITOR", "") // ensure create doesn't try to launch one

	out := &bytes.Buffer{}
	if err := runCreate(out, "demo", true); err != nil {
		t.Fatalf("create: %v", err)
	}
	// Find the scaffolded entry.sh under any %-prefixed namespace.
	matches, _ := filepath.Glob(filepath.Join(home, "installed", paths.ScriptPrefix+"*", "demo", paths.EntryPoint))
	if len(matches) != 1 {
		t.Fatalf("expected 1 %s, got %v", paths.EntryPoint, matches)
	}
	body, _ := os.ReadFile(matches[0])
	if !strings.Contains(string(body), "#!/usr/bin/env bash") {
		t.Errorf("missing shebang in %s", matches[0])
	}
}

func TestPublishInsideParentRefuses(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SHY_HOME", home)

	// Configure a fake user.name so the precondition passes.
	withGitUserName(t, "tester")

	// Build a parent repo with a script subdirectory.
	parent := filepath.Join(home, "parent")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	runGitWithName(t, parent, "tester", "init", "-q", "-b", "main")

	scriptDir := filepath.Join(home, "installed", paths.ScriptPrefix+"host", "inside")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scriptDir, paths.EntryPoint), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Plant a .git directory above by symlinking parent's .git up. Easier:
	// move scriptDir into parent.
	if err := os.Rename(scriptDir, filepath.Join(parent, "inside")); err != nil {
		t.Fatal(err)
	}
	// Re-layout so findScriptDir sees it: place a sibling tree under installed/.
	if err := os.MkdirAll(filepath.Join(home, "installed", paths.ScriptPrefix+"host"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(parent, "inside"), filepath.Join(home, "installed", paths.ScriptPrefix+"host", "inside")); err != nil {
		t.Fatal(err)
	}

	err := runPublish(strings.NewReader(""), &bytes.Buffer{}, "inside", "", false)
	if err == nil {
		t.Fatal("expected error for publish inside parent repo")
	}
	if !strings.Contains(err.Error(), "inside another git repository") {
		t.Errorf("error doesn't mention parent repo: %v", err)
	}
}

func TestPublishHappyPathInitsGitAndMoves(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SHY_HOME", home)
	withGitUserName(t, "alice")

	// Use a hostname that won't equal "alice" so the rename triggers.
	t.Setenv("HOSTNAME", "test-host")

	// Scaffold a script under the host namespace by calling create.
	out := &bytes.Buffer{}
	if err := runCreate(out, "demo", true); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := runPublish(strings.NewReader(""), out, "demo", "0.3.1", false); err != nil {
		t.Fatalf("publish: %v", err)
	}
	manifestPath := filepath.Join(paths.ScriptDir(home, "alice", "demo"), "manifest.toml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("manifest at %s: %v", manifestPath, err)
	}
	if !strings.Contains(string(data), `version = "0.3.1"`) {
		t.Errorf("expected version 0.3.1 in manifest: %s", data)
	}
	if !strings.Contains(string(data), `repo = "alice/demo"`) {
		t.Errorf("expected source repo alice/demo: %s", data)
	}
}

func withGitUserName(t *testing.T, name string) {
	t.Helper()
	// Use environment overrides; do not touch the user's real git
	// config.
	t.Setenv("GIT_AUTHOR_NAME", name)
	t.Setenv("GIT_AUTHOR_EMAIL", name+"@example.com")
	t.Setenv("GIT_COMMITTER_NAME", name)
	t.Setenv("GIT_COMMITTER_EMAIL", name+"@example.com")

	// Override `git config --global user.name` lookups by setting a
	// per-test global config file.
	gitConfig := filepath.Join(t.TempDir(), ".gitconfig")
	if err := os.WriteFile(gitConfig, []byte("[user]\n  name = "+name+"\n  email = "+name+"@example.com\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", filepath.Dir(gitConfig))
	t.Setenv("GIT_CONFIG_GLOBAL", gitConfig)
}

func runGitWithName(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	c.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME="+name, "GIT_AUTHOR_EMAIL="+name+"@example.com",
		"GIT_COMMITTER_NAME="+name, "GIT_COMMITTER_EMAIL="+name+"@example.com",
	)
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
