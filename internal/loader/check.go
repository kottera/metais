// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"io"
	"slices"
	"strings"

	"metais.org/metais/internal/fault"
	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/hostfs"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/liquid"
	"metais.org/metais/internal/liquid/ast"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/manifest"
	"metais.org/metais/internal/pkgtree"
	"metais.org/metais/internal/schema"
	"metais.org/metais/internal/template/filters"
	"metais.org/metais/internal/unitext"
	"metais.org/metais/internal/zipfmt"
)

// Dialect is the dialect a package's manifest expressions are parsed in
// when the package is opened (ADR-004 D16 item 5), and check's
// cross-reference parses them with: internal/liquid's standard catalog,
// Core(), with METAIS's own filters (internal/template/filters), so that a
// template and an expression know one set of filters, and METAIS's embed,
// asset and provenance tags known by name, so that a path holding one is
// template.tag_context rather than an unknown tag. A template is parsed in
// internal/template's dialect, which defines the tags and reserves the
// globals' names.
func Dialect() *liquid.Dialect { return dialect }

var dialect = func() *liquid.Dialect {
	d, err := liquid.Core().Extend(liquid.Spec{
		Filters: filters.All(),
		Tags:    []liquid.TagDef{namedTag("embed"), namedTag("asset"), namedTag("provenance")},
	})
	if err != nil {
		panic("loader: " + err.Error())
	}
	return d
}()

// namedTag is a tag a manifest expression knows only by its name: no
// expression may hold it, so it accepts any arguments and is never
// rendered.
type namedTag string

func (t namedTag) Name() string               { return string(t) }
func (namedTag) Check(*ast.Tag) *liquid.Error { return nil }
func (t namedTag) Render(liquid.TagContext, *ast.Tag) *liquid.Error {
	return &liquid.Error{Code: liquid.CodeInternal, Detail: "a manifest expression rendered the " + string(t) + " tag"}
}

// ParseLimits are the bounds a package's templates are parsed under:
// ADR-009's template-nesting and template-size, from lim.
func ParseLimits(lim limits.Set) liquid.ParseLimits {
	return liquid.ParseLimits{Nesting: int(lim.Value(limits.TemplateNesting)), Size: int(lim.Value(limits.TemplateSize))}
}

// Checked is a package as check reads it (ADR-007 D2): every step the
// loader runs, each run whatever an earlier step found, unless its input
// could not be made.
type Checked struct {
	// Form is how the package was given.
	Form Form
	// Tree is the package's entries, as Package.Tree.
	Tree pkgtree.Tree
	// Unknown is a source directory's top-level names pack would not
	// write and does not ignore silently (ADR-007 D7 item 3), other than a
	// misspelt manifest, which step 4 reports; nil for a packed package.
	Unknown []string
	// ManifestValue is the manifest's data model, and Positions locates
	// its values; nil when it did not parse.
	ManifestValue value.Value
	Positions     finding.Positions
	// Manifest is the manifest's model, nil unless it passed step 4's
	// schema and rules.
	Manifest *manifest.Manifest
	// Schemas holds every schema file compiled, nil when any did not.
	Schemas *schema.Compiler
	// Combined is the combined schema compiled, and CombinedValue its
	// data model; Combined is nil when it could not be made: with no
	// Manifest, or no Schemas.
	Combined      *schema.Schema
	CombinedValue value.Object
	// Compiled is, for a source directory, the compiled files pack would
	// write, as Package.Compiled; nil when they could not be made.
	Compiled []File

	// The findings of each step the loader runs (ADR-007 D2 items 3, 4,
	// 5 and 10, and step 6's manifest expressions).
	Limits, ManifestFindings, SchemaFindings, Expressions, CompiledFindings []finding.Finding

	closer interface{ Close() error }
}

// Close releases what the package holds open.
func (c *Checked) Close() error {
	if c.closer == nil {
		return nil
	}
	return c.closer.Close()
}

// Check reads the package at path as check does (ADR-007 D2). A package
// its read refuses — the container, the entry names, the digests, the
// walk of a source directory, or any finding of class refused, which
// every command refuses the package with — is not checked: Check returns
// no package and those findings, the findings every other command
// reports for it. Otherwise each of the loader's steps runs on whatever
// the steps before it made, and its findings are kept on the package.
func Check(path string, opt Options, faults fault.Set) (*Checked, []finding.Finding) {
	f, err := hostfs.OpenFile(path)
	if err != nil {
		switch r := hostfs.ReasonOf(err); r {
		case hostfs.IsDirectory:
			return checkSource(path, opt)
		case hostfs.NotFound:
			return nil, []finding.Finding{finding.New(CodeNotFound, pkgSubject, nil, nil)}
		default:
			return nil, []finding.Finding{notReadable(r)}
		}
	}
	c, fs := checkPacked(f, f.Size(), opt, faults)
	if c == nil {
		f.Close()
		return nil, fs
	}
	c.closer = f
	return c, nil
}

// checkPacked is ADR-001 D8 steps 1 to 6 over a packed package, then
// check's steps.
func checkPacked(f io.ReaderAt, size int64, opt Options, faults fault.Set) (*Checked, []finding.Finding) {
	o, _, fs := readPacked(f, size, opt, faults)
	if o == nil {
		return nil, fs
	}
	return o.check()
}

// checkSource is ADR-007 D2 step 1's walk of a source directory, then
// check's steps.
func checkSource(path string, opt Options) (*Checked, []finding.Finding) {
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
	o := &opening{form: Directory, tree: src, names: src.List(), unknown: src.Unknown(), lim: lim, c: newContents()}
	// ADR-002 D11 item 2 bounds the manifest before it is read.
	if fs := o.manifestSize(manifest.ManifestFile); len(fs) > 0 {
		src.Close()
		return nil, fs
	}
	for _, n := range o.names {
		if _, err := o.read(n); err != nil {
			src.Close()
			return nil, []finding.Finding{notReadable(hostfs.ReasonOf(err))}
		}
	}
	c, fs := o.check()
	if c == nil {
		src.Close()
		return nil, fs
	}
	c.closer = src
	return c, nil
}

// refused reports whether any of fs is of class refused: a finding every
// command refuses the package with (ADR-006 D12).
func refused(fs []finding.Finding) bool {
	return slices.ContainsFunc(fs, func(f finding.Finding) bool { return f.Class() == finding.ClassRefused })
}

// check runs the steps of ADR-007 D2 the loader owns over the entries
// already read: step 4, the manifest, in ADR-002 D11's order, each stage
// that can still run on what the earlier ones made; step 5's compile;
// step 6's manifest expressions; step 10, the compiled files; and step 3,
// the limits of the package pack would write, compiled files included.
func (o *opening) check() (*Checked, []finding.Finding) {
	c := &Checked{Form: o.form, Tree: o.tree}
	for _, n := range o.unknown {
		if !wrongManifestName(n) {
			c.Unknown = append(c.Unknown, n)
		}
	}
	p := &Package{Form: o.form, Tree: o.tree}
	var all []finding.Finding
	stop := func(fs []finding.Finding) bool {
		all = append(all, fs...)
		return refused(fs)
	}

	// Step 4. The manifest's stages each need the one before: a stage
	// that finds anything ends the manifest there.
	manifestOK := true
	for _, stage := range []func(*Package) []finding.Finding{o.locate, o.parse, o.compareManifest, o.gate, o.layout, o.validate} {
		fs := stage(p)
		c.ManifestFindings = append(c.ManifestFindings, fs...)
		if stop(fs) {
			return nil, all
		}
		if len(fs) > 0 {
			manifestOK = false
			break
		}
	}
	if o.doc.root != nil {
		c.ManifestValue, c.Positions = o.doc.root, o.doc.src.Positions
	}
	if manifestOK {
		c.Manifest = p.Manifest
		fs := o.paths(p)
		c.ManifestFindings = append(c.ManifestFindings, fs...)
		if stop(fs) {
			return nil, all
		}
	}

	// Step 5: every schema file compiled, whether or not an input names
	// it; then, with a manifest, the combined schema.
	compiler, values, fs := o.compileFiles()
	c.SchemaFindings = fs
	if stop(fs) {
		return nil, all
	}
	c.Schemas = compiler
	if compiler != nil && c.Manifest != nil {
		doc, ok := o.bundle(c.Manifest, values)
		if ok {
			combined, fs := compiler.Combined(doc)
			c.SchemaFindings = append(c.SchemaFindings, fs...)
			if stop(fs) {
				return nil, all
			}
			if len(fs) == 0 {
				p.Schemas, p.Combined, p.CombinedValue = compiler, combined, doc
				c.Combined, c.CombinedValue = combined, doc
			}
		}
	}

	// Step 6's manifest expressions.
	if c.Manifest != nil {
		c.Expressions = o.expressions(p)
		if stop(c.Expressions) {
			return nil, all
		}
	}

	// Step 10: the compiled files, computed; a packed package's compared
	// with those it carries.
	if c.Manifest != nil && c.Combined != nil {
		fs := o.derive(p)
		c.CompiledFindings = fs
		if stop(fs) {
			return nil, all
		}
		c.Compiled = o.derived
	}

	// Step 3: a source directory's entries were read within the limits;
	// the package pack would write adds the compiled files and the digest
	// file to them.
	if o.form == Directory && c.Compiled != nil {
		c.Limits = o.packLimits(c.Compiled)
		if stop(c.Limits) {
			return nil, all
		}
	}
	return c, nil
}

// compileFiles compiles every schema file of the package (ADR-003 D22
// item 1), returning the compiler and each file's data model, or the
// findings that refuse them.
func (o *opening) compileFiles() (*schema.Compiler, map[string]value.Value, []finding.Finding) {
	files := map[string][]byte{}
	for _, n := range o.names {
		if schema.IsSchemaFile(n) {
			files[n] = o.c.bytes(n)
		}
	}
	values := map[string]value.Value{}
	c, fs := schema.NewCompiler(newMemTree(files), o.lim, func(entry string, src []byte) (value.Value, []finding.Finding) {
		v, fs := decodeSchema(entry, src, o.lim)
		if len(fs) == 0 {
			values[entry] = v
		}
		return v, fs
	})
	if len(fs) > 0 {
		return nil, nil, fs
	}
	return c, values, nil
}

// bundle is the combined schema of the files and the manifest m (ADR-003
// D7), and false when an input names a schema that is not one of them,
// which step 4 has reported.
func (o *opening) bundle(m *manifest.Manifest, values map[string]value.Value) (value.Object, bool) {
	decl := schema.Declarations{Constraints: m.InputConstraints}
	for _, in := range m.Inputs {
		si := schema.Input{Name: in.Name, Title: in.Title, Schema: in.Schema, Required: in.Required}
		if in.Description != "" {
			d := in.Description
			si.Description = &d
		}
		decl.Inputs = append(decl.Inputs, si)
	}
	paths := make([]string, 0, len(values))
	for n := range values {
		paths = append(paths, n)
	}
	slices.Sort(paths)
	sf := make([]schema.File, 0, len(paths))
	for _, n := range paths {
		sf = append(sf, schema.File{Path: n, Root: values[n]})
	}
	doc, err := schema.Bundle(decl, sf)
	if err != nil {
		return value.Object{}, false
	}
	return doc, true
}

// packLimits is ADR-007 D2 step 3 for a source directory: the package
// pack would write, its source entries and the README.md, compiled files
// and digest file pack adds, within entries, entry-size and
// package-uncompressed (ADR-009 D3). The digest file is not computed here; it holds one line
// per entry, and its size is computed from them.
func (o *opening) packLimits(compiled []File) []finding.Finding {
	var total int64
	for _, n := range o.names {
		if info, err := o.tree.Stat(n); err == nil {
			total += info.Size
		}
	}
	var out []finding.Finding
	// The README.md pack writes first (ADR-001 A1 item 1), and the
	// digest file.
	readme := int64(len(Readme(o.spec)))
	total += readme
	if readme > o.lim.Value(limits.EntrySize) {
		out = append(out, o.lim.Breached(limits.EntrySize, entryLocation(zipfmt.ReadmeName), readme, true))
	}
	count := int64(len(o.names) + len(compiled) + 2)
	if count > o.lim.Value(limits.Entries) {
		out = append(out, o.lim.Breached(limits.Entries, finding.Location{Origin: finding.OriginPackage}, count, true))
	}
	digestSize := int64(0)
	for _, n := range o.names {
		digestSize += digestLine(n)
	}
	for _, f := range compiled {
		digestSize += digestLine(f.Name)
		total += int64(len(f.Content))
		if int64(len(f.Content)) > o.lim.Value(limits.EntrySize) {
			out = append(out, o.lim.Breached(limits.EntrySize, entryLocation(f.Name), int64(len(f.Content)), true))
		}
	}
	total += digestSize
	if total > o.lim.Value(limits.PackageUncompressed) {
		out = append(out, o.lim.Breached(limits.PackageUncompressed, finding.Location{Origin: finding.OriginPackage}, total, true))
	}
	return out
}

// digestLine is the length of the digest file's line for the entry name
// (ADR-001 D6 item 2): the hex digest, two spaces, the name and an LF.
func digestLine(name string) int64 {
	return int64(64 + 2 + len(name) + 1)
}

// Entries is every entry beneath dir, which ends in "/", in byte order.
func (c *Checked) Entries(dir string) []string {
	var out []string
	for _, n := range c.Tree.List() {
		if strings.HasPrefix(n, dir) {
			out = append(out, n)
		}
	}
	return out
}
