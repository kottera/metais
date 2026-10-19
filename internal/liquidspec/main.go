// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Command liquidspec is the test-only JSON-RPC server through which
// Shopify's liquid-spec drives the reference engine (ADR-011 D17 item 3,
// ADR-010's interface with ADR-011). It reads one JSON-RPC 2.0 message per
// line on standard input and writes one per line on standard output, as
// liquid-spec's driver protocol defines it at the commit
// spec/conformance/delegated/liquid-spec.json pins
// (docs/json-rpc-protocol.md there): initialize, compile, render and the
// quit notification.
//
// A template is parsed and rendered as metais render parses and renders a
// package's templates: in the dialect of internal/template, the standard
// catalog of internal/liquid with the globals' names reserved (ADR-004 D1,
// D7 item 8), under ADR-009's default limits. The compile request's file
// system is the package's templates/ directory, each file an entry named
// exactly as the request names it, and the render request's environment is
// the globals. Nothing else is bound, and nothing is read from the host.
//
// It is not part of the release binary: it is a main package, which
// nothing can import, and internal/guarantees' layering table holds it to
// that. spec/conformance/delegated/README.md is how a person runs the
// pinned selection against it.
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := serve(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "liquidspec:", err)
		os.Exit(1)
	}
}
