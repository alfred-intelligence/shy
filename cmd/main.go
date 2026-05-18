// SPDX-License-Identifier: MPL-2.0
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/alfred-intelligence/shy/internal/cmd"
)

// version is overridden at link time via -ldflags="-X main.version=...".
var version = "0.1.0-draft"

func main() {
	cmd.SetVersion(version)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(cmd.Execute(ctx))
}
