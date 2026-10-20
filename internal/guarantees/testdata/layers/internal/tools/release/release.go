// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Command release keeps only-referrers' exception for the release builder.
package main

import (
	"os"

	"github.com/kottera/metais/internal/tools/release/archive"
)

func main() {
	archive.Write()
	os.Exit(len(os.Args))
}
