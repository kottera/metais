// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package pkgtreetest helps test code that reads a package: New builds an
// in-memory pkgtree.Tree from a map of entry paths to their bytes, and Check
// holds any Tree to the pkgtree.Tree contract.
//
// It is for tests only. Like internal/pkgtree, it performs no host file
// access and imports nothing that does, so a test that builds a package in
// memory touches nothing outside it.
package pkgtreetest

import (
	"bytes"
	"errors"
	"io"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/kottera/metais/internal/pkgtree"
)

// tree is the in-memory Tree New builds.
type tree struct {
	paths []string          // every entry path, sorted
	files map[string][]byte // each entry's bytes, owned by the tree
}

// New returns a Tree whose entries are files: each key an entry path, each
// value its bytes. The tree keeps its own copy of every value, so changing
// files afterwards changes nothing it returns.
//
// New refuses a set of paths that would break the pkgtree.Tree contract —
// a path without ADR-001 D2's shape, or a path that is a directory prefix of
// another — and reports every such path at once, in path order. It does not
// apply ADR-001 D2's portable-path predicate, which is the business of a
// tree that reads untrusted bytes, so a test may build a tree from any
// well-shaped names it needs.
func New(files map[string][]byte) (pkgtree.Tree, error) {
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	slices.Sort(paths)
	if errs := pathProblems(paths); len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	t := &tree{paths: paths, files: make(map[string][]byte, len(files))}
	for _, p := range paths {
		t.files[p] = bytes.Clone(files[p])
	}
	return t, nil
}

func (t *tree) List() []string {
	return slices.Clone(t.paths)
}

func (t *tree) Stat(path string) (pkgtree.Info, error) {
	b, ok := t.files[path]
	if !ok {
		return pkgtree.Info{}, &pkgtree.PathError{Op: "stat", Path: path, Err: pkgtree.ErrNotExist}
	}
	return pkgtree.Info{Path: path, Size: int64(len(b))}, nil
}

func (t *tree) Open(path string) (io.ReadCloser, error) {
	b, ok := t.files[path]
	if !ok {
		return nil, &pkgtree.PathError{Op: "open", Path: path, Err: pkgtree.ErrNotExist}
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

// pathProblems returns one error for every path of sorted that has not
// ADR-001 D2's shape or is a directory prefix of another path of sorted,
// in path order.
func pathProblems(sorted []string) []error {
	var errs []error
	for _, p := range sorted {
		if why := shapeProblem(p); why != "" {
			errs = append(errs, pathError(p, why))
			continue
		}
		// Every path that begins with p/ sorts into one run, which starts
		// where p/ itself would.
		dir := p + "/"
		if j, _ := slices.BinarySearch(sorted, dir); j < len(sorted) && strings.HasPrefix(sorted[j], dir) {
			errs = append(errs, pathError(p, "is both an entry and a directory of "+strconv.Quote(sorted[j])))
		}
	}
	return errs
}

// shapeProblem says why p has not ADR-001 D2's shape (item 1's encoding and
// item 2's segments), or returns "" if it has.
func shapeProblem(p string) string {
	switch {
	case p == "":
		return "is empty"
	case !utf8.ValidString(p):
		return "is not valid UTF-8"
	case strings.HasPrefix(p, "\uFEFF"):
		return "begins with a byte order mark"
	case strings.HasPrefix(p, "/"):
		return "begins with /"
	case strings.HasSuffix(p, "/"):
		return "ends with /"
	case strings.ContainsRune(p, '\\'):
		return `holds \`
	case strings.ContainsRune(p, ':'):
		return "holds :"
	}
	for seg := range strings.SplitSeq(p, "/") {
		switch seg {
		case "":
			return "has an empty segment"
		case ".", "..":
			return "has a " + seg + " segment"
		}
	}
	return ""
}

func pathError(p, why string) error {
	return errors.New("pkgtreetest: entry path " + strconv.Quote(p) + " " + why)
}
