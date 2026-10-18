// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package specdata holds byte-for-byte copies of the spec files the binary
// embeds (ADR-010 D2 item 4). The canonical files live under spec/, which
// //go:embed cannot reach from here; go generate ./internal/specdata
// refreshes the copies, and the gate fails if that changes anything.
//
// Each copy keeps its path below spec/: spec/findings/registry.json is
// copied to findings/registry.json in this directory. The files copied are
// those named by specFiles in generate_test.go.
//
// The copier is a test, run by go generate with -specdata.generate, so this
// package's non-test files hold no file access.
package specdata

//go:generate go test -count=1 -run=^TestGenerate$ -args -specdata.generate
