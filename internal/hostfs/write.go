// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package hostfs

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
)

// The modes ADR-005 D14 item 4 creates with, less the process umask.
const (
	fileMode = 0o644
	execMode = 0o755
	dirMode  = 0o755
)

// Writer is a directory opened for writing as an os.Root, with the record
// of everything this render created through it (ADR-009 D8 item 1). Every
// name a method takes is relative to the directory and "/"-separated, and
// every write is confined beneath it by the root.
//
// No link is written through or replaced: every existing component of a
// name a method writes is examined with Lstat under the root first, and a
// link, even one that points inside, is refused (ADR-009 D8 item 3). The
// root confines whatever a link planted after that check would reach, so
// the operation fails rather than escapes.
//
// A Writer creates every file and directory exclusively and removes only
// what it created (ADR-005 D14 item 5): there is no method that removes a
// name the caller chooses. It holds each entry it created open, its pin,
// until the entry is removed or committed or the Writer is closed, so a
// render holds one handle per entry it has created and not committed.
type Writer struct {
	root *os.Root
	rec  *record

	// at is this Writer's directory relative to the record's first
	// Writer, "." for that Writer itself, so the record can tell which
	// of its entries lie beneath a directory a rename moves.
	at string

	// beforeRename, when set, runs before every rename. Only this
	// package's tests set it, to put a target in place in the window
	// between the destination's check and the rename.
	beforeRename func(from, to string)
}

// record is what one render created, shared by a Writer and every Writer
// Sub opens beneath it.
type record struct {
	made  []made
	roots []*os.Root // every root opened, closed by Close
}

// made is one file or directory a render created.
//
// Its pin is what proves a name still holds it. A device and inode number
// are not an identity on their own: once a file is deleted, a file system
// may give its number to the next file created (ext4 does), and the
// render's entry and a newcomer at its name would compare the same. An
// entry held open is never freed, so while the pin is held no other entry
// can have its number, and the pin's own Stat says what that number is.
type made struct {
	w    *Writer // the Writer that created it
	name string  // its name relative to w
	dir  bool
	info fs.FileInfo // the pin's Stat just after the entry was created
	pin  *os.File    // nil once released
}

// owns reports whether now, what Lstat says the name holds, is the entry
// m created.
func (m *made) owns(now fs.FileInfo) bool {
	if m.pin == nil {
		// Released for a rename on windows, whose file ids are never
		// given to another file (pinsBlockRename).
		return os.SameFile(m.info, now)
	}
	held, err := m.pin.Stat()
	return err == nil && os.SameFile(held, now)
}

// release closes m's pin.
func (m *made) release() {
	if m.pin != nil {
		m.pin.Close()
		m.pin = nil
	}
}

// OpenWriter opens the existing directory at path, the caller's own name
// for it, for writing. path may be a symbolic link, because the caller
// named it and is trusted (ADR-005 D15 item 1, ADR-009 D7 item 4); it must
// resolve to a directory.
func OpenWriter(path string) (*Writer, error) {
	root, err := openRoot(path)
	if err != nil {
		return nil, err
	}
	return &Writer{root: root, rec: &record{roots: []*os.Root{root}}, at: "."}, nil
}

// OpenNearest opens for writing the nearest existing ancestor of path,
// the caller's own name for an output directory (ADR-005 D14 item 1.1,
// ADR-009 D8 item 1), and returns it with the "/"-separated name of path
// relative to it: "." when path itself exists. Each ancestor is found by
// Stat, following links, because every component of path is the caller's.
func OpenNearest(path string) (*Writer, string, error) {
	p := filepath.Clean(path)
	var rest []string
	for {
		info, err := os.Stat(p)
		if err == nil {
			if !info.IsDir() {
				return nil, "", fail("open", NotDirectory)
			}
			break
		}
		// ENOTDIR is a file above p, which the next Stat up finds.
		if !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, syscall.ENOTDIR) {
			return nil, "", mapErr("open", err)
		}
		parent := filepath.Dir(p)
		if parent == p {
			return nil, "", fail("open", NotFound)
		}
		rest = append(rest, filepath.Base(p))
		p = parent
	}
	w, err := OpenWriter(p)
	if err != nil {
		return nil, "", err
	}
	if len(rest) == 0 {
		return w, ".", nil
	}
	slices.Reverse(rest)
	name := strings.Join(rest, "/")
	if _, ok := osName(name); !ok {
		w.Close()
		return nil, "", fail("open", BadName)
	}
	return w, name, nil
}

// Close releases the directory, every directory Sub opened beneath it and
// every pin. It removes nothing: a failed render calls RemoveCreated
// first, and nothing is the render's to remove after Close.
func (w *Writer) Close() error {
	for i := range w.rec.made {
		w.rec.made[i].release()
	}
	w.rec.made = nil
	var errs []error
	for _, r := range slices.Backward(w.rec.roots) {
		errs = append(errs, r.Close())
	}
	w.rec.roots = nil
	return mapErr("close", errors.Join(errs...))
}

// Lstat describes name without following it and without opening it.
// ADR-005 D15's destination checks are made from it: an existing file's
// content is never read (ADR-009 D8 item 4).
func (w *Writer) Lstat(name string) (Entry, error) {
	info, err := lstat(w.root, name)
	if err != nil {
		return Entry{}, err
	}
	return entryOf(info), nil
}

// List returns the entries of the directory name ("." for w itself) as
// Dir.List does. It reads names and Lstat modes only, never a file.
func (w *Writer) List(name string, max int) ([]Entry, error) {
	d := Dir{root: w.root}
	return d.List(name, max)
}

// checkParents examines every component of name but the last with Lstat
// under root: each that exists must be a directory and not a link. It
// returns the number of components that exist.
func checkParents(root *os.Root, op, name string) (int, error) {
	segs := strings.Split(name, "/")
	for i := range len(segs) - 1 {
		info, err := root.Lstat(filepath.FromSlash(strings.Join(segs[:i+1], "/")))
		if errors.Is(err, fs.ErrNotExist) {
			return i, nil
		}
		if err != nil {
			return i, mapErr(op, err)
		}
		switch kindOf(info.Mode()) {
		case KindDir:
		case KindLink:
			return i, fail(op, Link)
		default:
			return i, fail(op, NotDirectory)
		}
	}
	return len(segs) - 1, nil
}

// Mkdir creates the directory name, exclusively: it is refused with Exists
// if anything has that name. Its parent must exist.
func (w *Writer) Mkdir(name string) error {
	n, ok := osName(name)
	if !ok || name == "." {
		return fail("mkdir", BadName)
	}
	if _, err := checkParents(w.root, "mkdir", name); err != nil {
		return err
	}
	return w.mkdir(name, n)
}

func (w *Writer) mkdir(name, n string) error {
	if err := w.root.Mkdir(n, dirMode); err != nil {
		return mapErr("mkdir", err)
	}
	pin, err := w.root.Open(n)
	if err != nil {
		// The directory is this render's own, just made and empty, as
		// Create's file is when its write fails.
		w.root.Remove(n)
		return mapErr("mkdir", err)
	}
	info, err := pin.Stat()
	var now fs.FileInfo
	if err == nil {
		now, err = w.root.Lstat(n)
	}
	if err != nil {
		pin.Close()
		return mapErr("mkdir", err)
	}
	if !os.SameFile(info, now) {
		// The name was replaced before it was opened: what was opened
		// is not the render's, and nothing is removed.
		pin.Close()
		return fail("mkdir", Changed)
	}
	w.rec.made = append(w.rec.made, made{w: w, name: name, dir: true, info: info, pin: pin})
	return nil
}

// MkdirAll creates every missing directory of name, recording each one it
// created, and refuses a component that exists and is a link or is not a
// directory.
func (w *Writer) MkdirAll(name string) error {
	if name == "." {
		return nil
	}
	if _, ok := osName(name); !ok {
		return fail("mkdir", BadName)
	}
	segs := strings.Split(name, "/")
	for i := range segs {
		sub := strings.Join(segs[:i+1], "/")
		n := filepath.FromSlash(sub)
		info, err := w.root.Lstat(n)
		if err == nil {
			switch kindOf(info.Mode()) {
			case KindDir:
				continue
			case KindLink:
				return fail("mkdir", Link)
			}
			return fail("mkdir", NotDirectory)
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return mapErr("mkdir", err)
		}
		if err := w.mkdir(sub, n); err != nil {
			return err
		}
	}
	return nil
}

// Create writes the bytes r yields to a new file name, created
// exclusively: it is refused with Exists if anything has that name, a link
// included. Its parent must exist. The file is created 0644, or 0755 when
// executable, less the process umask (ADR-005 D14 item 4). A file that
// fails part way through is removed before Create returns.
func (w *Writer) Create(name string, executable bool, r io.Reader) error {
	n, ok := osName(name)
	if !ok || name == "." {
		return fail("create", BadName)
	}
	if _, err := checkParents(w.root, "create", name); err != nil {
		return err
	}
	mode := fs.FileMode(fileMode)
	if executable {
		mode = execMode
	}
	// O_EXCL refuses an existing final component without following it,
	// a link included.
	f, err := w.root.OpenFile(n, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return mapErr("create", err)
	}
	// The handle that created the file is its pin.
	_, err = io.Copy(f, r)
	var info fs.FileInfo
	if err == nil {
		info, err = f.Stat()
	}
	if err != nil {
		// The file is this render's own, just made and still held.
		w.root.Remove(n)
		f.Close()
		return mapErr("create", err)
	}
	w.rec.made = append(w.rec.made, made{w: w, name: name, info: info, pin: f})
	return nil
}

// Sub opens the directory name beneath w as a Writer of its own, sharing
// w's record, so what is created through it is removed by RemoveCreated
// on any of them. name must be a directory and not a link, and every
// component above it is checked the same way.
func (w *Writer) Sub(name string) (*Writer, error) {
	if name == "." {
		return w, nil
	}
	if _, err := checkParents(w.root, "open", name); err != nil {
		return nil, err
	}
	info, err := lstat(w.root, name)
	if err != nil {
		return nil, opAs("open", err)
	}
	switch kindOf(info.Mode()) {
	case KindDir:
	case KindLink:
		return nil, fail("open", Link)
	default:
		return nil, fail("open", NotDirectory)
	}
	n, _ := osName(name)
	root, err := w.root.OpenRoot(n)
	if err != nil {
		return nil, mapErr("open", err)
	}
	got, err := root.Lstat(".")
	if err != nil || !os.SameFile(info, got) {
		root.Close()
		if err != nil {
			return nil, mapErr("open", err)
		}
		return nil, fail("open", Changed)
	}
	w.rec.roots = append(w.rec.roots, root)
	return &Writer{root: root, rec: w.rec, at: join(w.at, name), beforeRename: w.beforeRename}, nil
}

func join(dir, name string) string {
	if dir == "." {
		return name
	}
	return dir + "/" + name
}

// RenameNoReplace renames from to to, both in the same directory, with a
// rename that refuses an existing target, file or directory, with Exists
// (ADR-005 D14 item 1.4): renameat2(RENAME_NOREPLACE) on linux,
// renamex_np(RENAME_EXCL) on darwin, MoveFileW on windows. Where the file
// system does not support the flag, the target is checked with Lstat and
// renamed, accepting the race between the two, as ADR-005 D14 item 1.4
// allows.
//
// It commits a staging directory as the output directory and a new file
// from its temporary name, so what it renames stops being this render's
// to remove.
func (w *Writer) RenameNoReplace(from, to string) error {
	return w.rename(from, to, false)
}

// RenameReplace renames from over to, both in the same directory, which
// replaces the directory entry and never writes into the file it replaces
// (ADR-005 D14 item 2.2, D15 item 6). An existing target must be a
// regular file: a link, a directory or anything else is refused.
func (w *Writer) RenameReplace(from, to string) error {
	return w.rename(from, to, true)
}

func (w *Writer) rename(from, to string, replace bool) error {
	fn, ok1 := osName(from)
	tn, ok2 := osName(to)
	if !ok1 || !ok2 || from == "." || to == "." {
		return fail("rename", BadName)
	}
	dir, fromBase := split(from)
	toDir, toBase := split(to)
	if dir != toDir {
		return fail("rename", BadName)
	}
	if _, err := checkParents(w.root, "rename", from); err != nil {
		return err
	}
	info, err := w.root.Lstat(fn)
	if err != nil {
		return mapErr("rename", err)
	}
	if k := kindOf(info.Mode()); k == KindLink || k == KindOther {
		return fail("rename", NotRegularFile)
	}
	if replace {
		if t, err := w.root.Lstat(tn); err == nil {
			if r := fileRefusal(t.Mode()); r != "" {
				return fail("rename", r)
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return mapErr("rename", err)
		}
	}
	if pinsBlockRename && info.IsDir() {
		w.rec.release(w, from, false)
	}
	if w.beforeRename != nil {
		w.beforeRename(from, to)
	}
	if replace {
		err = mapErr("rename", w.root.Rename(fn, tn))
	} else {
		err = w.renameNoReplace(dir, fromBase, toBase)
	}
	if err != nil {
		return err
	}
	w.rec.release(w, from, true)
	return nil
}

// renameNoReplace renames from to to, both single names in the directory
// dir beneath w. The platform's call is made relative to a handle on dir,
// opened through the root, so neither name is resolved anywhere else.
func (w *Writer) renameNoReplace(dir, from, to string) error {
	n, _ := osName(dir)
	d, err := w.root.Open(n)
	if err != nil {
		return mapErr("rename", err)
	}
	defer d.Close()
	if st, err := d.Stat(); err != nil || !st.IsDir() {
		return fail("rename", Changed)
	}
	err = platformRenameNoReplace(d, filepath.Join(w.root.Name(), n), from, to)
	if !errors.Is(err, errNoReplaceUnsupported) {
		return mapErr("rename", err)
	}
	// The file system refused the flag: check, then rename.
	tn, _ := osName(join(dir, to))
	if _, err := w.root.Lstat(tn); err == nil {
		return fail("rename", Exists)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return mapErr("rename", err)
	}
	sn, _ := osName(join(dir, from))
	return mapErr("rename", w.root.Rename(sn, tn))
}

// errNoReplaceUnsupported is a platform rename's answer when the file
// system does not support refusing an existing target.
var errNoReplaceUnsupported = errors.New("no-replace rename unsupported")

// release closes the pins of everything recorded beneath from, a name
// relative to w. When renamed, from has been renamed by w: its own record
// and every record beneath it are dropped too, because a committed staging
// directory is the output directory, and a committed temporary file is
// its target, neither of which is the render's to remove any longer.
func (r *record) release(w *Writer, from string, renamed bool) {
	moved := join(w.at, from)
	kept := r.made[:0]
	for _, m := range r.made {
		p := join(m.w.at, m.name)
		if strings.HasPrefix(p, moved+"/") || renamed && p == moved {
			m.release()
			if renamed {
				continue
			}
		}
		kept = append(kept, m)
	}
	clear(r.made[len(kept):])
	r.made = kept
}

// RemoveCreated removes what this render created and has not committed —
// its staging directory, its temporary files, and the directories it
// created that are empty — newest first, and nothing else (ADR-005 D14
// items 1.5, 2.1 and 5). An entry whose name now holds something other
// than the entry its pin holds is left alone, even one with the device and
// inode number the render saw when it created the entry, and so is a
// created directory that is not empty. Each pin is held until its entry's
// removal is done. It returns the first error other than those, after
// trying every entry.
func (w *Writer) RemoveCreated() error {
	var first error
	for i := range slices.Backward(w.rec.made) {
		m := &w.rec.made[i]
		n, _ := osName(m.name)
		now, err := m.w.root.Lstat(n)
		if err == nil && m.owns(now) {
			if err := m.w.root.Remove(n); err != nil && first == nil && !(m.dir && notEmpty(err)) {
				first = mapErr("remove", err)
			}
		}
		m.release()
	}
	w.rec.made = nil
	return first
}

// notEmpty reports whether err is the refusal to remove a directory that
// holds an entry.
func notEmpty(err error) bool {
	return errors.Is(err, fs.ErrExist) || errors.Is(err, errNotEmpty)
}
