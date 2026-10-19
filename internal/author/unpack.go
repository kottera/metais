// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package author

import (
	"errors"
	"io/fs"
	"path"
	"strings"

	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/pkgtree"
)

// compiledPrefix is the derived directory unpack never writes (ADR-007 D8
// item 4).
const compiledPrefix = ".compiled/"

// UnpackedEntries returns the entries of tree that unpack writes, in the
// order it writes them: every entry but those beneath .compiled/, in byte
// order (ADR-007 D8 item 4).
func UnpackedEntries(tree pkgtree.Tree) []string {
	var out []string
	for _, e := range tree.List() {
		if !strings.HasPrefix(e, compiledPrefix) {
			out = append(out, e)
		}
	}
	return out
}

// Unpack writes the entries of tree, a packed package that opened, into
// the directory to, the caller's own path for it, and returns the paths
// it wrote, relative to to, in the order written (ADR-007 D8, ADR-008 D9).
// The package's digests were verified when it was opened, so nothing is
// written before they are; Unpack runs no check.
//
// Every entry but those beneath .compiled/ is written, files 0644 and
// directories 0755 before the umask, whatever the package's zip records.
//
//   - A to that does not exist is written through a staging directory
//     beside it and renamed into place (hostfs.StageNew), so a failure
//     leaves nothing: its *WriteError lists no file.
//   - A to that is an existing directory must be empty, apart from the
//     quiet list (IsQuiet), or Unpack refuses with a *NotEmptyError and
//     writes nothing. It is written in place (hostfs.WriteInPlace), and a
//     failure removes nothing: its *WriteError lists the files written.
//   - A to that is not a directory, or beneath a file, is a *WriteError of
//     op open and reason hostfs.NotDirectory, with nothing written.
//
// An entry whose bytes cannot be read from the package stops the write
// as a failed write does, with reason hostfs.IOError.
//
// faults are the invocation's, passed through unchanged to the write
// shape, whose seam consults them (ADR-010 D13 item 2); the tool's are
// always zero.
func Unpack(tree pkgtree.Tree, to string, faults fault.Set) ([]string, error) {
	opt := hostfs.WriteOptions{Faults: faults}
	staged := true
	out, err := hostfs.StageNew(to, opt)
	if hostfs.ReasonOf(err) == hostfs.Exists {
		// The target exists, or the staging directory beside it does: a
		// killed unpack's leftover, or a concurrent one's, which refuses
		// this one with exists as it is.
		staged = false
		exists := err
		out, err = hostfs.WriteInPlace(to, opt)
		switch {
		case hostfs.ReasonOf(err) == hostfs.NotFound:
			err = exists
		case err == nil:
			err = unpackEmpty(out.List)
		}
	}
	if err != nil {
		if out != nil {
			out.Abort()
		}
		var ne *NotEmptyError
		if errors.As(err, &ne) {
			return nil, err
		}
		return nil, writeError(err, nil)
	}
	for _, e := range UnpackedEntries(tree) {
		if err = unpackEntry(tree, out, e); err != nil {
			break
		}
	}
	if err == nil {
		// Commit's own failure has already ended the write as Abort
		// does.
		err = out.Commit()
	} else {
		out.Abort()
	}
	if err == nil {
		return out.Written(), nil
	}
	var written []string
	if !staged {
		written = out.Written()
	}
	return written, writeError(err, written)
}

// unpackEntry writes the entry e of tree beneath out, creating its parent
// directories first.
func unpackEntry(tree pkgtree.Tree, out *hostfs.Output, e string) error {
	if dir := path.Dir(e); dir != "." {
		if err := out.MkdirAll(dir); err != nil {
			return err
		}
	}
	if beforeCreate != nil {
		beforeCreate(e)
	}
	r, err := tree.Open(e)
	if err != nil {
		return &hostfs.Error{Op: "read", Reason: hostfs.IOError}
	}
	defer r.Close()
	return out.Create(e, false, r)
}

// UnpackDryRun is Unpack with nothing written anywhere (ADR-008 D9,
// unpack): it returns the paths Unpack would write into to, in the order
// it would write them, or the error Unpack would refuse with before its
// first write, which is a *NotEmptyError or a *WriteError listing no
// file. Like Unpack it opens no file in to: a directory there is listed
// by name and Lstat mode, and a staging leftover beside a target that
// does not exist is found by Lstat.
//
// What it cannot foresee is what only writing finds out: a file system
// that runs out of space, or a name that appears after the check.
func UnpackDryRun(tree pkgtree.Tree, to string) ([]string, error) {
	w, rest, err := hostfs.OpenNearest(to)
	if err != nil {
		return nil, writeError(err, nil)
	}
	defer w.Close()
	if rest == "." {
		// Unpack's in-place write (hostfs.WriteInPlace) into an existing
		// directory, which must be empty apart from the quiet list.
		if err := unpackEmpty(w.List); err != nil {
			var ne *NotEmptyError
			if errors.As(err, &ne) {
				return nil, err
			}
			return nil, writeError(err, nil)
		}
		return UnpackedEntries(tree), nil
	}
	// The staged write (hostfs.StageNew) of a target that does not exist,
	// which an existing staging directory beside it refuses.
	dir, base := path.Split(rest)
	staging := path.Join(dir, hostfs.StagingName(base))
	switch _, err := w.Lstat(staging); {
	case err == nil:
		return nil, writeError(&hostfs.Error{Op: "mkdir", Reason: hostfs.Exists}, nil)
	case !errors.Is(err, fs.ErrNotExist):
		return nil, writeError(err, nil)
	}
	return UnpackedEntries(tree), nil
}

// unpackEmpty refuses the directory list lists when it holds anything but
// quiet entries (ADR-007 D8 item 3, with D1 item 3's meaning of empty).
func unpackEmpty(list func(name string, max int) ([]hostfs.Entry, error)) error {
	entries, err := list(".", int(limits.Get(limits.Entries).Default))
	if hostfs.ReasonOf(err) == hostfs.TooLarge {
		return &NotEmptyError{More: true}
	}
	if err != nil {
		return err
	}
	e := &NotEmptyError{}
	for _, en := range entries {
		if IsQuiet(en) {
			continue
		}
		if len(e.Entries) == MaxListed {
			e.More = true
			break
		}
		e.Entries = append(e.Entries, en.Name)
	}
	if len(e.Entries) > 0 {
		return e
	}
	return nil
}
