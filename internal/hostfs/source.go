// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package hostfs

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/pkgtree"
)

// The layout's roots: the top-level names of ADR-001 D5 item 1 a source
// directory's package is read from. Nothing outside them is read.
const (
	manifestName = "metais.yaml"
	compiledName = ".compiled"
)

var layoutDirs = []string{"assets", "examples", "inputs", "references", "templates"}

// SourceOptions configures OpenSource.
type SourceOptions struct {
	// Limits are the package limits the walk enforces: entries,
	// entry-size, package-uncompressed, path-length and path-depth, as a
	// packed package's reader does, and findings, which bounds how many
	// problems the walk collects before it stops. It must be a Set made
	// by limits.Defaults.
	Limits limits.Set

	// Names is ADR-001 D2's entry-name rule, applied to every entry path
	// the tree would list, all at once and in byte order, so that it can
	// refuse a collision between two of them. A non-nil answer refuses
	// the source directory and is returned inside the *SourceError. It is
	// required: a tree over untrusted names exists only for names that
	// passed it (pkgtree.Tree).
	//
	// The walk refuses on its own only what the pkgtree.Tree contract
	// needs and no file system name can be: invalid UTF-8, a "\" or ":",
	// and a leading byte order mark. Everything else ADR-001 D2 says —
	// NFC, the portable characters, case collisions, reserved names,
	// ".git" — is Names' to say.
	Names func(paths []string) error
}

// ProblemKind is why a source directory is refused. Each names the
// finding its caller reports.
type ProblemKind uint8

const (
	// ProblemNonRegularFile is a symbolic link, FIFO, socket, device or
	// other non-regular file in the layout's roots, which is
	// package.layout.non_regular_file (ADR-001 D5 item 6, ADR-009 D7 item
	// 8). It is never followed or opened.
	ProblemNonRegularFile ProblemKind = iota + 1

	// ProblemEmptyDirectory is a directory in the layout's roots that
	// holds no entry, which is package.layout.empty_directory (ADR-001 D2
	// item 6): the packed form cannot carry it.
	ProblemEmptyDirectory

	// ProblemCompiledInSource is a top-level .compiled, which is
	// package.layout.compiled_in_source (ADR-001 D5 item 5).
	ProblemCompiledInSource

	// ProblemMissingManifest is a source directory with no regular file
	// metais.yaml, which is package.layout.missing_manifest: it is not a
	// package (ADR-001 D5 item 5).
	ProblemMissingManifest

	// ProblemNameShape is a name no entry path can hold: invalid UTF-8,
	// a "\" or ":", or a leading byte order mark (ADR-001 D2 items 1 and
	// 2), which is package.name.encoding or package.name.shape.
	ProblemNameShape

	// ProblemLimit is a package limit breached, Limit naming it, which is
	// limit.package (ADR-009 D3, ADR-007 D2 step 3).
	ProblemLimit

	// ProblemUnreadable is a name in the layout's roots that could not be
	// examined or listed, Reason saying why (ADR-006 D9 item 8).
	ProblemUnreadable
)

// Problem is one reason a source directory is refused.
type Problem struct {
	Kind ProblemKind

	// Path is the name the problem is about, relative to the source
	// directory and "/"-separated: an entry path, a directory's path, or
	// "" for the source directory itself. It is a package path, never a
	// host path (ADR-009 D15 item 1). It may hold any bytes a name can,
	// so a caller that prints it neutralises it.
	Path string

	// Limit is the limit breached, for ProblemLimit.
	Limit limits.ID

	// Reason is why the name could not be read, for ProblemUnreadable.
	Reason Reason
}

// SourceError is the refusal of a source directory: every problem the walk
// found, at once, in byte order of path and then kind, and Names' answer.
type SourceError struct {
	Problems []Problem

	// Incomplete reports that the walk stopped at a bound — the entries
	// limit, or the findings limit's worth of problems — so there may be
	// more problems than Problems lists.
	Incomplete bool

	// Names is the refusal of SourceOptions.Names, or nil.
	Names error
}

func (e *SourceError) Error() string {
	n := len(e.Problems)
	if e.Names != nil {
		n++
	}
	return "hostfs: the source directory is refused: " + strconv.Itoa(n) + " problems"
}

// Unwrap returns Names' refusal, so errors.As finds the predicate's own
// error type.
func (e *SourceError) Unwrap() error {
	return e.Names
}

// Source is a source directory, read as a pkgtree.Tree (ADR-010 D11 item
// 1). Its entries are the regular files beneath the layout's roots, each
// addressed by its path relative to the directory.
//
// Open reads an entry in place, through the os.Root the directory was
// walked under. The file must still be the regular file the walk found,
// with the size it found: a file replaced, grown or shrunk since is an
// error from Open or Read, never different content.
type Source struct {
	root    *os.Root
	paths   []string
	files   map[string]sourceFile
	unknown []string
}

type sourceFile struct {
	info fs.FileInfo // the walk's Lstat
	size int64
}

var _ pkgtree.Tree = (*Source)(nil)

// OpenSource opens the source directory at path, the caller's own name for
// it, which may be a symbolic link to a directory (ADR-009 D7 item 4), and
// walks it under an os.Root (ADR-009 D7 item 8, ADR-001 D5).
//
// The layout's roots are metais.yaml and the directories assets, examples,
// inputs, references and templates. Beneath them every name must be a
// regular file or a non-empty directory: a symbolic link, FIFO, socket or
// device anywhere in them is refused from its Lstat mode, without being
// followed or opened. Nothing outside the roots is opened or listed, so a
// repository's .git never reaches a package: a top-level name beginning
// with "." and a top-level regular file whose name ends ".metais" are
// ignored silently, and any other top-level name is not part of the
// package and is returned by Unknown, for check's warning (ADR-001 D5
// item 5). A top-level .compiled is refused, and so is a directory with
// no regular file metais.yaml.
//
// Every problem is reported at once, in one *SourceError, and no Source is
// returned beside it. An error that is not a *SourceError is an *Error
// about the directory itself.
func OpenSource(path string, opt SourceOptions) (*Source, error) {
	if opt.Names == nil {
		return nil, errors.New("hostfs: OpenSource needs SourceOptions.Names")
	}
	root, err := openRoot(path)
	if err != nil {
		return nil, err
	}
	w := &walker{
		root:        root,
		lim:         opt.Limits,
		files:       map[string]sourceFile{},
		maxEntries:  opt.Limits.Value(limits.Entries),
		maxProblems: opt.Limits.Value(limits.Findings),
	}
	w.top()
	if err := w.refusal(opt.Names); err != nil {
		root.Close()
		return nil, err
	}
	slices.Sort(w.unknown)
	return &Source{root: root, paths: w.paths, files: w.files, unknown: w.unknown}, nil
}

// walker is one walk of a source directory.
type walker struct {
	root  *os.Root
	lim   limits.Set
	paths []string
	files map[string]sourceFile
	total int64

	unknown  []string
	problems []Problem

	maxEntries, maxProblems int64
	stopped                 bool
}

func (w *walker) problem(p Problem) {
	w.problems = append(w.problems, p)
	if int64(len(w.problems)) > w.maxProblems {
		w.stopped = true
	}
}

// listMax bounds the names one directory may hold. Every name beneath the
// layout's roots ends, at its deepest, in an entry or a problem, so a
// directory of more names than the entries limit and the problems bound
// together cannot open; refusing it there keeps the walk's work bounded by
// those two limits and path-depth.
func (w *walker) listMax() int {
	return int(w.maxEntries + w.maxProblems)
}

// list returns the entries of the directory rel, which Lstat described as
// info, sorted by name. It reports a problem and returns false when the
// directory cannot be listed.
func (w *walker) list(rel string, info fs.FileInfo) ([]fs.FileInfo, bool) {
	n, _ := osName(rel)
	f, err := w.root.Open(n)
	if err != nil {
		w.problem(Problem{Kind: ProblemUnreadable, Path: pathOf(rel), Reason: osReason(err)})
		return nil, false
	}
	defer f.Close()
	if got, err := f.Stat(); err != nil || !got.IsDir() || !os.SameFile(info, got) {
		w.problem(Problem{Kind: ProblemUnreadable, Path: pathOf(rel), Reason: Changed})
		return nil, false
	}
	var out []fs.FileInfo
	err = readDir(f, w.listMax(), func(de fs.DirEntry) error {
		// Info is the entry's Lstat: a link is described, not followed.
		fi, err := de.Info()
		if err != nil {
			return err
		}
		out = append(out, fi)
		return nil
	})
	if err != nil {
		if ReasonOf(err) == TooLarge {
			w.problem(Problem{Kind: ProblemLimit, Path: pathOf(rel), Limit: limits.Entries})
			w.stopped = true
		} else {
			w.problem(Problem{Kind: ProblemUnreadable, Path: pathOf(rel), Reason: osReasonOf(err)})
		}
		return nil, false
	}
	slices.SortFunc(out, func(a, b fs.FileInfo) int { return strings.Compare(a.Name(), b.Name()) })
	return out, true
}

func pathOf(rel string) string {
	if rel == "." {
		return ""
	}
	return rel
}

// osReasonOf is err's reason, mapping an OS error that is not yet an
// *Error.
func osReasonOf(err error) Reason {
	var e *Error
	if errors.As(err, &e) {
		return e.Reason
	}
	return osReason(err)
}

// top classifies the top-level names (ADR-001 D5 items 1 and 5) and walks
// the layout's roots.
func (w *walker) top() {
	info, err := w.root.Lstat(".")
	if err != nil {
		w.problem(Problem{Kind: ProblemUnreadable, Reason: osReason(err)})
		return
	}
	names, ok := w.list(".", info)
	if !ok {
		return
	}
	manifest := false
	for _, fi := range names {
		if w.stopped {
			return
		}
		name, kind := fi.Name(), kindOf(fi.Mode())
		switch {
		case name == compiledName:
			w.problem(Problem{Kind: ProblemCompiledInSource, Path: name})
		case name == manifestName && kind == KindFile:
			manifest = true
			w.file(name, fi)
		case (name == manifestName || slices.Contains(layoutDirs, name)) && (kind == KindLink || kind == KindOther):
			w.problem(Problem{Kind: ProblemNonRegularFile, Path: name})
		case slices.Contains(layoutDirs, name) && kind == KindDir:
			w.dir(name, fi, 1)
		case strings.HasPrefix(name, "."):
			// Ignored silently: .git, .gitignore, .DS_Store.
		case kind == KindFile && strings.HasSuffix(name, ".metais"):
			// Ignored silently: where pack writes by default.
		default:
			w.unknown = append(w.unknown, name)
		}
	}
	if !manifest && !w.stopped {
		w.problem(Problem{Kind: ProblemMissingManifest})
	}
}

// dir walks the directory rel, depth segments deep, which Lstat described
// as info.
func (w *walker) dir(rel string, info fs.FileInfo, depth int) {
	names, ok := w.list(rel, info)
	if !ok {
		return
	}
	if len(names) == 0 {
		w.problem(Problem{Kind: ProblemEmptyDirectory, Path: rel})
		return
	}
	for _, fi := range names {
		if w.stopped {
			return
		}
		name := fi.Name()
		child := rel + "/" + name
		if !nameShaped(name) {
			w.problem(Problem{Kind: ProblemNameShape, Path: child})
			continue
		}
		if int64(len(child)) > w.lim.Value(limits.PathLength) {
			w.problem(Problem{Kind: ProblemLimit, Path: child, Limit: limits.PathLength})
			continue
		}
		if int64(depth+1) > w.lim.Value(limits.PathDepth) {
			w.problem(Problem{Kind: ProblemLimit, Path: child, Limit: limits.PathDepth})
			continue
		}
		switch kindOf(fi.Mode()) {
		case KindFile:
			w.file(child, fi)
		case KindDir:
			w.dir(child, fi, depth+1)
		default:
			w.problem(Problem{Kind: ProblemNonRegularFile, Path: child})
		}
	}
}

// file admits the regular file rel, which Lstat described as info.
func (w *walker) file(rel string, info fs.FileInfo) {
	if int64(len(w.paths)) >= w.maxEntries {
		w.problem(Problem{Kind: ProblemLimit, Limit: limits.Entries})
		w.stopped = true
		return
	}
	size := info.Size()
	if size > w.lim.Value(limits.EntrySize) {
		w.problem(Problem{Kind: ProblemLimit, Path: rel, Limit: limits.EntrySize})
		return
	}
	if w.total += size; w.total > w.lim.Value(limits.PackageUncompressed) {
		w.problem(Problem{Kind: ProblemLimit, Limit: limits.PackageUncompressed})
		w.stopped = true
		return
	}
	w.paths = append(w.paths, rel)
	w.files[rel] = sourceFile{info: info, size: size}
}

// nameShaped reports whether name, one segment, can be part of an entry
// path under the pkgtree.Tree contract: valid UTF-8, no "\" or ":", and no
// leading byte order mark. A file system name is never empty, ".", ".."
// or holds a "/".
func nameShaped(name string) bool {
	return utf8.ValidString(name) && !strings.ContainsAny(name, "\\:") && !strings.HasPrefix(name, "\uFEFF")
}

// refusal returns the walk's *SourceError, or nil when the directory
// opens: no problem, and names admits every path.
func (w *walker) refusal(names func([]string) error) error {
	slices.Sort(w.paths)
	var nameErr error
	if !w.stopped {
		nameErr = names(slices.Clone(w.paths))
	}
	if len(w.problems) == 0 && nameErr == nil {
		return nil
	}
	slices.SortStableFunc(w.problems, func(a, b Problem) int {
		if c := strings.Compare(a.Path, b.Path); c != 0 {
			return c
		}
		return int(a.Kind) - int(b.Kind)
	})
	return &SourceError{Problems: w.problems, Incomplete: w.stopped, Names: nameErr}
}

// Unknown returns the top-level names that are not part of the package
// and are not ignored silently, in byte order: what check reports as
// package.layout.not_packed (ADR-001 D5 item 5). None of them was opened
// or listed.
func (s *Source) Unknown() []string {
	return slices.Clone(s.unknown)
}

// Close releases the directory. Readers Open returned stay readable.
func (s *Source) Close() error {
	return mapErr("close", s.root.Close())
}

func (s *Source) List() []string {
	return slices.Clone(s.paths)
}

func (s *Source) Stat(path string) (pkgtree.Info, error) {
	f, ok := s.files[path]
	if !ok {
		return pkgtree.Info{}, &pkgtree.PathError{Op: "stat", Path: path, Err: pkgtree.ErrNotExist}
	}
	return pkgtree.Info{Path: path, Size: f.size}, nil
}

// Open opens the entry at path. The file is examined with Lstat again
// first, and must still be the regular file the walk found, with its
// size; after opening, the handle must be the same file.
func (s *Source) Open(path string) (io.ReadCloser, error) {
	sf, ok := s.files[path]
	if !ok {
		return nil, &pkgtree.PathError{Op: "open", Path: path, Err: pkgtree.ErrNotExist}
	}
	now, err := lstat(s.root, path)
	if err != nil {
		return nil, opAs("open", err)
	}
	if r := fileRefusal(now.Mode()); r != "" {
		return nil, fail("open", r)
	}
	if !os.SameFile(sf.info, now) || now.Size() != sf.size {
		return nil, fail("open", Changed)
	}
	f, err := openSame(s.root, path, sf.info)
	if err != nil {
		return nil, err
	}
	return &sourceReader{f: f, left: sf.size}, nil
}

// sourceReader yields exactly the size the walk found, and an error rather
// than different content when the file has since grown or shrunk.
type sourceReader struct {
	f    *os.File
	left int64
	done bool
}

func (r *sourceReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, io.EOF
	}
	if r.left == 0 {
		// One byte more than the walk found means the file grew.
		n, err := r.f.Read(make([]byte, 1))
		if n > 0 {
			return 0, fail("read", Changed)
		}
		if err != nil && err != io.EOF {
			return 0, mapErr("read", err)
		}
		r.done = true
		return 0, io.EOF
	}
	if int64(len(p)) > r.left {
		p = p[:r.left]
	}
	n, err := r.f.Read(p)
	r.left -= int64(n)
	if err == io.EOF {
		if r.left > 0 {
			return n, fail("read", Changed)
		}
		err = nil
	}
	if err != nil {
		return n, mapErr("read", err)
	}
	return n, nil
}

func (r *sourceReader) Close() error {
	return mapErr("close", r.f.Close())
}
