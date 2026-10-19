// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package hostfs

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"slices"
	"strings"
)

// Kind is what a directory entry is, from its Lstat mode: nothing is
// followed to find it.
type Kind uint8

const (
	KindFile  Kind = iota + 1 // a regular file
	KindDir                   // a directory
	KindLink                  // a symbolic link, junction or other reparse point
	KindOther                 // a FIFO, socket, device or anything else
)

func kindOf(m fs.FileMode) Kind {
	switch {
	case m.IsRegular():
		return KindFile
	case m.IsDir():
		return KindDir
	case m&fs.ModeSymlink != 0:
		return KindLink
	}
	return KindOther
}

// Entry is one name in a directory, as Lstat saw it.
type Entry struct {
	Name string // the name within its directory
	Kind Kind
	Size int64 // the length of a regular file, and zero for anything else
}

// Dir is a directory the caller named, opened for reading as an os.Root.
// Every name a method takes is relative to it and "/"-separated.
type Dir struct {
	root *os.Root

	// beforeOpen, when set, runs between the Lstat that admits a file and
	// the open that reads it. Only this package's tests set it, to change
	// the file in that window.
	beforeOpen func(name string)
}

// OpenDir opens the directory at path, the caller's own name for it, for
// reading. path may be a symbolic link, because the caller named it and is
// trusted (ADR-009 D7 item 4); it must resolve to a directory.
func OpenDir(path string) (*Dir, error) {
	root, err := openRoot(path)
	if err != nil {
		return nil, err
	}
	return &Dir{root: root}, nil
}

// openRoot opens path as an os.Root, refusing anything but a directory.
func openRoot(path string) (*os.Root, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, mapErr("open", err)
	}
	if !info.IsDir() {
		return nil, fail("open", NotDirectory)
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, mapErr("open", err)
	}
	return root, nil
}

// Close releases the directory.
func (d *Dir) Close() error {
	return mapErr("close", d.root.Close())
}

// Lstat describes name without following it.
func (d *Dir) Lstat(name string) (Entry, error) {
	info, err := lstat(d.root, name)
	if err != nil {
		return Entry{}, err
	}
	return entryOf(info), nil
}

// lstat describes name under root without following it. Every component
// above it must be a directory and not a link: no link is followed, even
// one whose target stays inside (ADR-009 D7 item 2), and os.Root alone
// would follow one in the middle of a name.
func lstat(root *os.Root, name string) (fs.FileInfo, error) {
	n, ok := osName(name)
	if !ok {
		return nil, fail("stat", BadName)
	}
	if name != "." {
		if exist, err := checkParents(root, "stat", name); ReasonOf(err) == NotDirectory {
			// A file above the name, as the OS's ENOTDIR says: nothing
			// has that name.
			return nil, fail("stat", NotFound)
		} else if err != nil {
			return nil, err
		} else if exist < strings.Count(name, "/") {
			return nil, fail("stat", NotFound)
		}
	}
	info, err := root.Lstat(n)
	if err != nil {
		return nil, mapErr("stat", err)
	}
	return info, nil
}

func entryOf(info fs.FileInfo) Entry {
	e := Entry{Name: info.Name(), Kind: kindOf(info.Mode())}
	if e.Kind == KindFile {
		e.Size = info.Size()
	}
	return e
}

// List returns the entries of the directory name ("." for d itself), in
// byte order of name, refusing a directory with more than max entries
// with TooLarge, having read at most one batch of names past max. name
// must be a directory, and not a link to one.
func (d *Dir) List(name string, max int) ([]Entry, error) {
	f, _, err := openDir(d.root, name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Entry
	err = readDir(f, max, func(de fs.DirEntry) error {
		info, err := de.Info()
		if err != nil {
			return mapErr("list", err)
		}
		out = append(out, entryOf(info))
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.SortFunc(out, func(a, b Entry) int {
		switch {
		case a.Name < b.Name:
			return -1
		case a.Name > b.Name:
			return 1
		}
		return 0
	})
	return out, nil
}

// openDir opens the directory name under root, refusing a link or
// anything else that is not a directory from its Lstat mode, and a handle
// that is not the directory Lstat described.
func openDir(root *os.Root, name string) (*os.File, fs.FileInfo, error) {
	info, err := lstat(root, name)
	if err != nil {
		return nil, nil, opAs("list", err)
	}
	switch kindOf(info.Mode()) {
	case KindDir:
	case KindLink:
		return nil, nil, fail("list", Link)
	default:
		return nil, nil, fail("list", NotDirectory)
	}
	n, _ := osName(name)
	f, err := root.Open(n)
	if err != nil {
		return nil, nil, mapErr("list", err)
	}
	got, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, nil, mapErr("list", err)
	}
	if !got.IsDir() || !os.SameFile(info, got) {
		f.Close()
		return nil, nil, fail("list", Changed)
	}
	return f, info, nil
}

// listBatch is how many names a directory is read at a time, so a
// directory of any size is read in bounded memory until its bound.
const listBatch = 256

// readDir calls each for every entry of the open directory f, in the order
// the host returns them, and refuses a directory of more than max entries
// with TooLarge before calling each for the entry past max.
func readDir(f *os.File, max int, each func(fs.DirEntry) error) error {
	n := 0
	for {
		batch, err := f.ReadDir(listBatch)
		for _, de := range batch {
			if n++; n > max {
				return fail("list", TooLarge)
			}
			if err := each(de); err != nil {
				return err
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return mapErr("list", err)
		}
	}
}

// ReadFile returns the bytes of the regular file name, refusing it with
// TooLarge when it holds more than limit bytes. The read stops at limit
// plus one byte, whatever the file's size says.
func (d *Dir) ReadFile(name string, limit int64) ([]byte, error) {
	f, err := d.open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readLimited(f, limit)
}

// open opens the regular file name for reading (ADR-009 D7 item 3). A
// link, FIFO, socket or device is refused from its Lstat mode without
// being opened. The open itself does not block on a FIFO put there since,
// and the handle's own Stat must still be a regular file and the same file
// Lstat described.
func (d *Dir) open(name string) (*os.File, error) {
	info, err := lstat(d.root, name)
	if err != nil {
		return nil, opAs("open", err)
	}
	if r := fileRefusal(info.Mode()); r != "" {
		return nil, fail("open", r)
	}
	if d.beforeOpen != nil {
		d.beforeOpen(name)
	}
	return openSame(d.root, name, info)
}

// fileRefusal is why a file of mode m is not opened, or "" for a regular
// file.
func fileRefusal(m fs.FileMode) Reason {
	switch kindOf(m) {
	case KindFile:
		return ""
	case KindLink:
		return Link
	case KindDir:
		return IsDirectory
	}
	return NotRegularFile
}

// openSame opens name under root for reading and returns the handle only
// if it is a regular file and the same file as info.
func openSame(root *os.Root, name string, info fs.FileInfo) (*os.File, error) {
	n, ok := osName(name)
	if !ok {
		return nil, fail("open", BadName)
	}
	f, err := root.OpenFile(n, os.O_RDONLY|openFlags, 0)
	if err != nil {
		return nil, mapErr("open", err)
	}
	got, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, mapErr("open", err)
	}
	if !got.Mode().IsRegular() || !os.SameFile(info, got) {
		f.Close()
		return nil, fail("open", Changed)
	}
	return f, nil
}

// readLimited reads r to its end, refusing with TooLarge once it has read
// limit plus one byte.
func readLimited(r io.Reader, limit int64) ([]byte, error) {
	if limit < 0 {
		limit = 0
	}
	b, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, mapErr("read", err)
	}
	if int64(len(b)) > limit {
		return nil, fail("read", TooLarge)
	}
	return b, nil
}

// opAs returns err with its operation replaced by op.
func opAs(op string, err error) error {
	var e *Error
	if errors.As(err, &e) {
		return &Error{Op: op, Reason: e.Reason}
	}
	return mapErr(op, err)
}

// File is a regular file the caller named, opened for reading in place: a
// packed package, read by offset (ADR-010 D11 item 1).
type File struct {
	f    *os.File
	size int64
}

// OpenFile opens the regular file at path, the caller's own name for it.
// path may be a symbolic link, because the caller named it and is trusted
// (ADR-009 D7 item 4); it must resolve to a regular file, which is checked
// from its Stat before it is opened, so a FIFO cannot block the tool, and
// again on the handle.
func OpenFile(path string) (*File, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, mapErr("open", err)
	}
	if r := fileRefusal(info.Mode()); r != "" {
		return nil, fail("open", r)
	}
	f, err := os.OpenFile(path, os.O_RDONLY|openFlags, 0)
	if err != nil {
		return nil, mapErr("open", err)
	}
	got, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, mapErr("open", err)
	}
	if !got.Mode().IsRegular() || !os.SameFile(info, got) {
		f.Close()
		return nil, fail("open", Changed)
	}
	return &File{f: f, size: got.Size()}, nil
}

// Size is the file's length when it was opened.
func (f *File) Size() int64 {
	return f.size
}

// ReadAt reads len(p) bytes from offset off. Reading past the size the
// file had when it was opened returns io.EOF, whatever has been appended
// since.
func (f *File) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, fail("read", IOError)
	}
	if off >= f.size {
		return 0, io.EOF
	}
	short := false
	if rest := f.size - off; int64(len(p)) > rest {
		p, short = p[:rest], true
	}
	n, err := f.f.ReadAt(p, off)
	if err == io.EOF {
		// The file is shorter than it was when it was opened.
		return n, fail("read", Changed)
	}
	if err != nil {
		return n, mapErr("read", err)
	}
	if short {
		return n, io.EOF
	}
	return n, nil
}

// Close releases the file.
func (f *File) Close() error {
	return mapErr("close", f.f.Close())
}
