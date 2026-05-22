// SPDX-License-Identifier: MPL-2.0
package collection

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/alfred-intelligence/shy/internal/cache"
	"github.com/alfred-intelligence/shy/internal/install"
	"github.com/alfred-intelligence/shy/internal/paths"
)

// buildLocalRepo creates a bare git repo at <root>/<name>.git with an
// initial commit holding the given files. Returns the repo URL suitable
// for git clone.
func buildLocalRepo(t *testing.T, root, name string, files map[string]string) string {
	t.Helper()
	work := filepath.Join(root, name+"-work")
	bare := filepath.Join(root, name+".git")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	for path, content := range files {
		full := filepath.Join(work, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run := func(args ...string) {
		c := exec.Command("git", args...)
		c.Dir = work
		c.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
		)
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q", "-b", "main")
	run("add", ".")
	run("commit", "-q", "-m", "initial")
	c := exec.Command("git", "clone", "--bare", "-q", work, bare)
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git clone --bare: %v\n%s", err, out)
	}
	return bare
}

func TestSubscribeAndUnsubscribe(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "shy-home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}

	bare := buildLocalRepo(t, tmp, "alice-default", map[string]string{
		"manifest.toml": `
name = "alice-default"
version = "1.0.0"

[source]
repo = "alice/alice-default"

[[items]]
name = "git-autofetch"
type = "script"
path = "./git-autofetch.sh"

[[items]]
name = "ll"
type = "alias"
value = "ls -alh"
`,
		"git-autofetch.sh": "#!/usr/bin/env bash\necho ok\n",
	})

	c, _ := cache.Load(filepath.Join(home, "cache.json"))
	res, err := Subscribe(SubscribeOptions{
		Home:   home,
		Spec:   "file://" + bare,
		Policy: install.ConflictFail,
	}, c)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if len(res.Installed) < 2 {
		t.Errorf("installed=%d, want ≥2", len(res.Installed))
	}
	if _, err := os.Stat(filepath.Join(paths.ScriptDir(home, "alice", "git-autofetch"), "git-autofetch.sh")); err != nil {
		t.Errorf("script missing: %v", err)
	}
	if _, err := os.Stat(paths.AliasFile(home, "ll")); err != nil {
		t.Errorf("alias missing: %v", err)
	}
	if _, ok := c.Collections["alice-default"]; !ok {
		t.Error("subscription not recorded")
	}

	// Items should be tagged with the owning collection.
	owned := 0
	for _, it := range c.List() {
		if it.Owner == "alice-default" {
			owned++
		}
	}
	if owned < 2 {
		t.Errorf("owned items=%d, want ≥2", owned)
	}

	// Unsubscribe: collection clone and items go away.
	removed, err := Unsubscribe(home, "alice-default", c)
	if err != nil {
		t.Fatalf("unsubscribe: %v", err)
	}
	if removed < 2 {
		t.Errorf("removed=%d, want ≥2", removed)
	}
	if _, err := os.Stat(paths.CollectionDir(home, "alice-default")); err == nil {
		t.Error("collection clone still present after unsubscribe")
	}
	if _, err := os.Stat(paths.AliasFile(home, "ll")); err == nil {
		t.Error("alias still present after unsubscribe")
	}
}

func TestSubscribeBareManifestDiscoversSubItems(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "shy-home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}

	bare := buildLocalRepo(t, tmp, "bob-tools", map[string]string{
		"manifest.toml": `
name = "bob-tools"
version = "0.1.0"

[source]
repo = "bob/bob-tools"
`,
		"hello/manifest.toml": `
name = "hello"
version = "0.1.0"
type = "script"
entry = "./hello.sh"
`,
		"hello/hello.sh": "#!/usr/bin/env bash\n",
	})

	c, _ := cache.Load(filepath.Join(home, "cache.json"))
	res, err := Subscribe(SubscribeOptions{
		Home:   home,
		Spec:   "file://" + bare,
		Policy: install.ConflictFail,
	}, c)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if len(res.Installed) != 1 {
		t.Errorf("installed=%d, want 1", len(res.Installed))
	}
}

func TestParseSpec(t *testing.T) {
	cases := []struct {
		in       string
		wantRepo string
		wantRef  string
	}{
		{"github:alice/foo", "https://github.com/alice/foo.git", ""},
		{"github:alice/foo@v1.2.3", "https://github.com/alice/foo.git", "v1.2.3"},
		{"https://github.com/alice/foo.git", "https://github.com/alice/foo.git", ""},
		{"alice/foo", "https://github.com/alice/foo.git", ""},
		{"", "", ""},
	}
	for _, c := range cases {
		r, ref := parseSpec(c.in)
		if r != c.wantRepo || ref != c.wantRef {
			t.Errorf("parseSpec(%q) = (%q, %q), want (%q, %q)", c.in, r, ref, c.wantRepo, c.wantRef)
		}
	}
}
