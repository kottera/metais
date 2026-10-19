// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	pathpkg "path"
	"slices"
	"strconv"
	"strings"

	"github.com/kottera/metais/internal/cli/args"
	"github.com/kottera/metais/internal/dates"
	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/inputs"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/pkgtree"
	"github.com/kottera/metais/internal/sample"
	"github.com/kottera/metais/internal/schema"
	"github.com/kottera/metais/internal/tarw"
	"github.com/kottera/metais/internal/template"
	"github.com/kottera/metais/internal/unitext"
	"github.com/kottera/metais/internal/version"
	"github.com/kottera/metais/internal/zipfmt"
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
			Refusal{finding.CodeUsageMissingFlag, "--out is not given.", nextRenderOut},
			Refusal{finding.CodeUsageArchiveToTerminal, "--out - is given without --only, and standard output is a terminal.", nextRenderArchive},
			Refusal{finding.CodeUsageUnknownOutput, "--only names no file the render writes.", nextRenderDryRun},
			Refusal{finding.CodeUsageBadValue, "--now is not an RFC 3339 date-time with an offset, or --seed or --items is not a whole number in range.", nextRenderNow},
			Refusal{finding.CodeUsageFlagExcluded, "--data is given with --sample, whose generated data takes the data directory's place.", nextRenderSample},
			Refusal{finding.CodeUsageFlagRequires, "--optional, --seed or --items is given without --sample.", nextRenderSample},
			Refusal{finding.CodeUsageDataDirUnreadable, "The data directory given to --data does not exist, is not a directory, or cannot be read.", nextRenderData},
			Refusal{finding.CodeUsageConflictingFlags, "--inputs-json - is given twice, and standard input is read once.", nextInputsHelp},
			Refusal{finding.CodeDestinationNotDirectory, "--out, or a directory above it, exists and is not a directory; or, in --out, a file stands where an output's path needs a directory.", nextRenderInTheWay},
			Refusal{finding.CodeDestinationNotEmpty, "--out is a directory that holds entries, and --overwrite is not given.", nextRenderOverwrite},
			Refusal{finding.CodeDestinationLink, "In --out, an output's path runs through, or ends at, a symbolic link, a junction or another reparse point, which is never followed.", nextRenderInTheWay},
			Refusal{finding.CodeDestinationCaseVariant, "In --out, an entry's name differs only in case or normalization from a name an output's path holds.", nextRenderInTheWay},
			Refusal{finding.CodeDestinationNotRegularFile, "In --out, a directory, a named pipe, a socket or a device stands at an output's path.", nextRenderInTheWay},
			Refusal{finding.CodeDestinationWriteFailed, "A write failed, another file system is mounted on an output's path, a staging directory is left in --out, or an output's path begins with its name; whatever was written is removed.", nextRenderDest},
			Refusal{finding.CodeDestinationCommitFailed, "Moving the outputs into an existing --out failed part way; the files already moved are listed, and nothing else was changed.", nextRenderDest},
			Refusal{finding.CodeUsageFlagExcluded, "--overwrite is given with --out -, and an archive on standard output replaces nothing.", nextRenderOverwrite},
			Refusal{codeMirrorEmptyName, "A mirror would write a template whose file name is exactly .liquid with an empty name.", nextRenderTemplates},
			Refusal{codeMirrorEmpty, "A mirror renders no file, and --fail-on warning makes its warning fail the render.", nextRenderTemplates},
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
			r = Refusal{e.Code, "An input is refused (" + strings.ReplaceAll(strings.ReplaceAll(rest, "_", " "), ".", ": ") + ").", nextRenderInputs}
		case e.Code == sample.CodeUnsatisfiable || e.Code == finding.CodeSampleInvalid:
			r = Refusal{e.Code, "With --sample, the generated data cannot be made or does not validate (" + strings.ReplaceAll(rest, "_", " ") + ").", nextSampleSchema}
		case family == "template" && e.Severity == finding.SeverityError && !strings.HasPrefix(rest, "xref.") && !strings.HasPrefix(rest, "ooxml."):
			r = Refusal{e.Code, "A template is refused (" + strings.ReplaceAll(rest, "_", " ") + ").", nextRenderTemplates}
		case strings.HasPrefix(string(e.Code), "limit.exceeded."), strings.HasPrefix(string(e.Code), "output.path."),
			strings.HasPrefix(string(e.Code), "output.expression."), strings.HasPrefix(string(e.Code), "output.container."):
			r = Refusal{e.Code, "An output is refused (" + strings.ReplaceAll(strings.ReplaceAll(rest, "_", " "), ".", ": ") + ").", nextRenderTemplates}
		case family == "output" && e.Subject == finding.SubjectOutput:
			// An output's text its kind's reader refuses, or its schema
			// does, and a validate a rendered path's kind cannot take
			// (ADR-005 D11 items 2 and 4).
			r = Refusal{e.Code, "An output fails its validation (" + strings.ReplaceAll(strings.ReplaceAll(rest, "_", " "), ".", ": ") + ").", nextRenderTemplates}
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

// sampleRefusals adds to sample's declared refusals the codes a supply
// reports about an input, and the generator's own (ADR-008 D16 item 2).
func sampleRefusals(c *Command) {
	have := map[finding.Code]bool{}
	for _, r := range c.Refusals {
		have[r.Code] = true
	}
	for _, e := range finding.Registry() {
		_, rest, _ := strings.Cut(string(e.Code), ".")
		switch {
		case have[e.Code]:
		case inputFamily(e.Code):
			c.Refusals = append(c.Refusals, Refusal{e.Code, "A value given with --set, --set-json or --inputs-json is refused (" + strings.ReplaceAll(strings.ReplaceAll(rest, "_", " "), ".", ": ") + ").", nextSampleInputs})
		case e.Code == sample.CodeUnsatisfiable || e.Code == finding.CodeSampleInvalid:
			c.Refusals = append(c.Refusals, Refusal{e.Code, "The generated data cannot be made or does not validate (" + strings.ReplaceAll(rest, "_", " ") + ").", nextSampleSchema})
		}
	}
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

// planned is one file a render plans (ADR-005 D13 phase 1): its
// declaration, its path within its tree, where in the manifest the path
// comes from, the evaluation of a fan-out it is, what it is written from,
// and its bytes once rendered. A file copied from an asset holds no
// bytes: they are streamed from the package as they are written (D13 item
// 4).
type planned struct {
	decl manifest.Output
	path string
	ptr  string
	// template is the template a template output or a mirror renders the
	// file from, beneath templates/.
	template string
	// asset is the entry an asset declaration copies the file from, and
	// tree the package's tree it is read from.
	asset string
	tree  pkgtree.Tree
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
	// size and digest are a container's zip's, measured in phase 4, or an
	// asset's entry's, its digest taken as it is first read; deflated is
	// whether its container deflates an entry.
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

// source is the record's source of pf (ADR-005 D16 item 2): template for
// a file a template output or a mirror renders, asset, container, or
// profile for the mimetype entry a profile writes.
func (pf *planned) source() string {
	switch {
	case pf.tool:
		return "profile"
	case pf.asset != "":
		return "asset"
	case pf.isContainer():
		return "container"
	}
	return "template"
}

// kind is the kind of pf (ADR-005 D7), or nil for an asset, a container
// and a profile's mimetype, which have none (D7 item 3).
func (pf *planned) kind() *string {
	if pf.asset != "" || pf.isContainer() || pf.tool {
		return nil
	}
	k := kindOf(pf.path, pf.decl.Kind)
	return &k
}

// joinIn is the path p within the tree of the container at in.
func joinIn(in, p string) string {
	if in == "" {
		return p
	}
	return in + "/" + p
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
	// One renderer, whose work the manifest's expressions and the
	// templates share (ADR-009 D11).
	r := template.NewRenderer(ts, b, lim, s.inv.Options.Faults)
	// Phases 1 and 2: plan and verify paths.
	plan, pfs := planOutputs(pkg, lim, s, r)
	fs = append(fs, pfs...)
	// Phase 3: the destination, when it is a directory, every refusal
	// reported with the others (ADR-005 D15 item 8).
	var dest *destination
	if !toStream {
		var dfs []finding.Finding
		dest, dfs = checkDestination(out, p.Has("overwrite"), plan, lim, s.inv.Options.Faults)
		defer dest.close()
		fs = append(fs, dfs...)
	}
	// The output validation schemas, compiled before any output is
	// rendered (ADR-005 D11 item 3).
	schemas, sfs := outputSchemas(pkg, lim)
	fs = append(fs, sfs...)
	// Phase 4: render every output, each to its first error, and
	// transform and validate each file that rendered (ADR-005 D13 item 4).
	fs = append(fs, renderPlan(pkg, plan, ts, r, lim, schemas)...)
	// A warning that --fail-on warning makes a failure fails the render
	// before it writes anything (ADR-006 D2 item 5).
	if s.fails(fs) {
		return outcome{findings: fs, next: s.nextSteps(fs), dataDir: dataDir}
	}
	slices.SortFunc(plan, func(a, b *planned) int { return strings.Compare(a.path, b.path) })
	// --only chooses what is written from the whole render, which has
	// run and been verified (ADR-005 D18 items 1, 2 and 4).
	written := plan
	if hasOnly {
		i := slices.IndexFunc(plan, func(pf *planned) bool { return pf.path == only })
		if i < 0 {
			fs = append(fs, s.unknownOutput(p, plan))
			return outcome{findings: fs, next: s.nextSteps(fs), dataDir: dataDir}
		}
		written = plan[i : i+1]
	}
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
	// Phase 5: emit (ADR-005 D13 item 5), in exactly one of a dry run,
	// which writes nothing (D16), the archive (D17), one file's bytes
	// (D18 item 1), and the directory commit of every file, or of the
	// one selected (D14 item 1, D18 item 3).
	var stream func(io.Writer)
	switch {
	case dryRun:
	case toStream && hasOnly:
		stream = func(w io.Writer) { writeContent(w, written[0]) }
	case toStream:
		stream = func(w io.Writer) { writeArchive(w, written) }
	default:
		if f, ok := commit(out, dest, written, s); !ok {
			fs = append(fs, f)
			return outcome{findings: fs, next: s.nextSteps(fs)}
		}
	}
	// An asset's digest is taken as it is first read; a dry run, the
	// archive and --only read it after the records are made, or never,
	// so it is read for its digest here (ADR-005 D16 item 2).
	for _, pf := range written {
		if pf.asset != "" && pf.digest == "" {
			r, done := content(pf)
			_, err := io.Copy(io.Discard, r)
			if derr := done(err); err != nil || derr != nil {
				panic("cli: reading an asset the plan found: " + errors.Join(err, derr).Error())
			}
		}
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
		size, d := pf.measured()
		var entries strings.Builder
		rf := renderFile{
			Path: pf.path, Output: pf.decl.ID, Item: pf.item(), Source: pf.source(), Size: size, Digest: d,
			Executable: pf.decl.Executable, Entries: entryRecords(pf, &entries, 1), Kind: pf.kind(),
		}
		if !toStream {
			rf.Status = string(hostfs.Create)
			if dest != nil {
				rf.Status = string(dest.status[pf.path])
			}
		}
		result.Files = append(result.Files, rf)
		human.WriteString(strconv.FormatInt(size, 10) + "\t" + d + "\t")
		human.Write(finding.AppendText(nil, pf.path))
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
func (s *session) unknownOutput(p args.Parsed, plan []*planned) finding.Finding {
	var paths []string
	for _, pf := range plan[:min(len(plan), maxOutputsListed)] {
		paths = append(paths, pf.path)
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

// writeArchive writes plan, in path order, to w as ADR-005 D17's archive,
// built from the rendered bytes and never from disk (item 6). A failed
// write to w ends it unreported, as every write to standard output is
// (ADR-006 D12 item 5); any other error is a defect.
func writeArchive(w io.Writer, plan []*planned) {
	ew := &stickyWriter{w: w}
	tw := tarw.NewWriter(ew)
	for _, pf := range plan {
		size, _ := pf.measured()
		r, done := content(pf)
		err := tw.Add(pf.path, pf.decl.Executable, size, r)
		if derr := done(err); err == nil && derr != nil {
			panic("cli: a measured container does not write: " + derr.Error())
		}
		if err != nil {
			if ew.err != nil {
				return
			}
			panic("cli: " + err.Error())
		}
	}
	if err := tw.Close(); err != nil && ew.err == nil {
		panic("cli: " + err.Error())
	}
}

// writeContent writes pf's bytes alone to w (ADR-005 D18 item 1): an
// asset's as the package holds them, and a container's zip as it is
// assembled. A failed write ends it unreported.
func writeContent(w io.Writer, pf *planned) {
	ew := &stickyWriter{w: w}
	r, done := content(pf)
	_, err := io.Copy(ew, r)
	if derr := done(err); err == nil && derr != nil {
		panic("cli: a measured container does not write: " + derr.Error())
	}
	if err != nil && ew.err == nil {
		panic("cli: " + err.Error())
	}
}

// stickyWriter is w that keeps the first error a write returns.
type stickyWriter struct {
	w   io.Writer
	err error
}

func (sw *stickyWriter) Write(b []byte) (int, error) {
	if sw.err != nil {
		return 0, sw.err
	}
	n, err := sw.w.Write(b)
	sw.err = err
	return n, err
}

// measured is pf's size and digest: a container's zip's, an asset's
// entry's, or its bytes'.
func (pf *planned) measured() (int64, string) {
	if pf.isContainer() || pf.asset != "" {
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
		r := renderEntry{Path: e.path, Item: e.item(), Source: e.source(), Kind: e.kind(), Method: "stored", Size: size, Digest: d}
		if e.decl.ID != "" && !e.tool {
			id := e.decl.ID
			r.Output = &id
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
	p := &planner{pkg: pkg, lim: lim, s: s, r: r, counted: true,
		entries: pkg.Tree.List(), explicit: explicitTemplates(pkg.Manifest.Outputs)}
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
	// entries are the package's, and explicit the templates a template
	// output names, which no mirror renders (ADR-005 D6 item 3).
	entries  []string
	explicit map[string]bool
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
// and each container's entries; an asset's or a mirror's path names the
// destination of the files it expands into (D5, D6). Before each evaluation but the first, a
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
			pf := &planned{decl: d, fan: fan, ptr: d.Pointer + "/path", literal: x.Literal, owner: owner, in: in, template: d.Source}
			if x.Path == nil {
				pf.path, pf.ptr, pf.literal = d.Source, d.Pointer+"/"+string(d.Form), true
				if d.Form == manifest.OutputTemplate {
					pf.path = strings.TrimSuffix(d.Source, ".liquid")
				}
			} else {
				pf.path, f = p.r.Path(x, fan)
			}
			if f == nil {
				switch d.Form {
				case manifest.OutputAsset:
					expanded := p.asset(pf)
					p.all, files = append(p.all, expanded...), append(files, expanded...)
				case manifest.OutputMirror:
					expanded := p.mirror(pf)
					p.all, files = append(p.all, expanded...), append(files, expanded...)
				default:
					p.all = append(p.all, pf)
					if d.Form == manifest.OutputContainer {
						pf.entries = p.container(pf, depth+1)
					}
					files = append(files, pf)
				}
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

// asset is the files of the asset declaration's evaluation at, whose path
// names their destination (ADR-005 D5): the file assets/<path>, copied to
// the path; or every file beneath the directory assets/<path>, "."
// naming assets/ itself, each at its path relative to that directory
// beneath the destination, "." being the root of the tree. Each file's
// size is the entry's, and its bytes are not read. The loader has refused
// an asset that names no entry.
func (p *planner) asset(at *planned) []*planned {
	src := at.decl.Source
	file := func(entry, path string) *planned {
		info, err := p.pkg.Tree.Stat(entry)
		if err != nil {
			panic("cli: an entry the plan names has no Stat: " + entry)
		}
		pf := *at
		pf.path, pf.template, pf.asset, pf.tree, pf.size = path, "", entry, p.pkg.Tree, info.Size
		return &pf
	}
	if _, err := p.pkg.Tree.Stat(template.AssetDir + src); err == nil {
		return []*planned{file(template.AssetDir+src, at.path)}
	}
	var out []*planned
	for _, e := range p.entries {
		if rel, ok := strings.CutPrefix(e, treePrefix(template.AssetDir, src)); ok {
			out = append(out, file(e, beneath(at.path, rel)))
		}
	}
	return out
}

// mirror is the files of the mirror's evaluation at, whose path names
// their destination (ADR-005 D6): every template beneath templates/<dir>,
// "." naming templates/ itself, in byte order, but one with a name
// beginning with _ at any depth beneath the mirror's root and one a
// template field names anywhere in the manifest, each rendered to its path
// relative to the mirror's root, with a final .liquid removed, beneath the
// destination. A template named exactly .liquid would be written with an
// empty name, and is refused; a mirror that yields no file is a warning.
func (p *planner) mirror(at *planned) []*planned {
	d := at.decl
	var out []*planned
	refused := false
	subj := finding.Subject{Kind: finding.SubjectOutput, Name: at.owner}
	loc := p.at(d.Pointer + "/mirror")
	for _, e := range p.entries {
		rel, ok := strings.CutPrefix(e, treePrefix(template.Dir, d.Source))
		name := strings.TrimPrefix(e, template.Dir)
		if !ok || p.explicit[name] || slices.ContainsFunc(strings.Split(rel, "/"), func(seg string) bool { return strings.HasPrefix(seg, "_") }) {
			continue
		}
		if pathpkg.Base(rel) == ".liquid" {
			refused = true
			p.fs = append(p.fs, finding.New(codeMirrorEmptyName, subj, &loc, finding.Params{"template": finding.String(name)}))
			continue
		}
		pf := *at
		pf.path, pf.template = beneath(at.path, strings.TrimSuffix(rel, ".liquid")), name
		out = append(out, &pf)
	}
	if len(out) == 0 && !refused {
		p.fs = append(p.fs, finding.New(codeMirrorEmpty, subj, &loc, finding.Params{"directory": finding.String(d.Source)}))
	}
	return out
}

// The codes of an output declaration a render reports (ADR-005's
// interfaces name the family output.declaration).
const (
	codeMirrorEmptyName finding.Code = "output.declaration.empty_name"
	codeMirrorEmpty     finding.Code = "output.declaration.empty_mirror"
)

// treePrefix is the prefix of every entry beneath the directory rel of the
// layout directory dir, "." naming dir itself.
func treePrefix(dir, rel string) string {
	if rel == "." {
		return dir
	}
	return dir + rel + "/"
}

// beneath is the path rel beneath the destination directory dest, "."
// being the root of the tree. The paths are joined as written, never
// cleaned: a path is exactly what the manifest gives (ADR-005 D4 item 2).
func beneath(dest, rel string) string {
	if dest == "." {
		return rel
	}
	return dest + "/" + rel
}

// explicitTemplates is every template a template output names, at any
// depth of a container: no mirror renders one (ADR-005 D6 item 3).
func explicitTemplates(decls []manifest.Output) map[string]bool {
	out := map[string]bool{}
	var walk func([]manifest.Output)
	walk = func(ds []manifest.Output) {
		for _, d := range ds {
			switch d.Form {
			case manifest.OutputTemplate:
				out[d.Source] = true
			case manifest.OutputContainer:
				walk(d.Container.Entries)
			}
		}
	}
	walk(decls)
	return out
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

// destination is ADR-005 D13's phase 3 for an --out that exists: the
// directory, opened, and D15 item 4's check of each top-level planned
// path, as Amendment A1 items 4 and 5 extend it, by path. A render into a
// directory that does not exist has none.
type destination struct {
	x      *hostfs.Existing
	checks []hostfs.PathCheck
	status map[string]hostfs.Status
}

// close releases the directory, unless a commit already has.
func (d *destination) close() {
	if d != nil {
		d.x.Close()
	}
}

// checkDestination is ADR-005 D13's phase 3, D15, for out and the
// top-level files of plan, every refusal a finding reported with the
// others before anything is written (D15 item 8). An out that does not
// exist is D14 item 1's, and has no destination. An out that is, or lies
// beneath, a file is refused. An existing directory is D14 item 2's, as
// Amendment A1 amends it: one with no entry at all is written into as it
// is (D15 item 3); one that holds any entry, a hidden one included, is
// refused unless overwrite; a leftover staging directory in it refuses the
// render (A1 item 1); and each planned path that passed the path checks
// is checked through the directory (D15 item 4), so a link, a file where
// a directory must be, a case variant, a directory or non-regular target,
// another device, or the staging name refuses it. Nothing in the
// directory is read: only names and Lstat.
func checkDestination(out string, overwrite bool, plan []*planned, lim limits.Set, faults fault.Set) (*destination, []finding.Finding) {
	w, rest, err := hostfs.OpenNearest(out)
	if err != nil {
		if hostfs.ReasonOf(err) == hostfs.NotDirectory {
			return nil, []finding.Finding{finding.New(finding.CodeDestinationNotDirectory, invocation, nil, nil)}
		}
		return nil, []finding.Finding{writeFailed(hostfs.ReasonOf(err))}
	}
	w.Close()
	if rest != "." {
		return nil, nil
	}
	x, err := hostfs.OpenExisting(out, hostfs.WithFaults(faults))
	if err != nil {
		if hostfs.ReasonOf(err) == hostfs.NotDirectory {
			return nil, []finding.Finding{finding.New(finding.CodeDestinationNotDirectory, invocation, nil, nil)}
		}
		return nil, []finding.Finding{writeFailed(hostfs.ReasonOf(err))}
	}
	d := &destination{x: x, status: map[string]hostfs.Status{}}
	var fs []finding.Finding
	if !overwrite {
		entries, err := x.List(maxListed + 1)
		if err != nil && hostfs.ReasonOf(err) != hostfs.TooLarge {
			d.close()
			return nil, []finding.Finding{writeFailed(hostfs.ReasonOf(err))}
		}
		if len(entries) > 0 {
			var names []string
			for _, e := range entries[:min(len(entries), maxListed)] {
				names = append(names, e.Name)
			}
			slices.Sort(names)
			fs = append(fs, finding.New(finding.CodeDestinationNotEmpty, invocation, nil, finding.Params{
				"entries": stringList(names),
				"more":    finding.Bool(len(entries) > maxListed),
			}))
		}
	}
	if _, err := x.Lstat(hostfs.StagingDir); err == nil {
		fs = append(fs, writeFailed(hostfs.Exists))
	} else if r := hostfs.ReasonOf(err); r != hostfs.NotFound {
		fs = append(fs, writeFailed(r))
	}
	// A path that breaks D12 has its own finding, and is not checked here;
	// the output path check's seam (hostfs.CheckPaths) decides which.
	bad := map[string]bool{}
	for _, p := range hostfs.CheckPaths(hostfs.WithFaults(faults), func() []string {
		var bad []string
		for _, pf := range plan {
			if len(unitext.CheckPath(pf.path, lim)) > 0 {
				bad = append(bad, pf.path)
			}
		}
		return bad
	}) {
		bad[p] = true
	}
	var paths []string
	for _, pf := range plan {
		if !bad[pf.path] {
			paths = append(paths, pf.path)
		}
	}
	d.checks = x.Check(paths)
	for _, c := range d.checks {
		if c.Err != nil {
			fs = append(fs, pathRefusal(c.Path, c.Err))
			continue
		}
		d.status[c.Path] = c.Status
	}
	return d, fs
}

// pathRefusal is the finding of err, the refusal of ADR-005 D15 item 4's
// checks of the planned path p, as Amendment A1 items 4 and 5 extend them:
// a link anywhere on the path (ADR-009 D8 item 3), a file where a directory
// must be, a case variant, and a directory or non-regular target, each
// with its own code naming p; another device, the staging name, and an
// error that stopped the check, output.destination.write_failed.
func pathRefusal(p string, err error) finding.Finding {
	params := finding.Params{"path": finding.String(p)}
	switch r := hostfs.ReasonOf(err); r {
	case hostfs.Link:
		return finding.New(finding.CodeDestinationLink, invocation, nil, params)
	case hostfs.NotDirectory:
		return finding.New(finding.CodeDestinationNotDirectory, invocation, nil, params)
	case hostfs.CaseVariant:
		return finding.New(finding.CodeDestinationCaseVariant, invocation, nil, params)
	case hostfs.IsDirectory, hostfs.NotRegularFile:
		params["reason"] = finding.String(string(r))
		return finding.New(finding.CodeDestinationNotRegularFile, invocation, nil, params)
	default:
		return writeFailed(r)
	}
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
// of each file that rendered are held in its planned file; a file copied
// from an asset is not read, its size counted against the output limits
// with the bytes of every other file (ADR-005 D13 item 4), and the asset
// tag renders through the plan's placement map (D5 item 6). Then each
// container whose entries all rendered is measured, and held to
// output-size.
func renderPlan(pkg *loader.Package, plan []*planned, ts *template.Templates, r *template.Renderer, lim limits.Set, schemas map[string]*schema.Compiler) []finding.Finding {
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
	// A mirror's templates, and the placement map, from the plan.
	placed := map[string][]template.Placement{}
	var walk func(tree []*planned)
	walk = func(tree []*planned) {
		for _, pf := range tree {
			switch {
			case pf.isContainer():
				walk(pf.entries)
			case pf.asset != "":
				name := strings.TrimPrefix(pf.asset, template.AssetDir)
				placed[name] = append(placed[name], template.Placement{Tree: pf.in, Path: pf.path})
			case pf.decl.Form == manifest.OutputMirror:
				if _, seen := ready[pf.template]; seen {
					continue
				}
				ready[pf.template] = true
				sources = append(sources, pf.template)
				for _, name := range ts.Reach(pf.template) {
					_, pfs := ts.Parse(name)
					if len(pfs) > 0 {
						ready[pf.template] = false
					}
					if !reported[name] {
						reported[name] = true
						fs = append(fs, pfs...)
					}
				}
			}
		}
	}
	walk(plan)
	r.PlaceAssets(placed)
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
			case pf.asset != "":
				o := template.Output{ID: pf.owner, Path: pf.path, In: pf.in}
				if f := r.ChargeAsset(o, pf.size); f != nil {
					fs = append(fs, *f)
					pf.failed = true
				}
				continue
			case !ready[pf.template]:
				pf.failed = true
				continue
			}
			out, f := r.Render(template.Output{ID: pf.owner, Template: pf.template, Path: pf.path, Fan: pf.fan, Kind: kindOf(pf.path, pf.decl.Kind), In: pf.in, HTML: pf.decl.Escape == "html"})
			if f != nil {
				fs = append(fs, *f)
				pf.failed = true
				continue
			}
			pf.bytes = out
			fs = append(fs, transform(pkg, pf, schemas, lim)...)
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
			r, done := content(e)
			_, err := io.Copy(w, r)
			return done(err)
		}}
	}
	return zipfmt.WriteParts(w, parts)
}

// content is the reader of pf's bytes as they are written: an asset's are
// streamed from the package, its digest taken on the first whole read; a
// container's zip is written into a pipe as it is read; and done reports
// the error of its writing, once the reader is finished with.
func content(pf *planned) (r io.Reader, done func(read error) error) {
	if pf.asset != "" {
		rc, err := pf.tree.Open(pf.asset)
		if err != nil {
			panic("cli: opening " + pf.asset + ", which the plan found: " + err.Error())
		}
		h := sha256.New()
		return io.TeeReader(rc, h), func(read error) error {
			rc.Close()
			if read == nil && pf.digest == "" {
				pf.digest = "sha256:" + hex.EncodeToString(h.Sum(nil))
			}
			return nil
		}
	}
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

// commit writes plan, in path order, to out (ADR-005 D14): into dest,
// the existing directory phase 3 checked, by commitExisting; otherwise
// beneath a staging directory beside out, renamed to out (item 1). A
// failure removes what the render created and reports
// output.destination.write_failed, with nothing written.
func commit(out string, dest *destination, plan []*planned, s *session) (finding.Finding, bool) {
	if dest != nil {
		return commitExisting(dest, plan)
	}
	o, err := hostfs.StageNew(out, hostfs.WithFaults(s.inv.Options.Faults))
	if err != nil {
		if hostfs.ReasonOf(err) == hostfs.NotDirectory {
			return finding.New(finding.CodeDestinationNotDirectory, invocation, nil, nil), false
		}
		return writeFailed(hostfs.ReasonOf(err)), false
	}
	if f, ok := prepare(o, plan); !ok {
		return f, false
	}
	if err := o.Commit(); err != nil {
		return writeFailed(hostfs.ReasonOf(err)), false
	}
	return finding.Finding{}, true
}

// commitExisting writes plan into dest under ADR-005 D14 item 2, as
// Amendment A1 amends it: every file is prepared beneath the one staging
// directory inside it (item 1), which a leftover refuses with reason
// exists, and is committed by unit (item 2), every check made again
// first. A failure before the first unit is renamed leaves the directory
// as it was, and is output.destination.write_failed; one after is
// output.destination.commit_failed, listing the paths already committed
// (item 3).
func commitExisting(dest *destination, plan []*planned) (finding.Finding, bool) {
	defer dest.close()
	// --only writes the one file selected, by the same rules (ADR-005 D18
	// item 3); every planned path was checked all the same, so a selected
	// render is refused exactly when the whole one is (item 2).
	write := map[string]bool{}
	for _, pf := range plan {
		write[pf.path] = true
	}
	var checks []hostfs.PathCheck
	for _, c := range dest.checks {
		if write[c.Path] {
			checks = append(checks, c)
		}
	}
	o, err := dest.x.Stage(checks)
	if err != nil {
		return writeFailed(hostfs.ReasonOf(err)), false
	}
	if f, ok := prepare(o, plan); !ok {
		return f, false
	}
	err = o.Commit()
	if err == nil {
		return finding.Finding{}, true
	}
	var ce *hostfs.CommitError
	if !errors.As(err, &ce) || len(ce.Committed) == 0 {
		return writeFailed(hostfs.ReasonOf(err)), false
	}
	return finding.New(finding.CodeDestinationCommitFailed, invocation, nil, finding.Params{
		"reason":    finding.String(writeReason(hostfs.ReasonOf(err))),
		"committed": stringList(ce.Committed),
	}), false
}

// prepare writes every file of plan through o, in path order, with every
// directory it needs. A failure aborts o, which removes what it wrote.
func prepare(o *hostfs.Output, plan []*planned) (finding.Finding, bool) {
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
