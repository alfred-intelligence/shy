// SPDX-License-Identifier: MPL-2.0
package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alfred-intelligence/shy/internal/paths"
)

// TestInitWritesLayout exercises the init runtime against an isolated
// SHY_HOME and a fake .bashrc.
func TestInitWritesLayout(t *testing.T) {
	home := t.TempDir()
	bashrc := filepath.Join(t.TempDir(), ".bashrc")
	t.Setenv("SHY_HOME", home)
	t.Setenv("SHY_TEST_BASHRC", bashrc)

	out := &bytes.Buffer{}
	if err := runInit(out); err != nil {
		t.Fatalf("init: %v", err)
	}
	for _, sub := range []string{
		"installed",
		"helpers/aliases", "helpers/completions",
		"overrides.d/installed",
		"overrides.d/helpers/aliases", "overrides.d/helpers/completions",
	} {
		if _, err := os.Stat(filepath.Join(home, sub)); err != nil {
			t.Errorf("subdir %s missing: %v", sub, err)
		}
	}
	if _, err := os.Stat(filepath.Join(home, "init.bash")); err != nil {
		t.Errorf("init.bash missing: %v", err)
	}
	data, err := os.ReadFile(bashrc)
	if err != nil {
		t.Fatalf("bashrc: %v", err)
	}
	if !strings.Contains(string(data), "shy/init.bash") {
		t.Errorf("bashrc missing source line, got: %q", data)
	}
	if _, err := os.Stat(paths.CompletionFile(home, "shy")); err != nil {
		t.Errorf("completion bootstrap missing: %v", err)
	}

	// Idempotent: re-run should not duplicate the bashrc line.
	out.Reset()
	if err := runInit(out); err != nil {
		t.Fatalf("second init: %v", err)
	}
	data2, _ := os.ReadFile(bashrc)
	lines := strings.Split(string(data2), "\n")
	sourceLines := 0
	for _, l := range lines {
		if strings.Contains(l, "source") && strings.Contains(l, "shy/init.bash") {
			sourceLines++
		}
	}
	if sourceLines != 1 {
		t.Errorf("expected exactly 1 source line, got %d:\n%s", sourceLines, data2)
	}
}

// TestEndToEndAliasAndList walks the typical onboarding: shy alias →
// shy list --json. The data round-trips through cache.json.
func TestEndToEndAliasAndList(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SHY_HOME", home)
	t.Setenv("SHY_TEST_BASHRC", filepath.Join(t.TempDir(), ".bashrc"))
	if err := runInit(&bytes.Buffer{}); err != nil {
		t.Fatalf("init: %v", err)
	}

	if err := runAlias(&bytes.Buffer{}, []string{"ll=ls", "-alh"}); err != nil {
		t.Fatalf("alias: %v", err)
	}

	out := &bytes.Buffer{}
	if err := runList(out, "", false, true); err != nil {
		t.Fatalf("list: %v", err)
	}
	var resp struct {
		Items []struct {
			Type, Name string
		} `json:"items"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("decode json: %v\n%s", err, out.String())
	}
	// init also registers the shy bash completion, so expect 2 items.
	var aliasItem *struct{ Type, Name string }
	for i := range resp.Items {
		if resp.Items[i].Name == "ll" {
			aliasItem = &resp.Items[i]
		}
	}
	if aliasItem == nil {
		t.Errorf("expected alias 'll' in items: %+v", resp.Items)
	}

	aliasContent, err := os.ReadFile(paths.AliasFile(home, "ll"))
	if err != nil {
		t.Fatalf("alias file: %v", err)
	}
	if !strings.Contains(string(aliasContent), "ls -alh") {
		t.Errorf("alias content: %q", aliasContent)
	}
}

// TestRootBuilds is a cheap sanity check that the entire command tree
// constructs without panicking.
func TestRootBuilds(t *testing.T) {
	root := New()
	if root.Use != "shy" {
		t.Errorf("root.Use=%q", root.Use)
	}
	if len(root.Commands()) < 8 {
		t.Errorf("expected ≥8 subcommands, got %d", len(root.Commands()))
	}
	// Execute the help path with a no-op context; this would panic if
	// any RunE wired itself wrong.
	root.SetArgs([]string{"--help"})
	root.SetOut(&bytes.Buffer{})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Errorf("help: %v", err)
	}
}
