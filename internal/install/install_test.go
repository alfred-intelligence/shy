// SPDX-License-Identifier: MPL-2.0
package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alfred-intelligence/shy/internal/cache"
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
	want := filepath.Join(home, "scripts", "alice", "git-autofetch", "git-autofetch.sh")
	if _, err := os.Stat(want); err != nil {
		t.Errorf("expected %s, got %v", want, err)
	}
	helperPath := filepath.Join(home, "scripts", "alice", "git-autofetch", "_helper.sh")
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
	mustWrite(t, filepath.Join(src, "my-script.sh"), "#!/usr/bin/env bash\n")

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
	if _, err := os.Stat(filepath.Join(home, "aliases", "ll")); err != nil {
		t.Errorf("alias ll: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, "aliases", "la")); err != nil {
		t.Errorf("alias la: %v", err)
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
	data, _ := os.ReadFile(filepath.Join(home, "aliases", "ll"))
	if string(data) != "alias ll='ls -la'\n" {
		t.Errorf("alias after prefer-new: %q", data)
	}
}

func TestRemoveItem(t *testing.T) {
	home := t.TempDir()
	c, _ := cache.Load(filepath.Join(home, "cache.json"))
	c.Add(cache.Installed{Type: "alias", Name: "ll"})
	mustWrite(t, filepath.Join(home, "aliases", "ll"), "alias ll='ls -alh'\n")
	removed, err := RemoveItem(home, "alias", "", "ll", c)
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if !removed {
		t.Error("expected removed=true")
	}
	if _, err := os.Stat(filepath.Join(home, "aliases", "ll")); err == nil {
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
