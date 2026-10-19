// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"io"
	pathpkg "path"
	"slices"
	"strconv"
	"strings"

	"github.com/kottera/metais/internal/cli/args"
	"github.com/kottera/metais/internal/dates"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/template"
	"github.com/kottera/metais/internal/unitext"
	"github.com/kottera/metais/internal/version"
	"github.com/kottera/metais/internal/zipfmt"
)

// The steps render's refusals point to.
var (
	nextRenderOut = Next{Run: "metais render <package> --out <dir>",
		Why: "Names the directory to render into, which must not exist yet."}
	nextRenderNow = Next{Run: "metais render <package> --out <dir> --now 2026-01-31T09:00:00Z",
		Why: "Gives the render time as an RFC 3339 date-time with an offset."}
	nextRenderData = Next{Run: "metais render <package> --data <dir> --out <dir>",
		Why: "Names a data directory that exists and can be read."}
	nextRenderInputs = Next{Run: "metais inspect <package>",
		Why: "Lists the inputs the package takes, the files each is read from, and what each means."}
	nextRenderTemplates = Next{Run: "metais inspect <package>",
		Why: "Shows the package's outputs; a template's defect is the package's, for its author to fix."}
	nextRenderDest = Next{Run: "metais render <package> --out <dir>",
		Why: "Renders into a directory that does not exist yet, which metais creates."}
)

// renderDeclaration is metais render (ADR-008 D8).
func renderDeclaration() *Command {
	return &Command{
		Name: "render", Group: "Supplying and rendering",
		Synopsis:    "render <package> --out <dir> [supply flags] [--now <time>] [--json]",
		Description: "Render the package's outputs from the inputs supplied, into a new directory.",
		Detail: "The package is opened and verified whole, and the inputs are supplied and validated, as " +
			"metais inspect and metais validate do; an input that fails ends the render with every finding of " +
			"the inputs. Then every output is planned, its path checked by the portable-path rules, and " +
			"rendered in memory: every output is attempted, and each reports its first error. If any " +
			"finding is an error, nothing is written anywhere. A container output is one zip of its entries, " +
			"each rendered as an output is, in the order its profile and its first entries give; its zip is " +
			"written as it is assembled.\n\n" +
			"--out is required, and names a directory that does not exist yet. The outputs are written " +
			"beneath a staging directory beside it, .metais-staging- followed by 16 hex digits, which is " +
			"then renamed to it in one step that never replaces anything: the directory either does not " +
			"exist or holds every output. Its missing parents are created. An --out that exists is refused " +
			"before anything is written.\n\n" +
			"A template sees the inputs as inputs, the package's identity as package, where each input " +
			"came from as sources, the output being rendered as output, and the render time as now, which " +
			"is --now normalized to UTC, or nil without it. Nothing of the host — no environment variable, " +
			"directory, user or clock — reaches a template.\n\n" +
			"Each output's path is a Liquid template too. An output with for_each is rendered once for each " +
			"element of its array, which its when, its path and its template see as item, with its position " +
			"as fan_out; an output, or an element, whose when is false writes no file.",
		Role:     finding.RoleConsumer,
		Operands: []args.Operand{{Name: "<package>", Required: true, Description: "A packed package file, or a source directory; - is refused, since a package is read with random access."}},
		Flags: []args.Flag{
			{Name: "out", Value: "<dir>", Description: "The directory to render into, which must not exist; required."},
			dataFlag, setFlag, setJSONFlag, inputsJSONFlag,
			{Name: "now", Value: "<date-time>", Description: "The render time, an RFC 3339 date-time with an offset, which templates see as now; without it there is no time."},
			limitFlag, failOnFlag,
		},
		Format: "metais.render/1",
		Prints: "One line per file written, in path order: its size, its digest and its path, separated by tabs; a container's entries are indented beneath it, in its order.",
		Refusals: append(packageRefusals("render"),
			Refusal{finding.CodeUsageMissingFlag, "--out is not given.", nextRenderOut},
			Refusal{finding.CodeUsageBadValue, "--now is not an RFC 3339 date-time with an offset.", nextRenderNow},
			Refusal{finding.CodeUsageDataDirUnreadable, "The data directory given to --data does not exist, is not a directory, or cannot be read.", nextRenderData},
			Refusal{finding.CodeUsageConflictingFlags, "--inputs-json - is given twice, and standard input is read once.", nextInputsHelp},
			Refusal{finding.CodeDestinationNotDirectory, "--out, or a directory above it, exists and is not a directory.", nextRenderDest},
			Refusal{finding.CodeDestinationNotEmpty, "--out is a directory that holds entries.", nextRenderDest},
			Refusal{finding.CodeDestinationWriteFailed, "--out exists, or a write failed; whatever was written is removed.", nextRenderDest},
		),
		Examples: []Example{
			{"metais render report-1.0.0.metais --data data --out out", "Renders the package's outputs from the data directory data into the new directory out."},
			{"metais render report.metais --data data --set /vendor/name=Acme --out out", "Renders with one value set over the data directory's."},
			{"metais render . --data data --out out --now 2026-10-19T09:30:00+02:00 --json", "Renders the source directory in the working directory with a render time, and prints the files as one metais.render/1 document."},
		},
		run: runRender,
	}
}

// renderRefusals adds, to every command line's refusals, the codes of the
// families a supply and a render report (ADR-008 D16 item 2): the input
// families, every template and limit code, and the output path and
// expression codes.
func renderRefusals(c *Command) {
	have := map[finding.Code]bool{}
	for _, r := range c.Refusals {
		have[r.Code] = true
	}
	for _, e := range finding.Registry() {
		family, rest, _ := strings.Cut(string(e.Code), ".")
		var r Refusal
		switch {
		case have[e.Code]:
			continue
		case family == "input" || family == "json" || family == "yaml" || family == "csv" || family == "text" ||
			family == "markdown" || family == "collection" || e.Code == finding.CodeLimitExceeded:
			r = Refusal{e.Code, "An input is refused (" + strings.ReplaceAll(strings.ReplaceAll(rest, "_", " "), ".", ": ") + ").", nextRenderInputs}
		case family == "template" && e.Severity == finding.SeverityError && !strings.HasPrefix(rest, "xref.") && !strings.HasPrefix(rest, "ooxml."):
			r = Refusal{e.Code, "A template is refused (" + strings.ReplaceAll(rest, "_", " ") + ").", nextRenderTemplates}
		case strings.HasPrefix(string(e.Code), "limit.exceeded."), strings.HasPrefix(string(e.Code), "output.path."),
			strings.HasPrefix(string(e.Code), "output.expression."), strings.HasPrefix(string(e.Code), "output.container."):
			r = Refusal{e.Code, "An output is refused (" + strings.ReplaceAll(strings.ReplaceAll(rest, "_", " "), ".", ": ") + ").", nextRenderTemplates}
		default:
			continue
		}
		c.Refusals = append(c.Refusals, r)
	}
}

// renderResult is metais.render/1's result (ADR-008 D8 item 9).
type renderResult struct {
	Out    string       `json:"out"`
	DryRun bool         `json:"dry_run"`
	Only   *string      `json:"only"`
	Files  []renderFile `json:"files"`
	Sample *struct{}    `json:"sample"`
}

// renderFile is one file's record (ADR-005 D16 item 2).
type renderFile struct {
	Path       string  `json:"path"`
	Output     string  `json:"output"`
	Item       *int    `json:"item"`
	Source     string  `json:"source"`
	Kind       *string `json:"kind"`
	Size       int64   `json:"size"`
	Digest     string  `json:"digest"`
	Executable bool    `json:"executable"`
	Status     string  `json:"status"`
	// Entries are a container's, in its order (ADR-005 D16 item 2).
	Entries []renderEntry `json:"entries,omitempty"`
}

// renderEntry is one container entry's record (ADR-005 D16 item 2): its
// size and digest are its uncompressed content's.
type renderEntry struct {
	Path    string        `json:"path"`
	Output  *string       `json:"output"`
	Item    *int          `json:"item"`
	Source  string        `json:"source"`
	Kind    *string       `json:"kind"`
	Method  string        `json:"method"`
	Size    int64         `json:"size"`
	Digest  string        `json:"digest"`
	Entries []renderEntry `json:"entries,omitempty"`
}

// planned is one file a render plans (ADR-005 D13 phase 1): its
// declaration, its path within its tree, where in the manifest the path
// comes from, the evaluation of a fan-out it is, and its bytes once
// rendered.
type planned struct {
	decl manifest.Output
	path string
	ptr  string
	// literal is set for a path the manifest writes, or defaults, rather
	// than one a path template rendered.
	literal bool
	fan     *template.Fan
	bytes   []byte
	// owner is the id findings about the file name: its declaration's,
	// or, for a container's entry that has none, the nearest enclosing
	// declaration's.
	owner string
	// in is the path of the container the file is an entry of, through
	// every enclosing container, joined by "/", or "" at the top level.
	in string
	// entries are a container's files, in the container's order (ADR-005
	// D8 item 6).
	entries []*planned
	// tool marks the mimetype entry a profile writes itself (ADR-005 D9
	// item 4).
	tool bool
	// failed marks a file that did not render, and a container one of
	// whose entries did not.
	failed bool
	// size and digest are a container's zip's, measured in phase 4, and
	// deflated is whether its container deflates an entry.
	size     int64
	digest   string
	deflated bool
}

// item is the 0-based index of the fan-out evaluation pf is, or nil.
func (pf *planned) item() *int {
	if pf.fan == nil {
		return nil
	}
	i := pf.fan.Index()
	return &i
}

// isContainer reports whether pf is a container, whose bytes are its
// entries' zip.
func (pf *planned) isContainer() bool { return pf.decl.Form == manifest.OutputContainer && !pf.tool }

// source is the record's source of pf (ADR-005 D16 item 2): its form, or
// profile for the mimetype entry a profile writes.
func (pf *planned) source() string {
	if pf.tool {
		return "profile"
	}
	return string(pf.decl.Form)
}

// joinIn is the path p within the tree of the container at in.
func joinIn(in, p string) string {
	if in == "" {
		return p
	}
	return in + "/" + p
}

// runRender is metais render, of template outputs into a directory that
// does not exist (ADR-008 D8, ADR-005 D13 and D14 item 1).
func runRender(s *session, p args.Parsed) outcome {
	operand := p.Operands[0]
	if operand == "-" {
		f := finding.New(finding.CodeUsagePackageFromStdin, invocation, nil,
			finding.Params{"argument": finding.Int(int64(s.line.at[p.OperandArgs[0]] + 1))})
		return s.refuseWith(f)
	}
	out, ok := p.Value("out")
	if !ok {
		return s.refuseWith(finding.New(finding.CodeUsageMissingFlag, invocation, nil,
			finding.Params{"command": finding.String("render"), "flag": finding.String("--out")}))
	}
	if out == "-" {
		panic("cli: render to standard output is not part of this version of metais")
	}
	now, hasNow := p.Value("now")
	if hasNow {
		if _, err := dates.ParseNow(now); err != nil {
			// The value's own position: the flag's, written --now=value, or
			// the one after it.
			i, _ := p.Arg("now")
			at := s.line.at[i] + 1
			if !strings.Contains(s.inv.Args[s.line.at[i]], "=") {
				at++
			}
			return s.refuseWith(finding.New(finding.CodeUsageBadValue, invocation, nil, finding.Params{
				"argument": finding.Int(int64(at)),
				"flag":     finding.String("--now"),
				"reason":   finding.String("value_not_allowed"),
			}))
		}
	}
	lim, fs := s.limits(p)
	if len(fs) > 0 {
		return outcome{findings: fs, next: s.nextSteps(fs)}
	}
	if fs := s.stdinOnce(p); len(fs) > 0 {
		return s.refuseWith(fs[0])
	}
	pkg, fs := loader.Open(operand, loader.Options{Limits: lim}, s.inv.Options.Faults)
	if pkg == nil {
		return outcome{findings: fs, next: s.nextSteps(fs)}
	}
	defer pkg.Close()

	// Phase 0: the inputs (ADR-005 D13 item 0).
	dataDir, _ := p.Value(dataFlag.Name)
	res, fs := s.supply(p, pkg, lim)
	if res == nil || s.fails(fs) {
		return outcome{findings: fs, next: s.nextSteps(fs), dataDir: dataDir}
	}
	ts := template.NewTemplates(pkg.Tree, lim, pkg.Manifest.Inputs)
	b, err := template.NewBindings(template.Package{
		Tree: pkg.Tree, Manifest: pkg.Manifest, ManifestValue: pkg.ManifestValue,
		Digest: pkg.Digest, Combined: pkg.CombinedValue, Tool: toolName + "/" + version.String(),
	}, res.Inputs, now, lim)
	if err != nil {
		panic("cli: " + err.Error())
	}
	// One renderer, whose work the manifest's expressions and the
	// templates share (ADR-009 D11).
	r := template.NewRenderer(ts, b, lim, s.inv.Options.Faults)
	// Phases 1 and 2: plan and verify paths.
	plan, pfs := planOutputs(pkg, lim, s, r)
	fs = append(fs, pfs...)
	// Phase 3: the destination, every refusal reported with the others
	// (ADR-005 D15 item 8).
	fs = append(fs, checkDestination(out)...)
	// Phase 4: render every output, each to its first error.
	fs = append(fs, renderPlan(pkg, plan, ts, r, lim)...)
	// A warning that --fail-on warning makes a failure fails the render
	// before it writes anything (ADR-006 D2 item 5).
	if s.fails(fs) {
		return outcome{findings: fs, next: s.nextSteps(fs), dataDir: dataDir}
	}
	// Phase 5: commit (ADR-005 D14 item 1).
	slices.SortFunc(plan, func(a, b *planned) int { return strings.Compare(a.path, b.path) })
	if f, ok := commit(out, plan, s); !ok {
		fs = append(fs, f)
		return outcome{findings: fs, next: s.nextSteps(fs)}
	}
	result := &renderResult{Out: "directory", Files: []renderFile{}}
	var human strings.Builder
	for _, pf := range plan {
		size, d := pf.measured()
		var entries strings.Builder
		rf := renderFile{
			Path: pf.path, Output: pf.decl.ID, Item: pf.item(), Source: pf.source(), Size: size, Digest: d,
			Executable: pf.decl.Executable, Status: "create", Entries: entryRecords(pf, &entries, 1),
		}
		if !pf.isContainer() {
			k := kindOf(pf.path, pf.decl.Kind)
			rf.Kind = &k
		}
		result.Files = append(result.Files, rf)
		human.WriteString(strconv.FormatInt(size, 10) + "\t" + d + "\t")
		human.Write(finding.AppendText(nil, pf.path))
		human.WriteString("\n" + entries.String())
	}
	return outcome{findings: fs, result: result, human: []byte(human.String()), dataDir: dataDir}
}

// measured is pf's size and digest: a container's zip's, or its bytes'.
func (pf *planned) measured() (int64, string) {
	if pf.isContainer() {
		return pf.size, pf.digest
	}
	sum := sha256.Sum256(pf.bytes)
	return int64(len(pf.bytes)), "sha256:" + hex.EncodeToString(sum[:])
}

// entryRecords is the records of pf's entries, when pf is a container,
// and their human lines, each indented depth levels, written to human.
func entryRecords(pf *planned, human *strings.Builder, depth int) []renderEntry {
	if !pf.isContainer() {
		return nil
	}
	var lines strings.Builder
	out := make([]renderEntry, 0, len(pf.entries))
	for _, e := range pf.entries {
		size, d := e.measured()
		r := renderEntry{Path: e.path, Item: e.item(), Source: e.source(), Method: "stored", Size: size, Digest: d}
		if e.decl.ID != "" && !e.tool {
			id := e.decl.ID
			r.Output = &id
		}
		if e.decl.Form == manifest.OutputTemplate && !e.tool {
			k := kindOf(e.path, e.decl.Kind)
			r.Kind = &k
		}
		if e.deflated {
			r.Method = "deflated"
		}
		lines.WriteString(strings.Repeat("  ", depth) + strconv.FormatInt(size, 10) + "\t" + d + "\t")
		lines.Write(finding.AppendText(nil, e.path))
		lines.WriteString("\n")
		r.Entries = entryRecords(e, &lines, depth+1)
		out = append(out, r)
	}
	human.WriteString(lines.String())
	return out
}

// fails reports whether any of fs fails the command: an error, or under
// --fail-on warning a warning (ADR-006 D2 item 5).
func (s *session) fails(fs []finding.Finding) bool {
	return slices.ContainsFunc(fs, func(f finding.Finding) bool {
		return f.Severity() == finding.SeverityError || f.Severity() == finding.SeverityWarning && s.failOn() == finding.SeverityWarning
	})
}

// planOutputs is ADR-005 D13's phases 1 and 2 for the package's
// declarations. Each declaration's for_each gives its evaluations, each
// evaluation's when decides whether it writes a file, and its path is
// rendered, or defaulted (D3, D4); a declaration whose expressions fail
// contributes their finding and no file, and planning goes on with the
// next. A container is planned as a tree of its own, its entries in each
// of its evaluations, to container-depth, and held to its profile (D8,
// D9). Every path is held to the portable-path predicate, each alone and
// every path of its tree together (D12), and the count of planned files,
// every evaluation's and every container entry's, to the outputs limit
// (ADR-009 D3).
func planOutputs(pkg *loader.Package, lim limits.Set, s *session, r *template.Renderer) ([]*planned, []finding.Finding) {
	p := &planner{pkg: pkg, lim: lim, s: s, r: r, counted: true}
	plan := p.tree(pkg.Manifest.Outputs, "", "", 0, nil)
	if n, bound := int64(len(p.all)), lim.Value(limits.Outputs); n > bound {
		over := p.all[bound]
		if over.fan != nil {
			// The files the data fans out to take the plan beyond the
			// bound: the render's, not the package's (ADR-009 D19).
			subj := finding.Subject{Kind: finding.SubjectOutput, Name: over.owner}
			p.fs = append(p.fs, lim.ExceededOutput(limits.Outputs, subj, p.at(over.decl.Pointer), n, p.counted))
		} else {
			p.fs = append(p.fs, lim.Breached(limits.Outputs, p.at(over.decl.Pointer), n, p.counted))
		}
	}
	return plan, p.fs
}

// planner is one render's phases 1 and 2.
type planner struct {
	pkg *loader.Package
	lim limits.Set
	s   *session
	r   *template.Renderer
	fs  []finding.Finding
	all []*planned // every planned file, at every depth, in planning order
	// counted is cleared when a fan-out is left unevaluated because the
	// plan is already beyond outputs.
	counted bool
}

// at is the location of the manifest's value at ptr.
func (p *planner) at(ptr string) finding.Location { return manifestAt(p.pkg, ptr) }

// beyond reports whether the plan already holds more files than outputs.
func (p *planner) beyond() bool { return int64(len(p.all)) > p.lim.Value(limits.Outputs) }

// tree plans decls, the declarations of one tree: the output directory's,
// or the entries of the container at in, which depth containers enclose,
// in the enclosing container's evaluation fan. owner is the id an entry
// with none of its own is named by. The output directory's paths are
// held to D12 together here, and a container's by container.
func (p *planner) tree(decls []manifest.Output, owner, in string, depth int, fan *template.Fan) []*planned {
	var tree []*planned
	for _, d := range decls {
		switch {
		case d.Form != manifest.OutputTemplate && d.Form != manifest.OutputContainer:
			panic("cli: an output of form " + string(d.Form) + " is not part of this version of metais")
		case d.Escape == "html", d.Newline == "crlf", d.Validate != "":
			panic("cli: escape, newline and validate are not part of this version of metais")
		}
		id := cmp.Or(d.ID, owner)
		x, xfs := template.ParseExprs(d, id, p.at, p.lim)
		if len(xfs) > 0 {
			p.fs = append(p.fs, xfs...)
			continue
		}
		if x.ForEach != nil && p.beyond() {
			p.counted = false
			continue
		}
		if d.Form == manifest.OutputContainer && int64(depth) >= p.lim.Value(limits.ContainerDepth) {
			// D8 item 4: a container nested deeper than container-depth
			// is the package's own breach (ADR-009 D19), and plans
			// nothing.
			loc := p.at(d.Pointer + "/container")
			p.fs = append(p.fs, p.lim.Breached(limits.ContainerDepth, loc, int64(depth)+1, true))
			continue
		}
		tree = append(tree, p.declaration(d, x, id, in, depth, fan)...)
	}
	if in == "" {
		p.checkPaths(tree)
	}
	return tree
}

// declaration plans the files of the declaration d, whose expressions are
// x: one per evaluation whose when holds, in the order of the for_each
// array, or in the enclosing container's evaluation fan, each with its
// path rendered, or the default path when d gives none (ADR-005 D3, D4),
// and each container's entries. Before each evaluation but the first, a
// plan already beyond outputs stops the fan-out. The first error of an
// expression is the declaration's finding, and it plans no file.
func (p *planner) declaration(d manifest.Output, x *template.Exprs, owner, in string, depth int, fan *template.Fan) []*planned {
	fans := []*template.Fan{fan}
	if x.ForEach != nil {
		var f *finding.Finding
		if fans, f = p.r.Fans(x); f != nil {
			p.fs = append(p.fs, *f)
			return nil
		}
	}
	mark, marks := len(p.all), len(p.fs)
	var files []*planned
	for i, fan := range fans {
		if i > 0 && p.beyond() {
			p.counted = false
			break
		}
		ok, f := p.r.When(x, fan)
		if f == nil && ok {
			pf := &planned{decl: d, fan: fan, ptr: d.Pointer + "/path", literal: x.Literal, owner: owner, in: in}
			if x.Path == nil {
				pf.path, pf.ptr, pf.literal = strings.TrimSuffix(d.Source, ".liquid"), d.Pointer+"/template", true
			} else {
				pf.path, f = p.r.Path(x, fan)
			}
			if f == nil {
				p.all = append(p.all, pf)
				if d.Form == manifest.OutputContainer {
					pf.entries = p.container(pf, depth+1)
				}
				files = append(files, pf)
			}
		}
		if f != nil {
			// The declaration plans nothing: not the files of its earlier
			// evaluations, nor what their containers found.
			p.all, p.fs = p.all[:mark], append(p.fs[:marks], *f)
			return nil
		}
	}
	return files
}

// container plans the entries of the container pf, the depth-th
// container on its chain, in pf's evaluation: one tree, held to D12
// within it and to its profile (D8 items 3 and 8, D9), in the container's
// order (D8 item 6).
func (p *planner) container(pf *planned, depth int) []*planned {
	c := pf.decl.Container
	in := joinIn(pf.in, pf.path)
	var entries []*planned
	if m, ok := c.ToolMimetype(); ok {
		// D9 items 2 to 4: the profile's own mimetype entry, which takes
		// its place in the tree like any other.
		tool := &planned{decl: pf.decl, path: manifest.MimetypeEntry, ptr: pf.decl.Pointer + "/container/profile",
			literal: true, fan: pf.fan, owner: pf.owner, in: in, bytes: []byte(m), tool: true}
		p.all = append(p.all, tool)
		entries = append(entries, tool)
	}
	for _, e := range p.tree(c.Entries, pf.owner, in, depth, pf.fan) {
		subj := finding.Subject{Kind: finding.SubjectOutput, Name: e.owner}
		params := pathParams(e)
		params["profile"] = finding.String(c.Profile)
		switch c.CheckEntry(e.path) {
		case manifest.EntryReserved:
			p.fs = append(p.fs, finding.New(finding.CodeContainerReservedEntry, subj, ptrTo(p.at(e.ptr)), params))
			continue
		case manifest.EntryNonASCII:
			p.fs = append(p.fs, finding.New(finding.CodeContainerNonASCII, subj, ptrTo(p.at(e.ptr)), params))
		case manifest.EntrySpace:
			p.fs = append(p.fs, finding.New(finding.CodeContainerSpace, subj, ptrTo(p.at(e.ptr)), params))
		}
		entries = append(entries, e)
	}
	p.checkPaths(entries)
	subj := finding.Subject{Kind: finding.SubjectOutput, Name: pf.owner}
	if req := c.RequiredEntry(); req != "" && !slices.ContainsFunc(entries, func(e *planned) bool { return e.path == req }) {
		params := finding.Params{"entry": finding.String(req), "profile": finding.String(c.Profile)}
		if i := pf.item(); i != nil {
			params["item"] = finding.Int(int64(*i))
		}
		p.fs = append(p.fs, finding.New(finding.CodeContainerMissingEntry, subj, ptrTo(p.at(pf.decl.Pointer+"/container/profile")), params))
	}
	if len(entries) > zipfmt.MaxEntries {
		p.fs = append(p.fs, finding.New(finding.CodeContainerTooManyEntries, subj,
			ptrTo(p.at(pf.decl.Pointer+"/container/entries")), finding.Params{"count": finding.Int(int64(len(entries)))}))
	}
	return containerOrder(c, entries)
}

// ptrTo is a pointer to a copy of l.
func ptrTo(l finding.Location) *finding.Location { return &l }

// containerOrder is ADR-005 D8 item 6's order of a container's entries:
// the profile's first entries, the profile's own mimetype before the
// author's [Content_Types].xml; then the files of every first: true
// declaration, in declaration order, each declaration's in byte order of
// their paths; then every other file in byte order. A file appears once,
// at its earliest place.
func containerOrder(c *manifest.Container, entries []*planned) []*planned {
	sorted := slices.SortedFunc(slices.Values(entries), func(a, b *planned) int { return strings.Compare(a.path, b.path) })
	out := make([]*planned, 0, len(entries))
	placed := map[*planned]bool{}
	place := func(keep func(e *planned) bool) {
		for _, e := range sorted {
			if !placed[e] && keep(e) {
				placed[e] = true
				out = append(out, e)
			}
		}
	}
	place(func(e *planned) bool { return e.tool })
	if first := c.FirstEntry(); first != "" {
		place(func(e *planned) bool { return e.path == first })
	}
	for _, d := range c.Entries {
		if d.First {
			place(func(e *planned) bool { return !e.tool && e.decl.Pointer == d.Pointer })
		}
	}
	place(func(*planned) bool { return true })
	return out
}

// checkPaths applies the portable-path predicate to the paths of one
// tree (ADR-005 D12): each path alone, and every path of the tree
// together.
func (p *planner) checkPaths(tree []*planned) {
	at := func(pf *planned) *finding.Location { return ptrTo(p.at(pf.ptr)) }
	p.fs = append(p.fs, hostfs.CheckPaths(hostfs.WithFaults(p.s.inv.Options.Faults), func() []finding.Finding {
		return checkPaths(tree, p.lim, at)
	})...)
}

// manifestAt is the location of the manifest value at ptr, with its line
// and column when the manifest has them.
func manifestAt(pkg *loader.Package, ptr string) finding.Location {
	l := finding.Location{Origin: finding.OriginPackage, File: manifest.ManifestFile}.WithPointer(ptr)
	if pkg.Positions != nil {
		if pos, ok := pkg.Positions.Value(ptr); ok {
			l = l.At(pos)
		}
	}
	return l
}

// checkPaths applies the portable-path predicate to the paths of plan
// (ADR-005 D12): each path alone, and every path of the tree together,
// each finding located by at.
func checkPaths(plan []*planned, lim limits.Set, at func(*planned) *finding.Location) []finding.Finding {
	var fs []finding.Finding
	byPath := map[string]*planned{}
	paths := make([]string, len(plan))
	for i, pf := range plan {
		paths[i] = pf.path
		if _, seen := byPath[pf.path]; !seen {
			byPath[pf.path] = pf
		}
		for _, pr := range unitext.CheckPath(pf.path, lim) {
			fs = append(fs, pathFinding(pf, at(pf), pr, lim))
		}
	}
	repeats := map[string]int{}
	for _, c := range unitext.Collisions(paths) {
		pf, other := byPath[c.Path], byPath[c.Other]
		if c.Kind == unitext.Duplicate {
			// The n-th repeat of a path is its (n+1)-th declaration.
			repeats[c.Path]++
			pf = nthOf(plan, c.Path, repeats[c.Path])
		}
		fs = append(fs, collisionFinding(pf, at(pf), c, at(other)))
	}
	return fs
}

// nthOf is the planned file of plan at p after n others at p.
func nthOf(plan []*planned, p string, n int) *planned {
	for _, pf := range plan {
		if pf.path == p {
			if n == 0 {
				return pf
			}
			n--
		}
	}
	return nil
}

// The output path codes (ADR-005 D12; ADR-005's interfaces name the
// family output.path).
const (
	codePathEncoding     finding.Code = "output.path.encoding"
	codePathNotNFC       finding.Code = "output.path.not_nfc"
	codePathShape        finding.Code = "output.path.shape"
	codePathCharacter    finding.Code = "output.path.character"
	codePathNotPortable  finding.Code = "output.path.not_portable"
	codePathDuplicate    finding.Code = "output.path.duplicate"
	codePathCaseCollide  finding.Code = "output.path.case_collision"
	codePathFileDirClash finding.Code = "output.path.file_directory_conflict"
)

// pathFinding is the finding of pr, one rule of ADR-005 D12 items 1 to 5
// the path of pf breaks, at loc: the output path codes, and limit.package
// for path-length and path-depth, since a manifest's path is the
// package's own content (ADR-009 D19).
func pathFinding(pf *planned, loc *finding.Location, pr unitext.Problem, lim limits.Set) finding.Finding {
	subj := finding.Subject{Kind: finding.SubjectOutput, Name: pf.owner}
	p := pathParams(pf)
	var code finding.Code
	switch pr.Kind {
	case unitext.Encoding:
		code, p["reason"] = codePathEncoding, finding.String(pr.Reason)
	case unitext.NotNFC:
		code = codePathNotNFC
	case unitext.Shape:
		code, p["reason"] = codePathShape, finding.String(pr.Reason)
	case unitext.Character:
		code, p["code_point"] = codePathCharacter, finding.Int(int64(pr.CodePoint))
	case unitext.NotPortable:
		code, p["reason"] = codePathNotPortable, finding.String(pr.Reason)
	case unitext.Length, unitext.Depth:
		id := limits.PathLength
		if pr.Kind == unitext.Depth {
			id = limits.PathDepth
		}
		if !pf.literal {
			// A path template's path is the render's (ADR-009 D19).
			return lim.ExceededOutput(id, subj, *loc, pr.Observed, true)
		}
		return lim.Breached(id, *loc, pr.Observed, true)
	default:
		panic("cli: an unknown path problem")
	}
	return finding.New(code, subj, loc, p)
}

// pathParams is the parameters every output path finding about pf has:
// the path as rendered, and the index of the fan-out evaluation that
// rendered it (ADR-005 D4 item 2).
func pathParams(pf *planned) finding.Params {
	p := finding.Params{"path": finding.String(pf.path)}
	if i := pf.item(); i != nil {
		p["item"] = finding.Int(int64(*i))
	}
	return p
}

// collisionFinding is the finding of c, ADR-005 D12 item 6 or 7, about the
// planned file pf at loc, which collides with the one at other.
func collisionFinding(pf *planned, loc *finding.Location, c unitext.Collision, other *finding.Location) finding.Finding {
	subj := finding.Subject{Kind: finding.SubjectOutput, Name: pf.owner}
	p := pathParams(pf)
	switch c.Kind {
	case unitext.Duplicate:
		return finding.New(codePathDuplicate, subj, loc, p,
			finding.WithRelated(finding.Related{Location: *other, Note: "first occurrence"}))
	case unitext.CaseCollision:
		p["member"], p["other"] = finding.String(c.Member), finding.String(c.OtherMember)
		return finding.New(codePathCaseCollide, subj, loc, p,
			finding.WithRelated(finding.Related{Location: *other, Note: "collides with"}))
	}
	p["member"], p["other"] = finding.String(c.Member), finding.String(c.OtherMember)
	return finding.New(codePathFileDirClash, subj, loc, p,
		finding.WithRelated(finding.Related{Location: *other, Note: "collides with"}))
}

// checkDestination is ADR-005 D13's phase 3 for a render into the new
// directory out (D14 item 1): an out that exists, or that lies beneath a
// file, is refused before anything is written. A directory with entries
// is not empty, and lists them; any other existing out is a write the
// render cannot make, since it never writes into an existing name.
func checkDestination(out string) []finding.Finding {
	w, rest, err := hostfs.OpenNearest(out)
	if err != nil {
		if hostfs.ReasonOf(err) == hostfs.NotDirectory {
			return []finding.Finding{finding.New(finding.CodeDestinationNotDirectory, invocation, nil, nil)}
		}
		return []finding.Finding{writeFailed(hostfs.ReasonOf(err))}
	}
	defer w.Close()
	if rest != "." {
		return nil
	}
	entries, err := w.List(".", maxListed+1)
	if err != nil && hostfs.ReasonOf(err) != hostfs.TooLarge {
		return []finding.Finding{writeFailed(hostfs.ReasonOf(err))}
	}
	if len(entries) == 0 {
		return []finding.Finding{writeFailed(hostfs.Exists)}
	}
	var names []string
	for _, e := range entries[:min(len(entries), maxListed)] {
		names = append(names, e.Name)
	}
	slices.Sort(names)
	return []finding.Finding{finding.New(finding.CodeDestinationNotEmpty, invocation, nil, finding.Params{
		"entries": stringList(names),
		"more":    finding.Bool(len(entries) > maxListed),
	})}
}

// maxListed is how many entries a not_empty finding lists.
const maxListed = 10

// writeFailed is output.destination.write_failed for r, with nothing left
// written.
func writeFailed(r hostfs.Reason) finding.Finding {
	return finding.New(finding.CodeDestinationWriteFailed, invocation, nil, finding.Params{
		"reason":  finding.String(writeReason(r)),
		"written": stringList(nil),
	})
}

// renderPlan is ADR-005 D13's phase 4: every template a template output
// reaches, at any depth of containers, is parsed, each once, and every
// parse finding is reported, whether or not the output plans a file; then
// every embed of a raw form that does not exist, whose input and item are
// known, is reported before anything renders (ADR-004 D12 item 3); then
// every planned file whose templates parsed and need no such raw form is
// rendered with r, each to its first error (ADR-004 D16 item 4). The bytes
// of each file that rendered are held in its planned file. Then each
// container whose entries all rendered is measured, and held to
// output-size.
func renderPlan(pkg *loader.Package, plan []*planned, ts *template.Templates, r *template.Renderer, lim limits.Set) []finding.Finding {
	var fs []finding.Finding
	reported := map[string]bool{}
	ready := map[string]bool{}
	var sources []string // ready's keys, in manifest order
	var parse func(decls []manifest.Output)
	parse = func(decls []manifest.Output) {
		for _, d := range decls {
			if d.Container != nil {
				parse(d.Container.Entries)
			}
			if _, seen := ready[d.Source]; seen || d.Form != manifest.OutputTemplate {
				continue
			}
			ready[d.Source] = true
			sources = append(sources, d.Source)
			for _, name := range ts.Reach(d.Source) {
				_, pfs := ts.Parse(name)
				if len(pfs) > 0 {
					ready[d.Source] = false
				}
				if !reported[name] {
					reported[name] = true
					fs = append(fs, pfs...)
				}
			}
		}
	}
	parse(pkg.Manifest.Outputs)
	unavailable := map[string]bool{}
	for _, source := range sources {
		if !ready[source] {
			continue
		}
		for _, name := range ts.Reach(source) {
			if _, seen := unavailable[name]; !seen {
				ufs := r.Unavailable(name)
				unavailable[name] = len(ufs) > 0
				fs = append(fs, ufs...)
			}
			if unavailable[name] {
				ready[source] = false
			}
		}
	}
	var render func(tree []*planned)
	render = func(tree []*planned) {
		for _, pf := range tree {
			switch {
			case pf.isContainer():
				render(pf.entries)
				continue
			case pf.tool:
				continue
			case !ready[pf.decl.Source]:
				pf.failed = true
				continue
			}
			out, f := r.Render(template.Output{ID: pf.owner, Template: pf.decl.Source, Path: pf.path, Fan: pf.fan, Kind: kindOf(pf.path, pf.decl.Kind), In: pf.in})
			if f != nil {
				fs = append(fs, *f)
				pf.failed = true
				continue
			}
			pf.bytes = out
		}
	}
	render(plan)
	for _, pf := range plan {
		if !pf.isContainer() || !measure(pf) {
			continue
		}
		if pf.size > lim.Value(limits.OutputSize) {
			loc := finding.Location{Origin: finding.OriginOutput, File: pf.path}
			fs = append(fs, lim.ExceededOutput(limits.OutputSize, finding.Subject{Kind: finding.SubjectOutput, Name: pf.owner}, loc, pf.size, true))
		}
	}
	return fs
}

// measure measures the container pf, and every container among its
// entries, when every entry rendered: the size and digest of the zip it
// writes, and which of its entries it deflates. It writes the zip to
// nothing but a hash, so no container is held whole in memory.
func measure(pf *planned) bool {
	if len(pf.entries) > zipfmt.MaxEntries {
		// Refused in planning (D8 item 8); a zip cannot hold them.
		pf.failed = true
		return false
	}
	for _, e := range pf.entries {
		if e.failed || e.isContainer() && !measure(e) {
			pf.failed = true
			return false
		}
	}
	h := sha256.New()
	n := &byteCount{}
	written, err := writeContainer(io.MultiWriter(h, n), pf)
	if err != nil {
		panic("cli: a planned container does not write: " + err.Error())
	}
	for i, w := range written {
		pf.entries[i].deflated = w.Deflated
	}
	pf.size, pf.digest = n.n, "sha256:"+hex.EncodeToString(h.Sum(nil))
	return true
}

// byteCount counts the bytes written to it.
type byteCount struct{ n int64 }

func (c *byteCount) Write(b []byte) (int, error) {
	c.n += int64(len(b))
	return len(b), nil
}

// writeContainer writes the zip of the container pf to w, assembling it as
// it is written (ADR-005 D8, D13 item 5): each entry's content is its
// rendered bytes, or the zip of a container entry, itself written as it
// is written. stored: true and a profile's mimetype are stored; every
// other entry is ADR-001 D4's auto (D8 item 7).
func writeContainer(w io.Writer, pf *planned) ([]zipfmt.Written, error) {
	parts := make([]zipfmt.Part, len(pf.entries))
	for i, e := range pf.entries {
		method := zipfmt.Auto
		if e.tool || e.decl.Stored {
			method = zipfmt.Store
		}
		parts[i] = zipfmt.Part{Name: e.path, Method: method, Content: func(w io.Writer) error {
			if e.isContainer() {
				_, err := writeContainer(w, e)
				return err
			}
			_, err := w.Write(e.bytes)
			return err
		}}
	}
	return zipfmt.WriteParts(w, parts)
}

// content is the reader of pf's bytes as the commit writes them: a
// container's zip is written into a pipe as it is read, and done reports
// the error of its writing, once the reader is finished with.
func content(pf *planned) (r io.Reader, done func(read error) error) {
	if !pf.isContainer() {
		return bytes.NewReader(pf.bytes), func(error) error { return nil }
	}
	pr, pw := io.Pipe()
	errc := make(chan error, 1)
	go func() {
		_, err := writeContainer(pw, pf)
		pw.CloseWithError(err)
		errc <- err
	}()
	return pr, func(read error) error {
		// A reader that stopped early unblocks the writer.
		pr.CloseWithError(read)
		if err := <-errc; err != nil && read == nil {
			return err
		}
		return nil
	}
}

// commit writes plan, in path order, beneath a staging directory beside
// out, and renames it to out (ADR-005 D14 item 1). A failure removes what
// the render created and reports output.destination.write_failed, with
// nothing written.
func commit(out string, plan []*planned, s *session) (finding.Finding, bool) {
	o, err := hostfs.StageNew(out, hostfs.WithFaults(s.inv.Options.Faults))
	if err != nil {
		if hostfs.ReasonOf(err) == hostfs.NotDirectory {
			return finding.New(finding.CodeDestinationNotDirectory, invocation, nil, nil), false
		}
		return writeFailed(hostfs.ReasonOf(err)), false
	}
	for _, pf := range plan {
		err := o.MkdirAll(pathpkg.Dir(pf.path))
		if err == nil {
			r, done := content(pf)
			err = o.Create(pf.path, pf.decl.Executable, r)
			if derr := done(err); err == nil && derr != nil {
				panic("cli: a measured container does not write: " + derr.Error())
			}
		}
		if err != nil {
			o.Abort()
			return writeFailed(hostfs.ReasonOf(err)), false
		}
	}
	if err := o.Commit(); err != nil {
		return writeFailed(hostfs.ReasonOf(err)), false
	}
	return finding.Finding{}, true
}

// kindOf is a rendered file's kind (ADR-005 D7): the declaration's kind
// when it gives one, otherwise from the path's final extension, ASCII
// case-insensitively, then its whole file name.
func kindOf(p, declared string) string {
	if declared != "" {
		return declared
	}
	name := pathpkg.Base(p)
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		if k, ok := kindExtensions[strings.ToLower(name[i+1:])]; ok {
			return k
		}
	}
	if k, ok := kindNames[name]; ok {
		return k
	}
	return "text"
}

// kindExtensions and kindNames are ADR-005 D7 item 1's table.
var (
	kindExtensions = func() map[string]string {
		m := map[string]string{}
		for kind, exts := range map[string]string{
			"html":     "html htm",
			"xml":      "xml xhtml svg rels opf ncx xsd xsl xslt plist",
			"markdown": "md markdown",
			"json":     "json geojson webmanifest",
			"yaml":     "yaml yml",
			"csv":      "csv",
			"css":      "css scss less",
			"hash":     "sh bash zsh py rb pl r toml ps1 properties tf",
			"slash":    "js mjs cjs ts tsx jsx go c h cc cpp hpp java kt kts swift rs cs scala dart jsonc proto groovy gradle",
			"sql":      "sql",
		} {
			for _, e := range strings.Fields(exts) {
				m[e] = kind
			}
		}
		return m
	}()
	kindNames = map[string]string{
		"Makefile": "hash", "Dockerfile": "hash", "Containerfile": "hash", ".gitignore": "hash",
		".gitattributes": "hash", ".dockerignore": "hash", ".env": "hash",
	}
)
