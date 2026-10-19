// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package browse

import (
	"path"

	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/pkgtree"
)

// ExtractOptions is what Extract takes beside the tree, the entries and
// the destination.
type ExtractOptions struct {
	// Faults is the test-only fault set, handed to hostfs, whose seam
	// consults fault.SkipConfine (ADR-010 D13 item 2).
	Faults fault.Set
	// DryRun reports the files Extract would write, and writes nothing.
	DryRun bool

	// readBack, when set, receives the bytes of the first entry read back
	// from the file written: fault.UnpackToTemp's read.
	readBack *[]byte
}

// File is one file Extract writes: its entry path, which is its path
// beneath the destination, and its size.
type File struct {
	Path string
	Size int64
}

// MaxListed is how many of the names found in an existing destination a
// *DestinationError reports.
const MaxListed = 10

// DestinationError is Extract's refusal of a destination that exists.
// Wave 4's extract writes only where the destination does not exist yet
// (PLAN-2026-10-metais-v1 §10): replacing files in an existing directory
// is --overwrite's, which this tool does not offer.
type DestinationError struct {
	// NotDirectory is set when the destination, or a directory above it,
	// exists and is not a directory.
	NotDirectory bool
	// Entries are the first MaxListed names in an existing directory, in
	// byte order, and More whether there are others.
	Entries []string
	More    bool
}

func (e *DestinationError) Error() string { return "browse: the destination exists" }

// WriteError is a failure that stopped Extract. Reason is hostfs's;
// Written are the files written before it, in the order written, none of
// which was removed.
type WriteError struct {
	Reason  hostfs.Reason
	Written []string
}

func (e *WriteError) Error() string { return "browse: extract stopped (" + string(e.Reason) + ")" }

// Extract writes the entries of t named by entries, in their order, each
// to <to>/<entry path> (ADR-008 D5, extract), and returns the files
// written. to is the caller's own path, which must not exist: it is
// created, with its missing parents, by hostfs.CreateNew, and every file
// beneath it is created exclusively, confined beneath it (ADR-009 D8 item
// 1), with mode 0644 less nothing the umask takes from the owner. An
// existing to is a *DestinationError with nothing written. A failure
// stops Extract with a *WriteError; Extract removes nothing, so what it
// wrote is what the error lists, and nothing it did not create is ever
// touched.
//
// It is the one function of this package that reaches the host
// (layers.json): everything else reads a tree.
func Extract(t pkgtree.Tree, entries []string, to string, opt ExtractOptions) ([]File, error) {
	files := make([]File, 0, len(entries))
	for _, e := range entries {
		info, err := t.Stat(e)
		if err != nil {
			return nil, err
		}
		files = append(files, File{Path: e, Size: info.Size})
	}
	existing := func(err error) error {
		switch hostfs.ReasonOf(err) {
		case hostfs.NotDirectory:
			return &DestinationError{NotDirectory: true}
		case hostfs.Exists:
		default:
			return &WriteError{Reason: hostfs.ReasonOf(err)}
		}
		d, derr := hostfs.OpenDir(to)
		if derr != nil {
			if hostfs.ReasonOf(derr) == hostfs.NotDirectory {
				return &DestinationError{NotDirectory: true}
			}
			return &WriteError{Reason: hostfs.ReasonOf(derr)}
		}
		defer d.Close()
		de := &DestinationError{}
		found, lerr := d.List(".", int(limits.Get(limits.Entries).Default))
		if hostfs.ReasonOf(lerr) == hostfs.TooLarge {
			de.More = true
		} else if lerr != nil {
			return &WriteError{Reason: hostfs.ReasonOf(lerr)}
		}
		for _, f := range found {
			if len(de.Entries) == MaxListed {
				de.More = true
				break
			}
			de.Entries = append(de.Entries, f.Name)
		}
		return de
	}
	if opt.DryRun {
		w, rest, err := hostfs.OpenNearest(to)
		if err != nil {
			return nil, existing(err)
		}
		w.Close()
		if rest == "." {
			return nil, existing(&hostfs.Error{Op: "open", Reason: hostfs.Exists})
		}
		return files, nil
	}
	out, err := hostfs.CreateNew(to, hostfs.WriteOptions{Faults: opt.Faults})
	if err != nil {
		return nil, existing(err)
	}
	stop := func(err error) ([]File, error) {
		written := out.Written()
		out.Abort()
		return nil, &WriteError{Reason: hostfs.ReasonOf(err), Written: written}
	}
	for _, f := range files {
		if dir := path.Dir(f.Path); dir != "." {
			if err := out.MkdirAll(dir); err != nil {
				return stop(err)
			}
		}
		r, err := t.Open(f.Path)
		if err != nil {
			return stop(&hostfs.Error{Op: "read", Reason: hostfs.IOError})
		}
		err = out.Create(f.Path, false, r)
		r.Close()
		if err != nil {
			return stop(err)
		}
	}
	if err := out.Commit(); err != nil {
		return nil, &WriteError{Reason: hostfs.ReasonOf(err), Written: out.Written()}
	}
	if opt.readBack != nil && len(files) > 0 {
		d, err := hostfs.OpenDir(to)
		if err != nil {
			return nil, err
		}
		defer d.Close()
		b, err := d.ReadFile(files[0].Path, files[0].Size)
		if err != nil {
			return nil, err
		}
		*opt.readBack = b
	}
	return files, nil
}
