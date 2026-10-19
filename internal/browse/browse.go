// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package browse is the reading commands' work over an opened package
// (ADR-008 D5): listing, reading, outlining, searching and extracting the
// entries of a pkgtree.Tree, packed or source alike.
//
// A package is read in place (PRD R2 item 1): every entry is read through
// its Tree, into memory, and nothing is written to disk to read it. The one
// write is Extract's, the single entry point through which this package
// reaches internal/hostfs (layers.json), so listing, reading and searching
// cannot write. Content is the seam of fault.UnpackToTemp (ADR-010 D13 item
// 2), which the read-in-place detector must catch.
package browse

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/pkgtree"
)

// Options is what the reading functions take beside the tree.
type Options struct {
	// Faults is the test-only fault set (ADR-010 D13 item 2). The tool
	// builds it zero; browse consults only fault.UnpackToTemp.
	Faults fault.Set
}

// NotFoundError is a path that names no entry and no directory of the
// tree. Nearest is the nearest directory above it that exists, "." for the
// root.
type NotFoundError struct {
	Nearest string
}

func (e *NotFoundError) Error() string { return "browse: the path names no entry" }

// ErrIsDirectory is the refusal of a path that names a directory where an
// entry is wanted.
var ErrIsDirectory = errors.New("browse: the path names a directory")

// Clean returns an entry path as the command line writes it with the root
// written "." or "" and a directory's trailing / removed (ADR-008 D2 item
// 5): "" for the root, and otherwise the path to look up.
func Clean(p string) string {
	if p == "." || p == "./" {
		return ""
	}
	return strings.TrimSuffix(p, "/")
}

// isDir reports whether p, cleaned, is the root or a directory of the
// tree's entries: a proper prefix of an entry path, ending at a /.
func isDir(entries []string, p string) bool {
	if p == "" {
		return true
	}
	i, _ := slices.BinarySearch(entries, p+"/")
	return i < len(entries) && strings.HasPrefix(entries[i], p+"/")
}

// nearest is the nearest directory above p that exists, "." for the root.
func nearest(entries []string, p string) string {
	for {
		i := strings.LastIndexByte(p, '/')
		if i < 0 {
			return "."
		}
		p = p[:i]
		if isDir(entries, p) {
			return p + "/"
		}
	}
}

// Entry is one entry of a listing (ADR-008 D5, ls item 2): a file, with
// its uncompressed size, or a directory, implied by entry paths, with the
// number of files beneath it.
type Entry struct {
	// Path is the entry path; a directory's ends in /.
	Path  string
	Dir   bool
	Size  int64
	Files int
}

// List lists the tree at p (ADR-008 D5, ls): the root's or a directory's
// children, or with recursive every file and directory beneath it; a
// file's path lists that file. The listing is in byte order of the paths
// as written, a directory's / included. A path that names nothing is a
// *NotFoundError.
func List(t pkgtree.Tree, p string, recursive bool) ([]Entry, error) {
	entries := t.List()
	p = Clean(p)
	if p != "" {
		if info, err := t.Stat(p); err == nil {
			return []Entry{{Path: p, Size: info.Size}}, nil
		}
		if !isDir(entries, p) {
			return nil, &NotFoundError{Nearest: nearest(entries, p)}
		}
	}
	prefix := ""
	if p != "" {
		prefix = p + "/"
	}
	dirs := map[string]*Entry{}
	var out []*Entry
	for _, e := range entries {
		rest, ok := strings.CutPrefix(e, prefix)
		if !ok {
			continue
		}
		segs := strings.Split(rest, "/")
		// Every directory between p and the file is counted; a listing
		// that is not recursive shows only the first.
		for i := 1; i < len(segs); i++ {
			d := prefix + strings.Join(segs[:i], "/") + "/"
			de := dirs[d]
			if de == nil {
				de = &Entry{Path: d, Dir: true}
				dirs[d] = de
				if recursive || i == 1 {
					out = append(out, de)
				}
			}
			de.Files++
		}
		if recursive || len(segs) == 1 {
			info, err := t.Stat(e)
			if err != nil {
				return nil, fmt.Errorf("browse: a listed entry: %w", err)
			}
			out = append(out, &Entry{Path: e, Size: info.Size})
		}
	}
	slices.SortFunc(out, func(a, b *Entry) int { return strings.Compare(a.Path, b.Path) })
	res := make([]Entry, len(out))
	for i, e := range out {
		res[i] = *e
	}
	return res, nil
}

// Lookup resolves p, as the command line writes an entry, to the entry it
// names: ErrIsDirectory when it names a directory or the root, and a
// *NotFoundError when it names nothing.
func Lookup(t pkgtree.Tree, p string) (pkgtree.Info, error) {
	c := Clean(p)
	if c != "" {
		if info, err := t.Stat(c); err == nil {
			return info, nil
		}
	}
	entries := t.List()
	if isDir(entries, c) {
		return pkgtree.Info{}, ErrIsDirectory
	}
	return pkgtree.Info{}, &NotFoundError{Nearest: nearest(entries, c)}
}

// Content returns the bytes of the entry at path, read in place through t
// (ADR-008 D5; PRD R2 item 1).
//
// It is the seam of fault.UnpackToTemp (ADR-010 D13 item 2): under that
// fault the entry is extracted to a temporary file, in a directory named
// for it beneath the working directory, and read back from there, and
// the file is left behind. The read-in-place detector must catch it. The
// tool itself never holds a fault.
func Content(t pkgtree.Tree, path string, opt Options) ([]byte, error) {
	if opt.Faults.Has(fault.UnpackToTemp) {
		return unpackToTemp(t, path)
	}
	r, err := t.Open(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

// unpacked numbers UnpackToTemp's directories within one process.
var unpacked atomic.Int64

// unpackToTemp is Content under fault.UnpackToTemp: the entry is written
// by Extract to a new directory beneath the working directory, and read
// back from the file written.
func unpackToTemp(t pkgtree.Tree, path string) ([]byte, error) {
	dir := fmt.Sprintf(".metais-unpack-%d", unpacked.Add(1))
	var back []byte
	if _, err := Extract(t, []string{path}, dir, ExtractOptions{readBack: &back}); err != nil {
		return nil, err
	}
	return back, nil
}
