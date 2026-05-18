// SPDX-License-Identifier: MPL-2.0
package manifest

import "testing"

func TestParseSingleScript(t *testing.T) {
	data := []byte(`
name = "git-autofetch"
version = "1.0.0"
description = "Run git fetch in background"
license = "MIT"
type = "script"
entry = "./git-autofetch.sh"

[source]
repo = "alice/git-autofetch"

[requires]
bash = ">=4"
binaries = ["git"]
`)
	m, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if m.Name != "git-autofetch" {
		t.Errorf("name=%q", m.Name)
	}
	if m.Type != "script" {
		t.Errorf("type=%q", m.Type)
	}
	if m.Source == nil || m.Source.Repo != "alice/git-autofetch" {
		t.Errorf("source=%+v", m.Source)
	}
	if m.Requires == nil || len(m.Requires.Binaries) != 1 {
		t.Errorf("requires=%+v", m.Requires)
	}
	if err := m.Validate(); err != nil {
		t.Errorf("validate: %v", err)
	}
}

func TestParseCollection(t *testing.T) {
	data := []byte(`
name = "alice-default"
version = "2.5.0"

[source]
repo = "alfred-intelligence/shy-setup"

[[items]]
name = "git-autofetch"
type = "script"
path = "./scripts/git-autofetch.sh"

[[items]]
name = "gh-clone"
type = "plugin"
command = "gh-clone"
path = "./plugins/gh-clone.sh"

[aliases]
la = "ls -A"

[[completions]]
tool = "kubectl"
generate = "kubectl completion bash"

[capabilities]
network = ["github.com"]
binaries = ["git", "gh"]

[security]
fixes = "CVE-2026-12345"
severity = "high"
description = "Fix path traversal"
`)
	m, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(m.Items) != 2 {
		t.Fatalf("items=%d", len(m.Items))
	}
	if m.Aliases["la"] != "ls -A" {
		t.Errorf("alias la=%q", m.Aliases["la"])
	}
	if len(m.Completions) != 1 || m.Completions[0].Tool != "kubectl" {
		t.Errorf("completions=%+v", m.Completions)
	}
	if m.Capabilities == nil || len(m.Capabilities.Network) != 1 {
		t.Errorf("capabilities=%+v", m.Capabilities)
	}
	if m.Security == nil || m.Security.Severity != "high" {
		t.Errorf("security=%+v", m.Security)
	}
	if err := m.Validate(); err != nil {
		t.Errorf("validate: %v", err)
	}
}

func TestValidateMissingName(t *testing.T) {
	m := &Manifest{Version: "1.0.0"}
	if err := m.Validate(); err == nil {
		t.Error("expected error for missing name")
	}
}

func TestValidateItemTypes(t *testing.T) {
	cases := []struct {
		name    string
		item    Item
		wantErr bool
	}{
		{"script with path", Item{Name: "a", Type: "script", Path: "./a.sh"}, false},
		{"script no path", Item{Name: "a", Type: "script"}, true},
		{"plugin missing command", Item{Name: "a", Type: "plugin", Path: "./a.sh"}, true},
		{"plugin missing path", Item{Name: "a", Type: "plugin", Command: "a"}, true},
		{"alias no value", Item{Name: "a", Type: "alias"}, true},
		{"alias ok", Item{Name: "a", Type: "alias", Value: "ls -A"}, false},
		{"completion missing fields", Item{Name: "a", Type: "completion"}, true},
		{"completion ok", Item{Name: "a", Type: "completion", Tool: "x", Generate: "x completion bash"}, false},
		{"unknown type", Item{Name: "a", Type: "weird"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateItem(c.item)
			if c.wantErr && err == nil {
				t.Error("want error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
