// SPDX-License-Identifier: MPL-2.0

// Package manifest parses shy's TOML manifest schema. The manifest is
// metadata for sharing and distribution; the runtime layer (init.bash)
// never reads it. The CLI reads it for list/info/update/publish.
package manifest

import (
	"errors"
	"fmt"

	"github.com/pelletier/go-toml/v2"
)

// Manifest is the unified schema covering single-script repos, multi-item
// collections, and plugins. The CLI infers the form from what is present
// in the file and what is present on disk.
type Manifest struct {
	Name         string              `toml:"name"`
	Version      string              `toml:"version"`
	Description  string              `toml:"description,omitempty"`
	License      string              `toml:"license,omitempty"`
	Type         string              `toml:"type,omitempty"`
	Entry        string              `toml:"entry,omitempty"`
	Command      string              `toml:"command,omitempty"`
	Value        string              `toml:"value,omitempty"`
	Source       *Source             `toml:"source,omitempty"`
	Items        []Item              `toml:"items,omitempty"`
	Aliases      map[string]string   `toml:"aliases,omitempty"`
	Completions  []CompletionItem    `toml:"completions,omitempty"`
	Dependencies []Dependency        `toml:"dependencies,omitempty"`
	Requires     *Requires           `toml:"requires,omitempty"`
	Capabilities *Capabilities       `toml:"capabilities,omitempty"`
	Security     *Security           `toml:"security,omitempty"`
	Conformance  []map[string]string `toml:"conformance,omitempty"`
}

// Source identifies the upstream git repo this package came from.
// Missing source means the package is local to this host.
type Source struct {
	Repo string `toml:"repo"`
	Ref  string `toml:"ref,omitempty"`
}

// Item is one entry in a multi-item collection manifest.
type Item struct {
	Name        string `toml:"name"`
	Type        string `toml:"type"`
	Path        string `toml:"path,omitempty"`
	Command     string `toml:"command,omitempty"`
	Value       string `toml:"value,omitempty"`
	Tool        string `toml:"tool,omitempty"`
	Generate    string `toml:"generate,omitempty"`
	Description string `toml:"description,omitempty"`
	Source      string `toml:"source,omitempty"`
	Ref         string `toml:"ref,omitempty"`
}

// CompletionItem is an explicit completion entry.
type CompletionItem struct {
	Tool     string `toml:"tool"`
	Generate string `toml:"generate"`
}

// Dependency references an external package the install step must pull in.
type Dependency struct {
	Source     string `toml:"source"`
	Constraint string `toml:"constraint,omitempty"`
	Type       string `toml:"type"`
}

// Requires captures runtime preconditions checked at install time.
type Requires struct {
	Bash     string   `toml:"bash,omitempty"`
	Binaries []string `toml:"binaries,omitempty"`
}

// Capabilities is reserved for v2 sandboxing; parsed and ignored in v1.
// The shy audit plugin (v1.x) reads it for static-vs-declared analysis.
type Capabilities struct {
	Network    []string `toml:"network,omitempty"`
	Binaries   []string `toml:"binaries,omitempty"`
	Filesystem []string `toml:"filesystem,omitempty"`
}

// Security marks a release as a security fix; bypasses update-check
// throttle and snooze. Trust-based in v1; CVE-verified in v2.
type Security struct {
	Fixes       string `toml:"fixes,omitempty"`
	Severity    string `toml:"severity,omitempty"`
	Description string `toml:"description,omitempty"`
}

// Parse decodes a manifest document from TOML bytes.
func Parse(data []byte) (*Manifest, error) {
	var m Manifest
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}
	return &m, nil
}

// Validate checks required fields and item-type invariants. Returns the
// first failure encountered; callers wanting a list should accumulate
// upstream.
func (m *Manifest) Validate() error {
	if m.Name == "" {
		return errors.New("manifest: name is required")
	}
	if m.Version == "" {
		return errors.New("manifest: version is required")
	}
	for i, it := range m.Items {
		if err := validateItem(it); err != nil {
			return fmt.Errorf("manifest: item[%d] %q: %w", i, it.Name, err)
		}
	}
	return nil
}

func validateItem(it Item) error {
	if it.Name == "" {
		return errors.New("name is required")
	}
	switch it.Type {
	case "script", "":
		if it.Path == "" {
			return errors.New("script item requires path")
		}
	case "plugin":
		if it.Command == "" {
			return errors.New("plugin item requires command")
		}
		if it.Path == "" {
			return errors.New("plugin item requires path")
		}
	case "alias":
		if it.Value == "" {
			return errors.New("alias item requires value")
		}
	case "completion":
		if it.Tool == "" || it.Generate == "" {
			return errors.New("completion item requires tool and generate")
		}
	default:
		return fmt.Errorf("unknown item type %q", it.Type)
	}
	return nil
}
