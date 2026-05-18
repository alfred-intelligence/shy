// SPDX-License-Identifier: MPL-2.0
package cmd

import _ "embed"

// EmbeddedInitBash is the literal contents of init/init.bash baked into
// the binary so `shy init` can write it without an on-disk template.
//
//go:embed init.bash
var EmbeddedInitBash string
