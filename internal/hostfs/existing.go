// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package hostfs

import (
	"errors"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/unitext"
)

// StagingDir is the one staging directory a write into an existing output
// directory creates inside it (ADR-005 Amendment A1 items 1 and 5). The
// name is fixed: it reads no random source and no process identity, so a
// killed write's leftover, or a concurrent write into the same directory,
// is found by the exclusive create that makes it.
const StagingDir = ".metais-staging"

// Status is what a write into an existing directory does to one planned
// path (ADR-005 D15 item 4, D16 item 2).
type Status string

const (
	// Create is a planned path whose target is absent.
	Create Status = "create"

	// Replace is a planned path whose target is a regular file, which the
	// write replaces by renaming over it.
	Replace Status = "replace"
)

// PathCheck is ADR-005 D15 item 4's checks of one planned path, as
// Amendment A1 items 4 and 5 extend them: the path's Status, or Err, the
// *Error whose reason refuses it. A check is made from names and Lstat
// only, and writes nothing anywhere, so a dry run reports its Status (D16
// items 1 and 2).
type PathCheck struct {
	Path   string
	Status Status
	Err    error

	// unit is the path's commit unit (A1 item 2): its shallowest component
	// that did not exist at the check, or the path itself when its parent
	// exists. unitDir is whether the unit is a missing directory.
	unit    string
	unitDir bool
}

// Existing is an existing output directory a write goes into under ADR-005
// D14 item 2, as Amendment A1 amends it: render's, and extract's and
// sample's with their --overwrite. It is opened by OpenExisting, its
// planned paths are checked by Check, and Stage begins the write, which is
// an Output: Create prepares every planned file beneath one staging
// directory inside it, StagingDir, and Commit commits them by unit.
//
// One ownership rule governs what such a write removes (A1 item 6): only
// what lies within the staging directory, which it created exclusively and
// holds open, and beneath it only a name it created, while that name still
// holds the kind it created, a directory only when it is empty. Nothing
// outside the staging directory is created except by a commit rename, and
// nothing outside it is removed. The one declared gap is ruling R-W4-0's: a
// foreign file swapped in at a name the write created beneath its staging
// directory is removed.
//
// An existing file in the output directory is never opened: the checks use
// names and Lstat only (ADR-009 D8 item 4, ADR-005 D15 item 5), and a
// replace renames over the file's directory entry, never writing into it,
// so a target with another hard link keeps it with the old bytes (D15 item
// 6).
//
// The write holds a bounded number of handles however many files it plans
// (A1 item 7): the root on the output directory, the staging directory's
// pin and root, the file being written, and while checking or committing
// one directory at a time.
type Existing struct {
	w      *Writer // on the output directory; it pins the staging directory
	faults fault.Set

	// device is the device of a file from its Lstat, false when the host
	// does not say. Only this package's tests replace it, to put a
	// directory on another device without a privileged mount.
	device func(fs.FileInfo) (uint64, bool)

	staged bool // Stage handed the directory to an Output, which ends it
	closed bool
}

// OpenExisting opens the existing directory at path, the caller's own name
// for it, for a write under ADR-005 D14 item 2 and Amendment A1. path may be
// a symbolic link to a directory, because the caller named it (D15 item 1);
// a path that does not exist is refused with NotFound, and one that is not
// a directory with NotDirectory. The caller decides, from List, whether the
// directory is empty enough to write into (D15 item 3); this package does
// not.
func OpenExisting(path string, opt WriteOptions) (*Existing, error) {
	w, err := OpenWriter(path)
	if err != nil {
		return nil, err
	}
	w.faults = opt.Faults
	return &Existing{w: w, faults: opt.Faults, device: platformDevice}, nil
}

// List returns the entries of the output directory, as Writer.List does,
// from names and Lstat modes only.
func (x *Existing) List(max int) ([]Entry, error) {
	return x.w.List(".", max)
}

// Lstat describes the entry name of the output directory, as Writer.Lstat
// does, without following it and without opening it: a caller finds a
// leftover StagingDir by it before anything is written.
func (x *Existing) Lstat(name string) (Entry, error) {
	return x.w.Lstat(name)
}

// Close releases the output directory. After Stage it does nothing: the
// Output it returned ends the write.
func (x *Existing) Close() error {
	if x.staged || x.closed {
		return nil
	}
	x.closed = true
	return x.w.Close()
}

// Check makes ADR-005 D15 item 4's checks of each planned path, as
// Amendment A1 items 4 and 5 extend them, walking each segment beneath the
// output directory with Lstat through the root on it, and returns one
// PathCheck per path, in the order given, so a caller reports every
// refusal with the others (D15 item 8). A path is refused:
//
//   - with BadName, when it is not a relative "/"-separated name;
//   - with Exists, when its first segment is an ASCII-case-insensitive
//     match for StagingDir (A1 item 5);
//   - with Link, when a segment is a symbolic link, a junction or any other
//     reparse point, even one that points inside;
//   - with NotDirectory, when a segment that must be a directory is not;
//   - with CaseVariant, when the directory of a segment holds an entry that
//     is a canonical caseless match for it but not byte-equal to it;
//   - with IsDirectory or NotRegularFile, when the target is a directory
//     or another file that is not regular (a FIFO, socket or device);
//   - with IOError, when the deepest existing directory of the path is on
//     another device from the output directory (A1 item 4); the device is
//     compared and never reported (ADR-009 D12);
//   - with the reason of an error that stopped a check.
//
// Otherwise its Status is Create when the target is absent and Replace
// when it is a regular file. Check writes nothing and opens no file:
// directories are listed by name, for the caseless match, one at a time.
// Commit makes every check again.
func (x *Existing) Check(paths []string) []PathCheck {
	c := checker{x: x, seen: map[string]lstatResult{}, want: map[string]map[string][]segment{}}
	if info, err := c.lstat("."); err == nil {
		c.dev, c.hasDev = x.device(info)
	}
	out := make([]PathCheck, len(paths))
	for i, p := range paths {
		out[i] = c.walk(i, p)
	}
	c.caseless(out)
	if x.faults.Has(fault.ReadExisting) {
		c.readReplaced(out)
	}
	return out
}

// segment is one segment of a planned path, waiting for the caseless match
// against the names of its directory: the index of the path, and the
// segment.
type segment struct {
	i   int
	seg string
}

type lstatResult struct {
	info fs.FileInfo
	err  error
}

// checker is one run of Check.
type checker struct {
	x *Existing

	dev    uint64 // the output directory's device, when hasDev
	hasDev bool

	// seen is every name Lstat has described, so a directory shared by
	// many planned paths is examined once.
	seen map[string]lstatResult

	// want is, for each existing directory, the planned segments beneath
	// it, by caseless key.
	want map[string]map[string][]segment
}

func (c *checker) lstat(name string) (fs.FileInfo, error) {
	if r, ok := c.seen[name]; ok {
		return r.info, r.err
	}
	n, _ := osName(name)
	info, err := c.x.w.root.Lstat(n)
	c.seen[name] = lstatResult{info, err}
	return info, err
}

// walk checks path i, p, segment by segment. Every segment above the one
// examined has been found a directory and not a link, so the root follows
// no link to reach it.
func (c *checker) walk(i int, p string) PathCheck {
	pc := PathCheck{Path: p}
	if _, ok := osName(p); !ok || p == "." {
		if c.x.faults.Has(fault.SkipConfine) {
			// The output path check's seam (CheckPaths): under the
			// fault, a name that climbs out reaches the write.
			pc.Status, pc.unit = Create, p
			return pc
		}
		pc.Err = fail("check", BadName)
		return pc
	}
	segs := strings.Split(p, "/")
	if asciiEqualFold(segs[0], StagingDir) {
		pc.Err = fail("check", Exists)
		return pc
	}
	parent := "."
	for k, seg := range segs {
		name := join(parent, seg)
		c.wantCase(parent, seg, i)
		info, err := c.lstat(name)
		if errors.Is(err, fs.ErrNotExist) {
			pc.Status, pc.unit, pc.unitDir = Create, name, k < len(segs)-1
			pc.Err = c.sameDevice(parent)
			return pc
		}
		if err != nil {
			pc.Err = mapErr("check", err)
			return pc
		}
		if linkLike(info) {
			pc.Err = fail("check", Link)
			return pc
		}
		if k < len(segs)-1 {
			if !info.IsDir() {
				pc.Err = fail("check", NotDirectory)
				return pc
			}
			parent = name
			continue
		}
		if r := fileRefusal(info.Mode()); r != "" {
			pc.Err = fail("check", r)
			return pc
		}
		pc.Status, pc.unit = Replace, p
		pc.Err = c.sameDevice(parent)
	}
	return pc
}

// linkLike reports whether info is a link where a file or directory is
// wanted: a symbolic link, or on windows a junction or any other reparse
// point, which Lstat may report as irregular rather than as a link.
func linkLike(info fs.FileInfo) bool {
	return kindOf(info.Mode()) == KindLink || reparse(info)
}

// sameDevice refuses with IOError a planned path whose deepest existing
// directory, dir, is on another device from the output directory, so that
// every commit rename stays on one file system (ADR-005 A1 item 4).
func (c *checker) sameDevice(dir string) error {
	if !c.hasDev {
		return nil
	}
	info, err := c.lstat(dir)
	if err != nil {
		return mapErr("check", err)
	}
	if d, ok := c.x.device(info); ok && d != c.dev {
		return fail("check", IOError)
	}
	return nil
}

func (c *checker) wantCase(dir, seg string, i int) {
	keys := c.want[dir]
	if keys == nil {
		keys = map[string][]segment{}
		c.want[dir] = keys
	}
	k := caselessKey(seg)
	keys[k] = append(keys[k], segment{i, seg})
}

// caseless refuses with CaseVariant each path a segment of which has, in
// its directory, an entry that is a canonical caseless match for it and
// not byte-equal to it (ADR-005 D15 item 4). Each directory is listed
// once, by name, and the listing is not held: its names are matched
// against the planned segments as they are read.
func (c *checker) caseless(out []PathCheck) {
	dirs := make([]string, 0, len(c.want))
	for d := range c.want {
		dirs = append(dirs, d)
	}
	slices.Sort(dirs)
	for _, d := range dirs {
		keys := c.want[d]
		err := c.eachName(d, func(name string) {
			if d == "." && name == StagingDir {
				// The write's own, when the checks are made again at
				// commit; a leftover refuses Stage, and a first segment
				// that matches it is refused by walk.
				return
			}
			for _, s := range keys[caselessKey(name)] {
				if name != s.seg && out[s.i].Err == nil {
					out[s.i].Err = fail("check", CaseVariant)
				}
			}
		})
		if err == nil {
			continue
		}
		for _, segs := range keys {
			for _, s := range segs {
				if out[s.i].Err == nil {
					out[s.i].Err = opAs("check", err)
				}
			}
		}
	}
}

// eachName calls each with the name of every entry of the directory dir.
func (c *checker) eachName(dir string, each func(string)) error {
	f, _, err := openDir(c.x.w.root, dir)
	if err != nil {
		return err
	}
	defer f.Close()
	return readDir(f, math.MaxInt, func(de fs.DirEntry) error {
		each(de.Name())
		return nil
	})
}

// readReplaced is fault.ReadExisting's seam: it reads the content of every
// file a check found to replace, which ADR-009 D8 item 4 forbids and the
// existing-destination detector must catch. The tool never holds the
// fault.
func (c *checker) readReplaced(out []PathCheck) {
	for _, pc := range out {
		if pc.Err != nil || pc.Status != Replace {
			continue
		}
		info, err := c.lstat(pc.Path)
		if err != nil {
			continue
		}
		if f, err := openSame(c.x.w.root, pc.Path, info); err == nil {
			io.Copy(io.Discard, f)
			f.Close()
		}
	}
}

// caselessKey is s's canonical caseless key (unitext.CaselessKey), or s
// itself when it is not valid UTF-8, which matches only itself.
func caselessKey(s string) string {
	if !utf8.ValidString(s) {
		return s
	}
	return unitext.CaselessKey(s)
}

// asciiEqualFold reports whether a and b are equal under ASCII case
// folding alone.
func asciiEqualFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range len(a) {
		if lowerASCII(a[i]) != lowerASCII(b[i]) {
			return false
		}
	}
	return true
}

func lowerASCII(b byte) byte {
	if 'A' <= b && b <= 'Z' {
		return b + 'a' - 'A'
	}
	return b
}

// plan is what Stage was given: the planned paths, their checks in path
// order.
type plan struct {
	x       *Existing
	checks  []PathCheck
	planned map[string]bool

	// committed are the units renamed into place, which removeStaged no
	// longer reaches: they are not beneath the staging directory.
	committed map[string]bool
}

// isCommitted reports whether name, beneath the staging directory, lies
// within a unit already committed.
func (p *plan) isCommitted(name string) bool {
	for {
		if p.committed[name] {
			return true
		}
		i := strings.LastIndexByte(name, '/')
		if i < 0 {
			return false
		}
		name = name[:i]
	}
}

// Stage begins the write of the planned paths checks names, every one of
// which Check admitted (ADR-005 A1 item 1): it creates StagingDir inside
// the output directory, exclusively, and holds it open until the write
// ends. An existing one, a killed write's leftover or a concurrent write's,
// is refused with Exists, and nothing is written. A check that refused its
// path is refused with that check's reason.
//
// The Output it returns writes only the planned paths, each beneath the
// staging directory at its own path, with every directory that path needs,
// through a root on the staging directory: nothing is created in the
// existing tree until Commit. Abort removes the staging directory and
// changes nothing else.
func (x *Existing) Stage(checks []PathCheck) (*Output, error) {
	if x.staged || x.closed {
		return nil, fail("mkdir", IOError)
	}
	for _, c := range checks {
		if c.Err != nil {
			return nil, opAs("mkdir", c.Err)
		}
		if c.unit == "" {
			return nil, fail("mkdir", BadName)
		}
	}
	if err := x.w.Mkdir(StagingDir); err != nil {
		return nil, err
	}
	sw, err := x.w.Sub(StagingDir)
	if err != nil {
		x.w.RemoveCreated()
		return nil, err
	}
	// The staging directory is pinned; what is written beneath it is not.
	sw.pins = false
	p := &plan{x: x, checks: slices.Clone(checks), planned: map[string]bool{}, committed: map[string]bool{}}
	slices.SortStableFunc(p.checks, func(a, b PathCheck) int { return strings.Compare(a.Path, b.Path) })
	for _, c := range p.checks {
		p.planned[c.Path] = true
	}
	x.staged = true
	return &Output{parent: x.w, w: sw, plan: p}, nil
}

// CommitError is the failure of a write into an existing directory during
// its commit (ADR-005 D14 item 2.3, A1 item 3): Committed are the planned
// paths already committed, in path order, each one whose unit was renamed
// into place; Err is why the commit stopped. When Committed is empty,
// nothing in the output directory changed.
type CommitError struct {
	Committed []string
	Err       error
}

func (e *CommitError) Error() string { return e.Err.Error() }
func (e *CommitError) Unwrap() error { return e.Err }

// commitExisting is Commit for a Stage: it commits the planned paths by
// unit (ADR-005 A1 item 2). Before the first rename every planned path must
// have been written, and every check is made again, through the root, and
// must still give each path its status and unit; then each unit is renamed
// from the staging directory into place once, in path order, a missing
// directory and a Create with the no-replace rename, a Replace with the
// replacing rename. Each rename checks its target's directories again, so
// a link planted since fails the commit rather than being followed out.
//
// A failure stops the commit and ends the write as Abort does, returning a
// *CommitError that lists what was committed (A1 item 3). After the last
// unit the staging directory holds only directories the write created
// there, and they and it are removed.
func (o *Output) commitExisting() error {
	p := o.plan
	written := map[string]bool{}
	for _, f := range o.files {
		written[f] = true
	}
	paths := make([]string, len(p.checks))
	for i, c := range p.checks {
		if !written[c.Path] {
			return o.failCommit(nil, fail("rename", NotFound))
		}
		paths[i] = c.Path
	}
	for i, c := range p.x.Check(paths) {
		if c.Err != nil {
			return o.failCommit(nil, opAs("rename", c.Err))
		}
		if was := p.checks[i]; c.Status != was.Status || c.unit != was.unit || c.unitDir != was.unitDir {
			return o.failCommit(nil, fail("rename", Changed))
		}
	}
	var units []PathCheck
	beneath := map[string][]string{}
	for _, c := range p.checks {
		if _, ok := beneath[c.unit]; !ok {
			units = append(units, c)
		}
		beneath[c.unit] = append(beneath[c.unit], c.Path)
	}
	var committed []string
	var cd *commitDir
	defer func() { cd.close() }()
	for _, u := range units {
		if err := o.commitUnit(u, &cd); err != nil {
			slices.Sort(committed)
			return o.failCommit(committed, err)
		}
		committed = append(committed, beneath[u.unit]...)
		o.forget(u.unit)
		if p.x.faults.Has(fault.RemoveForeign) {
			o.removeForeign(u.unit)
		}
	}
	cd.close()
	slices.Sort(committed)
	o.ended = true
	errs := []error{o.removeStaged(), o.parent.RemoveCreated()}
	if _, err := o.parent.root.Lstat(StagingDir); err == nil {
		// Something the write did not create is beneath the staging
		// directory, which is therefore left, and which the next write
		// into this directory refuses.
		errs = append(errs, fail("remove", Exists))
	}
	errs = append(errs, o.parent.Close())
	for _, err := range errs {
		if err != nil {
			return &CommitError{Committed: committed, Err: err}
		}
	}
	return nil
}

// commitDir is the directory a run of units in path order is committed
// into: a handle on it in the output directory, opened once every segment
// of its name was found a directory and not a link, and proved by its own
// Stat to be the directory Lstat found, and a handle on its counterpart
// beneath the staging directory. Every rename of the run is made relative
// to the two handles, so a link planted at the directory's name since it
// was proved is never followed: the rename reaches the directory proved,
// or fails. A commit holds one commitDir at a time.
type commitDir struct {
	name             string
	to, from         *os.File
	toPath, fromPath string
}

func (d *commitDir) close() {
	if d != nil {
		d.to.Close()
		d.from.Close()
	}
}

// openCommitDir makes the checks of ADR-005 D15 item 4 again, at write
// time, on every segment of dir, a directory in the output directory, and
// opens it and its staged counterpart as a commitDir.
func (o *Output) openCommitDir(dir string) (*commitDir, error) {
	x := o.parent
	info, err := x.root.Lstat(".")
	if dir != "." {
		segs := strings.Split(dir, "/")
		for k := range segs {
			n, _ := osName(strings.Join(segs[:k+1], "/"))
			if info, err = x.root.Lstat(n); err != nil {
				break
			}
			if linkLike(info) {
				return nil, fail("rename", Link)
			}
			if !info.IsDir() {
				return nil, fail("rename", NotDirectory)
			}
		}
	}
	if err != nil {
		return nil, mapErr("rename", err)
	}
	n, _ := osName(dir)
	to, err := x.root.Open(n)
	if err != nil {
		return nil, mapErr("rename", err)
	}
	if got, err := to.Stat(); err != nil || !os.SameFile(info, got) {
		to.Close()
		return nil, fail("rename", Changed)
	}
	staged := StagingDir
	if dir != "." {
		staged = join(StagingDir, dir)
	}
	from, fromPath, err := x.openDirAt(staged)
	if err != nil {
		to.Close()
		return nil, err
	}
	return &commitDir{name: dir, to: to, toPath: filepath.Join(x.root.Name(), n), from: from, fromPath: fromPath}, nil
}

// commitUnit renames u's unit from the staging directory into place,
// through *cd, which it opens, or moves on to the unit's directory.
func (o *Output) commitUnit(u PathCheck, cd **commitDir) error {
	x := o.parent
	dir, base := split(u.unit)
	if x.beforeRename != nil {
		x.beforeRename(join(StagingDir, u.unit), u.unit)
	}
	if *cd == nil || (*cd).name != dir {
		(*cd).close()
		*cd = nil
		d, err := o.openCommitDir(dir)
		if err != nil {
			return err
		}
		*cd = d
	}
	d := *cd
	staged := StagingDir
	if dir != "." {
		staged = join(StagingDir, dir)
	}
	if u.unitDir || u.Status == Create {
		return x.renameNoReplaceAt(d.from, d.fromPath, staged, base, d.to, d.toPath, dir, base)
	}
	tn, _ := osName(u.unit)
	if t, err := x.root.Lstat(tn); err == nil {
		if linkLike(t) {
			return fail("rename", Link)
		}
		if r := fileRefusal(t.Mode()); r != "" {
			return fail("rename", r)
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return mapErr("rename", err)
	}
	err := platformRenameReplace(d.from, d.fromPath, base, d.to, d.toPath, base)
	if errors.Is(err, errHandleUnsupported) {
		fn, _ := osName(join(staged, base))
		err = x.root.Rename(fn, tn)
	}
	return mapErr("rename", err)
}

// forget marks unit committed: what its rename moved out of the staging
// directory is no longer the write's to remove (removeStaged). A
// directory whose mode a grant changed is now reached through the output
// directory's root, which gives it its own mode back when the write ends.
func (o *Output) forget(unit string) {
	under := func(name string) bool { return name == unit || strings.HasPrefix(name, unit+"/") }
	o.plan.committed[unit] = true
	for i, g := range o.parent.rec.grants {
		if g.w == o.w && under(g.name) {
			o.parent.rec.grants[i].w = o.parent
		}
	}
}

// removeForeign is fault.RemoveForeign's seam: it removes every regular
// file beside unit, in the output directory, that the write did not plan,
// which ADR-005 D15 item 7 forbids and the existing-destination detector
// must catch. The tool never holds the fault.
func (o *Output) removeForeign(unit string) {
	dir, _ := split(unit)
	entries, err := o.parent.List(dir, math.MaxInt)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := join(dir, e.Name)
		if e.Kind == KindFile && !o.plan.planned[name] {
			n, _ := osName(name)
			o.parent.root.Remove(n)
		}
	}
}

// failCommit ends a write whose commit failed, as Abort does, and returns
// the *CommitError that reports it.
func (o *Output) failCommit(committed []string, err error) error {
	o.Abort()
	return &CommitError{Committed: committed, Err: err}
}
