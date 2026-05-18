// SPDX-License-Identifier: MPL-2.0

// Package paths centralises filesystem location and namespace rules so
// the runtime layer (init.bash) and the CLI agree on where things live.
package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SystemSeed is the system-wide read-only seed directory populated by
// the .deb/.rpm packages. shy init copies from here into the user's
// SHY_HOME; nothing at runtime is sourced directly from here.
const SystemSeed = "/usr/share/shy"

// BashrcSourceLine is the literal line shy init appends to ~/.bashrc.
// Detection of an existing line is substring-based, so the form is
// stable.
const BashrcSourceLine = `[ -f "$HOME/.shy/init.bash" ] && source "$HOME/.shy/init.bash"`

// BashrcMarker is the substring used to detect a previously-added
// integration line.
const BashrcMarker = `shy/init.bash`

// Home returns the user's SHY_HOME, defaulting to $HOME/.shy.
func Home() (string, error) {
	if v := os.Getenv("SHY_HOME"); v != "" {
		return v, nil
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("paths: resolve home: %w", err)
	}
	return filepath.Join(h, ".shy"), nil
}

// Subdirs enumerates the directories created by shy init.
func Subdirs(home string) []string {
	return []string{
		filepath.Join(home, "bin"),
		filepath.Join(home, "scripts"),
		filepath.Join(home, "plugins"),
		filepath.Join(home, "aliases"),
		filepath.Join(home, "completions"),
		filepath.Join(home, "collections"),
		filepath.Join(home, "overrides.d", "scripts"),
		filepath.Join(home, "overrides.d", "aliases"),
		filepath.Join(home, "overrides.d", "completions"),
	}
}

// InitBash is the path of the user-level init.bash.
func InitBash(home string) string {
	return filepath.Join(home, "init.bash")
}

// CacheFile is the path of the (private) cache.json.
func CacheFile(home string) string {
	return filepath.Join(home, "cache.json")
}

// safeName lowercases, strips a .local suffix, and replaces any
// character outside [a-z0-9-] with -.
var nonSafe = regexp.MustCompile(`[^a-z0-9-]+`)

// HostNamespace returns the safe-name of the current hostname for use
// as the namespace of locally-authored items.
func HostNamespace() (string, error) {
	h, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("paths: hostname: %w", err)
	}
	return SafeName(strings.TrimSuffix(h, ".local")), nil
}

// SafeName normalises a string into a filesystem-safe namespace.
func SafeName(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSuffix(s, ".local")
	s = nonSafe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "local"
	}
	return s
}

// NamespaceFromRepo returns the author/org portion of a repo slug like
// "alice/git-autofetch" → "alice".
func NamespaceFromRepo(repo string) string {
	if i := strings.Index(repo, "/"); i > 0 {
		return SafeName(repo[:i])
	}
	return SafeName(repo)
}

// ScriptDir is the absolute path of an installed script.
func ScriptDir(home, namespace, name string) string {
	return filepath.Join(home, "scripts", namespace, name)
}

// PluginDir is the absolute path of an installed plugin.
func PluginDir(home, namespace, name string) string {
	return filepath.Join(home, "plugins", namespace, name)
}

// AliasFile is the path of a single alias file (flat).
func AliasFile(home, name string) string {
	return filepath.Join(home, "aliases", name)
}

// CompletionFile is the path of a single completion file (flat).
func CompletionFile(home, tool string) string {
	return filepath.Join(home, "completions", tool)
}

// CollectionDir is the local clone path of a subscribed collection.
func CollectionDir(home, name string) string {
	return filepath.Join(home, "collections", name)
}
