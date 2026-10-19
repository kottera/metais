// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package loader opens a package: a packed package file or a source
// directory, verified whole before anything of it is used (ADR-001 D8,
// RULINGS R1-013).
//
// Open composes the libraries a package is read with. A packed package is
// read by internal/zipfmt's strict reader, which applies ADR-001 D8's
// steps 1 to 5; the loader adds step 6, the digest file (internal/digest),
// and step 7: the manifest in ADR-002 D11's order, the layout for its
// declared version, every schema file compiled (ADR-003 D22 item 1), and
// the three compiled files re-derived from the sources and compared byte
// for byte with those the package carries (ADR-001 D7). A source directory
// is read by internal/hostfs under the same portable-path rule
// (ADR-001 D2), its layout checked as it is walked (ADR-001 D5), and its
// compiled files and package digest computed in memory (ADR-001 D6 item
// 3, D8 item 5).
//
// The loader only reads. Every refusal is a finding; nothing is written,
// to the package or anywhere else.
package loader

import (
	"bytes"
	"io"
	"slices"
	"strings"

	"github.com/kottera/metais/internal/digest"
	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/pkgtree"
	"github.com/kottera/metais/internal/schema"
	"github.com/kottera/metais/internal/unitext"
	"github.com/kottera/metais/internal/zipfmt"
)

// Options configures Open.
type Options struct {
	// Limits are the limits the open is bounded by, made by
	// limits.Defaults and lowered by the caller.
	Limits limits.Set
}

// Form is how a package was given.
type Form uint8

const (
	// Packed is a packed package file (ADR-001 D1).
	Packed Form = iota + 1
	// Directory is a source directory (ADR-001 D5).
	Directory
)

// String is the form as inspect's source names it (ADR-008 D6 item 1).
func (f Form) String() string {
	if f == Packed {
		return "packed"
	}
	return "directory"
}

// Package is a package that opened: verified whole, its manifest valid
// under the version it declares, every schema compiled.
type Package struct {
	// Form is how the package was given.
	Form Form
	// Tree is the package's entries: a packed package's are the zip's,
	// .compiled/ included; a source directory's are the files pack would
	// write from it, without .compiled/ (ADR-008 D2 items 2 and 3).
	Tree pkgtree.Tree
	// Digest is the package digest (ADR-001 D6): the verified one of a
	// packed package, the one pack would write for a source directory.
	Digest digest.Sum
	// Manifest is the manifest's model, and ManifestValue its data model.
	Manifest      *manifest.Manifest
	ManifestValue value.Value
	// Schemas holds every schema file compiled, and Combined the combined
	// schema, compiled (ADR-003 D7 item 5).
	Schemas  *schema.Compiler
	Combined *schema.Schema
	// CombinedValue is the combined schema's data model, which pack writes
	// as .compiled/schema.json.
	CombinedValue value.Object
	// References is the references index (ADR-008 D17).
	References []Reference
	// Compiled is, for a source directory, the files pack writes under
	// .compiled/ but the digest file, derived in memory in byte order of
	// their names (ADR-001 D7, ADR-008 D2 item 3); nil for a packed
	// package, which carries its own.
	Compiled []File

	closer io.Closer
}

// Close releases what the package holds open. Readers its Tree returned
// stay readable for a source directory; a packed package's are not.
func (p *Package) Close() error {
	if p.closer == nil {
		return nil
	}
	return p.closer.Close()
}

// The compiled files of a packed package at specification version 1.0
// (ADR-001 D7 item 1), in byte order.
const (
	compiledDir        = ".compiled/"
	compiledManifest   = ".compiled/manifest.json"
	compiledReferences = ".compiled/references.json"
	compiledSchema     = ".compiled/schema.json"
)

var compiledFiles = []string{digest.File, compiledManifest, compiledReferences, compiledSchema}

// topLevel is ADR-001 D5 item 1's closed set of top-level names: the
// manifest, and the reserved directories, .compiled included.
var topLevel = []string{".compiled", "assets", "examples", "inputs", "metais.yaml", "references", "templates"}

// Open opens the package at path, the caller's own name for it (ADR-008
// D2 item 1): a regular file is read as a packed package, a directory as a
// source directory, and anything else is refused. It returns the package,
// or no package and every finding that refuses it. Warnings never refuse a
// package, and Open reports none.
//
// faults are the faults a test injects (ADR-010 D13), the invocation's own,
// passed through unchanged; the tool's are always zero.
func Open(path string, opt Options, faults fault.Set) (*Package, []finding.Finding) {
	f, err := hostfs.OpenFile(path)
	if err != nil {
		switch r := hostfs.ReasonOf(err); r {
		case hostfs.IsDirectory:
			return openSource(path, opt)
		case hostfs.NotFound:
			return nil, []finding.Finding{finding.New(CodeNotFound, pkgSubject, nil, nil)}
		default:
			return nil, []finding.Finding{notReadable(r)}
		}
	}
	p, fs := openPacked(f, f.Size(), opt, faults)
	if p == nil {
		f.Close()
		return nil, fs
	}
	p.closer = f
	return p, nil
}

func notReadable(r hostfs.Reason) finding.Finding {
	return finding.New(CodeNotReadable, pkgSubject, nil, finding.Params{"reason": finding.String(string(r))})
}

// openPacked is ADR-001 D8 over the packed package r of size bytes.
func openPacked(r io.ReaderAt, size int64, opt Options, faults fault.Set) (*Package, []finding.Finding) {
	c := newContents()
	a, fs, err := zipfmt.OpenWith(r, size, opt.Limits, c.sink)
	if err != nil {
		return nil, []finding.Finding{notReadable(hostfs.ReasonOf(err))}
	}
	if len(fs) > 0 {
		return nil, fs
	}
	// The compiled files must all be there before any can be read: a zip
	// without them is not a package (ADR-001 D7 item 1).
	names := a.List()
	var missing []finding.Finding
	for _, n := range compiledFiles {
		if _, found := slices.BinarySearch(names, n); !found {
			missing = append(missing, entryFinding(CodeCompiledMissing, n, nil))
		}
	}
	if len(missing) > 0 {
		return nil, missing
	}
	// Step 6.
	var entries []digest.Entry
	for _, e := range a.Entries() {
		if e.Name != digest.File {
			entries = append(entries, digest.Entry{Name: e.Name, Sum: e.SHA256})
		}
	}
	if fs := pkgtree.VerifyDigests(faults, func() []finding.Finding {
		return digest.Verify(c.bytes(digest.File), entries, opt.Limits)
	}); len(fs) > 0 {
		return nil, fs
	}
	// Step 7.
	o := &opening{form: Packed, tree: a, names: names, lim: opt.Limits, c: c}
	p, fs := o.step7()
	if p == nil {
		return nil, fs
	}
	p.Digest = digest.Package(entries)
	return p, nil
}

// openSource reads the source directory at path (ADR-001 D5, D8 item 5).
func openSource(path string, opt Options) (*Package, []finding.Finding) {
	lim := opt.Limits
	src, err := hostfs.OpenSource(path, hostfs.SourceOptions{
		Limits: lim,
		Names:  func(paths []string) error { return unitext.CheckTree(paths, lim) },
	})
	if err != nil {
		if se, ok := err.(*hostfs.SourceError); ok {
			return nil, sourceFindings(se, lim)
		}
		return nil, []finding.Finding{notReadable(hostfs.ReasonOf(err))}
	}
	p, fs := openDirectory(src, src.Unknown(), lim)
	if p == nil {
		src.Close()
		return nil, fs
	}
	p.closer = src
	return p, nil
}

// OpenFiles opens files, a source directory's entries by their paths, held
// in memory: the files pack would write from it, with no .compiled/. Its
// names are held to ADR-001 D2 as a source directory's are, and it is
// otherwise opened exactly as one, with its compiled files and package
// digest computed in memory. It is how a program that builds packages
// from entries, such as the conformance suite's generator, derives what
// pack derives.
func OpenFiles(files map[string][]byte, opt Options) (*Package, []finding.Finding) {
	t := newMemTree(files)
	if err := unitext.CheckTree(t.List(), opt.Limits); err != nil {
		return nil, zipfmt.NameFindings(err.(*unitext.TreeError), opt.Limits)
	}
	var unknown []string
	for _, n := range t.List() {
		top, _, _ := strings.Cut(n, "/")
		if top == ".compiled" {
			return nil, []finding.Finding{entryFinding(CodeCompiledInSource, top, nil)}
		}
		if !contains(topLevel, top) && !contains(unknown, top) {
			unknown = append(unknown, top)
		}
	}
	if len(unknown) > 0 {
		var out []finding.Finding
		for _, n := range unknown {
			out = append(out, entryFinding(CodeUnknownTopLevel, n, nil))
		}
		return nil, out
	}
	return openDirectory(t, nil, opt.Limits)
}

// openDirectory opens the source tree t, whose names and layout have been
// checked, unknown being its top-level names that are not part of the
// package.
func openDirectory(t pkgtree.Tree, unknown []string, lim limits.Set) (*Package, []finding.Finding) {
	o := &opening{form: Directory, tree: t, names: t.List(), unknown: unknown, lim: lim, c: newContents()}
	if !o.has(manifest.ManifestFile) {
		return nil, []finding.Finding{finding.New(CodeMissingManifest, pkgSubject, nil, nil)}
	}
	// ADR-002 D11 item 2 bounds the manifest before it is read.
	if fs := o.manifestSize(manifest.ManifestFile); len(fs) > 0 {
		return nil, fs
	}
	var entries []digest.Entry
	for _, n := range o.names {
		sum, err := o.read(n)
		if err != nil {
			return nil, []finding.Finding{notReadable(hostfs.ReasonOf(err))}
		}
		entries = append(entries, digest.Entry{Name: n, Sum: sum})
	}
	p, fs := o.step7()
	if p == nil {
		return nil, fs
	}
	// The package digest of the package pack would write: the source
	// entries and the three compiled files, derived in memory (ADR-001 D6
	// item 3).
	for _, cf := range o.derived {
		entries = append(entries, digest.Entry{Name: cf.Name, Sum: sha256Sum(cf.Content)})
	}
	p.Digest = digest.Package(entries)
	p.Compiled = o.derived
	return p, nil
}

// read reads the source entry n once, hashing it and handing it to the
// contents' sink.
func (o *opening) read(n string) (digest.Sum, error) {
	r, err := o.tree.Open(n)
	if err != nil {
		return digest.Sum{}, err
	}
	defer r.Close()
	h := newHash()
	var w io.Writer = h
	if s := o.c.sink(n); s != nil {
		w = io.MultiWriter(h, s)
	}
	if _, err := io.Copy(w, r); err != nil {
		return digest.Sum{}, err
	}
	var sum digest.Sum
	copy(sum[:], h.Sum(nil))
	return sum, nil
}

// File is one compiled file, derived from the sources.
type File struct {
	Name    string
	Content []byte
}

// opening is one open on its way through step 7.
type opening struct {
	form    Form
	tree    pkgtree.Tree
	names   []string // every entry, in byte order
	unknown []string // a source directory's top-level names that are not part of the package
	lim     limits.Set
	c       *contents
	doc     manifestDoc
	derived []File
}

// step7 is ADR-001 D8 step 7, which ADR-002 D11 orders: the manifest
// located, bounded and parsed under its profile; for a packed package, its
// canonical form compared with .compiled/manifest.json; the version gate;
// the layout for that version; the manifest schema and rules; then the
// data specification's rules, every schema compiled, and the compiled
// files derived and, for a packed package, compared. Each stage that finds
// anything ends the open there, with every finding of the stage.
func (o *opening) step7() (*Package, []finding.Finding) {
	p := &Package{Form: o.form, Tree: o.tree}
	for _, stage := range []func(*Package) []finding.Finding{
		o.locate, o.parse, o.compareManifest, o.gate, o.layout, o.validate,
		o.paths, o.expressions, o.compile, o.derive,
	} {
		if fs := stage(p); len(fs) > 0 {
			return nil, fs
		}
	}
	return p, nil
}

// has reports whether name is an entry.
func (o *opening) has(name string) bool {
	_, ok := slices.BinarySearch(o.names, name)
	return ok
}

// under returns every entry beneath dir, which ends in "/".
func (o *opening) under(dir string) []string {
	i, _ := slices.BinarySearch(o.names, dir)
	var out []string
	for ; i < len(o.names) && strings.HasPrefix(o.names[i], dir); i++ {
		out = append(out, o.names[i])
	}
	return out
}

// topNames is every top-level name of the package's entries, each once, in
// byte order: a file's name, or a directory's without its "/".
func (o *opening) topNames() []string {
	var out []string
	for _, n := range o.names {
		top, _, _ := strings.Cut(n, "/")
		if len(out) == 0 || out[len(out)-1] != top {
			out = append(out, top)
		}
	}
	return out
}

// manifestSize is ADR-002 D11 item 2 for the entry name: manifest-size,
// before it is read.
func (o *opening) manifestSize(name string) []finding.Finding {
	info, err := o.tree.Stat(name)
	if err != nil {
		return nil
	}
	if info.Size > o.lim.Value(limits.ManifestSize) {
		return []finding.Finding{o.lim.Breached(limits.ManifestSize, entryLocation(name), info.Size, true)}
	}
	return nil
}

// sameBytes reports whether the stored compiled file name holds want.
func (o *opening) sameBytes(name string, want []byte) bool {
	return bytes.Equal(o.c.bytes(name), want)
}
