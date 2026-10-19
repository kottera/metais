// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package pkgtree is the one interface through which anything reads a
// package (ADR-010 D11 item 1). A packed package and a source directory are
// both a Tree, and nothing reads a package any other way: the JSON Schema
// compiler, the loader and the browse commands are all written against it.
//
// The package defines the interface and its contract, and the one seam of
// ADR-010 D13 that guards every tree's verification, VerifyDigests. It
// performs no host file access and imports nothing that does; the trees
// that read a file or a directory live with the code that reaches the host
// (ADR-010 D11 item 3), and internal/pkgtree/pkgtreetest holds an in-memory
// tree for tests and the check every implementation is run against.
package pkgtree

import (
	"errors"
	"io"
	"strconv"

	"github.com/kottera/metais/internal/fault"
)

// A Tree is a package, opened: a fixed set of regular files, each addressed
// by its package entry path (ADR-001 D2) and read in place (ADR-001 D8).
//
// Every implementation meets this contract, and pkgtreetest.Check holds it
// to every clause a test can observe:
//
//   - Opening is where a tree verifies and refuses. A Tree exists only for
//     a package that opened, so a Tree's methods never report that the
//     package itself is bad; the constructor of each implementation does
//     that, once, before it returns a Tree.
//
//   - A Tree is read-only and fixed. Nothing writes through it, and once
//     constructed it answers List, Stat and Open the same way on every
//     call, in any order.
//
//   - Entries are files. A package has no directory entries: a directory
//     exists only as a prefix of an entry path (ADR-001 D2 item 6), is not
//     itself an entry, and is never listed, stat'ed or opened.
//
//   - Every entry path has ADR-001 D2's shape: non-empty, valid UTF-8,
//     segments separated by "/", no leading or trailing "/", no empty, "."
//     or ".." segment, and no "\" or ":". No entry path is a directory
//     prefix of another ("a" beside "a/b"). A tree over untrusted bytes
//     refuses at construction every name ADR-001 D2's portable-path
//     predicate refuses, so its paths are also NFC, portable and free of
//     caseless collisions; a reader relies only on exact lookup, which
//     needs none of that.
//
//   - Lookup is exact. Stat and Open compare the path they are given,
//     byte for byte, against the tree's set of entry paths (ADR-009 D6 item
//     5). There is no case folding, no normalisation, no cleaning of "." or
//     "..", and no resolution against a file system. A path that is not an
//     entry — absent, a directory prefix, malformed, or differing from an
//     entry only in case or normalisation — is refused with a *PathError
//     wrapping ErrNotExist before anything is read, beside a zero Info from
//     Stat and a nil reader from Open. This is what confines a reader: no
//     path, whoever wrote it, names anything outside the tree.
//
// A Tree is not required to be safe for concurrent use: nothing that reads
// a package starts a goroutine (ADR-010 D2 item 3).
type Tree interface {
	// List returns the path of every entry, each once, sorted in ascending
	// byte order. The slice is the caller's: changing it changes nothing
	// the tree returns afterwards.
	List() []string

	// Stat returns the entry at path. Its Info.Path is path, and its
	// Info.Size is the number of bytes Open's reader yields.
	Stat(path string) (Info, error)

	// Open returns a reader of the entry at path, from its first byte. The
	// reader yields exactly Info.Size bytes and then io.EOF. A tree that
	// reads from beneath — the bytes of a packed file — and finds them
	// different from what it verified when it was opened returns an error
	// from Read, never different content. Nothing is written to disk to
	// read an entry.
	//
	// Every call returns a new reader, independent of every other. The
	// caller closes it, which releases whatever Open acquired.
	Open(path string) (io.ReadCloser, error)
}

// Info describes one entry of a Tree. An entry has no mode, owner or time:
// it is a regular file (ADR-001 D2 item 6), and nothing about where or when
// it was written reaches a reader.
type Info struct {
	// Path is the entry's path, as List returns it.
	Path string

	// Size is the entry's length in bytes, uncompressed.
	Size int64
}

// ErrNotExist is the refusal of a path that is not an entry of the tree.
// Stat and Open wrap it in a *PathError.
var ErrNotExist = errors.New("no such entry")

// PathError records a refused lookup: the method, the path it was given and
// why.
type PathError struct {
	// Op is the method: "stat" or "open".
	Op string

	// Path is the path the caller gave, exactly as given.
	Path string

	// Err is the reason, ErrNotExist for a path that is not an entry.
	Err error
}

// Error quotes the path, so a path holding a control or format character,
// which no entry path can, is shown as the bytes it is.
func (e *PathError) Error() string {
	return e.Op + " " + strconv.Quote(e.Path) + ": " + e.Err.Error()
}

// Unwrap returns Err, so errors.Is(err, ErrNotExist) finds the reason.
func (e *PathError) Unwrap() error {
	return e.Err
}

// VerifyDigests runs verify, ADR-001 D8 step 6's check of a packed
// package's digest file, and returns what it reports: its findings, whose
// type the package does not import so that it reaches nothing of the
// host. It is the seam of
// fault.SkipDigest (ADR-010 D13 item 2): with that fault in faults the
// check is not run and verification returns success, which the digest
// detector must catch. The tool itself never holds a fault.
func VerifyDigests[F any](faults fault.Set, verify func() []F) []F {
	if faults.Has(fault.SkipDigest) {
		return nil
	}
	return verify()
}
