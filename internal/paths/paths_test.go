// SPDX-License-Identifier: MPL-2.0
package paths

import "testing"

func TestSafeName(t *testing.T) {
	cases := map[string]string{
		"MacBook-Pro-2.local":   "macbook-pro-2",
		"laptop":                "laptop",
		"alice":                 "alice",
		"Alice/git":             "alice-git",
		"!!!.local":             "local",
		"  Weird Name  ":        "weird-name",
		"under_score.local":     "under-score",
		"":                      "local",
	}
	for in, want := range cases {
		if got := SafeName(in); got != want {
			t.Errorf("SafeName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNamespaceFromRepo(t *testing.T) {
	cases := map[string]string{
		"alice/git-autofetch":            "alice",
		"alfred-intelligence/shy-setup":  "alfred-intelligence",
		"single":                         "single",
	}
	for in, want := range cases {
		if got := NamespaceFromRepo(in); got != want {
			t.Errorf("NamespaceFromRepo(%q) = %q, want %q", in, got, want)
		}
	}
}
