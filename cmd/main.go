// SPDX-License-Identifier: MPL-2.0
package main

import "fmt"

// version is overridden at link time via -ldflags="-X main.version=...".
var version = "0.1.0-draft"

func main() {
	fmt.Printf("shy %s\n", version)
}
