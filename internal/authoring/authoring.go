// SPDX-License-Identifier: MPL-2.0

// Package authoring backs `shy create` and `shy publish`. It owns the
// three git-state detection rules from docs/01-whitepaper.md (no git
// here → init; here is its own repo → proceed; here is inside a parent
// repo → abort exit 1) and the Conventional Commits version inference
// (release-please's algorithm, executed locally).
package authoring

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// GitState classifies the git environment around a script directory.
type GitState int

const (
	// NoGit means neither the directory nor any ancestor is a git repo.
	NoGit GitState = iota
	// SelfRoot means the directory itself is the root of a git repo.
	SelfRoot
	// InsideParent means an ancestor is a git repo but this directory
	// is not the root. Publication is refused.
	InsideParent
)

// DetectGit walks up from dir looking for .git. Returns the state and,
// for SelfRoot/InsideParent, the path of the discovered .git directory.
// Symlinks are resolved before walking so a symlinked script directory
// is judged by its real location.
func DetectGit(dir string) (GitState, string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return NoGit, "", err
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	if _, err := os.Stat(filepath.Join(abs, ".git")); err == nil {
		return SelfRoot, filepath.Join(abs, ".git"), nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return NoGit, "", err
	}
	cur := filepath.Dir(abs)
	for cur != "/" && cur != "." && cur != "" {
		if _, err := os.Stat(filepath.Join(cur, ".git")); err == nil {
			return InsideParent, filepath.Join(cur, ".git"), nil
		}
		next := filepath.Dir(cur)
		if next == cur {
			break
		}
		cur = next
	}
	return NoGit, "", nil
}

// SemVer is a permissive parser; rejects strings that don't match
// X.Y.Z (with optional v prefix and -pre/+build suffixes).
type SemVer struct {
	Major, Minor, Patch int
	Pre                 string
	Build               string
}

var semverRe = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?(?:\+([0-9A-Za-z.-]+))?$`)

// ParseSemVer returns a structured semver.
func ParseSemVer(s string) (SemVer, error) {
	m := semverRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return SemVer{}, fmt.Errorf("authoring: %q is not a semantic version", s)
	}
	maj, _ := strconv.Atoi(m[1])
	min, _ := strconv.Atoi(m[2])
	pat, _ := strconv.Atoi(m[3])
	return SemVer{Major: maj, Minor: min, Patch: pat, Pre: m[4], Build: m[5]}, nil
}

// String renders without the leading v.
func (s SemVer) String() string {
	out := fmt.Sprintf("%d.%d.%d", s.Major, s.Minor, s.Patch)
	if s.Pre != "" {
		out += "-" + s.Pre
	}
	if s.Build != "" {
		out += "+" + s.Build
	}
	return out
}

// Bump represents the kind of version change inferred from commits.
type Bump int

const (
	BumpNone Bump = iota
	BumpPatch
	BumpMinor
	BumpMajor
)

func (b Bump) String() string {
	switch b {
	case BumpPatch:
		return "patch"
	case BumpMinor:
		return "minor"
	case BumpMajor:
		return "major"
	default:
		return "none"
	}
}

// Apply produces the next semver given a current version and a bump.
func (b Bump) Apply(cur SemVer) SemVer {
	out := cur
	out.Pre, out.Build = "", ""
	switch b {
	case BumpMajor:
		out.Major++
		out.Minor, out.Patch = 0, 0
	case BumpMinor:
		out.Minor++
		out.Patch = 0
	case BumpPatch:
		out.Patch++
	}
	return out
}

// ConvCommitCounts is the breakdown of types in a commit-message list.
type ConvCommitCounts struct {
	Feat       int
	Fix        int
	Breaking   int
	Chore      int
	Docs       int
	Other      int
}

// InferBump applies the release-please rules:
//   - feat! or BREAKING CHANGE → major
//   - any feat → minor
//   - any fix → patch
//   - otherwise → none
func InferBump(c ConvCommitCounts) Bump {
	switch {
	case c.Breaking > 0:
		return BumpMajor
	case c.Feat > 0:
		return BumpMinor
	case c.Fix > 0:
		return BumpPatch
	default:
		return BumpNone
	}
}

// ParseConventional turns a slice of commit subjects (and optional
// bodies, separated by \n) into counts.
func ParseConventional(messages []string) ConvCommitCounts {
	c := ConvCommitCounts{}
	for _, m := range messages {
		subject := strings.SplitN(m, "\n", 2)[0]
		typ, breaking := classify(subject)
		if breaking || strings.Contains(m, "BREAKING CHANGE:") {
			c.Breaking++
			continue
		}
		switch typ {
		case "feat":
			c.Feat++
		case "fix":
			c.Fix++
		case "chore", "ci", "test", "refactor", "style", "perf":
			c.Chore++
		case "docs":
			c.Docs++
		default:
			c.Other++
		}
	}
	return c
}

var conventionalRe = regexp.MustCompile(`^([a-z]+)(?:\([^)]+\))?(!)?:\s`)

func classify(subject string) (typ string, breaking bool) {
	m := conventionalRe.FindStringSubmatch(strings.ToLower(subject))
	if m == nil {
		return "", false
	}
	return m[1], m[2] == "!"
}

// CommitsSinceFile returns commit subjects since the last revision that
// touched manifest.toml within dir. If no such commit exists, returns
// every commit in the repo.
func CommitsSinceFile(dir, file string) ([]string, error) {
	since, err := lastCommitTouching(dir, file)
	if err != nil {
		return nil, err
	}
	args := []string{"-C", dir, "log", "--pretty=%B%x00"}
	if since != "" {
		args = append(args, since+"..HEAD")
	}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("authoring: git log: %w", err)
	}
	parts := strings.Split(strings.TrimRight(string(out), "\x00\n"), "\x00")
	msgs := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		msgs = append(msgs, p)
	}
	return msgs, nil
}

func lastCommitTouching(dir, file string) (string, error) {
	c := exec.Command("git", "-C", dir, "log", "-1", "--pretty=%H", "--", file)
	out, err := c.Output()
	if err != nil {
		return "", fmt.Errorf("authoring: git log -1 %s: %w", file, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// WorkingTreeClean returns true if dir's worktree has no uncommitted
// changes.
func WorkingTreeClean(dir string) (bool, error) {
	c := exec.Command("git", "-C", dir, "status", "--porcelain")
	out, err := c.Output()
	if err != nil {
		return false, fmt.Errorf("authoring: git status: %w", err)
	}
	return strings.TrimSpace(string(out)) == "", nil
}

// GlobalGitUserName returns git's configured user.name or an empty
// string. Errors propagate only for hard failures (missing git binary).
func GlobalGitUserName() (string, error) {
	c := exec.Command("git", "config", "--global", "user.name")
	out, err := c.Output()
	if err != nil {
		// `git config` exits non-zero when the key is unset; treat as ""
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return "", nil
		}
		return "", fmt.Errorf("authoring: git config user.name: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// GitInit runs `git init` plus an empty initial commit so version
// inference has a baseline.
func GitInit(dir, commitMsg string) error {
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"add", "."},
		{"commit", "-q", "-m", commitMsg, "--allow-empty"},
	} {
		c := exec.Command("git", append([]string{"-C", dir}, args...)...)
		c.Stdout, c.Stderr = os.Stderr, os.Stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("authoring: git %v: %w", args, err)
		}
	}
	return nil
}
