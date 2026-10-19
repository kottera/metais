// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package conformance is the conformance runner of ADR-011 D11 and ADR-010
// D14 item 2: TestConformance runs every vector under spec/conformance/
// against an implementation driven as a black box, through the reference
// tool's command line or through the driver protocol of
// spec/conformance/README.md.
//
// The runner is all test code. It starts processes, sets their environment,
// reads the clock and walks the file system, none of which the module's
// non-test code may do (ADR-009 D16 item 3), and it never links the
// implementation it judges (ADR-011 D11 item 2). Its own proof, TestSelfProof,
// runs it against the test binary re-executed as a fake implementation, once
// per fault the runner must catch (ADR-011 D18).
//
// Run one area with
//
//	go test ./internal/conformance -run 'TestConformance/out/'
//
// and pass the flags of ADR-011 D11 item 3 after -args.
package conformance
