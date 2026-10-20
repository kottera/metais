// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Command metais keeps only-referrers' exception, and breaks never-depends
// by importing a package of the release builder.
package main

import (
	"os"

	"github.com/kottera/metais/internal/cli"
	"github.com/kottera/metais/internal/tools/release/archive"
)

func main() {
	cli.Run()
	archive.Write()
	os.Exit(len(os.Args))
}
