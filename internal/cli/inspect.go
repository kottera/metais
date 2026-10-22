// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"metais.org/metais/internal/author"
	"metais.org/metais/internal/cli/args"
	"metais.org/metais/internal/digest"
	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/loader"
	"metais.org/metais/internal/manifest"
	"metais.org/metais/internal/render"
	"metais.org/metais/internal/template"
	"metais.org/metais/internal/version"
)

// The package operand's placeholder in a next step (ADR-008 D14 item 2).
const packagePlaceholder = "<package>"

// nextInspectAgain is the step after a package's own refusal: there is
// nothing to do with a refused package but fix or replace it.
var nextInspectAgain = Next{Run: "metais inspect <package>", Why: "Opens the package again once it is fixed or replaced; nothing of a refused package is used."}

// exampleIs is what the specification says an example is (spec §13.7),
// as the tool says it wherever it shows a package's examples (ADR-008 A8
// item 3), and that it is not a preview (A10 item 2). It says nothing of
// whether an example's facts are real, which the tool cannot know.
const exampleIs = "An example is the author's illustration of the inputs, and of the outputs a render of them should produce, kept in the package to test it: not your data, and not a preview of what the package produces, which is a render from generated data with metais render --sample."

// exampleNotFacts is that an example's inputs are never a real document's
// data, whose facts are asked of the person (ADR-008 A11 item 1, R-UX-2),
// and examplesEntries what the help of every command that reaches an
// example's entries says of them: ls, read, extract and grep, and unpack,
// which writes them (A11 item 1, A12 item 2).
const (
	exampleNotFacts = "An example's inputs are the author's illustration, never a real document's data: render a real document only from the facts the person supplies, and without them ask the person for them, never taking them from an example."
	examplesEntries = "The entries under examples/ are the package's examples. " + exampleIs + " " + exampleNotFacts
)

// checkStep is inspect's next step for a source directory: what check
// reports, as MTS-AUT-003 and metais help check say it (ADR-008 A12 item
// 1).
const checkStep = "The source directory is the package's editable source: edit its files, then check them. check reports every defect without data, or only the refusal of a source it cannot read."

// generatedServes is whom generated data serves, and generatedNever that
// it never takes the place of a real document's facts, as inspect's next
// steps and the help of render --sample and sample say them (ADR-008 A10
// item 1).
const (
	generatedServes = "a person who asked for an example, sample data or a preview of what the package produces"
	generatedNever  = "Generated data never stands in for the facts a real document needs: without them, ask the person for them."
)

var nextLimits = Next{Run: "metais limits", Why: "Lists every limit with its default; a limit can be lowered and never raised."}

// failOnFlag is --fail-on, which every command that reports findings
// takes (ADR-008 D4, ADR-006 D2 item 5).
var failOnFlag = args.Flag{
	Name: "fail-on", Value: "<severity>", Values: []string{"error", "warning"}, Default: "error",
	Description: "With warning, a warning fails the command with exit status 1 as an error does; no finding's severity changes.",
}

// limitFlag is --limit, which every command that reads a package takes
// (ADR-009 D4 item 1).
var limitFlag = args.Flag{
	Name: "limit", Value: "<name>=<value>", Repeatable: true,
	Description: "Lower one resource limit for this run, to a whole number no higher than its default (a size may end in KiB, MiB or GiB). metais limits lists them.",
}

// inspectDeclaration is metais inspect (ADR-008 D6).
func inspectDeclaration() *Command {
	return &Command{
		Name: "inspect", Group: "Reading a package",
		Synopsis:    "inspect <package> [--guide] [--limit <name>=<value>]... [--json]",
		Description: "Print what a supplier reads first: the package's identity, its guide, its inputs, outputs, references and examples, and the commands to run next.",
		Detail: "The package is a packed package file or a source directory. Before it prints anything, " +
			"inspect verifies the whole package: every entry's name and data, the digest of every entry " +
			"and of the package, the manifest under the specification version it declares, every schema, " +
			"and each compiled file against the sources it is derived from. A package that fails any " +
			"check is refused with every finding of the check, and nothing of it is printed.\n\n" +
			"The digest of a source directory is the digest of the package metais pack would write from it, " +
			"computed in memory; nothing is written.\n\n" +
			"A guide of more than 60 lines or 6144 bytes is cut to its first 40 lines, within 4096 bytes; " +
			"--guide prints it whole, and nothing else.\n\n" +
			"The Next: block lists the commands that fill in and render the package from the facts a person " +
			"supplies, then metais render --sample and metais sample, which serve only " + generatedServes +
			", with data generated from the inputs' schemas, and last the way to edit the package: metais " +
			"unpack for a packed package, metais check for a source directory. " + generatedNever + " " +
			exampleIs + " " + exampleNotFacts,
		Role:     finding.RoleConsumer,
		Operands: []args.Operand{{Name: "<package>", Required: true, Description: "A packed package file, or a source directory; - is refused, since a package is read with random access."}},
		Flags: []args.Flag{
			{Name: "guide", Description: "Print only the whole guide: its Markdown alone, or with --json the package's identity and the guide."},
			limitFlag,
			failOnFlag,
		},
		Format:       "metais.inspect/1",
		NextToStdout: true,
		Prints:       "The package's summary: its title, name and version, description, specification version, form and digest, then the sections Guide, Inputs, Outputs, References, Assets and Examples, then the Next: block; with --guide, the guide's Markdown alone.",
		Refusals:     packageRefusals("inspect"),
		Examples: []Example{
			{"metais inspect report-1.0.0.metais", "Prints the summary of a packed package."},
			{"metais inspect .", "Prints the summary of the source directory in the working directory."},
			{"metais inspect report-1.0.0.metais --guide", "Prints the package's whole guide."},
			{"metais inspect report-1.0.0.metais --json", "Prints the summary as one metais.inspect/1 document."},
		},
		run: runInspect,
	}
}

// packageRefusals are the refusals of the command called name, which opens
// a package: the operand's, the limits', and every registered code of the
// families the open reports (ADR-008 D16 item 2's declaration of each), but
// those only check reports.
func packageRefusals(name string) []Refusal {
	out := []Refusal{
		{Code: finding.CodeUsageMissingOperand, Cause: "The package operand is not given.", Next: nextCommandHelp(name)},
		{Code: finding.CodeUsagePackageFromStdin, Cause: "The package operand is -, standard input.", Next: nextCommandHelp(name)},
		{Code: finding.CodeLimitInvalid, Cause: "A --limit names no limit, gives a malformed value, or tries to raise one.", Next: nextLimits},
		{Code: finding.CodeLimitPackage, Cause: "The package breaches one of its limits.", Next: nextLimits},
	}
	checkOnly := author.CheckCodes()
	for _, e := range finding.Registry() {
		family, rest, _ := strings.Cut(string(e.Code), ".")
		opened := family == "package" || family == "manifest" || family == "schema" || openTemplateCodes[e.Code]
		// The generator's codes are about an input, and no open reports
		// them.
		if !opened || e.Severity != finding.SeverityError || e.Subject == finding.SubjectInput || slices.Contains(checkOnly, e.Code) {
			continue
		}
		what := map[string]string{
			"package":  "The package is refused",
			"manifest": "The manifest breaks one of its rules",
			"schema":   "An input schema is refused",
			"template": "An output's path, when or for_each does not parse",
		}[family]
		out = append(out, familyRefusal(e.Code, what, strings.ReplaceAll(strings.ReplaceAll(rest, "_", " "), ".", ": "), nextInspectAgain))
	}
	return out
}

// openTemplateCodes are the template family's codes an open reports: those
// of the manifest's own expressions, which the loader parses (ADR-004 D16
// item 5). A filter's input is not known when an expression is parsed, so
// template.filter_input is not among them.
var openTemplateCodes = map[finding.Code]bool{
	loader.CodeTemplateSyntax: true, loader.CodeTemplateInclude: true, loader.CodeTemplateUnknownFilter: true,
	loader.CodeTemplateNumberRange: true, loader.CodeTemplateTagContext: true, loader.CodeTemplateFilterArg: true,
	loader.CodeTemplateDivision: true, loader.CodeTemplateSafePosition: true, loader.CodeTemplateDateFormat: true,
}

// inspectResult is metais.inspect/1's result (ADR-008 D6). Under --guide,
// only Package and Guide are present (item 9).
type inspectResult struct {
	Package    inspectPackage     `json:"package"`
	Guide      inspectGuide       `json:"guide"`
	Inputs     *[]inspectInput    `json:"inputs,omitempty"`
	Outputs    *[]inspectOutput   `json:"outputs,omitempty"`
	References *inspectReferences `json:"references,omitempty"`
	Assets     *inspectAssets     `json:"assets,omitempty"`
	Examples   *[]inspectExample  `json:"examples,omitempty"`
}

type inspectPackage struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Specification string `json:"specification"`
	Implemented   bool   `json:"implemented"`
	Digest        string `json:"digest"`
	Source        string `json:"source"`
}

type inspectGuide struct {
	Text         string `json:"text"`
	ExcerptLines int    `json:"excerpt_lines"`
	TotalLines   int    `json:"total_lines"`
	Truncated    bool   `json:"truncated"`
}

type inspectInput struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description *string  `json:"description"`
	Required    bool     `json:"required"`
	Collection  bool     `json:"collection"`
	Forms       []string `json:"forms"`
	Files       []string `json:"files"`
	Meaning     *string  `json:"meaning"`
	SchemaEntry string   `json:"schema_entry"`
	Examples    []string `json:"examples"`
}

type inspectOutput struct {
	ID         *string           `json:"id"`
	Form       string            `json:"form"`
	Source     *string           `json:"source"`
	Path       string            `json:"path"`
	ForEach    *string           `json:"for_each"`
	When       *string           `json:"when"`
	Escape     string            `json:"escape"`
	Newline    string            `json:"newline"`
	Validate   *string           `json:"validate"`
	Kind       *string           `json:"kind"`
	Executable bool              `json:"executable"`
	Transforms []string          `json:"transforms"`
	Templates  *[]inspectMirror  `json:"templates,omitempty"`
	Container  *inspectContainer `json:"container,omitempty"`
}

type inspectMirror struct {
	Template string `json:"template"`
	Path     string `json:"path"`
}

type inspectContainer struct {
	Format  string          `json:"format"`
	Profile *string         `json:"profile"`
	Entries []inspectOutput `json:"entries"`
}

type inspectReferences struct {
	Count   int                `json:"count"`
	Size    int64              `json:"size"`
	Entries []inspectReference `json:"entries"`
}

type inspectReference struct {
	Path  string  `json:"path"`
	Title *string `json:"title"`
	Size  int64   `json:"size"`
}

type inspectAssets struct {
	Count int   `json:"count"`
	Size  int64 `json:"size"`
}

type inspectExample struct {
	Name     string   `json:"name"`
	Inputs   []string `json:"inputs"`
	Expected bool     `json:"expected"`
}

// runInspect is metais inspect.
func runInspect(s *session, p args.Parsed) outcome {
	pkg, refused := s.open(p)
	if pkg == nil {
		return refused
	}
	defer pkg.Close()
	return inspectOutcome(pkg, p.Has("guide"), s.table)
}

// open opens the package its first operand names (ADR-008 D2), within
// the command's limits, or returns the refusal: - as the package, a bad
// --limit, or every finding that refuses the package. The caller closes
// the package.
func (s *session) open(p args.Parsed) (*loader.Package, outcome) {
	operand := p.Operands[0]
	if operand == "-" {
		f := finding.New(finding.CodeUsagePackageFromStdin, invocation, nil,
			finding.Params{"argument": s.operandArg(p, 0)})
		return nil, s.refuseWith(f)
	}
	lim, fs := s.limits(p)
	if len(fs) > 0 {
		return nil, outcome{findings: fs, next: s.nextSteps(fs)}
	}
	pkg, fs := loader.Open(operand, loader.Options{Limits: lim}, s.inv.Options.Faults)
	if pkg == nil {
		return nil, outcome{findings: fs, next: s.nextSteps(fs)}
	}
	if opened != nil {
		opened(pkg)
	}
	return pkg, outcome{}
}

// opened, when set, is given every package a command opens. Only this
// package's tests set it, to count what a command reads of a package
// after it is verified (ADR-010 D15 item 2).
var opened func(*loader.Package)

// operandArg is the 1-based position on the whole line of the operand i,
// a finding's argument parameter.
func (s *session) operandArg(p args.Parsed, i int) finding.Value {
	return finding.Int(int64(s.line.at[p.OperandArgs[i]] + 1))
}

// flagArg is the 1-based position on the whole line of the flag called
// name, which was given.
func (s *session) flagArg(p args.Parsed, name string) finding.Value {
	i, _ := p.Arg(name)
	return finding.Int(int64(s.line.at[i] + 1))
}

// limits is the command's limits: the defaults, each lowered by a --limit
// in the order given (ADR-009 D4 item 1). The session keeps them, so the
// report lists at most the findings limit they give.
func (s *session) limits(p args.Parsed) (limits.Set, []finding.Finding) {
	lim := limits.Defaults()
	var fs []finding.Finding
	for _, v := range p.Values(limitFlag.Name) {
		if f, ok := lim.Lower(v); !ok {
			fs = append(fs, f)
		}
	}
	if len(fs) == 0 {
		s.lim = &lim
	}
	return lim, fs
}

// inspectOutcome is the summary of pkg (ADR-008 D6), its next steps the
// commands of t.
func inspectOutcome(pkg *loader.Package, guideOnly bool, t []*Command) outcome {
	m := pkg.Manifest
	res := &inspectResult{
		Package: inspectPackage{
			Name: m.Name, Version: m.Version, Title: m.Title, Description: m.Description,
			Specification: m.Metais.String(), Implemented: slices.Contains(version.Implements(), m.Metais.String()),
			Digest: digest.Format(pkg.Digest), Source: pkg.Form.String(),
		},
		Guide: guideOf(m.Guide, guideOnly),
	}
	if guideOnly {
		return outcome{result: res, human: []byte(ensureLF(m.Guide))}
	}
	entries := pkg.Tree.List()
	examples := examplesOf(entries)
	inputs := make([]inspectInput, 0, len(m.Inputs))
	for _, in := range m.Inputs {
		inputs = append(inputs, inputOf(in, examples))
	}
	outputs := outputsOf(m.Outputs, entries, render.ExplicitTemplates(m.Outputs))
	refs := &inspectReferences{Entries: []inspectReference{}}
	for _, r := range pkg.References {
		refs.Count++
		refs.Size += r.Size
		refs.Entries = append(refs.Entries, inspectReference{Path: r.Path, Title: r.Title, Size: r.Size})
	}
	assets := &inspectAssets{}
	for _, e := range entries {
		if strings.HasPrefix(e, "assets/") {
			info, _ := pkg.Tree.Stat(e)
			assets.Count++
			assets.Size += info.Size
		}
	}
	exList := make([]inspectExample, 0, len(examples))
	for _, name := range sortedKeys(examples) {
		exList = append(exList, *examples[name])
	}
	res.Inputs, res.Outputs, res.References, res.Assets, res.Examples = &inputs, &outputs, refs, assets, &exList
	return outcome{result: res, human: inspectHuman(res), next: inspectNext(res, t)}
}

// guideOf is the guide section (ADR-008 D6 item 2): the whole guide when it
// has at most 60 lines and 6144 bytes, or when whole is set; otherwise its
// first 40 lines, shortened to end within 4096 bytes.
func guideOf(g string, whole bool) inspectGuide {
	lines := strings.Split(g, "\n")
	out := inspectGuide{Text: g, ExcerptLines: len(lines), TotalLines: len(lines)}
	if whole || (len(lines) <= 60 && len(g) <= 6144) {
		return out
	}
	keep := lines[:min(40, len(lines))]
	for len(keep) > 1 && len(strings.Join(keep, "\n")) > 4096 {
		keep = keep[:len(keep)-1]
	}
	text := strings.Join(keep, "\n")
	if len(text) > 4096 {
		cut := 4096
		for cut > 0 && !utf8.RuneStart(text[cut]) {
			cut--
		}
		text = text[:cut]
	}
	out.Text, out.ExcerptLines, out.Truncated = text, len(keep), true
	return out
}

// inputOf is one input's entry (ADR-008 D6 item 3).
func inputOf(in manifest.Input, examples map[string]*inspectExample) inspectInput {
	out := inspectInput{
		Name: in.Name, Title: in.Title, Required: in.Required, Collection: in.Collection,
		SchemaEntry: in.Schema, Files: filesOf(in), Examples: []string{},
	}
	if in.Description != "" {
		d := in.Description
		out.Description = &d
	}
	if in.Meaning != "" {
		mn := in.Meaning
		out.Meaning = &mn
	}
	for _, f := range in.Forms {
		out.Forms = append(out.Forms, string(f))
	}
	for _, name := range sortedKeys(examples) {
		if slices.Contains(examples[name].Inputs, in.Name) {
			out.Examples = append(out.Examples, name)
		}
	}
	return out
}

// filesOf is the names a data directory may hold the input under
// (ADR-003 D15 item 2, D16 items 1 and 2): <name>/ for a collection, and
// otherwise <name>.<ext> for each extension of each accepted form, in the
// forms' order. .md and .markdown are text's only when markdown is not
// accepted (D15 item 4).
func filesOf(in manifest.Input) []string {
	if in.Collection {
		return []string{in.Name + "/"}
	}
	markdown := slices.Contains(in.Forms, manifest.FormMarkdown)
	var out []string
	for _, f := range in.Forms {
		var exts []string
		switch f {
		case manifest.FormJSON:
			exts = []string{".json"}
		case manifest.FormYAML:
			exts = []string{".yaml", ".yml"}
		case manifest.FormCSV:
			exts = []string{".csv"}
		case manifest.FormText:
			exts = []string{".txt"}
			if !markdown {
				exts = append(exts, ".md", ".markdown")
			}
		case manifest.FormMarkdown:
			exts = []string{".md", ".markdown"}
		}
		for _, e := range exts {
			out = append(out, in.Name+e)
		}
	}
	return out
}

// examplesOf is every example of the package (ADR-007), by name: the
// directories beneath examples/, the inputs each supplies, named by
// examples/<name>/inputs/'s entries as a data directory names them, and
// whether it has expected outputs.
func examplesOf(entries []string) map[string]*inspectExample {
	out := map[string]*inspectExample{}
	for _, e := range entries {
		rest, ok := strings.CutPrefix(e, "examples/")
		if !ok {
			continue
		}
		name, rest, ok := strings.Cut(rest, "/")
		if !ok {
			continue
		}
		ex := out[name]
		if ex == nil {
			ex = &inspectExample{Name: name, Inputs: []string{}}
			out[name] = ex
		}
		switch {
		case strings.HasPrefix(rest, "expected/"):
			ex.Expected = true
		case strings.HasPrefix(rest, "inputs/"):
			item := strings.TrimPrefix(rest, "inputs/")
			input, _, _ := strings.Cut(item, "/")
			input, _, _ = strings.Cut(input, ".")
			if !slices.Contains(ex.Inputs, input) {
				ex.Inputs = append(ex.Inputs, input)
			}
		}
	}
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// outputsOf is the outputs section (ADR-008 D6 item 4): each declaration's
// fields as ADR-005 names them, with its path's default where it omits one
// (ADR-005 D4 item 4), and each mirror expanded into the templates it
// renders (ADR-005 D6).
func outputsOf(decls []manifest.Output, entries []string, explicit map[string]bool) []inspectOutput {
	out := make([]inspectOutput, 0, len(decls))
	for _, d := range decls {
		o := inspectOutput{Form: string(d.Form), Escape: d.Escape, Newline: d.Newline, Executable: d.Executable, Transforms: []string{}}
		if d.ID != "" {
			id := d.ID
			o.ID = &id
		}
		if d.Form != manifest.OutputContainer {
			src := d.Source
			o.Source = &src
		}
		o.Path = d.Path
		if o.Path == "" {
			o.Path = strings.TrimSuffix(d.Source, ".liquid")
		}
		if d.ForEach != "" {
			v := d.ForEach
			o.ForEach = &v
		}
		if d.When != "" {
			v := d.When
			o.When = &v
		}
		if d.Validate != "" {
			v := d.Validate
			o.Validate = &v
			o.Transforms = append(o.Transforms, "validate")
		}
		if d.Kind != "" {
			v := d.Kind
			o.Kind = &v
		}
		if d.Newline == "crlf" {
			o.Transforms = append(o.Transforms, "crlf")
		}
		switch d.Form {
		case manifest.OutputMirror:
			ts := mirrorOf(d, entries, explicit)
			o.Templates = &ts
		case manifest.OutputContainer:
			c := &inspectContainer{Format: d.Container.Format, Entries: outputsOf(d.Container.Entries, entries, explicit)}
			if d.Container.Profile != "" {
				pr := d.Container.Profile
				c.Profile = &pr
			}
			o.Container = c
			o.Transforms = append(o.Transforms, "zip")
		}
		out = append(out, o)
	}
	return out
}

// mirrorOf is what a mirror renders (ADR-005 D6): every template beneath
// templates/<dir>, in byte order, but those with a segment beneath the
// mirror's root beginning with _ and those explicit names, each at the same
// relative path beneath the declaration's path with a final .liquid
// removed.
func mirrorOf(d manifest.Output, entries []string, explicit map[string]bool) []inspectMirror {
	dest := d.Path
	if dest == "" {
		dest = d.Source
	}
	out := []inspectMirror{}
	for _, e := range entries {
		rel, ok := strings.CutPrefix(e, render.TreePrefix(template.Dir, d.Source))
		name := strings.TrimPrefix(e, template.Dir)
		if !ok || explicit[name] || slices.ContainsFunc(strings.Split(rel, "/"), func(s string) bool { return strings.HasPrefix(s, "_") }) {
			continue
		}
		out = append(out, inspectMirror{Template: name, Path: render.Beneath(dest, strings.TrimSuffix(rel, ".liquid"))})
	}
	return out
}

// inspectNext is the summary's Next: block (ADR-008 D6 item 8), of the
// commands t declares.
func inspectNext(res *inspectResult, t []*Command) []Next {
	var out []Next
	if res.Guide.Truncated {
		out = append(out, Next{"metais inspect <package> --guide", "Prints the whole guide; this summary shows its start."})
	}
	for _, in := range *res.Inputs {
		if in.Meaning != nil {
			out = append(out, Next{"metais read <package> " + shellQuote(*in.Meaning), "Reads what the input " + in.Name + " means and what good data for it looks like."})
		}
	}
	for _, in := range *res.Inputs {
		out = append(out, Next{"metais schema <package> " + shellQuote(in.Name) + " --fields", "Lists the fields of the input " + in.Name + ", one line each with its type, whether it is required, its constraints and title; without --fields, metais schema prints the whole schema it is validated against."})
	}
	if res.References.Count > 0 {
		out = append(out, Next{"metais grep <package> <pattern> references/", "Searches the package's references."})
	}
	out = append(out,
		Next{"metais validate <package> --data <dir>", "Validates the inputs you gathered in a data directory."},
		Next{"metais render <package> --data <dir> --out <dir>", "Renders the outputs from the validated inputs into a new directory."},
		// The generator serves only a person who asked for an example,
		// sample data or a preview, and no step names a shipped example:
		// its expected outputs are no preview (ADR-008 A10, R-UX-2).
		Next{"metais render <package> --sample --out <dir>", "Only for a person who asked for an example or a preview of what the package produces: renders the outputs from generated data into a new directory. " + generatedNever},
		Next{"metais sample <package> --out <dir>", "Only for a person who asked for sample data or data to start from: writes generated data as a data directory. " + generatedNever},
	)
	// The route to editing the package (ADR-008 A8 item 2): a source
	// directory is already the tree unpack writes. The step says what
	// check reports as MTS-AUT-003 has it (A12 item 1): an edit can leave
	// a source the check cannot read.
	if res.Package.Source == "directory" {
		out = append(out, Next{"metais check <package>", checkStep})
	} else {
		out = append(out, Next{"metais unpack <package> --out <dir>", "Writes the package's source directory, to edit it and pack it again."})
	}
	// A step names a command this tool has, or it is not one a caller can
	// take: the reading, supplying and rendering commands join the list
	// as they are declared.
	return declaredSteps(out, t)
}

// shellQuote quotes s for a POSIX shell when it holds anything outside
// ADR-008 D14 item 2's safe set.
func shellQuote(s string) string {
	safe := s != ""
	for _, r := range s {
		if !(r < utf8.RuneSelf && (r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || strings.ContainsRune("_./:@%+=,-", r))) {
			safe = false
			break
		}
	}
	if safe {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func ensureLF(s string) string {
	if strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}

// inspectHuman is the human summary (ADR-008 D6 item 10).
func inspectHuman(res *inspectResult) []byte {
	var b strings.Builder
	pk := res.Package
	b.WriteString(pk.Title + " — " + pk.Name + " " + pk.Version + "\n")
	b.WriteString(pk.Description + "\n")
	source := "packed"
	if pk.Source == "directory" {
		source = "source directory"
	}
	b.WriteString("METAIS " + pk.Specification + ", " + source + ", " + pk.Digest + "\n")
	section := func(title string) { b.WriteString("\n" + title + "\n") }
	section("Guide")
	b.WriteString(ensureLF(res.Guide.Text))
	if res.Guide.Truncated {
		b.WriteString("… " + strconv.Itoa(res.Guide.TotalLines-res.Guide.ExcerptLines) + " more lines; metais inspect <package> --guide prints the whole guide.\n")
	}
	section("Inputs")
	if len(*res.Inputs) == 0 {
		b.WriteString("  none\n")
	}
	for _, in := range *res.Inputs {
		req := "optional"
		if in.Required {
			req = "required"
		}
		b.WriteString("  " + in.Name + " — " + in.Title + " (" + req + "; " + strings.Join(in.Files, ", ") + ")\n")
		if in.Description != nil {
			first, _, _ := strings.Cut(*in.Description, "\n")
			b.WriteString("    " + first + "\n")
		}
	}
	section("Outputs")
	var outs func([]inspectOutput, string)
	outs = func(os []inspectOutput, indent string) {
		for _, o := range os {
			line := indent + o.Path + " ← " + o.Form
			if o.Source != nil {
				line += " " + *o.Source
			}
			if o.ForEach != nil {
				line += ", for each " + *o.ForEach
			}
			if o.When != nil {
				line += ", when " + *o.When
			}
			b.WriteString(line + "\n")
			if o.Container != nil {
				outs(o.Container.Entries, indent+"  ")
			}
		}
	}
	outs(*res.Outputs, "  ")
	if r := res.References; r.Count > 0 {
		section("References")
		b.WriteString("  " + plural(r.Count, "entry", "entries") + ", " + humanSize(r.Size) + "\n")
		for i, e := range r.Entries {
			if i == 50 {
				b.WriteString("  … metais ls <package> references/ -r lists the rest\n")
				break
			}
			line := "  " + e.Path + " (" + humanSize(e.Size) + ")"
			if e.Title != nil {
				line += " — " + *e.Title
			}
			b.WriteString(line + "\n")
		}
	}
	if a := res.Assets; a.Count > 0 {
		section("Assets")
		b.WriteString("  " + plural(a.Count, "file", "files") + ", " + humanSize(a.Size) + "\n")
	}
	if len(*res.Examples) > 0 {
		section("Examples")
		b.WriteString("  " + exampleIs + "\n")
		b.WriteString("  " + exampleNotFacts + "\n")
		for _, ex := range *res.Examples {
			line := "  " + ex.Name + ": " + strings.Join(ex.Inputs, ", ")
			if ex.Expected {
				line += " (with expected outputs)"
			}
			b.WriteString(line + "\n")
		}
	}
	return []byte(b.String())
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + many
}

// humanSize is a size in bytes below 1024, and otherwise in KiB, MiB or
// GiB with one decimal place (ADR-008 D6 item 10).
func humanSize(n int64) string {
	if n < 1024 {
		return strconv.FormatInt(n, 10) + " B"
	}
	units := []string{"KiB", "MiB", "GiB"}
	v := float64(n) / 1024
	i := 0
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	return strconv.FormatFloat(v, 'f', 1, 64) + " " + units[i]
}
