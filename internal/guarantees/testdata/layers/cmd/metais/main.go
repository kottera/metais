// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Command metais keeps only-referrers' exception, and breaks never-depends
// by importing a package of the release builder and one of the UX runner.
package main

import (
	"os"

	"metais.org/metais/internal/cli"
	"metais.org/metais/internal/tools/release/archive"
	"metais.org/metais/internal/tools/ux/scenario"
)

func main() {
	cli.Run()
	archive.Write()
	scenario.Load()
	os.Exit(len(os.Args))
}
