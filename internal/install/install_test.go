// SPDX-License-Identifier: MPL-2.0
package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alfred-intelligence/shy/internal/cache"
	"github.com/alfred-intelligence/shy/internal/paths"
)

func TestBundleScriptWithSource(t *testing.T) {
	src := t.TempDir()
	home := t.TempDir()

	mustWrite(t, filepath.Join(src, "manifest.toml"), `
name = "git-autofetch"
version = "1.0.0"
type = "script"
entry = "./git-autofetch.sh"

[source]
repo = "alice/git-autofetch"
`)
	mustWrite(t, filepath.Join(src, "git-autofetch.sh"), "#!/usr/bin/env bash\necho 'ok'\n")
	mustWrite(t, filepath.Join(src, "_helper.sh"), "#!/usr/bin/env bash\n")

	c, _ := cache.Load(filepath.Join(home, "cache.json"))
	res, err := Bundle(src, Options{Home: home, Source: "alice/git-autofetch"}, c)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if len(res.Installed) != 1 {
		t.Fatalf("installed=%d", len(res.Installed))
	}
	// The entry is written under the fixed entry.sh name the runtime sources,
	// NOT its original filename — that rename is the load-bearing contract.
	want := filepath.Join(paths.ScriptDir(home, "alice", "git-autofetch"), paths.EntryPoint)
	if _, err := os.Stat(want); err != nil {
		t.Errorf("expected entry at %s, got %v", want, err)
	}
	// The original-named file must NOT exist alongside it.
	if _, err := os.Stat(filepath.Join(paths.ScriptDir(home, "alice", "git-autofetch"), "git-autofetch.sh")); err == nil {
		t.Error("original-named entry should be renamed to entry.sh, not copied verbatim")
	}
	// _-prefixed helpers belong to this item and are preserved.
	helperPath := filepath.Join(paths.ScriptDir(home, "alice", "git-autofetch"), "_helper.sh")
	if _, err := os.Stat(helperPath); err != nil {
		t.Errorf("expected helper at %s, got %v", helperPath, err)
	}
}

func TestBundleLocalNamespace(t *testing.T) {
	src := t.TempDir()
	home := t.TempDir()

	mustWrite(t, filepath.Join(src, "manifest.toml"), `
name = "my-script"
version = "0.1.0"
type = "script"
`)
	mustWrite(t, filepath.Join(src, paths.EntryPoint), "#!/usr/bin/env bash\n")

	c, _ := cache.Load(filepath.Join(home, "cache.json"))
	if _, err := Bundle(src, Options{Home: home, Source: "local"}, c); err != nil {
		t.Fatalf("install: %v", err)
	}
	items := c.List()
	if len(items) != 1 {
		t.Fatalf("cache items=%d", len(items))
	}
	if items[0].Namespace == "" {
		t.Error("expected hostname namespace, got empty")
	}
}

func TestBundleCollectionMultiItem(t *testing.T) {
	src := t.TempDir()
	home := t.TempDir()

	mustWrite(t, filepath.Join(src, "manifest.toml"), `
name = "alice-default"
version = "2.0.0"

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

[aliases]
la = "ls -A"
`)
	mustWrite(t, filepath.Join(src, "git-autofetch.sh"), "#!/usr/bin/env bash\n")

	c, _ := cache.Load(filepath.Join(home, "cache.json"))
	res, err := Bundle(src, Options{Home: home, Source: "alice/alice-default"}, c)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if len(res.Installed) != 3 {
		t.Fatalf("installed=%d items, want 3 (script, alias ll, alias la)", len(res.Installed))
	}
	if _, err := os.Stat(paths.AliasFile(home, "ll")); err != nil {
		t.Errorf("alias ll: %v", err)
	}
	if _, err := os.Stat(paths.AliasFile(home, "la")); err != nil {
		t.Errorf("alias la: %v", err)
	}
}

// TestBundleSharedDirNoFanout guards the bug where multiple script items
// share one collection dir (e.g. stdlib's scripts/): each item must get
// ONLY its own entry as entry.sh — siblings must NOT fan out into it.
func TestBundleSharedDirNoFanout(t *testing.T) {
	src := t.TempDir()
	home := t.TempDir()

	mustWrite(t, filepath.Join(src, "manifest.toml"), `
name = "tools"
version = "1.0.0"

[source]
repo = "alice/tools"

[[items]]
name = "aa"
type = "script"
path = "./scripts/aa.sh"

[[items]]
name = "bb"
type = "script"
path = "./scripts/bb.sh"
`)
	mustWrite(t, filepath.Join(src, "scripts", "aa.sh"), "aa() { echo aa; }\n")
	mustWrite(t, filepath.Join(src, "scripts", "bb.sh"), "bb() { echo bb; }\n")

	c, _ := cache.Load(filepath.Join(home, "cache.json"))
	if _, err := Bundle(src, Options{Home: home, Source: "alice/tools"}, c); err != nil {
		t.Fatalf("install: %v", err)
	}

	for _, item := range []struct{ name, body string }{{"aa", "aa()"}, {"bb", "bb()"}} {
		dir := paths.ScriptDir(home, "alice", item.name)
		// Its own entry.sh exists with its own content.
		got, err := os.ReadFile(filepath.Join(dir, paths.EntryPoint))
		if err != nil {
			t.Fatalf("%s entry.sh: %v", item.name, err)
		}
		if !strings.Contains(string(got), item.body) {
			t.Errorf("%s entry.sh has wrong content: %q", item.name, got)
		}
		// The OTHER item's script must not have fanned in.
		other := "bb.sh"
		if item.name == "bb" {
			other = "aa.sh"
		}
		if _, err := os.Stat(filepath.Join(dir, other)); err == nil {
			t.Errorf("%s dir leaked sibling %s (fan-out bug)", item.name, other)
		}
	}
}

func TestConflictPolicy(t *testing.T) {
	src := t.TempDir()
	home := t.TempDir()
	mustWrite(t, filepath.Join(src, "manifest.toml"), `
name = "bundle"
version = "0.1.0"

[aliases]
ll = "ls -alh"
`)
	c, _ := cache.Load(filepath.Join(home, "cache.json"))
	if _, err := Bundle(src, Options{Home: home, Source: "local"}, c); err != nil {
		t.Fatalf("first install: %v", err)
	}
	// Second install with conflicting value under default (fail) policy.
	mustWrite(t, filepath.Join(src, "manifest.toml"), `
name = "bundle"
version = "0.2.0"

[aliases]
ll = "ls -la"
`)
	if _, err := Bundle(src, Options{Home: home, Source: "local", Policy: ConflictFail}, c); err == nil {
		t.Error("expected conflict error under fail policy")
	}
	// Same path under prefer-new should overwrite without error.
	if _, err := Bundle(src, Options{Home: home, Source: "local", Policy: ConflictPreferNew}, c); err != nil {
		t.Errorf("prefer-new: %v", err)
	}
	data, _ := os.ReadFile(paths.AliasFile(home, "ll"))
	if string(data) != "alias ll='ls -la'\n" {
		t.Errorf("alias after prefer-new: %q", data)
	}
}

func TestRemoveItem(t *testing.T) {
	home := t.TempDir()
	c, _ := cache.Load(filepath.Join(home, "cache.json"))
	c.Add(cache.Installed{Type: "alias", Name: "ll"})
	mustWrite(t, paths.AliasFile(home, "ll"), "alias ll='ls -alh'\n")
	removed, err := RemoveItem(home, "alias", "", "ll", c)
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if !removed {
		t.Error("expected removed=true")
	}
	if _, err := os.Stat(paths.AliasFile(home, "ll")); err == nil {
		t.Error("alias file still present after remove")
	}
}

func TestValidateFlatName(t *testing.T) {
	bad := []string{"", ".", "..", "../escape", "foo/bar", "-l"}
	for _, n := range bad {
		if err := validateFlatName("alias", n); err == nil {
			t.Errorf("expected error for %q", n)
		}
	}
	ok := []string{"ll", "gst", "kubectl", "my_thing"}
	for _, n := range ok {
		if err := validateFlatName("alias", n); err != nil {
			t.Errorf("unexpected error for %q: %v", n, err)
		}
	}
}

func TestPolicyFromEnv(t *testing.T) {
	t.Setenv("SHY_ON_CONFLICT", "prefer-new")
	if PolicyFromEnv() != ConflictPreferNew {
		t.Error("prefer-new not parsed")
	}
	t.Setenv("SHY_ON_CONFLICT", "")
	if PolicyFromEnv() != ConflictFail {
		t.Error("default not fail")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
