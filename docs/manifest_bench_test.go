package manifest

import (
	"testing"
)

// Fixtures: stable inputs that won't drift between runs.
var (
	minimalManifest = []byte(`
name = "git-autofetch"
version = "1.0.0"
type = "script"
entry = "./git-autofetch.sh"

[source]
repo = "alice/git-autofetch"

[requires]
bash = ">=4"
binaries = ["git"]
`)

	collectionManifest = []byte(`
name = "test-collection"
version = "1.0.0"

[source]
repo = "alice/test-collection"

[[items]]
name = "foo"
type = "script"
path = "./scripts/foo.sh"

[[items]]
name = "bar"
type = "plugin"
command = "bar"
path = "./plugins/bar.sh"

[[items]]
name = "ll"
type = "alias"
value = "ls -alh"

[aliases]
la = "ls -A"
gst = "git status -sb"

[[completions]]
tool = "kubectl"
generate = "kubectl completion bash"

[[dependencies]]
source = "github:bob/git-helpers"
constraint = "^1.0"
type = "required"
`)
)

// Baseline cost for the common single-item case.
func BenchmarkParseMinimal(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := Parse(minimalManifest); err != nil {
			b.Fatal(err)
		}
	}
}

// Heavier shape with items, aliases, completions, and dependencies.
func BenchmarkParseCollection(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := Parse(collectionManifest); err != nil {
			b.Fatal(err)
		}
	}
}
