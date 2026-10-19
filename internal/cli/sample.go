// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"errors"
	"math"
	pathpkg "path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/kottera/metais/internal/cli/args"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/inputs"
	"github.com/kottera/metais/internal/inputs/csv"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/regexprofile"
	"github.com/kottera/metais/internal/sample"
)

// The generator's flags, which sample and render --sample take (ADR-008 D8
// item 8, D9; ADR-007 D9 item 1).
var (
	sampleFlag = args.Flag{Name: "sample",
		Description: "Render from sample data the generator makes from the inputs' schemas, in the data directory's place; values given with --set, --set-json and --inputs-json are kept as given, and the rest is made around them."}
	optionalFlag = args.Flag{Name: "optional", Value: "<mode>", Values: []string{"all", "none", "mixed"}, Default: "all",
		Description: "Which optional properties and inputs the generator makes: all of them, none (and null wherever null is allowed), or a mix the seed decides."}
	seedFlag = args.Flag{Name: "seed", Value: "<n>", Default: "0",
		Description: "The generator's seed, an unsigned 64-bit integer: the same package, options, supplied values and seed give the same data."}
	itemsFlag = args.Flag{Name: "items", Value: "<n>", Default: strconv.Itoa(sample.DefaultItems),
		Description: "The number of items the generator aims for in every array and collection, within the schema's bounds."}
)

// generatorFlags are the flags that only a run of the generator takes.
var generatorFlags = []args.Flag{optionalFlag, seedFlag, itemsFlag}

// The steps sample's refusals point to. No path the caller gave is echoed
// (ADR-008 D14 item 2).
var (
	nextSampleTo = Next{Run: "metais sample <package> --to <dir>",
		Why: "Writes the sample data as a data directory into a directory that is empty or does not exist yet."}
	nextSampleOverwrite = Next{Run: "metais sample <package> --to <dir> --overwrite",
		Why: "Writes into the directory anyway, replacing the files sample writes and leaving every other file as it is."}
	nextSampleExample = Next{Run: "metais sample --example <name>",
		Why: "Saves the sample data as a new example of the source directory, under a name no example has yet."}
	nextSampleUnpack = Next{Run: "metais unpack <package> --to <dir>",
		Why: "Writes the packed package's source directory, which sample --example can save an example into."}
	nextSampleInputs = Next{Run: "metais inspect <package>",
		Why: "Lists the inputs the package takes; a value given with --set, --set-json or --inputs-json is kept as given and must be one the schema allows."}
	nextSampleSchema = Next{Run: "metais schema <package> <name>",
		Why: "Prints the schema of the input named in the finding; add an examples value to the field named, and the generator uses it."}
)

// sampleDeclaration is metais sample (ADR-008 D9, ADR-007 D9).
func sampleDeclaration() *Command {
	refusals := packageRefusals("sample")
	refusals = append(refusals,
		Refusal{Code: finding.CodeUsageBadValue, Cause: "--seed or --items is not a whole number in range, or --example is not a name an example can have.", Next: nextCommandHelp("sample"),
			match: valueRefused("--seed", "--items", "--example")},
		Refusal{Code: finding.CodeUsageConflictingFlags, Cause: "--inputs-json - is given twice, and standard input is read once.", Next: nextInputsHelp},
		Refusal{Code: finding.CodeUsageFlagExcluded, Cause: "--to and --example are both given, or --overwrite is given with --example.", Next: nextCommandHelp("sample")},
		Refusal{Code: finding.CodeUsageNotSourceDir, Cause: "--example is given with a packed package, whose examples are not edited.", Next: nextSampleUnpack},
		Refusal{Code: finding.CodeDestinationNotEmpty, Cause: "--to names a directory that holds entries, and --overwrite is not given.", Next: nextSampleOverwrite},
		Refusal{Code: finding.CodeDestinationNotDirectory, Cause: "--to, or a directory above it, exists and is not a directory.", Next: nextSampleTo},
		Refusal{Code: finding.CodeDestinationWriteFailed, Cause: "A file sample writes is in the way (a link, a directory or a special file), the example already exists, or a write failed.", Next: nextSampleTo},
	)
	return &Command{
		Name: "sample", Group: "Authoring",
		Synopsis:    "sample [<package>] (--to <dir> | --example <name>) [--optional <mode>] [--seed <n>] [--items <n>] [supply flags] [--overwrite] [--dry-run] [--json]",
		Description: "Generate sample data from the inputs' schemas, and keep it as a data directory or an example.",
		Detail: "The generator makes a value for every input from its schema: the schema's const, examples, " +
			"default or enum where it has one, and otherwise placeholder text, numbers and dates of its own. " +
			"Values given with --set, --set-json and --inputs-json are fixed points: they are kept as given, " +
			"conditionals evaluated against them, and only what they leave out is made. The same package, " +
			"options, supplied values and seed give the same data on every platform.\n\n" +
			"The data is written as a data directory, one file per input in its first form and one directory " +
			"per collection, and is read back and validated exactly as metais render --data reads it, before " +
			"anything is written. A field the generator cannot make a valid value for is " +
			"schema.sample_unsatisfiable, and data that fails its schema, such as a supplied value the schema " +
			"refuses, is schema.sample_invalid; with any error nothing is written.\n\n" +
			"With --to, the data directory is <dir>: a <dir> that does not exist is created, with its missing " +
			"parents, beside it and renamed into place; an empty one is written into. A <dir> that holds any " +
			"entry, a hidden one included, is refused unless --overwrite is given, and then each file sample " +
			"writes replaces the file of that name, and nothing else in <dir> is read, changed or removed. A " +
			"link, a directory or a special file where sample writes a file is refused.\n\n" +
			"With --example, the source directory's examples/<name>/ is written: its inputs/ holds the data, " +
			"and its example.yaml records how it was made. It is refused with a packed package, and when " +
			"examples/<name>/ exists at all; nothing in an example is ever replaced. --dry-run lists the files " +
			"and writes nothing.",
		Role: finding.RoleAuthoring,
		Operands: []args.Operand{{Name: "<package>",
			Description: "A source directory, or a packed package file with --to; the current directory when omitted. - is refused, since a package is read with random access."}},
		Flags: append([]args.Flag{
			{Name: "to", Value: "<dir>", Description: "The directory to write the data directory into: one that does not exist yet, or an empty one, or with --overwrite any directory."},
			{Name: "example", Value: "<name>", Description: "Save the data as the inputs of a new example, examples/<name>/, of the source directory."},
		}, slices.Concat(generatorFlags, []args.Flag{setFlag, setJSONFlag, inputsJSONFlag},
			[]args.Flag{
				{Name: "overwrite", Description: "With --to, write into a directory that holds entries, replacing the files sample writes and nothing else."},
				{Name: "dry-run", Description: "Generate and validate the data, list the files sample would write, and write nothing."},
				limitFlag, failOnFlag,
			})...),
		Format:   "metais.sample/1",
		Prints:   "The files written, relative to <dir> or to examples/<name>/, in path order.",
		Refusals: refusals,
		Examples: []Example{
			{"metais sample --to data", "Writes sample data for the source directory in the working directory into the new directory data."},
			{"metais sample report-1.0.0.metais --to data --seed 7 --optional mixed", "Writes other data, with some optional fields left out."},
			{"metais sample --example staged --set /incident/stage=final", "Saves data whose stage is final as the example staged."},
			{"metais sample --to data --overwrite --json", "Replaces the data files in data, and lists them as one metais.sample/1 document."},
		},
		run: runSample,
	}
}

// sampleResult is metais.sample/1's result (ADR-008 D9 sample item 5).
type sampleResult struct {
	Optional string   `json:"optional"`
	Seed     uint64   `json:"seed"`
	Items    int      `json:"items"`
	Example  *string  `json:"example"`
	DryRun   bool     `json:"dry_run"`
	Files    []string `json:"files"`
}

// sampleOptions reads --optional, --seed and --items from p, each at its
// default when absent (ADR-008 D8 item 8, D9 sample item 4).
func (s *session) sampleOptions(p args.Parsed) (sample.Options, []finding.Finding) {
	opts := sample.Options{Optional: sample.All, Items: sample.DefaultItems}
	if v, ok := p.Value(optionalFlag.Name); ok {
		opts.Optional, _ = sample.ParseMode(v)
	}
	if v, ok := p.Value(seedFlag.Name); ok {
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil || !isDigits(v) {
			return opts, []finding.Finding{s.badValue(p, seedFlag.Name)}
		}
		opts.Seed = n
	}
	if v, ok := p.Value(itemsFlag.Name); ok {
		n, err := strconv.ParseUint(v, 10, 63)
		if err != nil || !isDigits(v) || n > math.MaxInt32 {
			return opts, []finding.Finding{s.badValue(p, itemsFlag.Name)}
		}
		opts.Items = int(n)
	}
	return opts, nil
}

// isDigits reports whether v is one or more ASCII digits, the only way a
// count is written: no sign, no space.
func isDigits(v string) bool {
	return v != "" && strings.Trim(v, "0123456789") == ""
}

// badValue is usage.bad_value for the value of the flag name in p, at the
// value's own position: the flag's, written --name=value, or the one after
// it.
func (s *session) badValue(p args.Parsed, name string) finding.Finding {
	i, _ := p.Arg(name)
	at := s.line.at[i] + 1
	if !strings.Contains(s.inv.Args[s.line.at[i]], "=") {
		at++
	}
	return finding.New(finding.CodeUsageBadValue, invocation, nil, finding.Params{
		"argument": finding.Int(int64(at)),
		"flag":     finding.String("--" + name),
		"reason":   finding.String("value_not_allowed"),
	})
}

// excluded is usage.flag_excluded for the flag name in p, given with
// other.
func (s *session) excluded(p args.Parsed, name, other string) finding.Finding {
	i, _ := p.Arg(name)
	return finding.New(finding.CodeUsageFlagExcluded, invocation, nil, finding.Params{
		"argument": finding.Int(int64(s.line.at[i] + 1)),
		"command":  finding.String(s.line.cmd.Name),
		"flag":     finding.String("--" + name),
		"other":    finding.String("--" + other),
	})
}

// requires is usage.flag_requires for the flag name in p, given without
// needs.
func (s *session) requires(p args.Parsed, name, needs string) finding.Finding {
	i, _ := p.Arg(name)
	return finding.New(finding.CodeUsageFlagRequires, invocation, nil, finding.Params{
		"argument": finding.Int(int64(s.line.at[i] + 1)),
		"command":  finding.String(s.line.cmd.Name),
		"flag":     finding.String("--" + name),
		"requires": finding.String("--" + needs),
	})
}

// sampled is what one run of the generator made of a package's inputs:
// the files of its data directory, each input's in manifest order, and
// what reading them back and validating them gave.
type sampled struct {
	files []inputs.File
	res   *inputs.Result
}

// generate is ADR-007 D9's generator over pkg under lim with opts: the
// supply flags of p are assembled, as ADR-003 D18 assembles them, into the
// fixed points (item 2); every input is generated around its fixed point
// and written in its first form (item 21); and the files are read back as
// a data directory, in memory, and validated like supplied data (item
// 22). It returns the data, or none and the findings that stop it: a
// supply argument refused, a field the generator cannot satisfy, or
// validation that fails, whose errors about an input are
// schema.sample_invalid.
func (s *session) generate(p args.Parsed, pkg *loader.Package, lim limits.Set, opts sample.Options) (*sampled, []finding.Finding) {
	layer := inputsLayer(pkg)
	sup := layer.NewSupply(lim)
	s.supplyArgs(p, sup, lim)
	fixed, fs := sup.Fixed()
	if hasError(fs) {
		return nil, fs
	}
	g := sample.New(pkg.Schemas, pkg.SchemaPositions(lim), lim, opts)
	var files []inputs.File
	for _, in := range pkg.Manifest.Inputs {
		si := sampleInput(in, fixed[in.Name])
		v, present, gfs, err := g.Generate(si)
		var le *regexprofile.LimitError
		switch {
		case errors.As(err, &le):
			// The command's matching budget is spent, so no value of this
			// input can be checked against its patterns.
			at := finding.Location{Origin: finding.OriginSample}.WithPointer("")
			fs = append(fs, lim.Exceeded(limits.RegexWork, finding.Subject{Kind: finding.SubjectInput, Name: in.Name}, at, 0, false))
			continue
		case err != nil:
			panic("cli: " + err.Error())
		}
		fs = append(fs, gfs...)
		if !present || len(gfs) > 0 {
			continue
		}
		written, err := g.Files(si, v)
		var ue *sample.UnwritableError
		switch {
		case errors.As(err, &ue):
			at := finding.Location{Origin: finding.OriginSample}.WithPointer(ue.Pointer)
			fs = append(fs, finding.New(finding.CodeSampleInvalid, finding.Subject{Kind: finding.SubjectInput, Name: in.Name}, &at,
				finding.Params{"code": finding.String("form")}))
			continue
		case err != nil:
			panic("cli: " + err.Error())
		}
		for _, f := range written {
			files = append(files, inputs.File{Path: f.Path, Data: f.Data})
		}
	}
	if hasError(fs) {
		return nil, fs
	}
	check := layer.NewSupply(lim)
	check.ReadFiles(files)
	res, err := check.Validate()
	if err != nil {
		panic("cli: " + err.Error())
	}
	for i, f := range res.Findings {
		if f.Severity() == finding.SeverityError && f.Subject().Kind == finding.SubjectInput {
			res.Findings[i] = sampleInvalid(f)
		}
	}
	return &sampled{files: files, res: res}, append(fs, res.Findings...)
}

// sampleInvalid is f, an error validating generated data, reported as
// schema.sample_invalid (ADR-007 D9 item 22): about the same input, the
// same field and the same schema keyword, located in the generated file
// at the same place, with f's code as its parameter.
func sampleInvalid(f finding.Finding) finding.Finding {
	var loc *finding.Location
	if l, ok := f.Location(); ok {
		l.Origin = finding.OriginSample
		loc = &l
	} else {
		l := finding.Location{Origin: finding.OriginSample}
		loc = &l
	}
	var opts []finding.Option
	if sc, ok := f.Schema(); ok {
		opts = append(opts, finding.WithSchema(sc))
	}
	if fd, ok := f.Field(); ok {
		opts = append(opts, finding.WithField(fd))
	}
	if v := f.Via(); len(v) > 0 {
		opts = append(opts, finding.WithVia(v...))
	}
	return finding.New(finding.CodeSampleInvalid, f.Subject(), loc, finding.Params{"code": finding.String(string(f.Code()))}, opts...)
}

// hasError reports whether any of fs is an error.
func hasError(fs []finding.Finding) bool {
	return slices.ContainsFunc(fs, func(f finding.Finding) bool { return f.Severity() == finding.SeverityError })
}

// sampleInput is the generator's view of the declared input in, with
// fixed, the value the caller supplied for it, as a fixed point.
func sampleInput(in manifest.Input, fixed value.Value) sample.Input {
	si := sample.Input{Name: in.Name, Schema: in.Schema, Collection: in.Collection, Required: in.Required, Supplied: fixed}
	for _, f := range in.Forms {
		si.Forms = append(si.Forms, sample.Form(f))
	}
	if d, ok := csv.ParseDelimiter(in.Delimiter); ok {
		si.Delimiter = d
	}
	return si
}

// runSample is metais sample: it opens the package, generates and
// validates the data, and writes it as a data directory or an example.
func runSample(s *session, p args.Parsed) outcome {
	operand := "."
	if len(p.Operands) > 0 {
		operand = p.Operands[0]
	}
	if operand == "-" {
		return s.refuseWith(finding.New(finding.CodeUsagePackageFromStdin, invocation, nil,
			finding.Params{"argument": finding.Int(int64(s.line.at[p.OperandArgs[0]] + 1))}))
	}
	to, hasTo := p.Value("to")
	name, hasExample := p.Value("example")
	switch {
	case hasTo && hasExample:
		return s.refuseWith(s.excluded(p, "example", "to"))
	case !hasTo && !hasExample:
		return s.refuseWith(finding.New(finding.CodeUsageMissingOperand, invocation, nil, finding.Params{
			"command": finding.String("sample"),
			"operand": finding.String("(--to <dir> | --example <name>)"),
		}))
	case hasExample && p.Has("overwrite"):
		return s.refuseWith(s.excluded(p, "overwrite", "example"))
	case hasTo && to == "":
		// An empty name names no directory; it is never read as the
		// working directory.
		return s.refuseWith(s.badValue(p, "to"))
	case hasExample && !manifest.ValidInputName(name):
		return s.refuseWith(s.badValue(p, "example"))
	}
	opts, fs := s.sampleOptions(p)
	if len(fs) > 0 {
		return s.refuseWith(fs[0])
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
	if hasExample && pkg.Form != loader.Directory {
		// The open runs first, so a hostile package is refused as every
		// opening command refuses it (ADR-011 D16 item 3).
		at := 0
		if len(p.OperandArgs) > 0 {
			at = s.line.at[p.OperandArgs[0]] + 1
		}
		return s.refuseWith(finding.New(finding.CodeUsageNotSourceDir, invocation, nil,
			finding.Params{"argument": finding.Int(int64(at))}))
	}
	data, fs := s.generate(p, pkg, lim, opts)
	if data == nil || s.fails(fs) {
		return outcome{findings: fs, next: s.nextSteps(fs)}
	}
	res := &sampleResult{Optional: opts.Optional.String(), Seed: opts.Seed, Items: opts.Items, DryRun: p.Has("dry-run")}
	files := data.files
	dest := to
	if hasExample {
		res.Example = &name
		dest = filepath.Join(operand, "examples", name)
		files = exampleFiles(name, opts, files)
	}
	for _, f := range files {
		if !strings.HasSuffix(f.Path, "/") {
			res.Files = append(res.Files, f.Path)
		}
	}
	slices.Sort(res.Files)
	if res.Files == nil {
		res.Files = []string{}
	}
	// A dry run makes every check the write makes, and writes nothing, so
	// it is refused exactly as the write would be.
	var f finding.Finding
	var ok bool
	if hasExample {
		f, ok = s.writeExample(dest, files, res.DryRun)
	} else {
		f, ok = s.writeData(dest, files, p.Has("overwrite"), res.DryRun)
	}
	if !ok {
		fs = append(fs, f)
		next := s.nextSteps(fs)
		if hasExample {
			next = []Next{nextSampleExample}
		}
		return outcome{findings: fs, next: next}
	}
	heading := "Wrote " + strconv.Itoa(len(res.Files)) + " files:"
	if res.DryRun {
		heading = "Would write " + strconv.Itoa(len(res.Files)) + " files:"
	}
	next := []Next{{Run: "metais render <package> --data <dir> --out <dir>", Why: "Renders the package's outputs from the data just written."}}
	if hasExample {
		next = []Next{{Run: "metais test <package> --example <name>", Why: "Renders the example; with --update it records the outputs as the example's expected ones."}}
	}
	return outcome{findings: fs, result: res, human: listing(heading, res.Files, false)[1:], next: declaredSteps(next, s.table)}
}

// exampleFiles are the files of the example name: the data under inputs/,
// and its example.yaml recording how the data was made (ADR-007 D9 item
// 21).
func exampleFiles(name string, opts sample.Options, data []inputs.File) []inputs.File {
	out := []inputs.File{{Path: "example.yaml", Data: exampleYAML(name, opts)}, {Path: "inputs/"}}
	for _, f := range data {
		out = append(out, inputs.File{Path: "inputs/" + f.Path, Data: f.Data})
	}
	return out
}

// exampleYAML is the example.yaml of an example sample saves: its title
// and a description of how the data was made and what to do with it.
func exampleYAML(name string, opts sample.Options) []byte {
	var b bytes.Buffer
	b.WriteString("title: Sample data\n")
	b.WriteString("description: \"Made by metais sample with --seed " + strconv.FormatUint(opts.Seed, 10) +
		", --optional " + opts.Optional.String() + " and --items " + strconv.Itoa(opts.Items) +
		". Replace the placeholder values in inputs/ with real ones, then run metais test --example " + name +
		" --update to record the outputs they render.\"\n")
	return b.Bytes()
}

// writeData writes files as the data directory to, the caller's own name
// for it: a directory that does not exist is staged beside it and renamed
// into place (ADR-005 D14 item 1); an existing one is written into under
// ADR-005 D14 item 2 as Amendment A1 amends it, when it holds no entry or
// overwrite is set, every file it does not plan left alone. A directory
// entry of files, a collection's, is created when it has no file to
// create it. A dry run makes every check and writes nothing.
func (s *session) writeData(to string, files []inputs.File, overwrite, dry bool) (finding.Finding, bool) {
	opt := hostfs.WithFaults(s.inv.Options.Faults)
	w, rest, err := hostfs.OpenNearest(to)
	if err != nil {
		return destinationRefusal(err), false
	}
	if rest != "." {
		w.Close()
		if dry {
			return finding.Finding{}, true
		}
		o, err := hostfs.StageNew(to, opt)
		if err != nil {
			return destinationRefusal(err), false
		}
		for _, f := range files {
			if err := writeFile(o, f); err != nil {
				o.Abort()
				return writeFailed(hostfs.ReasonOf(err)), false
			}
		}
		if err := o.Commit(); err != nil {
			return writeFailed(hostfs.ReasonOf(err)), false
		}
		return finding.Finding{}, true
	}
	entries, err := w.List(".", maxListed+1)
	w.Close()
	if err != nil && hostfs.ReasonOf(err) != hostfs.TooLarge {
		return writeFailed(hostfs.ReasonOf(err)), false
	}
	if len(entries) > 0 && !overwrite {
		var names []string
		for _, e := range entries[:min(len(entries), maxListed)] {
			names = append(names, e.Name)
		}
		slices.Sort(names)
		return finding.New(finding.CodeDestinationNotEmpty, invocation, nil, finding.Params{
			"entries": stringList(names),
			"more":    finding.Bool(len(entries) > maxListed),
		}), false
	}
	x, err := hostfs.OpenExisting(to, opt)
	if err != nil {
		return destinationRefusal(err), false
	}
	defer x.Close()
	var paths, dirs []string
	for _, f := range files {
		if d, ok := strings.CutSuffix(f.Path, "/"); ok {
			dirs = append(dirs, d)
		} else {
			paths = append(paths, f.Path)
		}
	}
	checks := x.Check(paths)
	for _, c := range checks {
		if c.Err != nil {
			return checkRefusal(c), false
		}
	}
	if dry {
		return finding.Finding{}, true
	}
	o, err := x.Stage(checks)
	if err != nil {
		return destinationRefusal(err), false
	}
	for _, f := range files {
		if strings.HasSuffix(f.Path, "/") {
			continue
		}
		if err := o.Create(f.Path, false, bytes.NewReader(f.Data)); err != nil {
			o.Abort()
			return writeFailed(hostfs.ReasonOf(err)), false
		}
	}
	if err := o.Commit(); err != nil {
		var ce *hostfs.CommitError
		if errors.As(err, &ce) {
			return finding.New(finding.CodeDestinationWriteFailed, invocation, nil, finding.Params{
				"reason":  finding.String(writeReason(hostfs.ReasonOf(ce.Err))),
				"written": stringList(ce.Committed),
			}), false
		}
		return writeFailed(hostfs.ReasonOf(err)), false
	}
	return s.emptyDirs(to, dirs, paths)
}

// emptyDirs creates each directory of dirs that no path of paths lies
// beneath, inside the existing directory to: a collection with no item is
// an empty directory (ADR-003 D15). Only a missing directory is created;
// nothing is replaced, and a link in the way is refused.
func (s *session) emptyDirs(to string, dirs, paths []string) (finding.Finding, bool) {
	var empty []string
	for _, d := range dirs {
		if !slices.ContainsFunc(paths, func(p string) bool { return strings.HasPrefix(p, d+"/") }) {
			empty = append(empty, d)
		}
	}
	if len(empty) == 0 {
		return finding.Finding{}, true
	}
	w, err := hostfs.OpenWriter(to)
	if err != nil {
		return writeFailed(hostfs.ReasonOf(err)), false
	}
	defer w.Close()
	for _, d := range empty {
		if err := w.MkdirAll(d); err != nil {
			return finding.New(finding.CodeDestinationWriteFailed, invocation, nil, finding.Params{
				"reason":  finding.String(writeReason(hostfs.ReasonOf(err))),
				"written": stringList(paths),
			}), false
		}
	}
	return finding.Finding{}, true
}

// writeExample writes files as the new example directory dest, which must
// not exist: staged beside it and renamed into place with a rename that
// never replaces (ADR-007 D9 item 21), so an example is never merged into
// or replaced. A dry run makes the check and writes nothing.
func (s *session) writeExample(dest string, files []inputs.File, dry bool) (finding.Finding, bool) {
	w, rest, err := hostfs.OpenNearest(dest)
	if err != nil {
		return destinationRefusal(err), false
	}
	w.Close()
	if rest == "." {
		return writeFailed(hostfs.Exists), false
	}
	if dry {
		return finding.Finding{}, true
	}
	o, err := hostfs.StageNew(dest, hostfs.WithFaults(s.inv.Options.Faults))
	if err != nil {
		return destinationRefusal(err), false
	}
	for _, f := range files {
		if err := writeFile(o, f); err != nil {
			o.Abort()
			return writeFailed(hostfs.ReasonOf(err)), false
		}
	}
	if err := o.Commit(); err != nil {
		return writeFailed(hostfs.ReasonOf(err)), false
	}
	return finding.Finding{}, true
}

// writeFile writes f beneath a new directory o: a directory, or a file
// with its missing parents.
func writeFile(o *hostfs.Output, f inputs.File) error {
	if d, ok := strings.CutSuffix(f.Path, "/"); ok {
		return o.MkdirAll(d)
	}
	if err := o.MkdirAll(pathpkg.Dir(f.Path)); err != nil {
		return err
	}
	return o.Create(f.Path, false, bytes.NewReader(f.Data))
}

// destinationRefusal is the finding for err, opening or staging a
// destination: a file in the way is not_directory, anything else a write
// that failed with nothing written.
func destinationRefusal(err error) finding.Finding {
	if hostfs.ReasonOf(err) == hostfs.NotDirectory {
		return finding.New(finding.CodeDestinationNotDirectory, invocation, nil, nil)
	}
	return writeFailed(hostfs.ReasonOf(err))
}

// checkRefusal is the finding for c, a planned path an existing
// destination refuses (ADR-005 D15 item 4): a write that cannot be made,
// with nothing written.
func checkRefusal(c hostfs.PathCheck) finding.Finding {
	return writeFailed(hostfs.ReasonOf(c.Err))
}
