// SPDX-License-Identifier: MPL-2.0
package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissing(t *testing.T) {
	p := filepath.Join(t.TempDir(), "cache.json")
	c, err := Load(p)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if c.Schema != SchemaVersion {
		t.Errorf("schema=%d want %d", c.Schema, SchemaVersion)
	}
	if len(c.Installed) != 0 {
		t.Errorf("installed=%d want 0", len(c.Installed))
	}
}

func TestRoundtrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "cache.json")
	c, _ := Load(p)
	c.Add(Installed{Type: "script", Namespace: "alice", Name: "git-autofetch", Version: "1.0.0", Source: "alice/git-autofetch"})
	c.Add(Installed{Type: "alias", Name: "ll", Source: "local"})
	c.SetCollection(Subscription{Name: "shy-stdlib", Repo: "alfred-intelligence/shy-stdlib", Commit: "deadbeef"})
	if err := c.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}
	c2, err := Load(p)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(c2.Installed) != 2 {
		t.Errorf("installed=%d want 2", len(c2.Installed))
	}
	if len(c2.Collections) != 1 {
		t.Errorf("collections=%d want 1", len(c2.Collections))
	}
	if got := c2.Installed["script:alice/git-autofetch"]; got.Version != "1.0.0" {
		t.Errorf("missing item: %+v", c2.Installed)
	}
	if !c2.Remove("alias:ll") {
		t.Error("remove alias:ll returned false")
	}
	if c2.Remove("alias:nope") {
		t.Error("removing nonexistent returned true")
	}
}

func TestStaleSchemaDiscarded(t *testing.T) {
	p := filepath.Join(t.TempDir(), "cache.json")
	// Write a bogus schema version directly.
	if err := os.WriteFile(p, []byte(`{"schema":999,"installed":{"x":{"name":"x"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(c.Installed) != 0 {
		t.Errorf("expected stale cache to be discarded, got %d items", len(c.Installed))
	}
}

