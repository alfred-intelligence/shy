// SPDX-License-Identifier: MPL-2.0
package authoring

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestParseSemVer(t *testing.T) {
	cases := []struct {
		in   string
		want SemVer
	}{
		{"1.2.3", SemVer{1, 2, 3, "", ""}},
		{"v0.1.0", SemVer{0, 1, 0, "", ""}},
		{"2.0.0-rc.1", SemVer{2, 0, 0, "rc.1", ""}},
		{"v1.0.0+build.7", SemVer{1, 0, 0, "", "build.7"}},
	}
	for _, c := range cases {
		got, err := ParseSemVer(c.in)
		if err != nil {
			t.Errorf("parse %q: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("parse %q = %+v, want %+v", c.in, got, c.want)
		}
	}
	if _, err := ParseSemVer("not a version"); err == nil {
		t.Error("expected error for invalid semver")
	}
}

func TestInferBump(t *testing.T) {
	cases := []struct {
		name string
		c    ConvCommitCounts
		want Bump
	}{
		{"breaking dominates", ConvCommitCounts{Feat: 5, Fix: 5, Breaking: 1}, BumpMajor},
		{"feat is minor", ConvCommitCounts{Feat: 1, Fix: 0}, BumpMinor},
		{"fix is patch", ConvCommitCounts{Fix: 3}, BumpPatch},
		{"chore only is none", ConvCommitCounts{Chore: 4, Docs: 1}, BumpNone},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := InferBump(c.c); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestBumpApply(t *testing.T) {
	cur := SemVer{1, 2, 3, "rc.1", "b"}
	cases := []struct {
		bump Bump
		want SemVer
	}{
		{BumpMajor, SemVer{2, 0, 0, "", ""}},
		{BumpMinor, SemVer{1, 3, 0, "", ""}},
		{BumpPatch, SemVer{1, 2, 4, "", ""}},
		{BumpNone, SemVer{1, 2, 3, "", ""}},
	}
	for _, c := range cases {
		if got := c.bump.Apply(cur); got != c.want {
			t.Errorf("bump %v: got %+v, want %+v", c.bump, got, c.want)
		}
	}
}

func TestParseConventional(t *testing.T) {
	msgs := []string{
		"feat: add subscribe command",
		"fix(install): handle missing path",
		"chore: bump deps",
		"feat!: rename SHY_HOME",
		"docs: update README",
		"refactor: split install.go",
		"not a conventional commit",
		"feat: another thing\n\nBREAKING CHANGE: removed old flag",
	}
	c := ParseConventional(msgs)
	if c.Feat != 1 { // the conventional feat (the breaking ones are counted as Breaking)
		t.Errorf("feat=%d", c.Feat)
	}
	if c.Fix != 1 {
		t.Errorf("fix=%d", c.Fix)
	}
	if c.Breaking != 2 {
		t.Errorf("breaking=%d", c.Breaking)
	}
	if c.Chore != 2 {
		t.Errorf("chore=%d", c.Chore)
	}
	if c.Docs != 1 {
		t.Errorf("docs=%d", c.Docs)
	}
	if c.Other != 1 {
		t.Errorf("other=%d", c.Other)
	}
}

func TestDetectGit(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "parent")
	child := filepath.Join(parent, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}

	// NoGit case
	state, _, err := DetectGit(child)
	if err != nil {
		t.Fatal(err)
	}
	if state != NoGit {
		t.Errorf("expected NoGit, got %v", state)
	}

	// SelfRoot case
	runGit(t, parent, "init", "-q", "-b", "main")
	state, _, _ = DetectGit(parent)
	if state != SelfRoot {
		t.Errorf("expected SelfRoot, got %v", state)
	}

	// InsideParent case
	state, _, _ = DetectGit(child)
	if state != InsideParent {
		t.Errorf("expected InsideParent, got %v", state)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	c.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
