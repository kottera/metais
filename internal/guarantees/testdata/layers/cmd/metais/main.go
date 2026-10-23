// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Command metais keeps only-referrers' exception, and breaks never-depends
// by importing a package of the release builder, one of the UX runner and
// one of the site's build step.
package main

import (
	"os"

	"metais.org/metais/internal/cli"
	"metais.org/metais/internal/tools/release/archive"
	"metais.org/metais/internal/tools/site/mdhtml"
	"metais.org/metais/internal/tools/ux/scenario"
)

func main() {
	cli.Run()
	archive.Write()
	scenario.Load()
	mdhtml.Convert()
	os.Exit(len(os.Args))
}
