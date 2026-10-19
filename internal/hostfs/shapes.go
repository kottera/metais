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

	"github.com/kottera/metais/internal/fault"
)

// WriteOptions is what a write shape takes beside the caller's path.
type WriteOptions struct {
	// Faults is the test-only fault set (ADR-010 D13 item 2). The tool
	// builds it zero; a write shape consults only fault.SkipConfine.
	Faults fault.Set
}

// WithFaults returns the write options of an invocation whose fault set is
// f: the tool's own, which is always zero, passed through unchanged.
func WithFaults(f fault.Set) WriteOptions { return WriteOptions{Faults: f} }

// CheckPaths is the output path check of a write's names, beside the
// confinement each write makes on disk: check applies the portable-path
// predicate to the paths a command will write (ADR-005 D12), whose items 1
// and 2 confine a path by its text (item 8), and CheckPaths returns what
// it finds. It is the seam of fault.SkipConfine with the confinement of
// Output (ADR-010 D13 item 2): with that fault in opt the check is not run
// and returns success, so a path that climbs out of the output directory
// reaches the write, which the confinement detector must catch. The tool
// itself never holds a fault.
func CheckPaths[F any](opt WriteOptions, check func() []F) []F {
	if opt.Faults.Has(fault.SkipConfine) {
		return nil
	}
	return check()
}

// Output is one write of a directory's entries in one of three shapes,
// each the write of a command that puts a package's or a render's entries
// on the host:
//
//   - StageNew writes a directory that does not exist through a staging
//     directory beside it, and commits it with the no-replace rename
//     (ADR-005 D14 item 1, which ADR-007 D8 item 3 reuses for unpack into
//     a target that does not exist). A failure leaves nothing.
//   - WriteInPlace writes into an existing directory, in place (ADR-007 D8
//     item 3, unpack into an existing empty directory). A failure removes
//     nothing that was written and reports it.
//   - CreateNew creates a directory that does not exist, with its missing
//     parents, and writes into it in place (ADR-008 D5, extract). A failure
//     removes nothing that was written and reports it.
//
// Every name an Output takes is relative to the output directory and
// "/"-separated, and every write is confined beneath that directory by an
// os.Root on it (ADR-009 D8 item 1): a name that climbs out of it or is not
// in the host's form is refused with BadName, and a component that is a
// link, even one that points inside, with Link (ADR-009 D8 item 3). Every
// file and directory is created exclusively: an existing name, a link
// included, is refused with Exists, and nothing is ever replaced.
//
// An Output holds a bounded number of handles however many entries it
// writes: the roots of the output directory and of its nearest existing
// ancestor, the file being written, and for StageNew the pins of the
// staging directory and of each ancestor it created. So a write of as many
// entries as ADR-009 D3's limits admit is not refused by the host's
// open-file limit before it reaches them. No write depends on the process
// umask granting the owner a permission (Writer).
//
// An Output is ended by exactly one call of Commit or Abort; a write that
// fails ends with Abort.
type Output struct {
	parent *Writer // on the nearest existing ancestor, or the output directory
	w      *Writer // on the directory the entries are written into

	// staging and rest are, for StageNew, the names of the staging
	// directory and of the output directory relative to parent.
	staging, rest string

	// plan is, for a write into an existing directory (Existing.Stage),
	// the planned paths and their checks.
	plan *plan

	made  []entry  // what this write created beneath w, in order
	files []string // the files written, in order
	ended bool
}

// entry is a name an Output created beneath its directory.
type entry struct {
	name string
	dir  bool
}

// StageNew begins the write of the directory at path, the caller's own
// name for it, which must not exist (ADR-005 D14 item 1):
//
//  1. its missing ancestors are created through a root on its nearest
//     existing ancestor (OpenNearest), and recorded;
//  2. the staging directory StagingName(base) is created exclusively beside
//     it, so a killed write's leftover or a concurrent write of the same
//     directory refuses this one with Exists;
//  3. every entry is written beneath the staging directory, through a root
//     on it;
//  4. Commit renames the staging directory to path with RenameNoReplace;
//  5. a target that appeared meanwhile, or any failure before the rename,
//     ends in Abort, which removes the staging directory and the ancestors
//     created in step 1. The staging directory and the ancestors are pinned
//     and removed only while their names still hold them; beneath the
//     staging directory, the staging directory is the ownership boundary
//     (removeStaged), which leaves one gap: a foreign file swapped in at a
//     name the write created there is removed.
//
// At every instant the directory at path either does not exist or holds
// every entry written. An existing path is refused with Exists.
func StageNew(path string, opt WriteOptions) (*Output, error) {
	w, rest, err := OpenNearest(path)
	if err != nil {
		return nil, err
	}
	w.faults = opt.Faults
	if rest == "." {
		w.Close()
		return nil, fail("open", Exists)
	}
	dir, base := split(rest)
	staging := join(dir, StagingName(base))
	o := &Output{parent: w, staging: staging, rest: rest}
	err = w.MkdirAll(dir)
	if err == nil {
		err = w.Mkdir(staging)
	}
	if err == nil {
		o.w, err = w.Sub(staging)
	}
	if err != nil {
		w.RemoveCreated()
		w.Close()
		return nil, err
	}
	// The staging directory and the ancestors are pinned; what is written
	// beneath the staging directory is not.
	o.w.pins = false
	return o, nil
}

// WriteInPlace begins a write into the existing directory at path, the
// caller's own name for it, in place (ADR-007 D8 item 3). path may be a
// symbolic link to a directory, because the caller named it (OpenWriter).
//
// Whether the directory is empty enough to write into is the caller's
// rule, which it applies with List before it writes; every create is
// exclusive, so an entry that appears afterwards at a name the write
// needs refuses that create with Exists rather than being replaced. A
// failure removes nothing that was written: Written reports it.
func WriteInPlace(path string, opt WriteOptions) (*Output, error) {
	w, err := OpenWriter(path)
	if err != nil {
		return nil, err
	}
	w.pins = false
	w.faults = opt.Faults
	return &Output{parent: w, w: w}, nil
}

// beforeCreateNew, when set, runs in CreateNew after OpenNearest and the
// parents, before the directory itself is created, with its name relative
// to the nearest existing ancestor. Only this package's tests set it, to
// put a target in place in that window.
var beforeCreateNew func(rest string)

// CreateNew begins a write of the directory at path, the caller's own
// name for it, which must not exist, in place (ADR-008 D5): its missing
// parents and then the directory itself are created, the directory
// exclusively, and every entry is written into it. An existing path is
// refused with Exists, a path that appears before the directory is created
// included. A failure removes nothing that was written: Written reports
// it, and the directories created stay.
func CreateNew(path string, opt WriteOptions) (*Output, error) {
	w, rest, err := OpenNearest(path)
	if err != nil {
		return nil, err
	}
	w.pins = false
	w.faults = opt.Faults
	if rest == "." {
		w.Close()
		return nil, fail("open", Exists)
	}
	dir, _ := split(rest)
	err = w.MkdirAll(dir)
	if err == nil && beforeCreateNew != nil {
		beforeCreateNew(rest)
	}
	if err == nil {
		err = w.Mkdir(rest)
	}
	var sub *Writer
	if err == nil {
		sub, err = w.Sub(rest)
	}
	if err != nil {
		w.Close()
		return nil, err
	}
	return &Output{parent: w, w: sub}, nil
}

// List returns the entries of the directory name beneath the output
// directory ("." for the output directory itself) as Writer.List does,
// from names and Lstat modes only (ADR-009 D8 item 4).
func (o *Output) List(name string, max int) ([]Entry, error) {
	return o.w.List(name, max)
}

// MkdirAll creates every missing directory of name, and refuses a
// component that exists and is a link or is not a directory.
func (o *Output) MkdirAll(name string) error {
	if o.ended {
		return fail("mkdir", IOError)
	}
	if !o.confined() {
		return mapErr("mkdir", os.MkdirAll(o.unconfined(name), dirMode))
	}
	if name == "." {
		return nil
	}
	if _, ok := osName(name); !ok {
		return fail("mkdir", BadName)
	}
	segs := splitAll(name)
	for i := range segs {
		sub := segs[i]
		info, err := o.w.root.Lstat(filepath.FromSlash(sub))
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
		if err := o.w.Mkdir(sub); err != nil {
			return err
		}
		o.made = append(o.made, entry{name: sub, dir: true})
	}
	return nil
}

// splitAll returns each leading run of name's segments: "a", "a/b", "a/b/c".
func splitAll(name string) []string {
	var out []string
	for i := range len(name) {
		if name[i] == '/' {
			out = append(out, name[:i])
		}
	}
	return append(out, name)
}

// Create writes the bytes r yields to the new file name, as Writer.Create
// does: created exclusively, 0644 or 0755 when executable less the umask
// (ADR-005 D14 item 4), its parent existing. A file whose bytes fail part
// way is removed before Create returns, proved the write's own by the
// handle that wrote it, which is still open.
func (o *Output) Create(name string, executable bool, r io.Reader) error {
	if o.ended {
		return fail("create", IOError)
	}
	if o.plan != nil {
		// A Stage writes only its planned paths, each with every
		// directory it needs.
		if !o.plan.planned[name] {
			return fail("create", BadName)
		}
		dir, _ := split(name)
		if err := o.MkdirAll(dir); err != nil {
			return opAs("create", err)
		}
	}
	if !o.confined() {
		return o.createUnconfined(name, executable, r)
	}
	if err := o.w.Create(name, executable, r); err != nil {
		return err
	}
	o.made = append(o.made, entry{name: name})
	o.files = append(o.files, name)
	return nil
}

// Written returns the files written, in the order written. After a failed
// WriteInPlace or CreateNew it is what the failure reports (ADR-007 D8 item
// 3); after a failed StageNew none of them is left.
func (o *Output) Written() []string {
	return slices.Clone(o.files)
}

// Wrote reports whether the write created anything beneath its directory,
// a file or a directory. After a failed WriteInPlace it is whether the
// failure left anything there; after a failed StageNew or Stage nothing is
// left whatever it reports.
func (o *Output) Wrote() bool {
	return len(o.made) > 0 || len(o.files) > 0
}

// Commit ends a write that succeeded. For StageNew it renames the staging
// directory to the output directory with RenameNoReplace (ADR-005 D14 item
// 1.4); a target that exists by then, or any other failure of the rename,
// is the failure of item 1.5, and Commit ends the write as Abort does and
// returns the rename's error. For the in-place shapes it only ends the
// write. Either way the directories whose modes a grant changed get their
// own modes back, and every handle is released.
func (o *Output) Commit() error {
	if o.ended {
		return fail("rename", IOError)
	}
	if o.plan != nil {
		return o.commitExisting()
	}
	if o.rest != "" {
		if err := o.parent.RenameNoReplace(o.staging, o.rest); err != nil {
			o.Abort()
			return err
		}
	}
	o.ended = true
	return o.parent.Close()
}

// Abort ends a write that failed. For StageNew it removes what the write
// created and did not commit, newest first, and nothing else (ADR-005 D14
// items 1.5 and 5): the entries written beneath the staging directory, the
// staging directory, and the ancestors StageNew created. For the in-place
// shapes it removes nothing; Written reports what was written. Either way
// every handle is released. It returns the first error other than a
// directory left because it is not empty, after trying every entry.
func (o *Output) Abort() error {
	if o.ended {
		return nil
	}
	o.ended = true
	var errs []error
	if o.rest != "" || o.plan != nil {
		errs = append(errs, o.removeStaged(), o.parent.RemoveCreated())
	}
	errs = append(errs, o.parent.Close())
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// removeStaged removes, newest first, every entry this write created
// beneath the staging directory, through the root on it.
//
// The ownership boundary is the staging directory (ADR-005 D14 item 1.5,
// as ruled for Wave 4 by its wave orchestrator): it was created
// exclusively under its fixed name, and the root is a handle on it, so
// nothing this removes lies outside it, even after the staging directory's
// name is taken by another. Beneath it no entry is pinned, so that a write
// holds a bounded number of handles, and ownership rests on the name:
// only a name this write created is removed, only while it still holds
// the kind created, and a directory only when it is empty. An entry at a
// name the write did not create is left, and so then is the staging
// directory. The residual gap: a foreign file swapped in at a name the
// write created inside the staging directory is removed.
func (o *Output) removeStaged() error {
	var first error
	for _, e := range slices.Backward(o.made) {
		if o.plan != nil && o.plan.isCommitted(e.name) {
			continue
		}
		n, _ := osName(e.name)
		info, err := o.w.root.Lstat(n)
		if err != nil {
			continue
		}
		if k := kindOf(info.Mode()); e.dir && k != KindDir || !e.dir && k != KindFile {
			continue
		}
		if err := o.w.root.Remove(n); err != nil && first == nil && !(e.dir && notEmpty(err)) {
			first = mapErr("remove", err)
		}
	}
	o.made = nil
	return first
}

// confined is the output path check's seam (ADR-010 D13 item 2): it
// reports whether this write's names are checked and confined beneath the
// output directory, which is always, except under fault.SkipConfine, when
// the check returns success and each write goes to its name joined onto
// the output directory's host path.
func (o *Output) confined() bool {
	return !o.w.faults.Has(fault.SkipConfine)
}

// unconfined is name joined onto the host path of the directory the
// entries are written into: the write path ADR-009 D8 item 1 forbids
// building, which only fault.SkipConfine uses.
func (o *Output) unconfined(name string) string {
	return filepath.Join(o.w.root.Name(), filepath.FromSlash(name))
}

// createUnconfined is Create under fault.SkipConfine: the file is created
// at the joined host path, through whatever links and ".." segments it
// holds.
func (o *Output) createUnconfined(name string, executable bool, r io.Reader) error {
	mode := fs.FileMode(fileMode)
	if executable {
		mode = execMode
	}
	f, err := os.OpenFile(o.unconfined(name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return mapErr("create", err)
	}
	_, err = io.Copy(f, r)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return mapErr("create", err)
	}
	o.files = append(o.files, name)
	return nil
}
