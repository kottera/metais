// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package hostfs

import (
	"bufio"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
)

// FileOut is one file written whole at a path the caller named: pack's
// package file (ADR-007 D7 item 5). It is written to a temporary file in
// the target's own directory, named as ADR-005 D14 item 2.1 names one
// (TempName) and created exclusively, synced, and renamed over the target,
// so a reader of the target sees its old bytes or its new ones, whole, and
// never part of a file.
//
// The target is nothing or a regular file; anything else at its name is
// refused, and never replaced or written through. A replace renames over
// the directory entry and never writes into the file it replaces
// (ADR-005 D15 item 6).
type FileOut struct {
	w    *Writer
	dir  string      // the caller's path of the directory the file is in
	name string      // the file's name in dir
	info fs.FileInfo // what Lstat found at name, nil when nothing
	// inDir is whether path named a directory, so that the file's name
	// is the one OpenFileOut was given.
	inDir bool
}

// OpenFileOut resolves path, the caller's own name for an output file. A
// path that names a directory, following links because every component of
// it is the caller's (ADR-005 D15 item 1), is the directory the file named
// name is written in; any other path names the file itself, in its parent
// directory, which must exist and be a directory, and a path ending in a
// separator must name a directory. Nothing is created.
//
// What stands at the file's name is examined with Lstat and never opened:
// a directory is refused with IsDirectory, a link with Link and anything
// else that is not a regular file with NotRegularFile, each of op "check".
// A leftover temporary file beside it, a killed write's or a concurrent
// one's, is refused with Exists, of op "create" (ADR-005 D14 item 2.1).
func OpenFileOut(path, name string) (*FileOut, error) {
	dir, base, inDir := path, name, false
	info, err := os.Stat(path)
	switch {
	case err == nil && info.IsDir():
		inDir = true
	case strings.HasSuffix(path, "/") || strings.HasSuffix(path, string(filepath.Separator)):
		if err != nil {
			return nil, mapErr("open", err)
		}
		return nil, fail("open", NotDirectory)
	case err == nil || errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR):
		dir, base = filepath.Dir(path), filepath.Base(path)
	default:
		return nil, mapErr("open", err)
	}
	if _, ok := osName(base); !ok || strings.Contains(base, "/") || base == "." {
		return nil, fail("open", BadName)
	}
	w, err := OpenWriter(dir)
	if err != nil {
		return nil, err
	}
	o := &FileOut{w: w, dir: dir, name: base, inDir: inDir}
	if err := o.check(); err != nil {
		w.Close()
		return nil, err
	}
	return o, nil
}

// check examines the target's name and its temporary file's name.
func (o *FileOut) check() error {
	info, err := o.w.root.Lstat(o.name)
	switch {
	case err == nil:
		if r := fileRefusal(info.Mode()); r != "" {
			return fail("check", r)
		}
		o.info = info
	case !errors.Is(err, fs.ErrNotExist):
		return mapErr("check", err)
	}
	if _, err := o.w.root.Lstat(TempName(o.name)); err == nil {
		return fail("create", Exists)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return mapErr("create", err)
	}
	return nil
}

// Name is the file's name in its directory.
func (o *FileOut) Name() string { return o.name }

// InDirectory reports whether the path OpenFileOut was given named a
// directory, the file's name being the one it was given beside it.
func (o *FileOut) InDirectory() bool { return o.inDir }

// Replaces reports whether a regular file stands at the target, which the
// write replaces.
func (o *FileOut) Replaces() bool { return o.info != nil }

// Close releases the directory. It removes nothing.
func (o *FileOut) Close() error { return o.w.Close() }

// Held is the identity of some top-level entries of a source directory
// and of every directory beneath them, which Within compares a file's
// directory and the file itself with.
type Held struct {
	dirs, files []heldEntry
}

type heldEntry struct {
	top  string // the top-level entry it is or lies beneath
	info fs.FileInfo
}

// Hold examines top, top-level entries of the directory at source, the
// caller's own path, and dirs, directories beneath them relative to
// source and "/"-separated, each with Lstat through a root on source, and
// holds the identity of each that is a directory, and of each top-level
// entry that is a regular file. A name that is absent is not held. For a
// source directory that opened, dirs are its entries' directories: every
// directory beneath a layout tree holds an entry, or the open refused it
// (ADR-001 D5 item 6). An error is the source directory's, never an
// output's.
func Hold(source string, top, dirs []string) (*Held, error) {
	root, err := openRoot(source)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	h := &Held{}
	for _, n := range append(slices.Clone(top), dirs...) {
		on, ok := osName(n)
		if !ok {
			return nil, fail("stat", BadName)
		}
		first, _, _ := strings.Cut(n, "/")
		if !slices.Contains(top, first) {
			continue
		}
		info, err := root.Lstat(on)
		switch {
		case errors.Is(err, fs.ErrNotExist):
		case err != nil:
			return nil, mapErr("stat", err)
		case info.IsDir():
			h.dirs = append(h.dirs, heldEntry{first, info})
		case info.Mode().IsRegular() && n == first:
			h.files = append(h.files, heldEntry{first, info})
		}
	}
	return h, nil
}

// Within reports which top-level entry of h holds the file: one whose
// directory, or a directory beneath it, is the file's directory, or a
// regular file that is the file itself. It returns "" when none does.
// Identity decides (os.SameFile), never a path's text, so a link, a
// second name or a case variant of a directory is seen for what it is.
func (o *FileOut) Within(h *Held) (string, error) {
	for _, f := range h.files {
		if o.info != nil && os.SameFile(f.info, o.info) {
			return f.top, nil
		}
	}
	here, err := o.w.root.Stat(".")
	if err != nil {
		return "", mapErr("stat", err)
	}
	for _, d := range h.dirs {
		if os.SameFile(here, d.info) {
			return d.top, nil
		}
	}
	return "", nil
}

// Write writes the file: write is called once with the temporary file,
// buffered, which is then flushed and synced; verify, when it is not
// nil, then reads the whole file back; and the file is closed and renamed
// over the target, a replacing rename when a regular file stood there and
// a no-replace rename otherwise (ADR-005 D14 items 1.4 and 2.2). It
// returns the number of bytes written. Any failure removes the temporary
// file while it still holds what this write created, and leaves the
// target as it was. A failure of write's or verify's own, other than a
// failed write to the file, is returned as they returned it; every other
// is an *Error.
func (o *FileOut) Write(write func(w io.Writer) error, verify func(r io.ReaderAt, size int64) error) (int64, error) {
	tmp := TempName(o.name)
	f, err := o.w.root.OpenFile(tmp, os.O_RDWR|os.O_CREATE|os.O_EXCL, fileMode)
	if err != nil {
		return 0, mapErr("create", err)
	}
	info, err := f.Stat()
	if err != nil {
		// The exclusive create made the name this write's, and nothing
		// since has proved otherwise or given it away.
		f.Close()
		o.w.root.Remove(tmp)
		return 0, mapErr("create", err)
	}
	cw := &countingWriter{w: f}
	bw := bufio.NewWriterSize(cw, 64<<10)
	own := write(bw)
	if own == nil {
		err = bw.Flush()
	}
	if cw.err != nil {
		own, err = nil, cw.err
	}
	if err == nil && own == nil {
		err = f.Sync()
	}
	if err == nil && own == nil && verify != nil {
		own = verify(f, cw.n)
	}
	if err != nil || own != nil {
		o.w.removeHeld(tmp, info)
		f.Close()
		if own != nil {
			return 0, own
		}
		return 0, mapErr("create", err)
	}
	if err := f.Close(); err != nil {
		o.w.removeHeld(tmp, info)
		return 0, mapErr("create", err)
	}
	if o.info != nil {
		err = o.w.RenameReplace(tmp, o.name)
	} else {
		err = o.w.RenameNoReplace(tmp, o.name)
	}
	if err != nil {
		o.w.removeHeld(tmp, info)
		return 0, err
	}
	return cw.n, nil
}

// countingWriter writes to w, counts what it wrote and keeps the first
// error.
type countingWriter struct {
	w   io.Writer
	n   int64
	err error
}

func (c *countingWriter) Write(b []byte) (int, error) {
	n, err := c.w.Write(b)
	c.n += int64(n)
	if err != nil && c.err == nil {
		c.err = err
	}
	return n, err
}
