// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package browse

import (
	"errors"
	"path"

	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/pkgtree"
)

// ExtractOptions is what Extract takes beside the tree, the entries and
// the destination.
type ExtractOptions struct {
	// DryRun reports the files Extract would write, and writes nothing.
	DryRun bool

	// Overwrite lets Extract write into an existing directory that holds
	// entries (ADR-008 D5, extract item 3): a file at a path it writes is
	// replaced, and nothing is deleted.
	Overwrite bool

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

// DestinationError is Extract's refusal of its destination as a whole,
// with nothing written: one that is not a directory, or an existing
// directory that holds entries when Overwrite is not given (ADR-005 D15
// item 3).
type DestinationError struct {
	// NotDirectory is set when the destination, or a directory above it,
	// exists and is not a directory.
	NotDirectory bool
	// Entries are the first MaxListed names in the existing directory, in
	// byte order, and More whether there are others.
	Entries []string
	More    bool
}

func (e *DestinationError) Error() string { return "browse: the destination exists" }

// CheckError is Extract's refusal, with nothing written, of the files an
// existing directory's checks refused (ADR-005 D15 items 4 and 8, as
// Amendment A1 extends them): every check that refused its path, in path
// order.
type CheckError struct {
	Refused []Refusal
}

// Refusal is one file an existing directory's checks refused: its path,
// and Err, the hostfs error whose reason refuses it.
type Refusal struct {
	Path string
	Err  error
}

func (e *CheckError) Error() string { return "browse: the destination's checks refused a file" }

// WriteError is a failure that stopped Extract. Reason is hostfs's
// reason, as its text; Written are the files written before it, in the
// order written, none of which was removed. A write into an existing
// directory leaves nothing it wrote when it fails before its commit, so
// its Written is empty.
type WriteError struct {
	Reason  string
	Written []string
}

func (e *WriteError) Error() string { return "browse: extract stopped (" + e.Reason + ")" }

// CommitError is a write into an existing directory that failed during
// its commit, after it had moved files into place (ADR-005 A1 item 3):
// Committed are those files, in path order, and Reason is why the commit
// stopped, as hostfs's reason's text. Every other file holds what it held.
type CommitError struct {
	Reason    string
	Committed []string
}

func (e *CommitError) Error() string { return "browse: extract's commit stopped (" + e.Reason + ")" }

// Extract writes the entries of t named by entries, in their order, each
// to <to>/<entry path> (ADR-008 D5, extract), and returns the files
// written. to is the caller's own path. Every file is written confined
// beneath it (ADR-009 D8 item 1), with mode 0644 less nothing the umask
// takes from the owner.
//
//   - A to that does not exist is created, with its missing parents, by
//     hostfs.CreateNew, and every file beneath it is created exclusively. A
//     failure stops Extract with a *WriteError; Extract removes nothing, so
//     what it wrote is what the error lists.
//   - A to that is an existing directory is written under ADR-005 D14 item
//     2 as Amendment A1 amends it, through hostfs.Existing, when it holds
//     no entry at all, or when it does and opt.Overwrite is given;
//     otherwise it is a *DestinationError listing what it holds. Every
//     file is checked first, and every refusal is reported together in a
//     *CheckError, with nothing written. An existing file is never opened:
//     a file at a path Extract writes is replaced by a rename, and nothing
//     else in the directory is touched. A failure before the commit leaves
//     the directory as it was, and is a *WriteError listing no file; a
//     failure during it is a *CommitError listing what was committed.
//   - A to that is not a directory, or lies beneath a file, is a
//     *DestinationError with NotDirectory set.
//
// With opt.DryRun, Extract makes the same checks, writes nothing, and
// returns the files it would write or the refusal it would make before
// writing.
//
// faults is the test-only fault set (ADR-010 D13 item 2), handed to
// hostfs, whose seams consult it; the tool builds it zero.
//
// It is the one function of this package that reaches the host
// (layers.json): everything else reads a tree.
func Extract(t pkgtree.Tree, entries []string, to string, opt ExtractOptions, faults fault.Set) ([]File, error) {
	// Every reach of hostfs is in this function, its closures included
	// (layers.json): the rest of the package reads a tree.
	reasonOf := func(err error) string { return string(hostfs.ReasonOf(err)) }
	// destinationError is a failure to open the destination: a
	// *DestinationError when it, or a directory above it, is not a
	// directory, and otherwise a *WriteError listing no file.
	destinationError := func(err error) error {
		if hostfs.ReasonOf(err) == hostfs.NotDirectory {
			return &DestinationError{NotDirectory: true}
		}
		return &WriteError{Reason: reasonOf(err)}
	}
	// staged reports whether the directory to holds a StagingDir, by
	// Lstat.
	staged := func() bool {
		d, err := hostfs.OpenDir(to)
		if err != nil {
			return false
		}
		defer d.Close()
		_, err = d.Lstat(hostfs.StagingDir)
		return err == nil
	}
	files := make([]File, 0, len(entries))
	for _, e := range entries {
		info, err := t.Stat(e)
		if err != nil {
			return nil, err
		}
		files = append(files, File{Path: e, Size: info.Size})
	}
	// extractExisting is Extract into to, an existing directory.
	extractExisting := func() ([]File, error) {
		x, err := hostfs.OpenExisting(to, hostfs.WithFaults(faults))
		if err != nil {
			return nil, destinationError(err)
		}
		defer x.Close()
		found, err := x.List(int(limits.Get(limits.Entries).Default))
		de := &DestinationError{}
		switch {
		case hostfs.ReasonOf(err) == hostfs.TooLarge:
			de.More = true
		case err != nil:
			return nil, &WriteError{Reason: reasonOf(err)}
		}
		for _, f := range found {
			if len(de.Entries) == MaxListed {
				de.More = true
				break
			}
			de.Entries = append(de.Entries, f.Name)
		}
		if (len(de.Entries) > 0 || de.More) && !opt.Overwrite {
			return nil, de
		}
		paths := make([]string, len(files))
		for i, f := range files {
			paths[i] = f.Path
		}
		checks := x.Check(paths)
		ce := &CheckError{}
		for _, c := range checks {
			if c.Err != nil {
				ce.Refused = append(ce.Refused, Refusal{c.Path, c.Err})
			}
		}
		if len(ce.Refused) > 0 {
			return nil, ce
		}
		if opt.DryRun {
			// Stage refuses a staging directory left in to; a dry run finds it
			// by name, as Stage's exclusive create would.
			if staged() {
				return nil, &WriteError{Reason: string(hostfs.Exists)}
			}
			return files, nil
		}
		out, err := x.Stage(checks)
		if err != nil {
			return nil, &WriteError{Reason: reasonOf(err)}
		}
		for _, f := range files {
			r, err := t.Open(f.Path)
			if err == nil {
				err = out.Create(f.Path, false, r)
				r.Close()
			} else {
				err = &hostfs.Error{Op: "read", Reason: hostfs.IOError}
			}
			if err != nil {
				out.Abort()
				return nil, &WriteError{Reason: reasonOf(err)}
			}
		}
		if err := out.Commit(); err != nil {
			var ce *hostfs.CommitError
			if errors.As(err, &ce) && len(ce.Committed) > 0 {
				return nil, &CommitError{Reason: reasonOf(ce.Err), Committed: ce.Committed}
			}
			return nil, &WriteError{Reason: reasonOf(err)}
		}
		return files, nil
	}
	w, rest, err := hostfs.OpenNearest(to)
	if err != nil {
		return nil, destinationError(err)
	}
	w.Close()
	if rest == "." {
		return extractExisting()
	}
	if opt.DryRun {
		return files, nil
	}
	out, err := hostfs.CreateNew(to, hostfs.WithFaults(faults))
	if err != nil {
		return nil, destinationError(err)
	}
	stop := func(err error) ([]File, error) {
		written := out.Written()
		out.Abort()
		return nil, &WriteError{Reason: reasonOf(err), Written: written}
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
		return nil, &WriteError{Reason: reasonOf(err), Written: out.Written()}
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
