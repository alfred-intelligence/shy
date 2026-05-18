// SPDX-License-Identifier: MPL-2.0
package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestEmbeddedInitMatchesCanonical guards against drift between the
// canonical init/init.bash (the file shellcheck runs against, the file
// packaged into .deb/.rpm) and the copy embedded into the binary for
// `shy init` to write.
func TestEmbeddedInitMatchesCanonical(t *testing.T) {
	_, here, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(here), "..", "..")
	canonical, err := os.ReadFile(filepath.Join(root, "init", "init.bash"))
	if err != nil {
		t.Fatalf("read canonical: %v", err)
	}
	if string(canonical) != EmbeddedInitBash {
		t.Fatal("init/init.bash and internal/cmd/init.bash drifted — run `cp init/init.bash internal/cmd/init.bash`")
	}
}
