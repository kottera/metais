// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"errors"
	"io"
	"maps"
	pathpkg "path"
	"path/filepath"
	"slices"
	"strings"

	"metais.org/metais/internal/cli/args"
	"metais.org/metais/internal/dates"
	"metais.org/metais/internal/digest"
	"metais.org/metais/internal/fault"
	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/hostfs"
	"metais.org/metais/internal/inputs"
	"metais.org/metais/internal/jsonx"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/loader"
	"metais.org/metais/internal/manifest"
	"metais.org/metais/internal/render"
	"metais.org/metais/internal/render/sink"
	"metais.org/metais/internal/template"
	"metais.org/metais/internal/yamlx"
)

// The example layout (ADR-007 D5 item 1).
const (
	examplesDir     = "examples/"
	exampleInputs   = "inputs/"
	exampleExpected = "expected/"
	exampleMetadata = "example.yaml"
)

// testTool is the provenance tool test renders under (ADR-004 D15 item 1).
const testTool = "test"

// The steps test's refusals and reports point to.
var (
	nextTestAgain = Next{Run: "metais test <package>",
		Why: "Tests every example again once the package or its examples are fixed."}
	nextTestUpdate = Next{Run: "metais test <package> --update --dry-run",
		Why: "Lists what --update would write into, and remove from, each example's expected/ to make it the render."}
	nextTestNames = Next{Run: "metais test <package> --example <name>",
		Why: "Names an example that exists: a directory under examples/."}
	nextTestSource = Next{Run: "metais test <dir> --update",
		Why: "Updates the examples of the package's source directory, which an author edits; a packed package is not changed."}
	nextTestDryRun = Next{Run: "metais test <package> --update --dry-run",
		Why: "Lists what --update would write and remove, and writes nothing."}
	nextTestWrite = Next{Run: "metais test <dir> --update",
		Why: "Updates the examples again once the cause is fixed; the files already written or removed are listed."}
	nextPack = Next{Run: "metais pack <package>", Why: "Packs the source directory, which checks and tests it again first."}
)

// testDeclaration is metais test (ADR-008 D9, ADR-007 D5 and D6).
func testDeclaration() *Command {
	return &Command{
		Name: "test", Group: "Authoring",
		Synopsis:    "test [<package>] [--example <name>]... [--update [--dry-run]] [--limit <name>=<value>]... [--json]",
		Description: "Render every example of a package and compare it with the outputs it should produce.",
		Detail: "The package is a source directory, the current directory when omitted, or a packed package " +
			"file, whose examples are read in place. The package is opened and verified whole, and every " +
			"template is parsed; an error there is reported and no example runs.\n\n" +
			"An example is a directory under examples/: inputs/ is a data directory, read as --data reads " +
			"one; expected/ is the output directory a render should write; example.yaml may give a title, a " +
			"description, and now, the render time. Every example runs, or each one --example names, in " +
			"byte order of name, whatever another found. Its inputs are validated, with findings located in " +
			"examples/<name>/inputs/; if they are valid it is rendered in memory as metais render renders, " +
			"with now from example.yaml or none, and nothing else of the host. The package digest a template " +
			"sees, and the provenance tag's package_digest, are sha256: and 64 zeros, and the provenance tool " +
			"is test, so an expected output records nothing that changes with the package or the tool.\n\n" +
			"When expected/ exists, its files are compared with the render's by path, byte for byte but for a container; " +
			"directories are ignored. A file the render did not write is example.output_missing, one it " +
			"wrote that expected/ lacks is example.output_unexpected, and different content is " +
			"example.output_differs: for text, a unified diff, of at most 40 lines here and 16 KiB in --json; " +
			"for a binary file, the first differing byte and both sizes and digests. Its cause names the " +
			"first likely explanation that applies: line endings, one final newline, a byte order mark, or " +
			"trailing whitespace.\n\n" +
			"A container output is compared by its entries, not its zip's bytes: their names in order, " +
			"each entry's uncompressed content, and the method of each entry declared stored: true and of a " +
			"profile's mimetype; any other entry's method is the encoder's choice and is not compared. The " +
			"expected container is read as a packed package is, with its bounds and profile, but in its " +
			"entries' own order, and one that does not read is example.expected_unreadable. An entry " +
			"expected and not written is example.entry_missing, one written and not expected " +
			"example.entry_unexpected, entries in another order example.entry_order, at the first index " +
			"where they differ, a stored entry the expected container deflates example.entry_method, and " +
			"different content example.output_differs, located at <output>!<entry>; a container entry is " +
			"compared by its own entries. An example without expected/ is rendered and not compared.\n\n" +
			"--update, on a source directory only, makes each selected example's expected/ exactly its " +
			"render: every file that is missing or differs is written, through a staging directory inside " +
			"expected/, and every file there the render no longer writes is removed, with any directory that " +
			"empties. Only an example whose inputs validated and that rendered is updated, and nothing " +
			"outside examples/<name>/expected/ is ever removed. Each file written or removed is listed; with " +
			"--dry-run they are listed and nothing is written.\n\n" +
			"test exits 1 when any example fails.",
		Role: finding.RoleAuthoring,
		Operands: []args.Operand{{Name: "<package>",
			Description: "A source directory, or a packed package file; the current directory when omitted. - is refused, since a package is read with random access."}},
		Flags: []args.Flag{
			{Name: "example", Value: "<name>", Repeatable: true, Description: "Test only this example; repeatable. Without it, every example is tested."},
			{Name: "update", Description: "Make each selected example's expected/ exactly its render, writing and removing files there; a source directory only."},
			{Name: "dry-run", Description: "With --update, list the files it would write and remove, and write nothing."},
			limitFlag, failOnFlag,
		},
		Format:            "metais.test/1",
		FindingsAreResult: true,
		Prints:            "Every finding, then a line counting the errors, warnings and notes, then one line per example with its status and each difference, a text difference's diff beneath it; with --update, each file written or removed.",
		Refusals:          testRefusals(),
		Examples: []Example{
			{"metais test", "Tests every example of the source directory in the working directory."},
			{"metais test report-1.0.0.metais --example basic", "Tests the example basic of a packed package."},
			{"metais test --update --dry-run", "Lists what --update would write into and remove from each example's expected/."},
			{"metais test --json", "Prints every finding and each example's result as one metais.test/1 document."},
		},
		run: runTest,
	}
}

// testRefusals are the codes test declares (ADR-008 D16 item 2): the
// operand's and the limits', every code a package's own read refuses it
// with, every template parse code, its own usage and destination codes,
// and every code an example's inputs, render and comparison report.
func testRefusals() []Refusal {
	var out []Refusal
	for _, r := range packageRefusals("test") {
		if r.Code == finding.CodeUsageMissingOperand {
			continue
		}
		if r.Next == nextInspectAgain {
			r.Next = nextTestAgain
		}
		out = append(out, r)
	}
	out = append(out,
		Refusal{Code: finding.CodeUsageUnknownExample, Cause: "An --example names no example of the package.", Next: nextTestNames},
		Refusal{Code: finding.CodeUsageNotSourceDir, Cause: "--update is given with a packed package, which is not edited.", Next: nextTestSource},
		Refusal{Code: finding.CodeUsageMissingFlag, Cause: "--dry-run is given without --update.", Next: nextTestDryRun},
		Refusal{Code: finding.CodeDestinationNotDirectory, Cause: "An example's expected/, or a directory in it, is not a directory where --update writes.", Next: nextTestWrite},
		Refusal{Code: finding.CodeDestinationWriteFailed, Cause: "--update's write into, or removal from, an example's expected/ failed after it had written or removed a file; what it wrote and removed is listed.", Next: nextTestWrite},
		Refusal{Code: finding.CodeDestinationNothingWritten, Cause: "--update's first write into, or removal from, an example's expected/ failed; nothing was written or removed.", Next: nextTestWrite},
		Refusal{Code: finding.CodeDestinationStagingExists, Cause: "An example's expected/ holds .metais-staging, left by a killed write or in use by another, before --update had written or removed anything.", Next: nextTestWrite},
	)
	have := map[finding.Code]bool{}
	for _, r := range out {
		have[r.Code] = true
	}
	for _, e := range finding.Registry() {
		family, rest, _ := strings.Cut(string(e.Code), ".")
		var what, words string
		switch {
		case have[e.Code]:
			continue
		case family == "example" && e.Code != "example.layout":
			what, words = "An example fails", strings.ReplaceAll(rest, "_", " ")
		case family == "input" || family == "json" || family == "yaml" || family == "csv" || family == "text" ||
			family == "markdown" || family == "collection" || e.Code == finding.CodeLimitExceeded:
			what, words = "An example's inputs are refused", strings.ReplaceAll(strings.ReplaceAll(rest, "_", " "), ".", ": ")
		case family == "template" && e.Severity == finding.SeverityError && !strings.HasPrefix(rest, "xref.") && !strings.HasPrefix(rest, "ooxml."):
			what, words = "A template is refused", strings.ReplaceAll(rest, "_", " ")
		case strings.HasPrefix(string(e.Code), "limit.exceeded."), strings.HasPrefix(string(e.Code), "output.path."),
			strings.HasPrefix(string(e.Code), "output.expression."), strings.HasPrefix(string(e.Code), "output.container."):
			what, words = "An example's output is refused", strings.ReplaceAll(strings.ReplaceAll(rest, "_", " "), ".", ": ")
		case family == "output" && e.Subject == finding.SubjectOutput:
			// An output its kind's reader or its schema refuses, and a
			// validate its rendered path's kind cannot take (ADR-005 D11).
			what, words = "An example's output fails its validation", strings.ReplaceAll(strings.ReplaceAll(rest, "_", " "), ".", ": ")
		default:
			continue
		}
		have[e.Code] = true
		out = append(out, familyRefusal(e.Code, what, words, nextTestAgain))
	}
	return out
}

// testResult is metais.test/1's result (ADR-008 D9 test item 2), with
// --update's files.
type testResult struct {
	Examples []testExample `json:"examples"`
	// Update is --update's record, or nil without it.
	Update *testUpdate `json:"update"`
}

// testExample is one example's result: its status is pass, fail or
// no-expected.
type testExample struct {
	Name        string           `json:"name"`
	Status      string           `json:"status"`
	Differences []testDifference `json:"differences"`
}

// testDifference is one path the comparison found different: kind is
// missing, unexpected or differs, and line is a text difference's first
// differing line in the expected file.
type testDifference struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
	Line *int   `json:"line,omitempty"`
}

// testUpdate is what --update wrote and removed, or with --dry-run would:
// each file by its path within the source directory, in the order written
// or removed.
type testUpdate struct {
	DryRun  bool     `json:"dry_run"`
	Written []string `json:"written"`
	Removed []string `json:"removed"`
}

// exampleRun is one example as test runs it.
type exampleRun struct {
	name string
	dir  string // its directory in the package, examples/<name>/
	// expected holds the paths of expected/'s files, relative to it, in
	// byte order; hasExpected is whether expected/ exists.
	expected    []string
	hasExpected bool
	// others holds every other path of the example, relative to its
	// directory: its inputs and example.yaml, which --update never removes.
	others []string
	// rendered holds the bytes of every file the render wrote, by path;
	// ok is whether the inputs validated and the render made no error.
	rendered map[string][]byte
	exec     map[string]bool
	// rewrite holds the paths of the rendered files the comparison found
	// different from expected/'s or absent from it: those --update writes.
	rewrite []string
	ok      bool
	result  testExample
	diffs   []string // the human report of each difference, in order
	fs      []finding.Finding
}

// runTest is metais test (ADR-007 D5 items 2 to 5, D6).
func runTest(s *session, p args.Parsed) outcome {
	operand, operandArg := ".", -1
	if len(p.Operands) > 0 {
		operand, operandArg = p.Operands[0], s.line.at[p.OperandArgs[0]]+1
		if operand == "-" {
			return s.refuseWith(finding.New(finding.CodeUsagePackageFromStdin, invocation, nil,
				finding.Params{"argument": finding.Int(int64(operandArg))}))
		}
	}
	update, dryRun := p.Has("update"), p.Has("dry-run")
	if dryRun && !update {
		return s.refuseWith(finding.New(finding.CodeUsageMissingFlag, invocation, nil,
			finding.Params{"command": finding.String("test"), "flag": finding.String("--update")}))
	}
	lim, fs := s.limits(p)
	if len(fs) > 0 {
		return outcome{findings: fs, next: s.nextSteps(fs)}
	}
	pkg, fs := loader.Open(operand, loader.Options{Limits: lim}, s.inv.Options.Faults)
	if pkg == nil {
		return outcome{findings: fs, next: s.nextSteps(fs)}
	}
	defer pkg.Close()
	if update && pkg.Form != loader.Directory {
		// The open runs first, so a hostile package is refused as every
		// opening command refuses it (ADR-011 D16 item 3).
		at := operandArg
		if at < 0 {
			i, _ := p.Arg("update")
			at = s.line.at[i] + 1
		}
		return s.refuseWith(finding.New(finding.CodeUsageNotSourceDir, invocation, nil,
			finding.Params{"argument": finding.Int(int64(at))}))
	}
	all := exampleRuns(pkg)
	selected := all
	if names := p.Values("example"); len(names) > 0 {
		selected = nil
		for i, n := range names {
			k := slices.IndexFunc(all, func(e *exampleRun) bool { return e.name == n })
			if k < 0 {
				listed := make([]string, len(all))
				for j, e := range all {
					listed[j] = e.name
				}
				// The value's own position: the flag's, written
				// --example=name, or the one after it.
				flag := s.line.at[p.ArgsOf("example")[i]]
				at := flag + 1
				if !strings.Contains(s.inv.Args[flag], "=") {
					at++
				}
				return s.refuseWith(finding.New(finding.CodeUsageUnknownExample, invocation, nil, finding.Params{
					"argument": finding.Int(int64(at)),
					"examples": stringList(listed),
				}))
			}
			if !slices.Contains(selected, all[k]) {
				selected = append(selected, all[k])
			}
		}
		slices.SortFunc(selected, func(a, b *exampleRun) int { return strings.Compare(a.name, b.name) })
	}

	pfs, ran := s.runExamples(pkg, lim, selected)
	if !ran {
		return outcome{findings: append(fs, pfs...), result: &testResult{Examples: []testExample{}}, report: true, next: []Next{nextTestAgain}}
	}
	fs = append(fs, pfs...)
	result := &testResult{Examples: []testExample{}}
	var upd *testUpdate
	if update {
		u, f, ok := s.update(operand, selected, dryRun)
		if !ok {
			// The destination's failure refuses the command; what was
			// written and removed before it is listed.
			for _, e := range selected {
				fs = append(fs, e.fs...)
			}
			return outcome{findings: append(fs, f), next: s.nextSteps([]finding.Finding{f}), human: []byte(updateLines(u))}
		}
		result.Update, upd = u, u
	}
	var human strings.Builder
	failed := false
	for _, e := range selected {
		if update && !dryRun && e.ok {
			// The update made expected/ the render: the example passes,
			// and its differences are the record of what changed.
			e.fs = slices.DeleteFunc(e.fs, func(f finding.Finding) bool {
				c := f.Code()
				return slices.Contains(sink.ComparisonCodes, c)
			})
			e.result.Status, e.diffs = "pass", nil
		}
		fs = append(fs, e.fs...)
		result.Examples = append(result.Examples, e.result)
		failed = failed || e.result.Status == "fail"
		human.WriteString(e.name + ": " + e.result.Status + "\n")
		for _, d := range e.diffs {
			human.WriteString(d)
		}
	}
	human.WriteString(updateLines(upd))
	o := outcome{findings: fs, result: result, report: true, human: []byte(human.String())}
	switch {
	case failed || s.fails(fs):
		o.next = []Next{nextTestAgain}
		if !update && slices.ContainsFunc(selected, func(e *exampleRun) bool { return len(e.result.Differences) > 0 }) && pkg.Form == loader.Directory {
			o.next = append(o.next, nextTestUpdate)
		}
	case lookup(s.table, "pack") != nil && pkg.Form == loader.Directory:
		o.next = []Next{nextPack}
	}
	return o
}

// runExamples runs each example of selected over pkg under lim. The
// package's templates are parsed once, and its output validation schemas
// compiled once; their findings are returned, and an error there stops
// test before any example runs, which runExamples reports with false
// (ADR-007 D5 item 2, ADR-005 D11 item 3).
func (s *session) runExamples(pkg *loader.Package, lim limits.Set, selected []*exampleRun) ([]finding.Finding, bool) {
	ts := template.NewTemplates(pkg.Tree, lim, pkg.Manifest.Inputs)
	pfs := parseTemplates(pkg.Manifest.Outputs, ts)
	schemas, sfs := render.OutputSchemas(pkg, lim)
	if pfs = append(pfs, sfs...); s.fails(pfs) {
		return pfs, false
	}
	for _, e := range selected {
		s.runExample(pkg, ts, schemas, lim, e)
	}
	return pfs, true
}

// exampleRuns is the examples of pkg, in byte order of name (ADR-007 D5
// item 1): each directory under examples/ whose name is an input's. An
// entry directly under examples/, and a directory otherwise named, is not
// an example; check reports it.
func exampleRuns(pkg *loader.Package) []*exampleRun {
	var out []*exampleRun
	for _, n := range pkg.Tree.List() {
		rest, ok := strings.CutPrefix(n, examplesDir)
		if !ok {
			continue
		}
		name, inside, isDir := strings.Cut(rest, "/")
		if !isDir || !manifest.ValidInputName(name) {
			continue
		}
		if len(out) == 0 || out[len(out)-1].name != name {
			out = append(out, &exampleRun{name: name, dir: examplesDir + name + "/"})
		}
		e := out[len(out)-1]
		if p, ok := strings.CutPrefix(inside, exampleExpected); ok {
			e.expected, e.hasExpected = append(e.expected, p), true
		} else {
			e.others = append(e.others, inside)
		}
	}
	return out
}

// parseTemplates parses every template an output of decls reaches, at
// any depth of containers, each once, and returns every parse finding.
func parseTemplates(decls []manifest.Output, ts *template.Templates) []finding.Finding {
	var fs []finding.Finding
	seen := map[string]bool{}
	var walk func(decls []manifest.Output)
	walk = func(decls []manifest.Output) {
		for _, d := range decls {
			if d.Container != nil {
				walk(d.Container.Entries)
			}
			if d.Form != manifest.OutputTemplate {
				continue
			}
			for _, name := range ts.Reach(d.Source) {
				if !seen[name] {
					seen[name] = true
					_, pfs := ts.Parse(name)
					fs = append(fs, pfs...)
				}
			}
		}
	}
	walk(decls)
	return fs
}

// runExample runs e (ADR-007 D5 item 2): its example.yaml, its inputs,
// and its render under the fixed identity (ADR-004 D15) to the comparison
// sink, which holds it in memory and compares it with expected/ (items 3
// and 4). Every finding is e's.
func (s *session) runExample(pkg *loader.Package, ts *template.Templates, schemas *render.Schemas, lim limits.Set, e *exampleRun) {
	e.result = testExample{Name: e.name, Status: "fail", Differences: []testDifference{}}
	now, fs := exampleNow(pkg, e, lim)
	e.fs = append(e.fs, fs...)
	if len(fs) > 0 {
		return
	}
	layer, err := inputs.New(inputs.Package{Manifest: pkg.Manifest, Compiler: pkg.Schemas, Combined: pkg.CombinedValue, Schema: pkg.Combined})
	if err != nil {
		panic("cli: " + err.Error())
	}
	sup := layer.NewSupply(lim)
	sup.ReadPackageDir(pkg.Tree, e.dir+exampleInputs)
	res, err := sup.Validate()
	if err != nil {
		panic("cli: " + err.Error())
	}
	e.fs = append(e.fs, res.Findings...)
	if s.fails(res.Findings) {
		return
	}
	b, err := template.NewBindings(template.Package{
		Tree: pkg.Tree, Manifest: pkg.Manifest, ManifestValue: pkg.ManifestValue,
		Digest: digest.Sum{}, Combined: pkg.CombinedValue, Tool: testTool,
	}, res.Inputs, now, lim)
	if err != nil {
		panic("cli: " + err.Error())
	}
	c := sink.Compare(pkg.Tree, lim, e.name, e.dir+exampleExpected, e.expected, e.hasExpected)
	rr := render.Run(render.Options{
		Package: pkg, Templates: ts, Bindings: b, Limits: lim, Seams: s.inv.Options.Faults,
		Schemas: schemas, Fails: s.fails,
	}, c)
	e.fs = append(e.fs, rr.Findings...)
	if !rr.Produced {
		return
	}
	e.ok, e.rendered, e.exec = true, c.Rendered, c.Executable
	if !e.hasExpected {
		e.result.Status = "no-expected"
		return
	}
	e.fs = append(e.fs, c.Findings...)
	for _, d := range c.Differences {
		e.result.Differences = append(e.result.Differences, testDifference{Path: d.Path, Kind: d.Kind, Line: d.Line})
	}
	e.diffs, e.rewrite = c.Human, c.Rewrite
	if len(e.result.Differences) == 0 {
		e.result.Status = "pass"
	}
}

// exampleNow reads e's example.yaml, when it has one (ADR-007 D5 item 1):
// YAML, or JSON when its first byte other than white space is "{", under
// the profiles metais.yaml is read under, a mapping of title, description
// and now and nothing else. It returns now, or "" when there is none, and
// example.metadata for whatever breaks the rule.
func exampleNow(pkg *loader.Package, e *exampleRun, lim limits.Set) (string, []finding.Finding) {
	entry := e.dir + exampleMetadata
	info, err := pkg.Tree.Stat(entry)
	if err != nil {
		return "", nil
	}
	subj := finding.Subject{Kind: finding.SubjectExample, Name: e.name}
	at := finding.Location{Origin: finding.OriginPackage, File: entry}
	if info.Size > lim.Value(limits.ManifestSize) {
		// manifest-size bounds the file in either form, and is checked
		// before it is read (ADR-009 Amendment A6 item 2); its reader
		// would refuse it as it refuses a syntax error.
		return "", []finding.Finding{finding.New(codeExampleMetadata, subj, &at, finding.Params{"reason": finding.String("syntax")})}
	}
	src, err := readEntry(pkg, entry, info.Size)
	if err != nil {
		panic("cli: reading " + entry + ": " + err.Error())
	}
	bad := func(reason, member string, pos finding.Positions, ptr string) []finding.Finding {
		l := at
		if pos != nil {
			if p, ok := pos.Value(ptr); ok {
				l = l.At(p)
			}
		}
		ps := finding.Params{"reason": finding.String(reason)}
		if member != "" {
			ps["member"] = finding.String(member)
		}
		return []finding.Finding{finding.New(codeExampleMetadata, subj, &l, ps)}
	}
	var root value.Value
	var pos finding.Positions
	var pfs []finding.Finding
	if isJSONText(src) {
		doc, fs := jsonx.Parse(src, jsonx.Source{Input: finding.Subject{Kind: finding.SubjectInput, Name: exampleMetadata}, At: at, Size: limits.ManifestSize}, lim)
		root, pos, pfs = doc.Value, doc.Positions(), fs
	} else {
		doc, fs := yamlx.Parse(src, yamlx.Source{Subject: finding.Subject{Kind: finding.SubjectInput, Name: exampleMetadata}, At: at, Size: limits.ManifestSize}, lim)
		root, pos, pfs = doc.Value, doc.Positions(), fs
	}
	if len(pfs) > 0 {
		l := at
		if fl, ok := pfs[0].At(); ok {
			l = fl
			l.Pointer = ""
		}
		return "", []finding.Finding{finding.New(codeExampleMetadata, subj, &l, finding.Params{"reason": finding.String("syntax")})}
	}
	obj, ok := root.(value.Object)
	if !ok {
		return "", bad("not_an_object", "", pos, "")
	}
	now := ""
	for name, v := range obj.All() {
		ptr := "/" + strings.ReplaceAll(strings.ReplaceAll(name, "~", "~0"), "/", "~1")
		if name != "title" && name != "description" && name != "now" {
			return "", bad("member", name, pos, ptr)
		}
		str, ok := v.(value.String)
		if !ok {
			return "", bad("type", name, pos, ptr)
		}
		text := str.Text()
		switch {
		case name == "title" && strings.ContainsAny(text, "\r\n\u0085\u2028\u2029"):
			return "", bad("title", name, pos, ptr)
		case name == "now":
			if _, err := dates.ParseNow(text); err != nil {
				return "", bad("now", name, pos, ptr)
			}
			now = text
		}
	}
	return now, nil
}

// codeExampleMetadata is example.yaml's code (ADR-007 D5 item 1).
const codeExampleMetadata finding.Code = "example.metadata"

// isJSONText is ADR-002 D11 item 3's test: the first byte other than
// U+0020, U+0009, U+000A and U+000D is "{".
func isJSONText(src []byte) bool {
	t := bytes.TrimLeft(src, " \t\n\r")
	return len(t) > 0 && t[0] == '{'
}

// readEntry reads the package entry name of size bytes. The caller has
// admitted size under the limit that bounds the entry: entry-size, which
// the package's open checks of every entry, or a tighter one of the
// entry's own (ADR-009 Amendment A6 item 2).
func readEntry(pkg *loader.Package, name string, size int64) ([]byte, error) {
	r, err := pkg.Tree.Open(name)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	b := make([]byte, size)
	_, err = io.ReadFull(r, b)
	return b, err
}

// update is --update (ADR-007 D5 item 5): for each of examples that
// validated and rendered, it removes each file of expected/ the render no
// longer writes, with each directory of expected/ that empties, and writes
// each file that is missing or differs, through the write shape of an
// existing directory, or of a new one when expected/ does not exist yet.
// Nothing outside examples/<name>/expected/ is removed. With dryRun it
// lists what it would do and does nothing. It reports what it wrote and
// removed, in that order, and on a failure, the finding that stopped it.
func (s *session) update(source string, examples []*exampleRun, dryRun bool) (*testUpdate, finding.Finding, bool) {
	u := &testUpdate{DryRun: dryRun, Written: []string{}, Removed: []string{}}
	for _, e := range examples {
		if !e.ok {
			continue
		}
		exampleHost := filepath.Join(source, filepath.FromSlash(strings.TrimSuffix(e.dir, "/")))
		expectedHost := filepath.Join(exampleHost, "expected")
		// What is removed is reckoned over expected/ alone, and removed
		// beneath it; the fault reckons it over the whole example.
		base, prefix, within := expectedHost, e.dir+exampleExpected, e.expected
		produced := func(n string) bool { _, ok := e.rendered[n]; return ok }
		if s.inv.Options.Faults.Has(fault.UpdateBeyondExpected) {
			base, prefix, within = exampleHost, e.dir, nil
			for _, n := range e.expected {
				within = append(within, exampleExpected+n)
			}
			within = append(within, e.others...)
			produced = func(n string) bool {
				p, ok := strings.CutPrefix(n, exampleExpected)
				_, done := e.rendered[p]
				return ok && done
			}
		}
		var stale, writes []string
		for _, n := range within {
			if !produced(n) {
				stale = append(stale, n)
			}
		}
		writes = append(writes, e.rewrite...)
		if !e.hasExpected {
			writes = slices.Sorted(maps.Keys(e.rendered))
		}
		if dryRun {
			for _, n := range writes {
				u.Written = append(u.Written, e.dir+exampleExpected+n)
			}
			for _, n := range stale {
				u.Removed = append(u.Removed, prefix+n)
			}
			continue
		}
		for _, n := range stale {
			if err := hostfs.RemoveFile(base, n); err != nil {
				return u, updateFailed(err, u), false
			}
			u.Removed = append(u.Removed, prefix+n)
		}
		// Ruling R-W5-11: the directories removed are only those beneath
		// expected/, never expected/ itself, that its own removals emptied.
		for _, d := range emptied(stale, within) {
			if err := hostfs.RemoveEmptyDir(base, d); err != nil {
				return u, updateFailed(err, u), false
			}
		}
		if len(writes) == 0 {
			continue
		}
		committed, err := writeExpected(expectedHost, e, writes, s.inv.Options.Faults)
		for _, n := range committed {
			u.Written = append(u.Written, e.dir+exampleExpected+n)
		}
		if err != nil {
			return u, updateFailed(err, u), false
		}
	}
	return u, finding.Finding{}, true
}

// emptied is every directory of within, deepest first, that holds nothing
// once stale is removed from it: a source directory holds no empty
// directory (ADR-001 D2 item 6).
func emptied(stale, within []string) []string {
	left := map[string]bool{}
	for _, n := range within {
		if !slices.Contains(stale, n) {
			for d := pathpkg.Dir(n); d != "."; d = pathpkg.Dir(d) {
				left[d] = true
			}
		}
	}
	var out []string
	for _, n := range stale {
		for d := pathpkg.Dir(n); d != "." && !left[d]; d = pathpkg.Dir(d) {
			if !slices.Contains(out, d) {
				out = append(out, d)
			}
		}
	}
	slices.SortFunc(out, func(a, b string) int {
		if c := strings.Count(b, "/") - strings.Count(a, "/"); c != 0 {
			return c
		}
		return strings.Compare(a, b)
	})
	return out
}

// writeExpected writes the rendered files writes of e into the directory
// at dir, its expected/: through a staging directory beside it when it
// does not exist (ADR-005 D14 item 1), and otherwise beneath one staging
// directory inside it, committed unit by unit (D14 item 2, Amendment A1).
// It returns the files committed, in path order, and why it stopped.
func writeExpected(dir string, e *exampleRun, writes []string, faults fault.Set) ([]string, error) {
	opt := hostfs.WithFaults(faults)
	x, err := hostfs.OpenExisting(dir, opt)
	if hostfs.ReasonOf(err) == hostfs.NotFound {
		o, err := hostfs.StageNew(dir, opt)
		if err != nil {
			return nil, err
		}
		for _, n := range writes {
			err := o.MkdirAll(pathpkg.Dir(n))
			if err == nil {
				err = o.Create(n, e.exec[n], bytes.NewReader(e.rendered[n]))
			}
			if err != nil {
				o.Abort()
				return nil, err
			}
		}
		if err := o.Commit(); err != nil {
			return nil, err
		}
		return writes, nil
	}
	if err != nil {
		return nil, err
	}
	defer x.Close()
	checks := x.Check(writes)
	for _, c := range checks {
		if c.Err != nil {
			return nil, c.Err
		}
	}
	o, err := x.Stage(checks)
	if err != nil {
		return nil, err
	}
	for _, n := range writes {
		if err := o.Create(n, e.exec[n], bytes.NewReader(e.rendered[n])); err != nil {
			o.Abort()
			return nil, err
		}
	}
	if err := o.Commit(); err != nil {
		var ce *hostfs.CommitError
		if errors.As(err, &ce) {
			return ce.Committed, err
		}
		return nil, err
	}
	return writes, nil
}

// updateFailed is the finding of an --update write or removal that failed,
// listing what was written and removed before it. A failure before the
// first write or removal changed nothing.
func updateFailed(err error, u *testUpdate) finding.Finding {
	switch {
	case hostfs.ReasonOf(err) == hostfs.NotDirectory:
		return finding.New(finding.CodeDestinationNotDirectory, invocation, nil, nil)
	case len(u.Written) == 0 && len(u.Removed) == 0:
		return sink.StageFailed(err)
	}
	return finding.New(finding.CodeDestinationWriteFailed, invocation, nil, finding.Params{
		"reason":  finding.String(sink.WriteReason(hostfs.ReasonOf(err))),
		"written": stringList(u.Written),
	})
}

// updateLines is the human report of u: each file written, then each
// removed, or with --dry-run each that would be.
func updateLines(u *testUpdate) string {
	if u == nil {
		return ""
	}
	wrote, removed := "Wrote ", "Removed "
	if u.DryRun {
		wrote, removed = "Would write ", "Would remove "
	}
	var b strings.Builder
	for _, n := range u.Written {
		b.WriteString(wrote + string(finding.AppendText(nil, n)) + "\n")
	}
	for _, n := range u.Removed {
		b.WriteString(removed + string(finding.AppendText(nil, n)) + "\n")
	}
	if b.Len() == 0 {
		b.WriteString("Nothing to update.\n")
	}
	return b.String()
}
