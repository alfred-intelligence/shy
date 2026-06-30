// SPDX-License-Identifier: MPL-2.0
package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alfred-intelligence/shy/internal/cache"
	"github.com/alfred-intelligence/shy/internal/paths"
)

func writeManifest(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverSingleItemPlugin(t *testing.T) {
	home := t.TempDir()
	dir := paths.PluginDir(home, "alice", "gh-clone")
	writeManifest(t, dir, `
name = "gh-clone"
version = "0.1.0"
type = "plugin"
command = "gh-clone"
entry = "./gh-clone.sh"
description = "Clone a GitHub repo with default org"

[source]
repo = "alice/gh-clone"
`)
	os.WriteFile(filepath.Join(dir, "gh-clone.sh"), []byte("#!/usr/bin/env bash\n"), 0o755)

	entries, err := Discover(home)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries=%d, want 1", len(entries))
	}
	got := entries[0]
	if got.Command != "gh-clone" || got.Namespace != "alice" {
		t.Errorf("unexpected entry: %+v", got)
	}
	if got.Description == "" {
		t.Errorf("description should propagate from manifest")
	}
}

func TestDiscoverMultiItemPlugin(t *testing.T) {
	home := t.TempDir()
	dir := paths.PluginDir(home, "bob", "tools")
	writeManifest(t, dir, `
name = "tools"
version = "0.1.0"

[source]
repo = "bob/tools"

[[items]]
name = "do-x"
type = "plugin"
command = "do-x"
path = "./do-x.sh"

[[items]]
name = "do-y"
type = "plugin"
command = "do-y"
path = "./do-y.sh"
`)
	os.WriteFile(filepath.Join(dir, "do-x.sh"), []byte("#!/usr/bin/env bash\n"), 0o755)
	os.WriteFile(filepath.Join(dir, "do-y.sh"), []byte("#!/usr/bin/env bash\n"), 0o755)

	entries, err := Discover(home)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries=%d, want 2", len(entries))
	}
}

func TestRebuildAndLookup(t *testing.T) {
	home := t.TempDir()
	dir := paths.PluginDir(home, "alice", "gh-clone")
	writeManifest(t, dir, `
name = "gh-clone"
version = "0.1.0"
type = "plugin"
command = "gh-clone"
entry = "./gh-clone.sh"
`)
	c, _ := cache.Load(filepath.Join(home, "cache.json"))
	if err := Rebuild(home, c); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	if len(c.Plugins) != 1 {
		t.Fatalf("cache plugins=%d", len(c.Plugins))
	}
	if _, ok := Lookup(c, "gh-clone"); !ok {
		t.Error("Lookup did not find gh-clone")
	}
	if _, ok := Lookup(c, "nonexistent"); ok {
		t.Error("Lookup matched a nonexistent command")
	}
}

// TestDispatchResolvesEntryPoint verifies that Discover always builds
// EntryScript from paths.EntryPoint, never from the manifest's original
// entry/path filename. This is the regression test for the acceptance
// failure: "fork/exec .../hello-world.sh: no such file or directory".
// Install copies the source entry to entry.sh and the original name is
// absent on disk; dispatching via the manifest filename breaks every plugin.
func TestDispatchResolvesEntryPoint(t *testing.T) {
	home := t.TempDir()
	dir := paths.PluginDir(home, "alfred-intelligence", "hello-world")
	writeManifest(t, dir, `
name = "hello-world"
version = "0.1.0"
type = "plugin"
command = "hello-world"
entry = "./hello-world.sh"

[source]
repo = "alfred-intelligence/hello-world"
`)
	// Only entry.sh exists — that is what install writes.
	if err := os.WriteFile(filepath.Join(dir, paths.EntryPoint), []byte("#!/usr/bin/env bash\necho hello\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	entries, err := Discover(home)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries=%d, want 1", len(entries))
	}
	got := entries[0]
	if !strings.HasSuffix(got.EntryScript, "/"+paths.EntryPoint) {
		t.Errorf("EntryScript=%q must end with /%s", got.EntryScript, paths.EntryPoint)
	}
	if strings.Contains(got.EntryScript, "hello-world.sh") {
		t.Errorf("EntryScript=%q must not reference original filename hello-world.sh", got.EntryScript)
	}
	// Verify the resolved path is actually accessible on disk.
	if _, err := os.Stat(got.EntryScript); err != nil {
		t.Errorf("EntryScript %q not accessible: %v", got.EntryScript, err)
	}
}

func TestEnsureFreshSelfHeals(t *testing.T) {
	home := t.TempDir()
	dir := paths.PluginDir(home, "alice", "gh-clone")
	writeManifest(t, dir, `
name = "gh-clone"
version = "0.1.0"
type = "plugin"
command = "gh-clone"
entry = "./gh-clone.sh"
`)
	c, _ := cache.Load(filepath.Join(home, "cache.json"))
	// Cache starts empty even though plugins/ has content.
	if len(c.Plugins) != 0 {
		t.Fatalf("expected empty plugins, got %d", len(c.Plugins))
	}
	if err := EnsureFresh(home, c); err != nil {
		t.Fatalf("ensure fresh: %v", err)
	}
	if len(c.Plugins) != 1 {
		t.Errorf("plugins after ensure=%d", len(c.Plugins))
	}
}
