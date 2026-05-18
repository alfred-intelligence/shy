// SPDX-License-Identifier: MPL-2.0

// Package plugin discovers installed plugins by walking
// $SHY_HOME/plugins/<namespace>/<name>/manifest.toml and indexing each
// item whose type is "plugin". The index lives in cache.json's Plugins
// map so `shy <command>` can resolve dispatchable plugin commands
// without re-walking the filesystem on every invocation.
package plugin

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/alfred-intelligence/shy/internal/cache"
	"github.com/alfred-intelligence/shy/internal/manifest"
	"github.com/alfred-intelligence/shy/internal/paths"
)

// Entry is one dispatchable plugin command.
type Entry struct {
	Command     string `json:"command"`
	Namespace   string `json:"namespace"`
	Name        string `json:"name"`
	EntryScript string `json:"entry"`
	Description string `json:"description,omitempty"`
}

// Discover walks the plugins/ tree, parses each manifest, and returns
// one entry per item declared with type="plugin". Plugins missing the
// `command` field are skipped (the manifest validator rejects them at
// install time, but discovery is forgiving).
func Discover(home string) ([]Entry, error) {
	root := filepath.Join(home, "plugins")
	info, err := os.Stat(root)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("plugin: stat %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, nil
	}

	out := []Entry{}
	nss, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("plugin: readdir %s: %w", root, err)
	}
	for _, ns := range nss {
		if !ns.IsDir() {
			continue
		}
		nsDir := filepath.Join(root, ns.Name())
		names, err := os.ReadDir(nsDir)
		if err != nil {
			continue
		}
		for _, n := range names {
			if !n.IsDir() {
				continue
			}
			dir := filepath.Join(nsDir, n.Name())
			entries, err := parseManifest(dir, ns.Name(), n.Name())
			if err != nil {
				continue
			}
			out = append(out, entries...)
		}
	}
	return out, nil
}

func parseManifest(dir, namespace, name string) ([]Entry, error) {
	data, err := os.ReadFile(filepath.Join(dir, "manifest.toml"))
	if err != nil {
		return nil, err
	}
	m, err := manifest.Parse(data)
	if err != nil {
		return nil, err
	}
	out := []Entry{}
	// Single-item form: top-level type=plugin + command + entry.
	if m.Type == "plugin" && m.Command != "" {
		entry := m.Entry
		if entry == "" {
			entry = "./" + name + ".sh"
		}
		out = append(out, Entry{
			Command:     m.Command,
			Namespace:   namespace,
			Name:        name,
			EntryScript: filepath.Join(dir, entry),
			Description: m.Description,
		})
	}
	// Multi-item form: items with type=plugin.
	for _, it := range m.Items {
		if it.Type != "plugin" || it.Command == "" {
			continue
		}
		entry := it.Path
		if entry == "" {
			entry = "./" + it.Name + ".sh"
		}
		out = append(out, Entry{
			Command:     it.Command,
			Namespace:   namespace,
			Name:        it.Name,
			EntryScript: filepath.Join(dir, entry),
			Description: it.Description,
		})
	}
	return out, nil
}

// Rebuild refreshes the plugin index on the given cache. Call after
// install/remove/update operations so `shy <cmd>` dispatch stays in
// sync with disk.
func Rebuild(home string, c *cache.Cache) error {
	entries, err := Discover(home)
	if err != nil {
		return err
	}
	c.SetPlugins(toCacheEntries(entries))
	return nil
}

func toCacheEntries(entries []Entry) []cache.PluginEntry {
	out := make([]cache.PluginEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, cache.PluginEntry{
			Command:     e.Command,
			Namespace:   e.Namespace,
			Name:        e.Name,
			EntryScript: e.EntryScript,
			Description: e.Description,
		})
	}
	return out
}

// Lookup finds a plugin entry by command name.
func Lookup(c *cache.Cache, command string) (cache.PluginEntry, bool) {
	for _, e := range c.Plugins {
		if e.Command == command {
			return e, true
		}
	}
	return cache.PluginEntry{}, false
}

// EnsureFresh runs Rebuild only if the cache has no plugin entries but
// the plugins directory has subdirectories. Used as a self-heal so a
// stale cache.json doesn't permanently hide installed plugins.
func EnsureFresh(home string, c *cache.Cache) error {
	if len(c.Plugins) > 0 {
		return nil
	}
	dir := filepath.Join(home, "plugins")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if e.IsDir() {
			return Rebuild(home, c)
		}
	}
	return nil
}

// _ keeps paths imported even when no exported function references it.
var _ = paths.Home
