// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/kottera/metais/internal/cli/args"
	"github.com/kottera/metais/internal/dates"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/inputs"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/render"
	"github.com/kottera/metais/internal/render/sink"
	"github.com/kottera/metais/internal/sample"
	"github.com/kottera/metais/internal/template"
	"github.com/kottera/metais/internal/version"
)

// The steps render's refusals point to.
var (
	nextRenderOut = Next{Run: "metais render <package> --out <dir>",
		Why: "Names the directory to render into: a new one, or an empty one; --overwrite renders into one that holds entries."}
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
	nextRenderArchive = Next{Run: "metais render <package> --out - --only <path>",
		Why: "Writes one file's bytes, which may go to a terminal; or redirect standard output to a file for the archive."}
	nextRenderDryRun = Next{Run: "metais render <package> --out <dir> --dry-run",
		Why: "Lists every file the render writes, by the path --only takes, and writes nothing."}
	nextRenderSample = Next{Run: "metais render <package> --sample --out <dir>",
		Why: "Renders from sample data in place of a data directory; --optional, --seed and --items tune the generator."}
	nextRenderOverwrite = Next{Run: "metais render <package> --out <dir> --overwrite",
		Why: "Replaces the files the render writes in the directory, and leaves every other file in it alone."}
	nextRenderInTheWay = Next{Run: "metais render <package> --out <dir>",
		Why: "Renders into a new directory; or remove, from the existing one, what stands at the path the finding names."}
)

// renderDeclaration is metais render (ADR-008 D8).
func renderDeclaration() *Command {
	return &Command{
		Name: "render", Group: "Supplying and rendering",
		Synopsis:    "render <package> --out <dir|-> [supply flags] [--only <path>] [--overwrite] [--dry-run] [--now <time>] [--sample [--optional <mode>] [--seed <n>] [--items <n>]] [--json]",
		Description: "Render the package's outputs from the inputs supplied, into a directory or to standard output.",
		Detail: "The package is opened and verified whole, and the inputs are supplied and validated, as " +
			"metais inspect and metais validate do; an input that fails ends the render with every finding of " +
			"the inputs. Then every output is planned, an asset or a mirror as each file it writes, its " +
			"path checked by the portable-path rules, and rendered in memory: every output is attempted, " +
			"and each reports its first error. An asset is copied as it is, never rendered: its size counts " +
			"against the output limits, and its bytes are read from the package as it is written. An output " +
			"declaring newline: crlf then has its line endings made CR LF, and one naming a validate schema " +
			"is parsed and validated against it, every finding of every output reported. If any finding is " +
			"an error, nothing is written anywhere. A container output is one zip of its entries, " +
			"each rendered as an output is, in the order its profile and its first entries give; its zip is " +
			"written as it is assembled.\n\n" +
			"--out is required, and names a directory. When it does not exist yet — the way to render — " +
			"the outputs are written beneath a staging directory beside it, .metais-staging- followed by 16 " +
			"hex digits, which is then renamed to it in one step that never replaces anything: the directory " +
			"either does not exist or holds every output. Its missing parents are created.\n\n" +
			"An --out that exists must be a directory. One with no entry at all is rendered into as it is; " +
			"one that holds any entry, a hidden one included, is refused unless --overwrite is given. The " +
			"outputs are then written beneath one staging directory inside it, .metais-staging, and moved " +
			"into place one by one, in path order: a file whose path exists replaces it, by a rename that " +
			"never writes into it or reads it, and a directory the render needs is moved in whole. Every " +
			"other file in the directory is left alone. Each file is whole, its old bytes or its new, and a " +
			"failure part way through lists the files already moved. Before anything is written, every " +
			"output's path is checked in the directory: a symbolic link or junction on it, a file where a " +
			"directory must be, a name that differs only in case from an existing one, a directory or a " +
			"special file at it, another file system mounted on it, or a first segment that is the staging " +
			"directory's name refuses the render, as a staging directory left by an earlier render does.\n\n" +
			"--out - writes every file to standard output as one pax archive instead, in path order, each " +
			"entry with mode 0644, or 0755 for an executable output, owner 0 and the time " +
			"1980-01-01T00:00:00Z; with --json the document then goes to standard error, after the archive. " +
			"Standard output carries the archive and nothing else, and nothing at all when the render is " +
			"refused. An archive is not written to a terminal.\n\n" +
			"--only <path> selects one file by the path --dry-run lists. The whole render still runs and is " +
			"checked, so --only never makes a refused render succeed; it chooses what is written: with " +
			"--out -, the file's bytes alone, which may go to a terminal, and with --out <dir>, that file " +
			"alone.\n\n" +
			"--dry-run renders everything and checks --out as a render would, writes nothing anywhere, and " +
			"lists every file the render would write with its size and digest, and, into a directory, " +
			"whether it would create the file or replace it. It is refused exactly when the render would be.\n\n" +
			"A template sees the inputs as inputs, the package's identity as package, where each input " +
			"came from as sources, the output being rendered as output, and the render time as now, which " +
			"is --now normalized to UTC, or nil without it. Nothing of the host — no environment variable, " +
			"directory, user or clock — reaches a template.\n\n" +
			"Each output's path is a Liquid template too. An output with for_each is rendered once for each " +
			"element of its array, which its when, its path and its template see as item, with its position " +
			"as fan_out; an output, or an element, whose when is false writes no file.\n\n" +
			"--sample renders from sample data, the author's preview: the generator makes every input from its " +
			"schema, as metais sample does, in the data directory's place, so --data is refused with it. Values " +
			"given with --set, --set-json and --inputs-json are kept as given and the rest is made around them, " +
			"conditionals included, so one value can be pinned to see one branch. The data is validated as " +
			"supplied data is before anything is rendered. --optional, --seed and --items are taken only with " +
			"--sample. A render with --sample is an authoring command: a defect of the package fails it with exit " +
			"status 1, as check does.",
		Role:     finding.RoleConsumer,
		Operands: []args.Operand{{Name: "<package>", Required: true, Description: "A packed package file, or a source directory; - is refused, since a package is read with random access."}},
		Flags: []args.Flag{
			{Name: "out", Value: "<dir|->", Description: "The directory to render into — a new one, an empty one, or with --overwrite one that holds entries — or - for an archive on standard output; required."},
			dataFlag, setFlag, setJSONFlag, inputsJSONFlag,
			{Name: "overwrite", Description: "Renders into an existing directory that holds entries, replacing each file the render writes; nothing else in it is touched. Not with --out -."},
			{Name: "only", Value: "<path>", Description: "Write only the file at this path, as --dry-run lists it: its bytes alone with --out -."},
			{Name: "dry-run", Description: "List every file the render would write, with its size and digest, and write nothing."},
			{Name: "now", Value: "<date-time>", Description: "The render time, an RFC 3339 date-time with an offset, which templates see as now; without it there is no time."},
			sampleFlag, optionalFlag, seedFlag, itemsFlag,
			limitFlag, failOnFlag,
		},
		Format: "metais.render/1",
		Prints: "One line per file written, in path order: its size, its digest and its path, separated by tabs; " +
			"a container's entries are indented beneath it, in its order. On a dry run the lines are headed would write. " +
			"With --out -, standard output is the archive, or the one file's bytes.",
		Refusals: append(packageRefusals("render"),
			Refusal{Code: finding.CodeUsageMissingFlag, Cause: "--out is not given.", Next: nextRenderOut},
			Refusal{Code: finding.CodeUsageArchiveToTerminal, Cause: "--out - is given without --only, and standard output is a terminal.", Next: nextRenderArchive},
			Refusal{Code: finding.CodeUsageUnknownOutput, Cause: "--only names no file the render writes.", Next: nextRenderDryRun},
			Refusal{Code: finding.CodeUsageBadValue, Cause: "--now is not an RFC 3339 date-time with an offset, or --seed or --items is not a whole number in range.", Next: nextRenderNow,
				match: valueRefused("--now", "--seed", "--items")},
			Refusal{Code: finding.CodeUsageFlagExcluded, Cause: "--data is given with --sample, whose generated data takes the data directory's place.", Next: nextRenderSample,
				match: flagGiven("--data")},
			Refusal{Code: finding.CodeUsageFlagRequires, Cause: "--optional, --seed or --items is given without --sample.", Next: nextRenderSample},
			Refusal{Code: finding.CodeUsageDataDirUnreadable, Cause: "The data directory given to --data does not exist, is not a directory, or cannot be read.", Next: nextRenderData},
			Refusal{Code: finding.CodeUsageConflictingFlags, Cause: "--inputs-json - is given twice, and standard input is read once.", Next: nextInputsHelp},
			Refusal{Code: finding.CodeDestinationNotDirectory, Cause: "--out, or a directory above it, exists and is not a directory; or, in --out, a file stands where an output's path needs a directory.", Next: nextRenderInTheWay},
			Refusal{Code: finding.CodeDestinationNotEmpty, Cause: "--out is a directory that holds entries, and --overwrite is not given.", Next: nextRenderOverwrite},
			Refusal{Code: finding.CodeDestinationLink, Cause: "In --out, an output's path runs through, or ends at, a symbolic link, a junction or another reparse point, which is never followed.", Next: nextRenderInTheWay},
			Refusal{Code: finding.CodeDestinationCaseVariant, Cause: "In --out, an entry's name differs only in case or normalization from a name an output's path holds.", Next: nextRenderInTheWay},
			Refusal{Code: finding.CodeDestinationNotRegularFile, Cause: "In --out, a directory, a named pipe, a socket or a device stands at an output's path.", Next: nextRenderInTheWay},
			Refusal{Code: finding.CodeDestinationNothingWritten, Cause: "A write failed, another file system is mounted on an output's path, or an output's path begins with the staging directory's name; whatever was written is removed.", Next: nextRenderDest},
			Refusal{Code: finding.CodeDestinationStagingExists, Cause: "--out holds .metais-staging, left by a killed render or in use by another; nothing is written.", Next: nextRenderDest},
			Refusal{Code: finding.CodeDestinationCommitFailed, Cause: "Moving the outputs into an existing --out failed part way; the files already moved are listed, and nothing else was changed.", Next: nextRenderDest},
			Refusal{Code: finding.CodeUsageFlagExcluded, Cause: "--overwrite is given with --out -, and an archive on standard output replaces nothing.", Next: nextRenderOverwrite,
				match: flagGiven("--overwrite")},
			Refusal{Code: render.CodeMirrorEmptyName, Cause: "A mirror would write a template whose file name is exactly .liquid with an empty name.", Next: nextRenderTemplates},
			Refusal{Code: render.CodeMirrorEmpty, Cause: "A mirror renders no file, and --fail-on warning makes its warning fail the render.", Next: nextRenderTemplates},
		),
		Examples: []Example{
			{"metais render report-1.0.0.metais --data data --out out", "Renders the package's outputs from the data directory data into the new directory out."},
			{"metais render report.metais --data data --set /vendor/name=Acme --out out", "Renders with one value set over the data directory's."},
			{"metais render . --data data --out out --now 2026-10-19T09:30:00+02:00 --json", "Renders the source directory in the working directory with a render time, and prints the files as one metais.render/1 document."},
			{"metais render report.metais --data data --out out --dry-run", "Lists every file the render would write into out, with its size and digest, and writes nothing."},
			{"metais render report.metais --data data --out - --only report.md", "Renders the whole package and writes the bytes of report.md alone to standard output."},
			{"metais render . --sample --out preview", "Previews the source directory's outputs from sample data."},
			{"metais render . --sample --set /incident/stage=final --seed 7 --out preview", "Previews them with one value pinned and other generated values."},
			{"metais render report.metais --data data --out site --overwrite", "Renders into the existing directory site, replacing the files the render writes there and leaving every other one."},
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
		case inputFamily(e.Code):
			r = familyRefusal(e.Code, "An input is refused", strings.ReplaceAll(strings.ReplaceAll(rest, "_", " "), ".", ": "), nextRenderInputs)
		case e.Code == sample.CodeUnsatisfiable || e.Code == finding.CodeSampleInvalid:
			r = familyRefusal(e.Code, "With --sample, the generated data cannot be made or does not validate", strings.ReplaceAll(rest, "_", " "), nextSampleSchema)
		case family == "template" && e.Severity == finding.SeverityError && !strings.HasPrefix(rest, "xref.") && !strings.HasPrefix(rest, "ooxml."):
			r = familyRefusal(e.Code, "A template is refused", strings.ReplaceAll(rest, "_", " "), nextRenderTemplates)
		case strings.HasPrefix(string(e.Code), "limit.exceeded."), strings.HasPrefix(string(e.Code), "output.path."),
			strings.HasPrefix(string(e.Code), "output.expression."), strings.HasPrefix(string(e.Code), "output.container."):
			r = familyRefusal(e.Code, "An output is refused", strings.ReplaceAll(strings.ReplaceAll(rest, "_", " "), ".", ": "), nextRenderTemplates)
		case family == "output" && e.Subject == finding.SubjectOutput:
			// An output's text its kind's reader refuses, or its schema
			// does, and a validate a rendered path's kind cannot take
			// (ADR-005 D11 items 2 and 4).
			r = familyRefusal(e.Code, "An output fails its validation", strings.ReplaceAll(strings.ReplaceAll(rest, "_", " "), ".", ": "), nextRenderTemplates)
		default:
			continue
		}
		c.Refusals = append(c.Refusals, r)
	}
}

// inputFamily reports whether code is one a supply reports about an
// input: the input families, and limit.exceeded.
func inputFamily(code finding.Code) bool {
	family, _, _ := strings.Cut(string(code), ".")
	return family == "input" || family == "json" || family == "yaml" || family == "csv" || family == "text" ||
		family == "markdown" || family == "collection" || code == finding.CodeLimitExceeded
}

// renderResult is metais.render/1's result (ADR-008 D8 item 9).
type renderResult struct {
	Out    string        `json:"out"`
	DryRun bool          `json:"dry_run"`
	Only   *string       `json:"only"`
	Files  []renderFile  `json:"files"`
	Sample *renderSample `json:"sample"`
}

// renderSample is the result's sample: the generator's options of a
// render with --sample (ADR-008 D8 item 9).
type renderSample struct {
	Optional string `json:"optional"`
	Seed     uint64 `json:"seed"`
	Items    int    `json:"items"`
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
	// Status is set only when --out is a directory (ADR-005 D16 item 2).
	Status string `json:"status,omitempty"`
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

// runRender is metais render, of template outputs, assets, mirrors and
// containers into a directory — one that does not exist, or an existing
// one, which --overwrite needs when it holds entries — or to standard
// output, or as a dry run (ADR-008 D8, ADR-005 D13 to D18).
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
	// --out - writes bytes to standard output, which then carries them
	// and nothing else (ADR-008 D11 item 2, emit's streamsBytes); a dry
	// run writes none.
	toStream, dryRun := out == "-", p.Has("dry-run")
	only, hasOnly := p.Value("only")
	if i, ok := p.Arg("overwrite"); ok && toStream {
		// An archive replaces nothing (ADR-008 D8 item 4).
		return s.refuseWith(finding.New(finding.CodeUsageFlagExcluded, invocation, nil, finding.Params{
			"argument": finding.Int(int64(s.line.at[i] + 1)),
			"command":  finding.String("render"),
			"flag":     finding.String("--overwrite"),
			"other":    finding.String("--out -"),
		}))
	}
	if toStream && !dryRun {
		if !hasOnly && isTerminal(s.inv.Stdout) {
			return s.refuseWith(finding.New(finding.CodeUsageArchiveToTerminal, invocation, nil,
				finding.Params{"argument": s.flagValueArg(p, "out")}))
		}
	}
	now, hasNow := p.Value("now")
	if hasNow {
		if _, err := dates.ParseNow(now); err != nil {
			return s.refuseWith(finding.New(finding.CodeUsageBadValue, invocation, nil, finding.Params{
				"argument": s.flagValueArg(p, "now"),
				"flag":     finding.String("--now"),
				"reason":   finding.String("value_not_allowed"),
			}))
		}
	}
	// A render from sample data is the author's preview, an authoring
	// command (ADR-008 D8 item 8, ADR-006 D12 item 2).
	isSample := p.Has(sampleFlag.Name)
	var opts sample.Options
	if isSample {
		s.role = finding.RoleAuthoring
		if p.Has(dataFlag.Name) {
			return s.refuseWith(s.excluded(p, dataFlag.Name, sampleFlag.Name))
		}
		var fs []finding.Finding
		if opts, fs = s.sampleOptions(p); len(fs) > 0 {
			return s.refuseWith(fs[0])
		}
	} else {
		for _, f := range generatorFlags {
			if p.Has(f.Name) {
				return s.refuseWith(s.requires(p, f.Name, sampleFlag.Name))
			}
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
	// A stream reads the package as emit writes it, after runRender has
	// returned, so the package is closed by the stream when there is one.
	streaming := false
	defer func() {
		if !streaming {
			pkg.Close()
		}
	}()

	// Phase 0: the inputs (ADR-005 D13 item 0).
	// With --sample, the generated data, validated, is the data
	// directory's (ADR-007 D9 item 22).
	dataDir, _ := p.Value(dataFlag.Name)
	var res *inputs.Result
	if isSample {
		var data *sampled
		if data, fs = s.generate(p, pkg, lim, opts); data != nil {
			res = data.res
		}
	} else {
		res, fs = s.supply(p, pkg, lim)
	}
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
	// Phase 3: the destination, when it is a directory, every refusal
	// reported with the others (ADR-005 D15 item 8). Phase 5: emit
	// (ADR-005 D13 item 5), to exactly one of one file's bytes (D18 item
	// 1), the archive (D17), and the directory, every file or the one
	// selected (D14, D18 item 3), or with --dry-run to the sink's checks
	// alone, which write nothing (D16).
	faults := s.inv.Options.Faults
	var sk sink.Sink
	switch {
	case toStream && hasOnly:
		sk = sink.Content()
	case toStream:
		sk = sink.Archive()
	default:
		sk = sink.Directory(out, p.Has("overwrite"), lim, faults)
	}
	if dryRun {
		sk = sink.DryRun(sk)
	}
	defer sk.Close()
	// Phases 1 to 5, with --only's choice of what is written made from the
	// whole render, which has run and been verified (ADR-005 D18 items 1,
	// 2 and 4).
	rr := render.Run(render.Options{
		Package: pkg, Templates: ts, Bindings: b, Limits: lim, Seams: faults,
		Fails: s.fails, Only: only, HasOnly: hasOnly,
	}, sk)
	fs = append(fs, rr.Findings...)
	switch {
	case !rr.Produced:
		// A warning that --fail-on warning makes a failure fails the
		// render before it writes anything (ADR-006 D2 item 5).
		return outcome{findings: fs, next: s.nextSteps(fs), dataDir: dataDir}
	case !rr.Selected:
		fs = append(fs, s.unknownOutput(p, rr.Files))
		return outcome{findings: fs, next: s.nextSteps(fs), dataDir: dataDir}
	case !rr.Emitted:
		return outcome{findings: fs, next: s.nextSteps(fs)}
	}
	written, stream := rr.Written, rr.Stream
	result := &renderResult{Out: "directory", DryRun: dryRun, Files: []renderFile{}}
	if isSample {
		result.Sample = &renderSample{Optional: opts.Optional.String(), Seed: opts.Seed, Items: opts.Items}
	}
	if toStream {
		result.Out = "stdout"
	}
	if hasOnly {
		result.Only = &only
	}
	if stream != nil {
		streaming = true
		emit := stream
		stream = func(w io.Writer) {
			defer pkg.Close()
			emit(w)
		}
	}
	var human strings.Builder
	if dryRun {
		human.WriteString("would write\n")
	}
	for _, pf := range written {
		size, d := pf.Measured()
		var entries strings.Builder
		rf := renderFile{
			Path: pf.Path(), Output: pf.Output(), Item: pf.Item(), Source: pf.Source(), Size: size, Digest: d,
			Executable: pf.Executable(), Entries: entryRecords(pf, &entries, 1), Kind: pf.Kind(),
		}
		rf.Status = sk.Status(pf.Path())
		result.Files = append(result.Files, rf)
		human.WriteString(strconv.FormatInt(size, 10) + "\t" + d + "\t")
		human.Write(finding.AppendText(nil, pf.Path()))
		human.WriteString("\n" + entries.String())
	}
	o := outcome{findings: fs, result: result, dataDir: dataDir, stream: stream}
	if stream == nil {
		// With --out -, standard output is the bytes, and the listing is
		// in the document alone (ADR-008 D8 item 9).
		o.human = []byte(human.String())
	}
	return o
}

// maxOutputsListed is how many paths usage.unknown_output lists, as many
// as ADR-006 D6 lists of an enum's allowed values.
const maxOutputsListed = 20

// unknownOutput is usage.unknown_output for --only's value, which no file
// of plan, in path order, has (ADR-005 D18 item 4).
func (s *session) unknownOutput(p args.Parsed, plan []*render.File) finding.Finding {
	var paths []string
	for _, pf := range plan[:min(len(plan), maxOutputsListed)] {
		paths = append(paths, pf.Path())
	}
	return finding.New(finding.CodeUsageUnknownOutput, invocation, nil, finding.Params{
		"argument":   s.flagValueArg(p, "only"),
		"paths":      stringList(paths),
		"path_count": finding.Int(int64(len(plan))),
	})
}

// flagValueArg is the 1-based position on the whole line of the value of
// the flag called name, which was given: the flag's own, written
// --name=value, or the one after it.
func (s *session) flagValueArg(p args.Parsed, name string) finding.Value {
	i, _ := p.Arg(name)
	at := s.line.at[i] + 1
	if !strings.Contains(s.inv.Args[s.line.at[i]], "=") {
		at++
	}
	return finding.Int(int64(at))
}

// entryRecords is the records of pf's entries, when pf is a container,
// and their human lines, each indented depth levels, written to human.
func entryRecords(pf *render.File, human *strings.Builder, depth int) []renderEntry {
	if !pf.IsContainer() {
		return nil
	}
	var lines strings.Builder
	out := make([]renderEntry, 0, len(pf.Entries()))
	for _, e := range pf.Entries() {
		size, d := e.Measured()
		r := renderEntry{Path: e.Path(), Item: e.Item(), Source: e.Source(), Kind: e.Kind(), Method: "stored", Size: size, Digest: d}
		if id := e.Output(); id != "" {
			r.Output = &id
		}
		if e.Deflated() {
			r.Method = "deflated"
		}
		lines.WriteString(strings.Repeat("  ", depth) + strconv.FormatInt(size, 10) + "\t" + d + "\t")
		lines.Write(finding.AppendText(nil, e.Path()))
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
