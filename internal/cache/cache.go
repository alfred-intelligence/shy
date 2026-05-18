// SPDX-License-Identifier: MPL-2.0

// Package cache manages the private cache.json under SHY_HOME. The
// schema is intentionally not exposed to plugins; they must read via
// shy <cmd> --json instead.
package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"sync"
	"time"
)

// SchemaVersion bumps whenever the on-disk shape changes. Old caches
// with a different version are silently discarded and rebuilt.
const SchemaVersion = 1

// Cache is the in-memory representation of cache.json.
type Cache struct {
	Schema      int                     `json:"schema"`
	Installed   map[string]Installed    `json:"installed"`
	Collections map[string]Subscription `json:"collections"`
	Plugins     []PluginEntry           `json:"plugins,omitempty"`
	UpdateCheck *UpdateCheck            `json:"update_check,omitempty"`

	path string
	mu   sync.Mutex
}

// PluginEntry indexes one dispatchable plugin command for `shy <cmd>`.
type PluginEntry struct {
	Command     string `json:"command"`
	Namespace   string `json:"namespace"`
	Name        string `json:"name"`
	EntryScript string `json:"entry"`
	Description string `json:"description,omitempty"`
}

// Installed records one user-visible item — script, plugin, alias, or
// completion — and where it came from.
type Installed struct {
	Type      string `json:"type"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
	Source    string `json:"source,omitempty"`
	Ref       string `json:"ref,omitempty"`
	Version   string `json:"version,omitempty"`
	Owner     string `json:"owner,omitempty"`
}

// Key returns the canonical map key for an installed item. Aliases and
// completions are flat (no namespace); scripts and plugins are
// namespaced.
func (i Installed) Key() string {
	switch i.Type {
	case "alias", "completion":
		return i.Type + ":" + i.Name
	default:
		return i.Type + ":" + i.Namespace + "/" + i.Name
	}
}

// Subscription records a subscribed collection.
type Subscription struct {
	Name   string `json:"name"`
	Repo   string `json:"repo"`
	Ref    string `json:"ref,omitempty"`
	Commit string `json:"commit,omitempty"`
}

// UpdateCheck records the last upstream-version poll.
type UpdateCheck struct {
	LastChecked time.Time `json:"last_checked"`
	Latest      string    `json:"latest,omitempty"`
	Seen        bool      `json:"seen,omitempty"`
}

// Load reads cache.json from path. A missing or unreadable file
// produces a fresh empty cache; a stale-schema file is also discarded.
func Load(path string) (*Cache, error) {
	c := &Cache{
		Schema:      SchemaVersion,
		Installed:   map[string]Installed{},
		Collections: map[string]Subscription{},
		path:        path,
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cache: read %s: %w", path, err)
	}
	var on Cache
	if err := json.Unmarshal(data, &on); err != nil {
		return c, nil
	}
	if on.Schema != SchemaVersion {
		return c, nil
	}
	c.Installed = on.Installed
	if c.Installed == nil {
		c.Installed = map[string]Installed{}
	}
	c.Collections = on.Collections
	if c.Collections == nil {
		c.Collections = map[string]Subscription{}
	}
	c.UpdateCheck = on.UpdateCheck
	return c, nil
}

// Save writes the cache atomically (write-rename) under the same path.
func (c *Cache) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Schema = SchemaVersion
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("cache: marshal: %w", err)
	}
	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("cache: write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, c.path); err != nil {
		return fmt.Errorf("cache: rename %s: %w", c.path, err)
	}
	return nil
}

// Add records an installed item, overwriting any prior entry with the
// same key.
func (c *Cache) Add(i Installed) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Installed[i.Key()] = i
}

// Remove drops an installed item by key. Returns true if anything was
// removed.
func (c *Cache) Remove(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.Installed[key]; !ok {
		return false
	}
	delete(c.Installed, key)
	return true
}

// List returns all installed items sorted by key for stable output.
func (c *Cache) List() []Installed {
	c.mu.Lock()
	defer c.mu.Unlock()
	keys := make([]string, 0, len(c.Installed))
	for k := range c.Installed {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]Installed, 0, len(keys))
	for _, k := range keys {
		out = append(out, c.Installed[k])
	}
	return out
}

// SetCollection upserts a subscription record.
func (c *Cache) SetCollection(s Subscription) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Collections[s.Name] = s
}

// DropCollection removes a subscription. Returns true if anything was
// removed.
func (c *Cache) DropCollection(name string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.Collections[name]; !ok {
		return false
	}
	delete(c.Collections, name)
	return true
}

// SetPlugins replaces the plugin index in one shot. Called by the
// plugin package after Rebuild walks the filesystem.
func (c *Cache) SetPlugins(p []PluginEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Plugins = p
}
