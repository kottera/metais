// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Command metais keeps only-referrers' exception.
package main

import (
	"os"

	"github.com/kottera/metais/internal/cli"
)

func main() {
	cli.Run()
	os.Exit(len(os.Args))
}
