// SPDX-License-Identifier: MPL-2.0

// Package install lays installed items onto disk under SHY_HOME and
// keeps cache.json in sync. The runtime layer reads only the
// filesystem; manifests live with their items for sharing purposes.
package install

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/alfred-intelligence/shy/internal/cache"
	"github.com/alfred-intelligence/shy/internal/manifest"
	"github.com/alfred-intelligence/shy/internal/paths"
)

// ConflictPolicy decides what to do when an alias or completion file
// already exists with different content. Set via SHY_ON_CONFLICT.
type ConflictPolicy string

const (
	ConflictPreferExisting ConflictPolicy = "prefer-existing"
	ConflictPreferNew      ConflictPolicy = "prefer-new"
	ConflictSkip           ConflictPolicy = "skip"
	ConflictFail           ConflictPolicy = "fail"
)

// PolicyFromEnv reads SHY_ON_CONFLICT, defaulting to ConflictFail.
func PolicyFromEnv() ConflictPolicy {
	switch ConflictPolicy(strings.ToLower(os.Getenv("SHY_ON_CONFLICT"))) {
	case ConflictPreferExisting:
		return ConflictPreferExisting
	case ConflictPreferNew:
		return ConflictPreferNew
	case ConflictSkip:
		return ConflictSkip
	default:
		return ConflictFail
	}
}

// ConflictError is returned when an existing file would be overwritten
// and the policy is ConflictFail.
type ConflictError struct {
	Path string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("conflict at %s: existing file would be overwritten (set SHY_ON_CONFLICT=prefer-new to overwrite, =prefer-existing to keep)", e.Path)
}

// Options drives a single install run.
type Options struct {
	Home     string
	Source   string         // origin descriptor: path, url, or @user/repo
	Ref      string         // pinned commit or branch
	Policy   ConflictPolicy
	Silent   bool           // suppress stdout progress
}

// Result reports what one Install call did.
type Result struct {
	Installed []cache.Installed
}

// Bundle installs a directory containing manifest.toml plus payload
// files. The directory layout is the source of truth for what lives
// where; manifest items only refine type-specific metadata.
func Bundle(dir string, opts Options, c *cache.Cache) (*Result, error) {
	mPath := filepath.Join(dir, "manifest.toml")
	data, err := os.ReadFile(mPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("install: no manifest.toml in %s", dir)
	}
	if err != nil {
		return nil, fmt.Errorf("install: read manifest: %w", err)
	}
	m, err := manifest.Parse(data)
	if err != nil {
		return nil, err
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}

	ns, err := chooseNamespace(m)
	if err != nil {
		return nil, err
	}
	res := &Result{}

	// Single-item form: top-level type=script|plugin with entry. Skip
	// when the manifest is a pure collection (only [aliases] /
	// [[completions]] / [[items]]).
	if len(m.Items) == 0 && (m.Type == "script" || m.Type == "plugin" || m.Entry != "") {
		entry := m.Entry
		if entry == "" {
			entry = "./" + paths.EntryPoint
		}
		typ := m.Type
		if typ == "" {
			typ = "script"
		}
		installed, err := installScriptOrPlugin(dir, opts.Home, ns, m.Name, typ, entry, opts.Policy)
		if err != nil {
			return nil, err
		}
		installed.Source = opts.Source
		installed.Ref = opts.Ref
		installed.Version = m.Version
		c.Add(*installed)
		res.Installed = append(res.Installed, *installed)
	}

	// Multi-item items.
	for _, it := range m.Items {
		switch it.Type {
		case "", "script":
			installed, err := installScriptOrPlugin(dir, opts.Home, ns, it.Name, "script", it.Path, opts.Policy)
			if err != nil {
				return nil, err
			}
			installed.Source = opts.Source
			installed.Ref = opts.Ref
			installed.Version = m.Version
			c.Add(*installed)
			res.Installed = append(res.Installed, *installed)
		case "plugin":
			installed, err := installScriptOrPlugin(dir, opts.Home, ns, it.Name, "plugin", it.Path, opts.Policy)
			if err != nil {
				return nil, err
			}
			installed.Source = opts.Source
			installed.Ref = opts.Ref
			installed.Version = m.Version
			c.Add(*installed)
			res.Installed = append(res.Installed, *installed)
		case "alias":
			installed, err := installAlias(opts.Home, it.Name, it.Value, opts.Policy, opts.Source, m.Version)
			if err != nil {
				return nil, err
			}
			c.Add(*installed)
			res.Installed = append(res.Installed, *installed)
		case "completion":
			installed, err := installCompletion(opts.Home, it.Tool, it.Generate, opts.Policy, opts.Source, m.Version)
			if err != nil {
				return nil, err
			}
			c.Add(*installed)
			res.Installed = append(res.Installed, *installed)
		default:
			return nil, fmt.Errorf("install: unknown item type %q", it.Type)
		}
	}

	// Top-level inline aliases and completions.
	for name, value := range m.Aliases {
		installed, err := installAlias(opts.Home, name, value, opts.Policy, opts.Source, m.Version)
		if err != nil {
			return nil, err
		}
		c.Add(*installed)
		res.Installed = append(res.Installed, *installed)
	}
	for _, ci := range m.Completions {
		installed, err := installCompletion(opts.Home, ci.Tool, ci.Generate, opts.Policy, opts.Source, m.Version)
		if err != nil {
			return nil, err
		}
		c.Add(*installed)
		res.Installed = append(res.Installed, *installed)
	}

	return res, nil
}

func chooseNamespace(m *manifest.Manifest) (string, error) {
	if m.Source != nil && m.Source.Repo != "" {
		return paths.NamespaceFromRepo(m.Source.Repo), nil
	}
	return paths.HostNamespace()
}

func installScriptOrPlugin(srcDir, home, namespace, name, kind, entry string, policy ConflictPolicy) (*cache.Installed, error) {
	var destDir string
	if kind == "plugin" {
		destDir = paths.PluginDir(home, namespace, name)
	} else {
		destDir = paths.ScriptDir(home, namespace, name)
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, fmt.Errorf("install: mkdir %s: %w", destDir, err)
	}
	if entry == "" {
		entry = "./" + paths.EntryPoint
	}
	entry = filepath.Clean(entry)
	srcEntry := filepath.Join(srcDir, entry)
	if _, err := os.Stat(srcEntry); err != nil {
		return nil, fmt.Errorf("install: entry %s: %w", srcEntry, err)
	}

	// Copy the entry script and any sibling .sh files plus README and
	// manifest, preserving _-prefixed helpers.
	entryDir := filepath.Dir(srcEntry)
	if err := copyDirShallow(entryDir, destDir, policy); err != nil {
		return nil, err
	}
	// Always include the manifest at the destination so list/info works
	// even when the entry lives in a subdirectory.
	if entryDir != srcDir {
		if err := copyFileIfExists(filepath.Join(srcDir, "manifest.toml"), filepath.Join(destDir, "manifest.toml"), policy); err != nil {
			return nil, err
		}
		if err := copyFileIfExists(filepath.Join(srcDir, "README.md"), filepath.Join(destDir, "README.md"), policy); err != nil {
			return nil, err
		}
	}
	return &cache.Installed{
		Type:      kind,
		Namespace: namespace,
		Name:      name,
	}, nil
}

func installAlias(home, name, value string, policy ConflictPolicy, source, version string) (*cache.Installed, error) {
	if err := validateFlatName("alias", name); err != nil {
		return nil, err
	}
	dst := paths.AliasFile(home, name)
	content := fmt.Sprintf("alias %s=%s\n", name, shellQuote(value))
	if err := writeWithPolicy(dst, []byte(content), 0o644, policy); err != nil {
		return nil, err
	}
	return &cache.Installed{
		Type:    "alias",
		Name:    name,
		Source:  source,
		Version: version,
	}, nil
}

func installCompletion(home, tool, generate string, policy ConflictPolicy, source, version string) (*cache.Installed, error) {
	if tool == "" || generate == "" {
		return nil, errors.New("install: completion requires tool and generate")
	}
	if err := validateFlatName("completion", tool); err != nil {
		return nil, err
	}
	out, err := runShell(generate)
	if err != nil {
		return nil, fmt.Errorf("install: generate completion for %s: %w", tool, err)
	}
	dst := paths.CompletionFile(home, tool)
	if err := writeWithPolicy(dst, out, 0o644, policy); err != nil {
		return nil, err
	}
	return &cache.Installed{
		Type:    "completion",
		Name:    tool,
		Source:  source,
		Version: version,
	}, nil
}

func runShell(cmd string) ([]byte, error) {
	c := exec.Command("bash", "-lc", cmd)
	return c.Output()
}

// shellQuote wraps a value in single quotes, escaping any embedded
// single quote.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// writeWithPolicy writes data to dst, honouring the conflict policy
// when dst already exists with different content.
func writeWithPolicy(dst string, data []byte, mode os.FileMode, policy ConflictPolicy) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("install: mkdir for %s: %w", dst, err)
	}
	if existing, err := os.ReadFile(dst); err == nil {
		if string(existing) == string(data) {
			return nil
		}
		switch policy {
		case ConflictPreferExisting, ConflictSkip:
			return nil
		case ConflictPreferNew:
			// fall through to overwrite
		default:
			return &ConflictError{Path: dst}
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("install: stat %s: %w", dst, err)
	}
	return os.WriteFile(dst, data, mode)
}

func copyFileIfExists(src, dst string, policy ConflictPolicy) error {
	in, err := os.Open(src)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("install: open %s: %w", src, err)
	}
	defer in.Close()
	data, err := io.ReadAll(in)
	if err != nil {
		return fmt.Errorf("install: read %s: %w", src, err)
	}
	return writeWithPolicy(dst, data, 0o644, policy)
}

func copyDirShallow(src, dst string, policy ConflictPolicy) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("install: readdir %s: %w", src, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		srcPath := filepath.Join(src, e.Name())
		info, err := os.Stat(srcPath)
		if err != nil {
			return fmt.Errorf("install: stat %s: %w", srcPath, err)
		}
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("install: read %s: %w", e.Name(), err)
		}
		// Preserve the executable bit when copying scripts so plugin
		// entry scripts can be exec'd; data files keep 0o644.
		mode := os.FileMode(0o644)
		if info.Mode()&0o111 != 0 {
			mode = 0o755
		}
		if err := writeWithPolicy(filepath.Join(dst, e.Name()), data, mode, policy); err != nil {
			return err
		}
	}
	return nil
}

// validateFlatName rejects alias and completion names that would be
// invalid or unsafe as a flat filename: empty, containing path
// separators, equal to "." or "..", or starting with a dash (which
// causes options-confusion in many tools).
func validateFlatName(kind, name string) error {
	if name == "" {
		return fmt.Errorf("install: %s name is empty", kind)
	}
	if name == "." || name == ".." || strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("install: %s name %q would resolve to a directory or escape SHY_HOME", kind, name)
	}
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("install: %s name %q starts with a dash; pick a name shells handle cleanly", kind, name)
	}
	return nil
}

// RemoveItem deletes one installed item from disk and from the cache.
// Returns true if anything was removed. Aliases and completions are
// flat; scripts and plugins are namespaced.
func RemoveItem(home string, kind, namespace, name string, c *cache.Cache) (bool, error) {
	var target string
	switch kind {
	case "alias":
		target = paths.AliasFile(home, name)
	case "completion":
		target = paths.CompletionFile(home, name)
	case "script":
		target = paths.ScriptDir(home, namespace, name)
	case "plugin":
		target = paths.PluginDir(home, namespace, name)
	default:
		return false, fmt.Errorf("install: unknown type %q", kind)
	}
	if err := os.RemoveAll(target); err != nil {
		return false, fmt.Errorf("install: remove %s: %w", target, err)
	}
	key := (&cache.Installed{Type: kind, Namespace: namespace, Name: name}).Key()
	return c.Remove(key), nil
}
