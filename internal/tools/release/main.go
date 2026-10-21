// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Command release builds a release of metais (ADR-010 D12):
//
//	go run ./internal/tools/release -version <semver>
//
// run from a clean checkout whose HEAD is the tag v<semver>. It builds each
// shipped target twice, in two fresh clones of HEAD with separate GOCACHEs,
// with go.mod's toolchain whatever the developer's GOTOOLCHAIN, and with
// no workspace, go env file, proxy or flags of the developer's. It refuses
// unless the two binaries are equal byte for byte and go version -m shows
// go.mod's toolchain and the release's build settings. It refuses a tool whose
// implements names an unreleased specification revision (ADR-011 D5 item
// 6). It then writes dist/ at the work tree's root: one
// metais_<version>_<os>_<arch>.tar.gz per shipped target and SHA256SUMS.
//
// The builder is a developer's tool, never linked into metais, and is the
// one package ADR-009 Amendment A2 lets run commands, read the environment
// and write files outside internal/hostfs.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	os.Exit(cli(os.Args[1:], os.Stdout, os.Stderr, os.Environ()))
}

// cli parses args and runs the builder from the working directory, writing
// its report to stdout and a refusal to stderr. It returns the exit status:
// 0 when dist/ is written, 1 on a refusal, 2 on a usage error.
func cli(args []string, stdout, stderr io.Writer, environ []string) int {
	fs := flag.NewFlagSet("release", flag.ContinueOnError)
	fs.SetOutput(stderr)
	version := fs.String("version", "", "the release's semantic version, without a leading v; HEAD must be the tag v<version>")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 || *version == "" {
		fmt.Fprintln(stderr, "usage: go run ./internal/tools/release -version <semver>")
		return 2
	}
	c := config{version: *version, stdout: stdout, environ: environ, proxy: releaseProxy}
	if err := c.run(); err != nil {
		fmt.Fprintln(stderr, "release:", err)
		return 1
	}
	return 0
}
