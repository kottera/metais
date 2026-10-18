// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Command metais is the reference tool of the METAIS package format. It is a
// shell around internal/cli: it isolates the process from its host, then
// hands the arguments and the standard streams to cli.Main.
package main

import (
	"os"

	"github.com/kottera/metais/internal/cli"
	"github.com/kottera/metais/internal/hostenv"
)

func main() {
	d := hostenv.Isolate()
	os.Exit(cli.Main(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, d))
}
