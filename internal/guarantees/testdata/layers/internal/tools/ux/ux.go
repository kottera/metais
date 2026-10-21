// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Command ux keeps only-referrers' exception for the UX runner.
package main

import (
	"os"

	"github.com/kottera/metais/internal/tools/ux/scenario"
)

func main() {
	scenario.Load()
	os.Exit(len(os.Args))
}
