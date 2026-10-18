// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package fault holds the test-only fault switches of ADR-010 D13 item 2.
//
// A fault deliberately breaks one headline guarantee at one named seam, so
// that the guarantee's detector can be shown to catch it in every go test.
// Faults are injected by options, never by a flag or the environment: the
// pipeline's options carry a Set, internal/cli builds that Set zero, and a
// seam consults it with opts.Faults.Has(fault.X).
//
// Each named fault is a Set with one bit, declared here beside the detector
// that needs it.
package fault

// Set is a set of faults, one bit per fault. The zero Set holds no fault and
// is the only Set the tool itself ever builds.
type Set uint64

// Has reports whether every fault of f is in s. The empty set names no fault,
// so Has(0) is false.
func (s Set) Has(f Set) bool {
	return f != 0 && s&f == f
}
