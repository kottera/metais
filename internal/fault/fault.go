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

// SkipDigest makes the verification of a packed package's digest file
// return success (ADR-010 D13 item 2): the digest detector must catch a
// package whose entries are not those its digest file names. Its seam is
// internal/pkgtree.
const SkipDigest Set = 1 << 0

// FirstFinding makes a report keep only the first error it is given and
// drop every later one (ADR-010 D13 item 2): the SC8 planted-errors test
// must catch a report that stops at its first error. Its seam is
// internal/finding.
const FirstFinding Set = 1 << 1

// UnpackToTemp makes browse read an entry by extracting it to a temporary
// file and reading it back (ADR-010 D13 item 2): the read-in-place detector
// must catch a reading command that writes to the host. Its seam is
// internal/browse.
const UnpackToTemp Set = 1 << 2

// SkipConfine makes the output path check return success (ADR-010 D13
// item 2): a write goes to its name joined onto the output directory's host
// path, through no root, so a name that climbs out of the directory or
// through a link lands where it points. The confinement detector must catch
// a write outside the output directory. Its seam is internal/hostfs.
const SkipConfine Set = 1 << 3

// MapOrder makes a render iterate an object's members in the order of a Go
// map built from them, rather than in canonical order (ADR-010 D13 item 2,
// Amendment A2): the determinism detector must catch outputs that differ
// between renders. Its seam is internal/template, which sets the engine's
// Env.UnorderedObjects from it.
const MapOrder Set = 1 << 4

// ReadExisting makes a write into an existing output directory read the
// content of every file it is about to replace, as it checks it (ADR-010
// D13 item 2): the existing-destination detector must catch a write that
// reads what the output directory holds, which ADR-009 D8 item 4 forbids.
// Its seam is internal/hostfs.
const ReadExisting Set = 1 << 5

// RemoveForeign makes a write into an existing output directory remove,
// as it commits, every regular file beside a committed one that the write
// did not plan (ADR-010 D13 item 2): the existing-destination detector
// must catch a write that removes a name it did not create, which ADR-005
// D15 item 7 and Amendment A1 item 6 forbid. Its seam is internal/hostfs.
const RemoveForeign Set = 1 << 6

// StdoutNotice makes a render that writes bytes to standard output write a
// notice line there before them, on a refusal too (ADR-008 D16 item 5):
// the standard-output purity test must catch a standard output that holds
// anything but the bytes. Its seam is internal/cli.
const StdoutNotice Set = 1 << 7

// UpdateBeyondExpected makes test --update reckon the files it removes
// over the whole example directory rather than its expected/ (ADR-010 D13
// item 2): the example's inputs and example.yaml, which its render does
// not produce, are removed with the stale expected files. The update
// confinement detector must catch a removal outside
// examples/<name>/expected/, which ADR-007 D5 item 5 and RULINGS R1-006
// forbid. Its seam is internal/cli.
const UpdateBeyondExpected Set = 1 << 8
