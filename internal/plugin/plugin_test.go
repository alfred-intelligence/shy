// SPDX-License-Identifier: MPL-2.0
package plugin

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alfred-intelligence/shy/internal/cache"
	"github.com/alfred-intelligence/shy/internal/install"
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

// TestDiscoverMultiItemPlugin exercises Discover the way install.Bundle
// actually lays a multi-item collection on disk: one directory per item
// (installed/@ns/<item.Name>/), each carrying a copy of the *whole*
// collection manifest.toml (installScriptOrPlugin copies manifest.toml +
// README into every item's directory so `shy list`/`shy info` still see
// sibling items). Regression coverage for the multi-item-manifest
// dispatch bug: Discover used to replay every item in that manifest once
// per directory it found it in, instead of scoping to the directory's
// own item — see TestMultiItemDispatchNoCrossItemMisroute for the
// install.Bundle -> Discover -> Lookup pipeline version of this same bug.
func TestDiscoverMultiItemPlugin(t *testing.T) {
	home := t.TempDir()
	collection := `
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
`
	dirX := paths.PluginDir(home, "bob", "do-x")
	dirY := paths.PluginDir(home, "bob", "do-y")
	writeManifest(t, dirX, collection)
	writeManifest(t, dirY, collection)
	os.WriteFile(filepath.Join(dirX, paths.EntryPoint), []byte("#!/usr/bin/env bash\n"), 0o755)
	os.WriteFile(filepath.Join(dirY, paths.EntryPoint), []byte("#!/usr/bin/env bash\n"), 0o755)

	entries, err := Discover(home)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries=%d, want 2 (got: %+v)", len(entries), entries)
	}
	byCommand := map[string]Entry{}
	for _, e := range entries {
		byCommand[e.Command] = e
	}
	if e, ok := byCommand["do-x"]; !ok || e.EntryScript != filepath.Join(dirX, paths.EntryPoint) {
		t.Errorf("do-x entry wrong (must point at its own directory): %+v", e)
	}
	if e, ok := byCommand["do-y"]; !ok || e.EntryScript != filepath.Join(dirY, paths.EntryPoint) {
		t.Errorf("do-y entry wrong (must point at its own directory): %+v", e)
	}
}

// TestMultiItemDispatchNoCrossItemMisroute is the end-to-end regression
// test for the multi-item-manifest dispatch bug: install a real
// multi-item plugin collection through install.Bundle (the actual code
// path `shy install` uses), rebuild the dispatch cache the way `shy
// <command>` does, and verify each command executes ITS OWN script.
//
// Before the fix, Discover replayed every item in the shared
// manifest.toml once per item directory. Because EntryScript always
// resolves to paths.EntryPoint (this directory's own entry.sh — see
// TestDispatchResolvesEntryPoint), the replayed sibling entries pointed
// at the WRONG script: `shy do-y` would silently execute do-x's
// entry.sh whenever Lookup's first match for "do-y" came from do-x's
// directory (alphabetically first). That is a silent wrong-command
// execution, not a loud failure — the most dangerous shape of dispatch
// bug.
func TestMultiItemDispatchNoCrossItemMisroute(t *testing.T) {
	src := t.TempDir()
	home := t.TempDir()

	writeManifest(t, src, `
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
	os.WriteFile(filepath.Join(src, "do-x.sh"), []byte("#!/usr/bin/env bash\necho x\n"), 0o755)
	os.WriteFile(filepath.Join(src, "do-y.sh"), []byte("#!/usr/bin/env bash\necho y\n"), 0o755)

	c, _ := cache.Load(filepath.Join(home, "cache.json"))
	if _, err := install.Bundle(src, install.Options{Home: home, Source: "bob/tools"}, c); err != nil {
		t.Fatalf("bundle: %v", err)
	}
	if err := Rebuild(home, c); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	if len(c.Plugins) != 2 {
		t.Fatalf("cache plugins=%d, want 2 (got: %+v)", len(c.Plugins), c.Plugins)
	}

	dx, ok := Lookup(c, "do-x")
	if !ok {
		t.Fatal("lookup do-x: not found")
	}
	if out, err := exec.Command(dx.EntryScript).CombinedOutput(); err != nil || string(out) != "x\n" {
		t.Errorf("shy do-x: got output %q, err %v (entryScript=%s) — want \"x\\n\"", out, err, dx.EntryScript)
	}

	dy, ok := Lookup(c, "do-y")
	if !ok {
		t.Fatal("lookup do-y: not found")
	}
	if out, err := exec.Command(dy.EntryScript).CombinedOutput(); err != nil || string(out) != "y\n" {
		t.Errorf("shy do-y: got output %q, err %v (entryScript=%s) — want \"y\\n\" (a wrong output here means do-y is dispatching to do-x's script)", out, err, dy.EntryScript)
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
