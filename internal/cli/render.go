// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	pathpkg "path"
	"slices"
	"strconv"
	"strings"

	"github.com/kottera/metais/internal/cli/args"
	"github.com/kottera/metais/internal/dates"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/inputs"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/template"
	"github.com/kottera/metais/internal/unitext"
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
			"finding is an error, nothing is written anywhere.\n\n" +
			"--out is required, and names a directory that does not exist yet. The outputs are written " +
			"beneath a staging directory beside it, .metais-staging- followed by 16 hex digits, which is " +
			"then renamed to it in one step that never replaces anything: the directory either does not " +
			"exist or holds every output. Its missing parents are created. An --out that exists is refused " +
			"before anything is written.\n\n" +
			"A template sees the inputs as inputs, the package's identity as package, where each input " +
			"came from as sources, the output being rendered as output, and the render time as now, which " +
			"is --now normalized to UTC, or nil without it. Nothing of the host — no environment variable, " +
			"directory, user or clock — reaches a template.",
		Role:     finding.RoleConsumer,
		Operands: []args.Operand{{Name: "<package>", Required: true, Description: "A packed package file, or a source directory; - is refused, since a package is read with random access."}},
		Flags: []args.Flag{
			{Name: "out", Value: "<dir>", Description: "The directory to render into, which must not exist; required."},
			dataFlag, setFlag, setJSONFlag, inputsJSONFlag,
			{Name: "now", Value: "<date-time>", Description: "The render time, an RFC 3339 date-time with an offset, which templates see as now; without it there is no time."},
			limitFlag, failOnFlag,
		},
		Format: "metais.render/1",
		Prints: "One line per file written, in path order: its size, its digest and its path, separated by tabs.",
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
// families, every template and limit code, and the output path codes.
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
		case strings.HasPrefix(string(e.Code), "limit.exceeded."), strings.HasPrefix(string(e.Code), "output.path."):
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
	Path       string `json:"path"`
	Output     string `json:"output"`
	Item       *int   `json:"item"`
	Source     string `json:"source"`
	Kind       string `json:"kind"`
	Size       int64  `json:"size"`
	Digest     string `json:"digest"`
	Executable bool   `json:"executable"`
	Status     string `json:"status"`
}

// planned is one file a render plans (ADR-005 D13 phase 1): its
// declaration, its path, where in the manifest the path comes from, and
// its bytes once rendered.
type planned struct {
	decl  manifest.Output
	path  string
	ptr   string
	bytes []byte
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
	records := res.Inputs
	// Phases 1 and 2: plan and verify paths.
	plan, pfs := planOutputs(pkg, lim, s)
	fs = append(fs, pfs...)
	// Phase 3: the destination, every refusal reported with the others
	// (ADR-005 D15 item 8).
	fs = append(fs, checkDestination(out)...)
	// Phase 4: render every output, each to its first error.
	fs = append(fs, renderPlan(pkg, plan, records, now, lim, s)...)
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
		sum := sha256.Sum256(pf.bytes)
		d := "sha256:" + hex.EncodeToString(sum[:])
		result.Files = append(result.Files, renderFile{
			Path: pf.path, Output: pf.decl.ID, Source: "template", Kind: kindOf(pf.path, pf.decl.Kind),
			Size: int64(len(pf.bytes)), Digest: d, Executable: pf.decl.Executable, Status: "create",
		})
		human.WriteString(strconv.Itoa(len(pf.bytes)) + "\t" + d + "\t")
		human.Write(finding.AppendText(nil, pf.path))
		human.WriteString("\n")
	}
	return outcome{findings: fs, result: result, human: []byte(human.String()), dataDir: dataDir}
}

// fails reports whether any of fs fails the command: an error, or under
// --fail-on warning a warning (ADR-006 D2 item 5).
func (s *session) fails(fs []finding.Finding) bool {
	return slices.ContainsFunc(fs, func(f finding.Finding) bool {
		return f.Severity() == finding.SeverityError || f.Severity() == finding.SeverityWarning && s.failOn() == finding.SeverityWarning
	})
}

// planOutputs is ADR-005 D13's phases 1 and 2 for the package's
// declarations: each template output's path, given or defaulted (D4 item
// 4), held to the portable-path predicate, each path alone and every path
// of the tree together (D12), and the count of planned files to the
// outputs limit (ADR-009 D3).
func planOutputs(pkg *loader.Package, lim limits.Set, s *session) ([]*planned, []finding.Finding) {
	var plan []*planned
	for _, d := range pkg.Manifest.Outputs {
		switch {
		case d.Form != manifest.OutputTemplate:
			panic("cli: an output of form " + string(d.Form) + " is not part of this version of metais")
		case d.When != "", d.ForEach != "", d.Escape == "html", d.Newline == "crlf", d.Validate != "",
			strings.Contains(d.Path, "{{"), strings.Contains(d.Path, "{%"):
			panic("cli: when, for_each, escape, newline, validate and path templates are not part of this version of metais")
		}
		pf := &planned{decl: d, path: d.Path, ptr: d.Pointer + "/path"}
		if d.Path == "" {
			pf.path, pf.ptr = strings.TrimSuffix(d.Source, ".liquid"), d.Pointer+"/template"
		}
		plan = append(plan, pf)
	}
	var fs []finding.Finding
	at := func(pf *planned) *finding.Location {
		l := finding.Location{Origin: finding.OriginPackage, File: manifest.ManifestFile}.WithPointer(pf.ptr)
		if pkg.Positions != nil {
			if pos, ok := pkg.Positions.Value(pf.ptr); ok {
				l = l.At(pos)
			}
		}
		return &l
	}
	fs = hostfs.CheckPaths(hostfs.WithFaults(s.inv.Options.Faults), func() []finding.Finding {
		return checkPaths(plan, lim, at)
	})
	if n := int64(len(plan)); n > lim.Value(limits.Outputs) {
		over := plan[lim.Value(limits.Outputs)]
		l := finding.Location{Origin: finding.OriginPackage, File: manifest.ManifestFile}.WithPointer(over.decl.Pointer)
		if pkg.Positions != nil {
			if pos, ok := pkg.Positions.Value(over.decl.Pointer); ok {
				l = l.At(pos)
			}
		}
		fs = append(fs, lim.Breached(limits.Outputs, l, n, true))
	}
	return plan, fs
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
	subj := finding.Subject{Kind: finding.SubjectOutput, Name: pf.decl.ID}
	p := finding.Params{"path": finding.String(pf.path)}
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
	case unitext.Length:
		return lim.Breached(limits.PathLength, *loc, pr.Observed, true)
	case unitext.Depth:
		return lim.Breached(limits.PathDepth, *loc, pr.Observed, true)
	default:
		panic("cli: an unknown path problem")
	}
	return finding.New(code, subj, loc, p)
}

// collisionFinding is the finding of c, ADR-005 D12 item 6 or 7, about the
// planned file pf at loc, which collides with the one at other.
func collisionFinding(pf *planned, loc *finding.Location, c unitext.Collision, other *finding.Location) finding.Finding {
	subj := finding.Subject{Kind: finding.SubjectOutput, Name: pf.decl.ID}
	p := finding.Params{"path": finding.String(pf.path)}
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

// renderPlan is ADR-005 D13's phase 4 for template outputs: every template
// an output reaches is parsed, each once, and every parse finding is
// reported; then every output whose templates parsed is rendered, each to
// its first error (ADR-004 D16 item 4). The bytes of each output that
// rendered are held in its planned file.
func renderPlan(pkg *loader.Package, plan []*planned, records []inputs.Record, now string, lim limits.Set, s *session) []finding.Finding {
	ts := template.NewTemplates(pkg.Tree, lim)
	var fs []finding.Finding
	reported := map[string]bool{}
	ready := make([]bool, len(plan))
	for i, pf := range plan {
		ready[i] = true
		for _, name := range ts.Reach(pf.decl.Source) {
			_, pfs := ts.Parse(name)
			if len(pfs) > 0 {
				ready[i] = false
			}
			if !reported[name] {
				reported[name] = true
				fs = append(fs, pfs...)
			}
		}
	}
	b, err := template.NewBindings(template.Package{
		Tree: pkg.Tree, Manifest: pkg.Manifest, ManifestValue: pkg.ManifestValue,
		Digest: pkg.Digest, Combined: pkg.CombinedValue,
	}, records, now, lim)
	if err != nil {
		panic("cli: " + err.Error())
	}
	r := template.NewRenderer(ts, b, lim, s.inv.Options.Faults)
	for i, pf := range plan {
		if !ready[i] {
			continue
		}
		out, f := r.Render(template.Output{ID: pf.decl.ID, Template: pf.decl.Source, Path: pf.path})
		if f != nil {
			fs = append(fs, *f)
			continue
		}
		pf.bytes = out
	}
	return fs
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
			err = o.Create(pf.path, pf.decl.Executable, bytes.NewReader(pf.bytes))
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
