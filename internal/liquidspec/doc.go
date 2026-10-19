// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package liquidspec is the test-only JSON-RPC server through which
// Shopify's liquid-spec drives the reference engine (ADR-011 D17 item 3).
// Its tests are the program, as internal/vectorgen's are: the test binary
// run with -serve reads one JSON-RPC 2.0 message per line on standard
// input and writes one per line on standard output, as liquid-spec's
// driver protocol defines it at the commit
// spec/conformance/delegated/liquid-spec.json pins
// (docs/json-rpc-protocol.md there): initialize, compile, render and the
// quit notification. The same binary run with -results judges a run's
// result against the departures that file lists (TestDelegatedRun).
//
//	go test -c -o liquidspec.test ./internal/liquidspec
//	./liquidspec.test -serve
//
// A template is parsed and rendered as metais render parses and renders a
// package's templates: in the dialect of internal/template, the standard
// catalog of internal/liquid with the globals' names reserved (ADR-004 D1,
// D7 item 8), under ADR-009's default limits. The compile request's file
// system is the package's templates/ directory, each file an entry named
// exactly as the request names it, and the render request's environment is
// the globals. Nothing else is bound, and nothing is read from the host.
//
// Nothing imports the package, so no release build links it; the layering
// table of internal/guarantees holds it to that.
// spec/conformance/delegated/README.md is how a person runs the pinned
// selection against it.
package liquidspec
