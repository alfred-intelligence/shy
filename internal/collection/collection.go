// SPDX-License-Identifier: MPL-2.0

// Package collection implements subscribe/update/unsubscribe semantics
// on top of internal/install. A collection is just a git repo with a
// manifest at the root; items are discovered from that manifest or, if
// the manifest is bare, from sub-manifests under sub-directories.
package collection

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/alfred-intelligence/shy/internal/cache"
	"github.com/alfred-intelligence/shy/internal/install"
	"github.com/alfred-intelligence/shy/internal/manifest"
	"github.com/alfred-intelligence/shy/internal/paths"
)

// SubscribeOptions controls one `shy collection subscribe` call.
type SubscribeOptions struct {
	Home   string
	Spec   string // github:user/name[@ref] or https://...
	Policy install.ConflictPolicy
}

// Subscribe clones the collection repo, installs every declared item,
// then recursively follows [[dependencies]]. Cycle prevention is by
// skip-if-already-installed: shy's namespacing makes two distinct
// owners with the same name coexist, so cycles can only emerge if a
// collection lists itself.
type SubResult struct {
	Name      string
	Repo      string
	Commit    string
	Installed []cache.Installed
}

func Subscribe(opts SubscribeOptions, c *cache.Cache) (*SubResult, error) {
	repo, ref := parseSpec(opts.Spec)
	if repo == "" {
		return nil, fmt.Errorf("collection: cannot parse spec %q (expected github:user/name[@ref])", opts.Spec)
	}
	name := nameFromRepo(repo)
	target := paths.CollectionDir(opts.Home, name)

	if _, err := os.Stat(target); err == nil {
		return nil, fmt.Errorf("collection: %q is already subscribed; use `shy collection update`", name)
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return nil, fmt.Errorf("collection: mkdir: %w", err)
	}
	commit, err := gitCloneTo(repo, ref, target)
	if err != nil {
		return nil, err
	}

	result := &SubResult{Name: name, Repo: repo, Commit: commit}
	if err := installCollection(target, repo, commit, name, opts.Home, opts.Policy, c, result, map[string]bool{}); err != nil {
		// Best-effort rollback of the clone if install failed.
		_ = os.RemoveAll(target)
		return nil, err
	}

	c.SetCollection(cache.Subscription{
		Name:   name,
		Repo:   repo,
		Ref:    ref,
		Commit: commit,
	})
	return result, nil
}

// Update re-clones each subscribed collection and reinstalls items.
// Returns per-collection results so the caller can print diffs.
func Update(home string, only string, policy install.ConflictPolicy, c *cache.Cache) ([]*SubResult, error) {
	out := []*SubResult{}
	for name, sub := range c.Collections {
		if only != "" && only != name {
			continue
		}
		tmp, err := os.MkdirTemp("", "shy-update-")
		if err != nil {
			return nil, err
		}
		commit, err := gitCloneTo(sub.Repo, sub.Ref, tmp)
		if err != nil {
			os.RemoveAll(tmp)
			return out, err
		}
		// Replace the existing clone atomically.
		target := paths.CollectionDir(home, name)
		_ = os.RemoveAll(target)
		if err := os.Rename(tmp, target); err != nil {
			return out, fmt.Errorf("collection: rename clone: %w", err)
		}

		res := &SubResult{Name: name, Repo: sub.Repo, Commit: commit}
		if err := installCollection(target, sub.Repo, commit, name, home, policy, c, res, map[string]bool{}); err != nil {
			return out, err
		}
		c.SetCollection(cache.Subscription{Name: name, Repo: sub.Repo, Ref: sub.Ref, Commit: commit})
		out = append(out, res)
	}
	return out, nil
}

// Unsubscribe drops the collection clone and removes items recorded
// with owner=<name>, unless those items are also owned by another
// subscription.
func Unsubscribe(home, name string, c *cache.Cache) (removed int, err error) {
	if _, ok := c.Collections[name]; !ok {
		return 0, fmt.Errorf("collection: %q is not subscribed", name)
	}
	c.DropCollection(name)

	for _, it := range c.List() {
		if it.Owner != name {
			continue
		}
		// Check whether any other collection still owns this item.
		ownedElsewhere := false
		for other := range c.Collections {
			if other == name {
				continue
			}
			if !isOwnedBy(c, it, other) {
				continue
			}
			ownedElsewhere = true
			break
		}
		if ownedElsewhere {
			continue
		}
		if _, e := install.RemoveItem(home, it.Type, it.Namespace, it.Name, c); e != nil {
			return removed, e
		}
		removed++
	}

	target := paths.CollectionDir(home, name)
	if err := os.RemoveAll(target); err != nil {
		return removed, fmt.Errorf("collection: remove clone: %w", err)
	}
	return removed, nil
}

func isOwnedBy(c *cache.Cache, want cache.Installed, owner string) bool {
	for _, it := range c.List() {
		if it.Owner == owner && it.Type == want.Type && it.Namespace == want.Namespace && it.Name == want.Name {
			return true
		}
	}
	return false
}

// installCollection installs every item declared by the collection's
// manifest. If the root manifest is bare, sub-manifests in immediate
// sub-directories are installed instead.
func installCollection(dir, repoSource, commit, owner, home string, policy install.ConflictPolicy, c *cache.Cache, res *SubResult, visited map[string]bool) error {
	if visited[repoSource] {
		return nil
	}
	visited[repoSource] = true

	mPath := filepath.Join(dir, "manifest.toml")
	data, err := os.ReadFile(mPath)
	if err != nil {
		return fmt.Errorf("collection: read %s: %w", mPath, err)
	}
	m, err := manifest.Parse(data)
	if err != nil {
		return err
	}

	bare := m.Entry == "" && len(m.Items) == 0 && len(m.Aliases) == 0 && len(m.Completions) == 0
	if bare {
		// Discover sub-manifests one level deep.
		entries, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("collection: readdir: %w", err)
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			sub := filepath.Join(dir, e.Name())
			if _, err := os.Stat(filepath.Join(sub, "manifest.toml")); errors.Is(err, fs.ErrNotExist) {
				continue
			} else if err != nil {
				return err
			}
			ir, err := install.Bundle(sub, install.Options{
				Home:   home,
				Source: repoSource + "/" + e.Name(),
				Ref:    commit,
				Policy: policy,
			}, c)
			if err != nil {
				return err
			}
			tagOwner(c, ir.Installed, owner)
			res.Installed = append(res.Installed, ir.Installed...)
		}
	} else {
		ir, err := install.Bundle(dir, install.Options{
			Home:   home,
			Source: repoSource,
			Ref:    commit,
			Policy: policy,
		}, c)
		if err != nil {
			return err
		}
		tagOwner(c, ir.Installed, owner)
		res.Installed = append(res.Installed, ir.Installed...)
	}

	// Recurse into [[dependencies]].
	for _, dep := range m.Dependencies {
		if dep.Source == "" {
			continue
		}
		if dep.Type == "optional" {
			continue
		}
		depRepo, depRef := parseSpec(dep.Source)
		if depRepo == "" {
			continue
		}
		depTarget := paths.CollectionDir(home, nameFromRepo(depRepo))
		if _, err := os.Stat(depTarget); err == nil {
			// Already cloned by another path — skip.
			continue
		}
		if _, err := gitCloneTo(depRepo, depRef, depTarget); err != nil {
			if dep.Type == "recommended" {
				continue
			}
			return err
		}
		if err := installCollection(depTarget, depRepo, "", nameFromRepo(depRepo), home, policy, c, res, visited); err != nil {
			return err
		}
		c.SetCollection(cache.Subscription{Name: nameFromRepo(depRepo), Repo: depRepo, Ref: depRef})
	}
	return nil
}

func tagOwner(c *cache.Cache, items []cache.Installed, owner string) {
	for _, it := range items {
		it.Owner = owner
		c.Add(it)
	}
}

// parseSpec accepts "github:user/name", "github:user/name@ref",
// "https://...", "file://...", or "user/name" (assumed GitHub). Returns
// the repo URL (or file path) and an optional ref.
func parseSpec(spec string) (repo, ref string) {
	switch {
	case strings.HasPrefix(spec, "github:"):
		s := strings.TrimPrefix(spec, "github:")
		if i := strings.Index(s, "@"); i > 0 {
			return "https://github.com/" + s[:i] + ".git", s[i+1:]
		}
		return "https://github.com/" + s + ".git", ""
	case strings.HasPrefix(spec, "https://"), strings.HasPrefix(spec, "http://"):
		return spec, ""
	case strings.HasPrefix(spec, "file://"):
		return spec, ""
	case strings.HasPrefix(spec, "/"):
		return "file://" + spec, ""
	case strings.Contains(spec, "/") && !strings.Contains(spec, " "):
		return "https://github.com/" + spec + ".git", ""
	default:
		return "", ""
	}
}

func nameFromRepo(repo string) string {
	base := repo
	base = strings.TrimSuffix(base, ".git")
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	return paths.SafeName(base)
}

func gitCloneTo(repo, ref, dst string) (commit string, err error) {
	if _, err := exec.LookPath("git"); err != nil {
		return "", fmt.Errorf("collection: git not found on PATH")
	}
	args := []string{"clone", "--depth", "1"}
	if ref != "" {
		args = append(args, "--branch", ref)
	}
	args = append(args, repo, dst)
	cmd := exec.Command("git", args...)
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("collection: clone %s: %w", repo, err)
	}
	c := exec.Command("git", "-C", dst, "rev-parse", "HEAD")
	out, err := c.Output()
	if err == nil {
		commit = strings.TrimSpace(string(out))
	}
	return commit, nil
}
