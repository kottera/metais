// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package author

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io"
	"net/url"
	pathpkg "path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/kottera/metais/internal/author/ooxml"
	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/regexprofile"
	"github.com/kottera/metais/internal/schema"
	templates "github.com/kottera/metais/internal/template"
	"github.com/kottera/metais/internal/unitext"
	"github.com/kottera/metais/internal/xref"
)

// check's own codes (ADR-007 D2). Each is class content.
const (
	CodeStepSkipped                  finding.Code = "report.step_skipped"          // D2 item 2, info
	CodeOSLitter                     finding.Code = "package.layout.os_litter"     // step 1, a warning
	CodeNotPacked                    finding.Code = "package.layout.not_packed"    // step 1, D7 item 3, a warning
	CodeGuideLinkMissing             finding.Code = "manifest.guide_link_missing"  // step 4, a warning
	CodeInvalidDefault               finding.Code = "schema.invalid_default"       // step 5
	CodeInvalidExample               finding.Code = "schema.invalid_example"       // step 5
	CodeUnsatisfiable                finding.Code = "schema.unsatisfiable"         // step 5
	CodeUnusableEnumMember           finding.Code = "schema.unusable_enum_member"  // step 5, a warning
	CodeRequiredNotDeclared          finding.Code = "schema.required_not_declared" // step 5, a warning
	CodeTemplateCRLF                 finding.Code = "template.crlf"                // step 6, a warning
	CodeExampleLayout                finding.Code = "example.layout"               // step 9, D5 item 1
	codeTemplateNotUTF8                           = loader.CodeTemplateNotUTF8     // step 6
	codeTemplateSyntax                            = loader.CodeTemplateSyntax      // step 6
	templatesDir                                  = xref.TemplatesDir              // every template lies under it
	examplesDir                                   = "examples/"                    // every example lies under it
	assetsDir                                     = "assets/"                      // every asset lies under it
	manifestFile                                  = manifest.ManifestFile          // where the manifest's findings are
	guidePointer                                  = "/guide"                       // where the guide is in the manifest
	examplesInputs, examplesExpected              = "inputs/", "expected/"         // an example's two trees
	exampleFile                                   = "example.yaml"                 // an example's own file
)

// CheckCodes are the codes only check reports: its own, the
// cross-reference's and the split-run scan's. A command that opens a
// package for any other purpose reports none of them.
func CheckCodes() []finding.Code {
	out := []finding.Code{CodeStepSkipped, CodeOSLitter, CodeNotPacked, CodeGuideLinkMissing, CodeInvalidDefault,
		CodeInvalidExample, CodeUnsatisfiable, CodeUnusableEnumMember, CodeRequiredNotDeclared, CodeTemplateCRLF,
		CodeExampleLayout, CodeFormShape, CodeCollectionShape, CodeValidateKind, codeTemplateNotUTF8, ooxml.CodeSplitConstruct, ooxml.CodeSplitDelimiter}
	return append(out, xref.Codes()...)
}

// PathCodes are the codes of ADR-005 D12's portable-path predicate, which
// check reports over every output path the manifest writes literally, and
// a render over every path it plans.
func PathCodes() []finding.Code { return templates.PathCodes() }

// ContainerCodes are the codes of ADR-005 D9's profile rules, which check
// reports statically and a render on the planned entry paths (D9 item 6).
func ContainerCodes() []finding.Code {
	return []finding.Code{finding.CodeContainerMissingEntry, finding.CodeContainerNonASCII,
		finding.CodeContainerReservedEntry, finding.CodeContainerSpace}
}

// The codes of ADR-003's and ADR-005's rules that only check can judge,
// statically: form and shape agreement (ADR-003 D1 item 6), a
// collection's schema (D15 item 1), and the kind of an output validate
// names (ADR-005 D11 item 2). Each is an error of class content.
const (
	CodeFormShape       finding.Code = "manifest.input.form_shape"
	CodeCollectionShape finding.Code = "manifest.input.collection_shape"
	CodeValidateKind                 = finding.CodeDeclarationValidateKind
)

// The names of the steps and parts of steps report.step_skipped names
// (ADR-007 D2 item 2), in the order check runs them.
const (
	stepCompiledLimits = "limits of the compiled files"
	stepManifestPaths  = "manifest paths"
	stepGuideLinks     = "guide links"
	stepCombined       = "combined schema"
	stepExpressions    = "manifest expressions"
	stepXref           = "cross-reference"
	stepCompiled       = "compiled form"
)

var (
	pkgSubject      = finding.Subject{Kind: finding.SubjectPackage}
	manifestSubject = finding.Subject{Kind: finding.SubjectManifest}
	reportSubject   = finding.Subject{Kind: finding.SubjectReport}
)

// Check runs ADR-007 D2 over the package at path, a source directory or a
// packed package, under lim, and returns every finding of every step. It
// reads nothing but the package, and writes nothing. A package its read
// refuses is refused with the findings every command gives it (ADR-001
// D8): check's steps run on a package whose entries were read.
//
// faults are the faults a test injects (ADR-010 D13), passed through.
func Check(path string, lim limits.Set, faults fault.Set) []finding.Finding {
	c, refused := loader.Check(path, loader.Options{Limits: lim}, faults)
	if c == nil {
		return refused
	}
	defer c.Close()
	k := &checker{c: c, lim: lim}
	return k.run()
}

// checker is one run of check over a package that was read.
type checker struct {
	c       *loader.Checked
	lim     limits.Set
	out     []finding.Finding
	skipped []string
}

func (k *checker) add(fs ...finding.Finding) { k.out = append(k.out, fs...) }

func (k *checker) skip(step string) { k.skipped = append(k.skipped, step) }

// run is the steps, in ADR-007 D2's order. A step whose input could not be
// made is named in one report.step_skipped finding, at the end.
func (k *checker) run() []finding.Finding {
	c := k.c
	noManifest := c.Manifest == nil

	// Step 1: the source tree (a source directory only). Its refusals
	// were the read's; what is left are the two warnings.
	if c.Form == loader.Directory {
		for _, n := range c.Unknown {
			k.add(entryFinding(CodeNotPacked, n))
		}
		for _, n := range c.Tree.List() {
			if osLitter(n) {
				k.add(entryFinding(CodeOSLitter, n))
			}
		}
	}
	// Step 2, the entries, was the read. Step 3: the package pack would
	// write, compiled files included.
	if c.Form == loader.Directory && noManifest {
		k.skip(stepCompiledLimits)
	}
	k.add(c.Limits...)

	// Step 4: the manifest, then the links of its guide.
	k.add(c.ManifestFindings...)
	if noManifest {
		k.skip(stepManifestPaths)
		k.skip(stepGuideLinks)
	} else {
		k.add(k.validateKinds(c.Manifest.Outputs, "")...)
		k.add(k.outputPaths(c.Manifest.Outputs, nil, "", 0)...)
		k.add(k.containers(c.Manifest.Outputs, "", 0)...)
		k.add(k.guideLinks()...)
	}

	// Step 5: every schema file compiled, then each statically; and every
	// output validation schema compiled, without data (ADR-005 D11 item
	// 5).
	k.add(c.SchemaFindings...)
	if !noManifest {
		k.add(k.outputSchemas()...)
	}
	if c.Schemas != nil {
		k.add(k.schemaStatics()...)
	}
	if noManifest {
		k.skip(stepCombined)
	} else if c.Combined != nil {
		k.add(k.shapes()...)
	}

	// Step 6: every template, then the manifest's expressions.
	templates := k.templates()
	if noManifest {
		k.skip(stepExpressions)
	} else {
		k.add(c.Expressions...)
	}

	// Step 7: the cross-reference.
	if noManifest {
		k.skip(stepXref)
	} else {
		k.add(k.xref(templates)...)
	}

	// Step 8: Office Open XML text runs.
	ts := make([]ooxml.Template, len(templates))
	for i, t := range templates {
		ts[i] = ooxml.Template{Path: t.path, Text: t.src}
	}
	k.add(ooxml.Scan(ts)...)

	// Step 9: the examples' layout.
	k.add(k.examples()...)

	// Step 10: the compiled form.
	if noManifest {
		k.skip(stepCompiled)
	} else {
		k.add(c.CompiledFindings...)
	}

	if len(k.skipped) > 0 {
		steps := make([]finding.Value, len(k.skipped))
		for i, st := range k.skipped {
			steps[i] = finding.String(st)
		}
		k.add(finding.New(CodeStepSkipped, reportSubject, nil, finding.Params{"steps": finding.List(steps...)}))
	}
	return k.out
}

// entryLocation is the location of the package entry name as a whole.
func entryLocation(name string) finding.Location {
	return finding.Location{Origin: finding.OriginPackage, File: name}
}

func entryFinding(code finding.Code, name string) finding.Finding {
	l := entryLocation(name)
	return finding.New(code, pkgSubject, &l, nil)
}

// layoutTrees are the top-level directories of ADR-001 D5's layout that
// pack writes whole.
var layoutTrees = []string{"assets/", "examples/", "inputs/", "references/", "templates/"}

// osLitter reports whether the entry name is operating-system litter
// beneath a layout tree (ADR-007 D2 step 1): .DS_Store, Thumbs.db,
// desktop.ini, or a name beginning ._.
func osLitter(name string) bool {
	if !slices.ContainsFunc(layoutTrees, func(t string) bool { return strings.HasPrefix(name, t) }) {
		return false
	}
	base := pathpkg.Base(name)
	return base == ".DS_Store" || base == "Thumbs.db" || base == "desktop.ini" || strings.HasPrefix(base, "._")
}

// manifestAt is the location of the manifest value at ptr.
func (k *checker) manifestAt(ptr string) *finding.Location {
	l := finding.Location{Origin: finding.OriginPackage, File: manifestFile}.WithPointer(ptr)
	if pos := k.c.Positions; pos != nil {
		if p, ok := pos.Value(ptr); ok {
			l = l.At(p)
		}
	}
	return &l
}

// guideLinks is ADR-007 D2 step 4's last check: a relative link of the
// guide whose target is no package entry is manifest.guide_link_missing,
// a warning (ADR-002 D4 item 9).
func (k *checker) guideLinks() []finding.Finding {
	names := k.c.Tree.List()
	var out []finding.Finding
	seen := map[string]bool{}
	for _, target := range markdownLinks(k.c.Manifest.Guide) {
		p, ok := packagePath(target)
		if !ok || seen[target] {
			continue
		}
		seen[target] = true
		if _, found := slices.BinarySearch(names, p); found {
			continue
		}
		dir := strings.TrimSuffix(p, "/") + "/"
		if i, _ := slices.BinarySearch(names, dir); i < len(names) && strings.HasPrefix(names[i], dir) {
			continue
		}
		out = append(out, finding.New(CodeGuideLinkMissing, manifestSubject, k.manifestAt(guidePointer),
			finding.Params{"target": finding.String(target)}))
	}
	return out
}

var schemeRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)

// packagePath is the package path a link target names, and false when the
// target is not relative: it has a scheme, starts with / or #, or names
// nothing but a fragment or a query.
func packagePath(target string) (string, bool) {
	if target == "" || schemeRE.MatchString(target) || strings.HasPrefix(target, "/") || strings.HasPrefix(target, "#") {
		return "", false
	}
	if i := strings.IndexAny(target, "?#"); i >= 0 {
		target = target[:i]
	}
	if target == "" {
		return "", false
	}
	if u, err := url.PathUnescape(target); err == nil {
		target = u
	}
	return pathpkg.Clean(target), true
}

// markdownLinks returns the targets of the CommonMark links and images
// of src, inline and by reference definition, outside code: what the
// guide links to.
func markdownLinks(src string) []string {
	var out []string
	fence := ""
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimLeft(line, " ")
		if fence != "" {
			if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
			continue
		}
		if len(line)-len(trimmed) <= 3 {
			if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
				fence = trimmed[:3]
				continue
			}
			if m := refDefRE.FindStringSubmatch(trimmed); m != nil {
				out = append(out, strings.Trim(m[1], "<>"))
				continue
			}
		}
		out = append(out, inlineLinks(line)...)
	}
	return out
}

var refDefRE = regexp.MustCompile(`^\[[^\]]+\]:\s*(\S+)`)

// inlineLinks returns the destinations of the inline links of one line,
// skipping code spans.
func inlineLinks(line string) []string {
	var out []string
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '`':
			n := 1
			for i+n < len(line) && line[i+n] == '`' {
				n++
			}
			ticks := line[i : i+n]
			if end := strings.Index(line[i+n:], ticks); end >= 0 {
				i += n + end + n - 1
			} else {
				i += n - 1
			}
		case ']':
			if i+1 < len(line) && line[i+1] == '(' {
				if d, ok := destination(line[i+2:]); ok {
					out = append(out, d)
				}
			}
		}
	}
	return out
}

// destination reads a link destination at the start of s, after "(".
func destination(s string) (string, bool) {
	s = strings.TrimLeft(s, " \t")
	if strings.HasPrefix(s, "<") {
		end := strings.IndexByte(s, '>')
		if end < 0 {
			return "", false
		}
		return s[1:end], true
	}
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			if depth == 0 {
				return s[:i], i > 0
			}
			depth--
		case ' ', '\t':
			return s[:i], i > 0
		}
	}
	return "", false
}

// shapes is ADR-003 D1 item 6 and D15 item 1, judged on each input's
// statically declared type set (D14 item 1): a collection's schema
// declares an array type and an item schema, and each form agrees with
// the schema of what a file holds — the input's, or a collection's item
// schema: csv an array whose items include object, text a string,
// markdown an object; json and yaml anything.
func (k *checker) shapes() []finding.Finding {
	w, err := schema.NewWalker(k.c.CombinedValue, schema.CombinedURI)
	if err != nil {
		panic(fmt.Sprintf("author: the combined schema, which compiled: %v", err))
	}
	walk := func(path ...schema.Step) (schema.Static, bool) {
		s, ok, err := w.Walk(path)
		if err != nil {
			panic(fmt.Sprintf("author: walking the combined schema, which compiled: %v", err))
		}
		return s, ok
	}
	var out []finding.Finding
	for _, in := range k.c.Manifest.Inputs {
		file, ok := walk(schema.Member(in.Name))
		if !ok {
			continue
		}
		if in.Collection {
			item, ok := walk(schema.Member(in.Name), schema.Index(maxIndex))
			if !file.Types.Has(schema.TypeArray) || !ok {
				out = append(out, finding.New(CodeCollectionShape, manifestSubject, k.manifestAt(in.Pointer+"/schema"), nil))
				continue
			}
			file = item
		}
		for i, form := range in.Forms {
			var good bool
			switch form {
			case manifest.FormCSV:
				row, ok := walk(append(pathTo(in), schema.Index(maxIndex))...)
				good = file.Types.Has(schema.TypeArray) && ok && row.Types.Has(schema.TypeObject)
			case manifest.FormText:
				good = file.Types.Has(schema.TypeString)
			case manifest.FormMarkdown:
				good = file.Types.Has(schema.TypeObject)
			default:
				good = true
			}
			if !good {
				out = append(out, finding.New(CodeFormShape, manifestSubject, k.manifestAt(in.Pointer+"/forms/"+strconv.Itoa(i)),
					finding.Params{"form": finding.String(string(form)), "types": typeNames(file.Types)}))
			}
		}
	}
	return out
}

// maxIndex is an index beyond any prefixItems a schema may hold within
// the limits, so that walking it reaches items.
const maxIndex = 1 << 30

// pathTo is the walk to what one file of the input in holds: the input,
// or a collection's item.
func pathTo(in manifest.Input) []schema.Step {
	if in.Collection {
		return []schema.Step{schema.Member(in.Name), schema.Index(maxIndex)}
	}
	return []schema.Step{schema.Member(in.Name)}
}

// typeNames is the type set t as a parameter: its names, in the order
// TypeSet lists them.
func typeNames(t schema.TypeSet) finding.Value {
	var vs []finding.Value
	for _, n := range []struct {
		t    schema.TypeSet
		name string
	}{{schema.TypeNull, "null"}, {schema.TypeBoolean, "boolean"}, {schema.TypeObject, "object"}, {schema.TypeArray, "array"},
		{schema.TypeNumber, "number"}, {schema.TypeString, "string"}, {schema.TypeInteger, "integer"}} {
		if t.Has(n.t) {
			vs = append(vs, finding.String(n.name))
		}
	}
	return finding.List(vs...)
}

// outputSchemas compiles every schema an output's validate names, at
// every depth of a container (ADR-005 D11 item 5), and reports what
// refuses any of them.
func (k *checker) outputSchemas() []finding.Finding {
	var meter *regexprofile.Meter
	if k.c.Schemas != nil {
		meter = k.c.Schemas.Meter()
	}
	_, fs := loader.OutputSchemas(k.c.Tree, k.c.Manifest.Outputs, k.lim, meter)
	return fs
}

// validateKinds is ADR-005 D11 item 2 for outs, at every depth of a
// container: an output with validate has kind json or yaml. The kind is
// the declared one, or the one the output's path gives (D7 item 1) when
// check can know it: a path omitted, or one whose text after its last
// Liquid construct holds the file's extension. A kind only the render can
// know is the render's to judge. owner is the id of the declaration a
// container entry with no id of its own is part of.
func (k *checker) validateKinds(outs []manifest.Output, owner string) []finding.Finding {
	var out []finding.Finding
	for _, d := range outs {
		id := d.ID
		if id == "" {
			id = owner
		}
		if d.Validate != "" {
			kind, known := d.Kind, d.Kind != ""
			if !known {
				kind, known = staticKind(d)
			}
			if known && kind != "json" && kind != "yaml" {
				out = append(out, finding.New(CodeValidateKind, finding.Subject{Kind: finding.SubjectOutput, Name: id},
					k.manifestAt(d.Pointer+"/validate"), finding.Params{"kind": finding.String(kind)}))
			}
		}
		if d.Container != nil {
			out = append(out, k.validateKinds(d.Container.Entries, id)...)
		}
	}
	return out
}

// containers is ADR-005 D9 item 6 and D8 item 4 for outs, the
// declarations of one tree within depth containers: no container nests
// deeper than container-depth (limit.package, ADR-009 D19), and each
// profile is verified as far as it can be without data. An entry whose
// path is literal, or the default of a template, is held to the
// profile's rules on entry paths; and a profile's required entry is
// missing when no entry has it as such a path and no entry's path is
// one only a render can know: a path template, an asset or a mirror.
func (k *checker) containers(outs []manifest.Output, owner string, depth int) []finding.Finding {
	var out []finding.Finding
	for _, d := range outs {
		if d.Container == nil {
			continue
		}
		id := d.ID
		if id == "" {
			id = owner
		}
		if int64(depth) >= k.lim.Value(limits.ContainerDepth) {
			out = append(out, k.lim.Breached(limits.ContainerDepth, *k.manifestAt(d.Pointer + "/container"), int64(depth)+1, true))
			continue
		}
		c := d.Container
		found, unknown := false, false
		for _, e := range c.Entries {
			p, ptr, ok := staticPath(e)
			if !ok {
				unknown = true
				continue
			}
			found = found || p == c.RequiredEntry()
			subj := finding.Subject{Kind: finding.SubjectOutput, Name: cmp.Or(e.ID, id)}
			params := finding.Params{"path": finding.String(p), "profile": finding.String(c.Profile)}
			switch c.CheckEntry(p) {
			case manifest.EntryReserved:
				out = append(out, finding.New(finding.CodeContainerReservedEntry, subj, k.manifestAt(ptr), params))
			case manifest.EntryNonASCII:
				out = append(out, finding.New(finding.CodeContainerNonASCII, subj, k.manifestAt(ptr), params))
			case manifest.EntrySpace:
				out = append(out, finding.New(finding.CodeContainerSpace, subj, k.manifestAt(ptr), params))
			}
		}
		if req := c.RequiredEntry(); req != "" && !found && !unknown {
			out = append(out, finding.New(finding.CodeContainerMissingEntry, finding.Subject{Kind: finding.SubjectOutput, Name: id},
				k.manifestAt(d.Pointer+"/container/profile"), finding.Params{"entry": finding.String(req), "profile": finding.String(c.Profile)}))
		}
		out = append(out, k.containers(c.Entries, id, depth+1)...)
	}
	return out
}

// outputPaths holds every output path the manifest writes literally, in
// the tree of outs, to the portable-path predicate a render holds it to
// (ADR-005 D12), and reports what a render would refuse it with: a path
// is refused by check exactly when every render refuses it, whatever the
// data (PLAN-2026-10-metais-v1 Wave 7, the plan orchestrator's ruling
// that check refuses a literal output path render would refuse). Each
// literal path is checked alone; an asset's or a mirror's literal
// destination is checked too, other than ".", the tree's root, since
// every file it expands into lies beneath it. The literal paths of the
// declarations with no when and no for_each, which every render plans
// once, are checked together, and a container's entries as a tree of
// their own, to container-depth, as a render plans them; a reserved
// entry, which a render plans nowhere, is left out of its tree, and a
// container nested deeper than container-depth is left out whole. in is the
// container whose entries outs are, and nil for the output directory.
func (k *checker) outputPaths(outs []manifest.Output, in *manifest.Container, owner string, depth int) []finding.Finding {
	var out []finding.Finding
	type literal struct {
		path string
		subj finding.Subject
		loc  finding.Location
	}
	var tree []literal
	for _, d := range outs {
		if d.Container != nil && int64(depth) >= k.lim.Value(limits.ContainerDepth) {
			// A render plans nothing of a container nested deeper than
			// container-depth, its own path included; containers reports it.
			continue
		}
		id := cmp.Or(d.ID, owner)
		subj := finding.Subject{Kind: finding.SubjectOutput, Name: id}
		p, ptr, ok := staticPath(d)
		if !ok && (d.Form == manifest.OutputAsset || d.Form == manifest.OutputMirror) && d.Path != "." {
			p, ptr, ok = d.Path, d.Pointer+"/path", d.Path != "" && !strings.Contains(d.Path, "{{") && !strings.Contains(d.Path, "{%")
		}
		if ok {
			loc := *k.manifestAt(ptr)
			problems := unitext.CheckPath(p, k.lim)
			for _, pr := range problems {
				out = append(out, templates.PathFinding(pr, subj, loc, finding.Params{"path": finding.String(p)}, true, k.lim))
			}
			_, _, planned := staticPath(d)
			if planned && d.When == "" && d.ForEach == "" && (in == nil || in.CheckEntry(p) != manifest.EntryReserved) {
				tree = append(tree, literal{p, subj, loc})
			}
		}
		if d.Container != nil {
			out = append(out, k.outputPaths(d.Container.Entries, d.Container, id, depth+1)...)
		}
	}
	paths := make([]string, len(tree))
	for i, l := range tree {
		paths[i] = l.path
	}
	first := map[string]int{}
	for i := len(tree) - 1; i >= 0; i-- {
		first[tree[i].path] = i
	}
	repeats := map[string]int{}
	for _, c := range unitext.Collisions(paths) {
		at, other := first[c.Path], first[c.Other]
		if c.Kind == unitext.Duplicate {
			// The n-th repeat of a path is its (n+1)-th declaration.
			repeats[c.Path]++
			at = nth(paths, c.Path, repeats[c.Path])
		}
		l := tree[at]
		out = append(out, templates.CollisionFinding(c, l.subj, l.loc, finding.Params{"path": finding.String(l.path)}, tree[other].loc))
	}
	return out
}

// nth is the index in paths of p after n others equal to it.
func nth(paths []string, p string, n int) int {
	for i, q := range paths {
		if q == p {
			if n == 0 {
				return i
			}
			n--
		}
	}
	return -1
}

// staticPath is the one path d's file has, and the pointer it is written
// at, when check can know it: a literal path, or a template's default;
// false for a path template, an asset and a mirror, whose files a render
// plans.
func staticPath(d manifest.Output) (string, string, bool) {
	switch {
	case strings.Contains(d.Path, "{{"), strings.Contains(d.Path, "{%"),
		d.Form == manifest.OutputAsset, d.Form == manifest.OutputMirror:
		return "", "", false
	case d.Path != "":
		return d.Path, d.Pointer + "/path", true
	case d.Form == manifest.OutputTemplate:
		return strings.TrimSuffix(d.Source, ".liquid"), d.Pointer + "/template", true
	}
	return "", "", false
}

// staticKind is the kind ADR-005 D7's table gives a template output's
// path, and false when the path's extension is computed.
func staticKind(d manifest.Output) (string, bool) {
	p := d.Path
	if p == "" {
		if d.Form != manifest.OutputTemplate {
			return "", false
		}
		p = strings.TrimSuffix(d.Source, ".liquid")
	}
	if i := max(strings.LastIndex(p, "}}"), strings.LastIndex(p, "%}")); i >= 0 {
		p = p[i+2:]
		if !strings.Contains(p, ".") {
			return "", false
		}
	}
	return outputKind(pathpkg.Base(p)), true
}

// outputKind is ADR-005 D7 item 1's table: the kind of a file named name,
// by its final extension, ASCII-case-insensitively, then by its name.
func outputKind(name string) string {
	ext := strings.ToLower(pathpkg.Ext(name))
	for _, row := range kindTable {
		if slices.Contains(row.exts, ext) || slices.Contains(row.names, name) {
			return row.kind
		}
	}
	return "text"
}

var kindTable = []struct {
	kind        string
	exts, names []string
}{
	{"html", []string{".html", ".htm"}, nil},
	{"xml", []string{".xml", ".xhtml", ".svg", ".rels", ".opf", ".ncx", ".xsd", ".xsl", ".xslt", ".plist"}, nil},
	{"markdown", []string{".md", ".markdown"}, nil},
	{"json", []string{".json", ".geojson", ".webmanifest"}, nil},
	{"yaml", []string{".yaml", ".yml"}, nil},
	{"csv", []string{".csv"}, nil},
	{"css", []string{".css", ".scss", ".less"}, nil},
	{"hash", []string{".sh", ".bash", ".zsh", ".py", ".rb", ".pl", ".r", ".toml", ".ps1", ".properties", ".tf"},
		[]string{"Makefile", "Dockerfile", "Containerfile", ".gitignore", ".gitattributes", ".dockerignore", ".env"}},
	{"slash", []string{".js", ".mjs", ".cjs", ".ts", ".tsx", ".jsx", ".go", ".c", ".h", ".cc", ".cpp", ".hpp", ".java", ".kt", ".kts",
		".swift", ".rs", ".cs", ".scala", ".dart", ".jsonc", ".proto", ".groovy", ".gradle"}, nil},
	{"sql", []string{".sql"}, nil},
}

// schemaStatics is ADR-007 D2 step 5's static checks, over every schema
// object of every schema file: each default and each examples entry
// validates against the object holding it; a const that fails the rest of
// its object is unsatisfiable, and an enum member that does is unusable;
// and a name in required that the object does not declare (D3 item 4) is
// almost always a typo.
func (k *checker) schemaStatics() []finding.Finding {
	s := &statics{c: k.c.Schemas, lim: k.lim, patterns: map[string]*regexprofile.Regexp{}}
	for _, n := range k.c.Tree.List() {
		if !schema.IsSchemaFile(n) {
			continue
		}
		if root, ok := s.c.Lookup(n, ""); ok {
			s.walk(n, "", root)
		}
	}
	return s.out
}

// statics is one run of the schema statics.
type statics struct {
	c        *schema.Compiler
	lim      limits.Set
	patterns map[string]*regexprofile.Regexp
	out      []finding.Finding
}

// The keywords whose value is a schema, an object of schemas, or an array
// of schemas, in 2020-12 (Core §10, Validation §6 and §8).
var (
	schemaKeywords      = []string{"additionalProperties", "contains", "contentSchema", "else", "if", "items", "not", "propertyNames", "then", "unevaluatedItems", "unevaluatedProperties"}
	schemaMapKeywords   = []string{"$defs", "dependentSchemas", "patternProperties", "properties"}
	schemaArrayKeywords = []string{"allOf", "anyOf", "oneOf", "prefixItems"}
)

func (s *statics) walk(entry, ptr string, v value.Value) {
	o, ok := v.(value.Object)
	if !ok {
		return
	}
	if d, ok := o.Get("default"); ok && s.fails(entry, ptr, ptr+"/default", d) {
		s.add(CodeInvalidDefault, entry, ptr+"/default", nil)
	}
	if ex, ok := o.Get("examples"); ok {
		if a, ok := ex.(value.Array); ok {
			for i, e := range a.All() {
				at := ptr + "/examples/" + strconv.Itoa(i)
				if s.fails(entry, ptr, at, e) {
					s.add(CodeInvalidExample, entry, at, nil)
				}
			}
		}
	}
	if cv, ok := o.Get("const"); ok && s.fails(entry, ptr, ptr+"/const", cv) {
		s.add(CodeUnsatisfiable, entry, ptr+"/const", nil)
	}
	if en, ok := o.Get("enum"); ok {
		if a, ok := en.(value.Array); ok {
			for i, e := range a.All() {
				at := ptr + "/enum/" + strconv.Itoa(i)
				if s.fails(entry, ptr, at, e) {
					s.add(CodeUnusableEnumMember, entry, at, nil)
				}
			}
		}
	}
	if rq, ok := o.Get("required"); ok {
		if a, ok := rq.(value.Array); ok {
			for i, e := range a.All() {
				name, ok := e.(value.String)
				if ok && !s.declared(entry, ptr, name.Text()) {
					s.add(CodeRequiredNotDeclared, entry, ptr+"/required/"+strconv.Itoa(i), finding.Params{"name": finding.String(name.Text())})
				}
			}
		}
	}
	for _, kw := range schemaKeywords {
		if sub, ok := o.Get(kw); ok {
			s.walk(entry, ptr+"/"+escapeToken(kw), sub)
		}
	}
	for _, kw := range schemaMapKeywords {
		if m, ok := o.Get(kw); ok {
			if mo, ok := m.(value.Object); ok {
				for name, sub := range mo.All() {
					s.walk(entry, ptr+"/"+escapeToken(kw)+"/"+escapeToken(name), sub)
				}
			}
		}
	}
	for _, kw := range schemaArrayKeywords {
		if a, ok := o.Get(kw); ok {
			if aa, ok := a.(value.Array); ok {
				for i, sub := range aa.All() {
					s.walk(entry, ptr+"/"+kw+"/"+strconv.Itoa(i), sub)
				}
			}
		}
	}
}

// fails reports whether v fails the schema object at ptr in entry, the
// value lying at at. A regular expression that reaches regex-work is a
// limit the package breaches, reported at at.
func (s *statics) fails(entry, ptr, at string, v value.Value) bool {
	sub, err := s.c.Subschema(entry, ptr)
	if err != nil {
		panic(fmt.Sprintf("author: compiling %s#%s, which compiled whole: %v", entry, ptr, err))
	}
	failures, err := sub.Validate(v)
	if err != nil {
		var le *regexprofile.LimitError
		if errors.As(err, &le) {
			l := entryLocation(entry).WithPointer(at)
			s.out = append(s.out, s.lim.Breached(le.Limit, l, 0, false))
			return false
		}
		panic(fmt.Sprintf("author: validating at %s#%s: %v", entry, at, err))
	}
	return len(failures) > 0
}

func (s *statics) add(code finding.Code, entry, ptr string, p finding.Params) {
	l := entryLocation(entry).WithPointer(ptr)
	s.out = append(s.out, finding.New(code, pkgSubject, &l, p))
}

// declared reports whether the schema object at ptr in entry declares the
// property name by ADR-007 D3 item 4: some node of its applicable set is
// open, or has name in properties, a pattern of patternProperties that
// matches it, or an additionalProperties other than false — or, with no
// additionalProperties, an unevaluatedProperties other than false.
func (s *statics) declared(entry, ptr, name string) bool {
	type node struct{ entry, ptr string }
	seen := map[node]bool{}
	var visit func(n node) bool
	visit = func(n node) bool {
		if seen[n] {
			return false
		}
		seen[n] = true
		v, ok := s.c.Lookup(n.entry, n.ptr)
		if !ok {
			return true
		}
		if isOpen(v) {
			return true
		}
		o, ok := v.(value.Object)
		if !ok {
			return false
		}
		if p, ok := o.Get("properties"); ok {
			if po, ok := p.(value.Object); ok {
				if _, ok := po.Get(name); ok {
					return true
				}
			}
		}
		if pp, ok := o.Get("patternProperties"); ok {
			if po, ok := pp.(value.Object); ok {
				for pat := range po.All() {
					if s.matches(pat, name) {
						return true
					}
				}
			}
		}
		if ap, ok := o.Get("additionalProperties"); ok {
			if ap != value.Bool(false) {
				return true
			}
		} else if up, ok := o.Get("unevaluatedProperties"); ok && up != value.Bool(false) {
			return true
		}
		if e, p, ok := s.c.Ref(n.entry, n.ptr); ok && visit(node{e, p}) {
			return true
		}
		if e, p, ok := s.c.DynamicRef(n.entry, n.ptr); ok && visit(node{e, p}) {
			return true
		}
		for _, kw := range []string{"if", "then", "else"} {
			if _, ok := o.Get(kw); ok && visit(node{n.entry, n.ptr + "/" + kw}) {
				return true
			}
		}
		for _, kw := range []string{"allOf", "anyOf", "oneOf"} {
			if a, ok := o.Get(kw); ok {
				if aa, ok := a.(value.Array); ok {
					for i := range aa.Len() {
						if visit(node{n.entry, n.ptr + "/" + kw + "/" + strconv.Itoa(i)}) {
							return true
						}
					}
				}
			}
		}
		if d, ok := o.Get("dependentSchemas"); ok {
			if do, ok := d.(value.Object); ok {
				for dn := range do.All() {
					if visit(node{n.entry, n.ptr + "/dependentSchemas/" + escapeToken(dn)}) {
						return true
					}
				}
			}
		}
		return false
	}
	return visit(node{entry, ptr})
}

// matches reports whether the patternProperties pattern pat matches name,
// by the package regular-expression profile the compiler compiled it with.
func (s *statics) matches(pat, name string) bool {
	re, ok := s.patterns[pat]
	if !ok {
		var err error
		re, err = regexprofile.Compile(pat, s.lim, nil)
		if err != nil {
			// The compiler compiled the pattern under the same limits.
			panic(fmt.Sprintf("author: compiling the pattern %q, which compiled: %v", pat, err))
		}
		s.patterns[pat] = re
	}
	return re.MatchString(name)
}

// annotations are the keywords an open schema holds (ADR-007 D3 item 4.2).
var annotations = map[string]bool{
	"title": true, "description": true, "$comment": true, "examples": true, "default": true, "deprecated": true,
	"readOnly": true, "writeOnly": true, "$id": true, "$anchor": true, "$schema": true, "$defs": true,
}

// isOpen reports whether a schema is open: the boolean schema true, or an
// object schema holding annotations only (ADR-007 D3 item 4.2).
func isOpen(v value.Value) bool {
	switch s := v.(type) {
	case value.Bool:
		return bool(s)
	case value.Object:
		for name := range s.All() {
			if !annotations[name] {
				return false
			}
		}
		return true
	}
	return false
}

// escapeToken escapes a JSON Pointer reference token (RFC 6901 §3).
func escapeToken(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "~", "~0"), "/", "~1")
}

// template is one template as step 6 read it.
type template struct {
	path  string // relative to templates/
	entry string // its entry name
	src   []byte
	root  *liquid.Template
}

// templates is ADR-007 D2 step 6 for every file under templates/: it is
// valid UTF-8, it parses with the one parse the render uses (ADR-004 D16),
// a parse error naming its cause (D2 item 4), and a CR LF in it is a
// warning.
func (k *checker) templates() []template {
	var ts []template
	for _, n := range k.c.Tree.List() {
		if !strings.HasPrefix(n, templatesDir) {
			continue
		}
		src, err := readEntry(k.c, n)
		if err != nil {
			// The read already read every entry whole.
			panic(fmt.Sprintf("author: reading %s again: %v", n, err))
		}
		ts = append(ts, template{path: strings.TrimPrefix(n, templatesDir), entry: n, src: src})
	}
	ots := make([]ooxml.Template, len(ts))
	for i, t := range ts {
		ots[i] = ooxml.Template{Path: t.path, Text: t.src}
	}
	scanned := ooxml.Scanned(ots)
	for i := range ts {
		t := &ts[i]
		subj := finding.Subject{Kind: finding.SubjectTemplate, Name: t.entry}
		if off := bytes.Index(t.src, []byte("\r\n")); off >= 0 && utf8.Valid(t.src) {
			l := at(t.entry, t.src, off)
			k.add(finding.New(CodeTemplateCRLF, subj, &l, nil))
		}
		// The one parse render makes of the same template (W4-3.12).
		root, fs := templates.ParseEntry(t.entry, t.src, k.lim, scanned[i])
		k.add(fs...)
		if root != nil && k.c.Manifest != nil {
			// The embed tags' index: as the declarations decide it; a name
			// no input has is the cross-reference's (ADR-007 D3 item 12).
			k.add(templates.EmbedFindings(t.entry, root, k.c.Manifest.Inputs)...)
		}
		t.root = root
	}
	return ts
}

// readEntry reads the entry name of c whole.
func readEntry(c *loader.Checked, name string) ([]byte, error) {
	r, err := c.Tree.Open(name)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

// at is the location of byte off of the entry name's source src: its
// 1-based line, and its column in code points (ADR-006 D5 item 3), an
// invalid byte counting as one.
func at(name string, src []byte, off int) finding.Location {
	line, start := 1, 0
	for i := 0; i < off; i++ {
		if src[i] == '\n' {
			line, start = line+1, i+1
		}
	}
	col := 1
	for i := start; i < off; {
		_, n := utf8.DecodeRune(src[i:])
		i += n
		col++
	}
	return finding.Location{Origin: finding.OriginPackage, File: name, Line: line, Column: col}
}

// xref is ADR-007 D2 step 7 (D3), over every template that parsed and
// every manifest expression. With no combined schema, every input is
// unknown and every other reference is still checked.
func (k *checker) xref(ts []template) []finding.Finding {
	c := k.c
	p := xref.Package{
		Manifest:    c.Manifest,
		Positions:   c.Positions,
		Schema:      c.CombinedValue,
		NoSchemas:   c.Combined == nil,
		Dialect:     loader.Dialect(),
		ParseLimits: loader.ParseLimits(k.lim),
		Limits:      k.lim,
	}
	for _, t := range ts {
		xt := xref.Template{Path: t.path}
		if t.root != nil {
			xt.Root = t.root.Root
		}
		p.Templates = append(p.Templates, xt)
	}
	for _, n := range c.Tree.List() {
		if strings.HasPrefix(n, assetsDir) {
			p.Assets = append(p.Assets, strings.TrimPrefix(n, assetsDir))
		}
	}
	r, err := xref.Analyze(p)
	if err != nil {
		// The loader made the package xref reads: a refusal is a defect.
		panic(fmt.Sprintf("author: %v", err))
	}
	return r.Findings
}

// examples is ADR-007 D2 step 9 (D5 item 1): an example directory is
// named as an input is, and holds inputs/, expected/ and example.yaml and
// nothing else. Validating and rendering them is test's.
func (k *checker) examples() []finding.Finding {
	var out []finding.Finding
	badName := map[string]bool{}
	for _, n := range k.c.Tree.List() {
		rest, ok := strings.CutPrefix(n, examplesDir)
		if !ok {
			continue
		}
		name, inside, dir := strings.Cut(rest, "/")
		subj := finding.Subject{Kind: finding.SubjectExample, Name: name}
		l := entryLocation(n)
		if !dir {
			out = append(out, finding.New(CodeExampleLayout, subj, &l, finding.Params{"reason": finding.String("entry")}))
			continue
		}
		if !manifest.ValidInputName(name) {
			if !badName[name] {
				badName[name] = true
				out = append(out, finding.New(CodeExampleLayout, subj, &l, finding.Params{"reason": finding.String("name")}))
			}
			continue
		}
		if inside == exampleFile || strings.HasPrefix(inside, examplesInputs) || strings.HasPrefix(inside, examplesExpected) {
			continue
		}
		out = append(out, finding.New(CodeExampleLayout, subj, &l, finding.Params{"reason": finding.String("entry")}))
	}
	return out
}
