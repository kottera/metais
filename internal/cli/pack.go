// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"errors"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/kottera/metais/internal/author"
	"github.com/kottera/metais/internal/cli/args"
	"github.com/kottera/metais/internal/digest"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/zipfmt"
)

// packResult is metais.pack/1's result (ADR-008 D9 pack item 4, ADR-007
// D7 item 6). File is the package file's name when pack chose it, - for
// standard output, and absent when the caller named the file, whose path
// is never echoed (ADR-008 D11 item 5).
type packResult struct {
	DryRun  bool   `json:"dry_run"`
	File    string `json:"file,omitempty"`
	Size    int64  `json:"size"`
	Digest  string `json:"digest"`
	Entries int    `json:"entries"`
}

// The steps pack's refusals and reports point to. No path the caller gave
// is echoed: the source directory is <dir> (ADR-008 D14 item 2).
var (
	nextPackCheck = Next{Run: "metais check <dir>",
		Why: "Lists the errors that stop pack; fix them, then pack again, which checks and tests the package first."}
	nextPackTest = Next{Run: "metais test <dir>",
		Why: "Renders every example and compares it; fix the package or the examples that fail, then pack again."}
	nextPackAgain = Next{Run: "metais pack <dir>",
		Why: "Packs the source directory again once the cause is fixed; nothing was written."}
	nextPackOut = Next{Run: "metais pack <dir> --out <path>",
		Why: "Writes the package to a file outside the layout trees: in the source directory itself, or anywhere outside it."}
	nextPackUnpack = Next{Run: "metais unpack <package> --to <dir>",
		Why: "Writes the packed package's source directory, which pack reads."}
	nextPackStream = Next{Run: "metais pack <dir> --out <path>",
		Why: "Writes the package to a file; or redirect standard output to a file or a pipe, since a package is not written to a terminal."}
	nextPackWrite = Next{Run: "metais pack <dir>",
		Why: "Writes the package the dry run described; the dry run wrote nothing."}
	nextPackInspect = Next{Run: "metais inspect <package>",
		Why: "Reads the packed package as a supplier first reads it."}
)

// packDeclaration is metais pack (ADR-007 D7, ADR-008 D9).
func packDeclaration() *Command {
	return &Command{
		Name: "pack", Group: "Authoring",
		Synopsis:    "pack [<dir>] [--out <file>|-] [--dry-run] [--limit <name>=<value>]... [--json]",
		Description: "Check and test a source directory, then write it as one package file.",
		Detail: "Runs metais check on the source directory, the current directory when omitted. Any error " +
			"stops pack: every finding is reported and nothing is written. With no error it runs metais test " +
			"over every example; an example that fails stops it the same way. Warnings are reported and do " +
			"not stop it. There is no way to skip a step: the bytes a supplier receives are the bytes whose " +
			"examples passed.\n\n" +
			"The package holds metais.yaml and every file of inputs/, templates/, references/, assets/ and " +
			"examples/, names beginning with . included, and .compiled/, computed afresh. A top-level name " +
			"beginning with ., and a top-level file ending .metais, is left out silently; any other top-level " +
			"name is left out and reported as package.layout.not_packed. Each file is packed byte for byte. " +
			"The package's bytes depend only on its entries' paths and contents: never on a file's time, " +
			"mode or owner, the order a directory lists its names in, the platform, the locale or the clock. " +
			"One build of metais packs a source directory to the same bytes every time. The package is " +
			"read back as every reader reads one before it is kept, and a limit only its bytes show, such " +
			"as entry-ratio, refuses it as limit.package. An entry whose bytes put the end of central " +
			"directory signature in the package's final 65,557 bytes refuses it as " +
			"package.container.second_end_record, and the next step names that entry.\n\n" +
			"--out names the package file. Without it, the file is <name>-<version>.metais, from the " +
			"manifest, in the working directory; an --out that names an existing directory writes that file " +
			"inside it. The file is written to a temporary file .metais-tmp-<h> beside it, created " +
			"exclusively, synced, and renamed over the file, so a reader never sees part of a package and a " +
			"package file already there is replaced: replacing its own previous package is pack's edit loop, " +
			"so there is no --overwrite. A file inside one of the layout trees, or the manifest itself, is " +
			"refused, and so is an --out that exists and is not a regular file or a directory. --out - writes " +
			"the package to standard output, and is refused when standard output is a terminal; with --json " +
			"the document then goes to standard error, after the package.\n\n" +
			"--dry-run checks, tests and builds the package as pack does, and checks where it would write " +
			"it, and writes nothing: it reports the package's size, digest and entries as pack would, and " +
			"is refused exactly when pack would be refused before writing. A packed package is refused as " +
			"the operand: unpack it first.",
		Role: finding.RoleAuthoring,
		Operands: []args.Operand{{Name: "<dir>",
			Description: "The source directory to pack; the current directory when omitted. A packed package file is refused."}},
		Flags: []args.Flag{
			{Name: "out", Value: "<file>", Description: "The package file to write, an existing directory to write <name>-<version>.metais in, or - for standard output. Default: <name>-<version>.metais in the working directory."},
			{Name: "dry-run", Description: "Check, test and build the package, and write nothing."},
			limitFlag,
			failOnFlag,
		},
		Format:   "metais.pack/1",
		Prints:   "The package file's name when pack chose it, its size, its package digest and its entry count; with --dry-run, the entries it would pack.",
		Refusals: packRefusals(),
		Examples: []Example{
			{"metais pack", "Checks, tests and packs the source directory in the working directory into <name>-<version>.metais there."},
			{"metais pack report --out dist", "Packs the source directory report into dist/<name>-<version>.metais, dist being an existing directory."},
			{"metais pack --out -", "Writes the package to standard output, which must be a file or a pipe, not a terminal."},
			{"metais pack --dry-run --json", "Checks, tests and builds the package, writes nothing, and prints its size and digest as one metais.pack/1 document."},
		},
		run: runPack,
	}
}

// packSecondEnd is the refusal of a second end record in the package pack
// built (R1-014), which packFailed gives the read-back's finding: the
// code's own row, written for every reader of a package, says to pack it
// again, which writes the same bytes. The entry is the one whose records
// hold the signature, which pack knows and a reader does not; the
// finding's members stay a reader's.
var packSecondEnd = Refusal{Code: zipfmt.CodeSecondEndRecord,
	Cause: "An entry of the package pack built puts the end of central directory signature in the package's final 65,557 bytes, so a reader refuses the package; nothing was written.",
	Hint:  "Change the entry the next step names, in the source directory, so that its bytes no longer hold 50 4B 05 06, the end of central directory signature; a nested zip can be shipped in another archive format.",
	Next: Next{Run: "metais pack <dir>",
		Why: "Packs the source directory again once its entry <entry> no longer holds the signature; packing it unchanged writes the same bytes."},
	// No other path reports the code as pack's own: the operand's open
	// keeps the code's own row.
	match: func(finding.Finding) bool { return false },
}

// packRefusals are the codes pack declares (ADR-008 D16 item 2): every
// code check and test report, each pointing at the step whose errors stop
// pack, and pack's own.
func packRefusals() []Refusal {
	var out []Refusal
	have := map[finding.Code]bool{}
	for _, r := range checkRefusals() {
		switch {
		case r.Code == finding.CodeLimitInvalid || r.Code == finding.CodeUsagePackageFromStdin:
			r.Next = nextCommandHelp("pack")
		case r.Code == finding.CodeLimitPackage:
			// check's breach, or one only the package's bytes show,
			// such as entry-ratio, which pack finds reading them back.
			r.Cause = "The package, or the package's bytes as pack writes them, breach one of its limits; nothing was written."
			r.Next = nextLimits
		case r.Next == nextCheckAgain:
			r.Next = nextPackCheck
		}
		have[r.Code] = true
		out = append(out, r)
	}
	for _, r := range testRefusals() {
		family, _, _ := strings.Cut(string(r.Code), ".")
		if have[r.Code] || family == "usage" || strings.HasPrefix(string(r.Code), "output.destination.") {
			continue
		}
		if r.Next == nextTestAgain {
			r.Next = nextPackTest
		}
		have[r.Code] = true
		out = append(out, r)
	}
	return append(out,
		packSecondEnd,
		Refusal{Code: finding.CodeUsageNotSourceDir, Cause: "The operand is a packed package, which pack does not read: it packs a source directory.", Next: nextPackUnpack},
		Refusal{Code: finding.CodeUsageArchiveToTerminal, Cause: "--out - is given, and standard output is a terminal.", Next: nextPackStream},
		Refusal{Code: finding.CodeUsageBadValue, Cause: "--out is empty, or names a file whose name metais will not write.", Next: nextCommandHelp("pack"),
			match: valueRefused("--out")},
		Refusal{Code: finding.CodeDestinationInPackage, Cause: "The package file would be inside one of the layout trees of the source directory, or be its manifest.", Next: nextPackOut},
		Refusal{Code: finding.CodeDestinationNotDirectory, Cause: "A directory above the package file exists and is not a directory.", Next: nextPackOut},
		Refusal{Code: finding.CodeDestinationNothingWritten, Cause: "The package file could not be written, and nothing was: its directory does not exist, what stands at its name is not a regular file, a temporary file .metais-tmp-<h> is left beside it, or a write failed.", Next: nextPackAgain},
	)
}

// runPack is metais pack (ADR-007 D7): it opens the operand as every
// command that opens a package does, so a package refused anywhere is
// refused here with the same findings and nothing written, refuses a
// packed one, then checks, tests and writes.
func runPack(s *session, p args.Parsed) outcome {
	operand, operandArg := ".", -1
	if len(p.Operands) > 0 {
		operand, operandArg = p.Operands[0], s.line.at[p.OperandArgs[0]]+1
		if operand == "-" {
			return s.refuseWith(finding.New(finding.CodeUsagePackageFromStdin, invocation, nil,
				finding.Params{"argument": finding.Int(int64(operandArg))}))
		}
	}
	out, hasOut := p.Value("out")
	dry := p.Has("dry-run")
	toStream := hasOut && out == "-"
	switch {
	case hasOut && out == "":
		// An empty name names no file; it is never read as the working
		// directory.
		return s.refuseWith(s.badOut(p))
	case toStream && !dry && isTerminal(s.inv.Stdout):
		return s.refuseWith(finding.New(finding.CodeUsageArchiveToTerminal, invocation, nil,
			finding.Params{"argument": s.flagValueArg(p, "out")}))
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
	if pkg.Form != loader.Directory {
		// The open runs first, so a hostile package is refused as every
		// opening command refuses it (ADR-011 D16 item 3).
		// The operand was given: "." is always a directory.
		return s.refuseWith(finding.New(finding.CodeUsageNotSourceDir, invocation, nil,
			finding.Params{"argument": finding.Int(int64(operandArg))}))
	}

	// Check, then test (ADR-007 D7 item 2).
	fs = author.Check(operand, lim, s.inv.Options.Faults)
	if refusal(fs) {
		return outcome{findings: fs, next: s.nextSteps(fs)}
	}
	if s.fails(fs) {
		return outcome{findings: fs, next: []Next{nextPackCheck}}
	}
	examples := exampleRuns(pkg)
	pfs, ran := s.runExamples(pkg, lim, examples)
	if !ran {
		return outcome{findings: append(fs, pfs...), next: []Next{nextPackCheck}}
	}
	failed := false
	for _, e := range examples {
		fs = append(fs, e.fs...)
		failed = failed || e.result.Status == "fail"
	}
	if failed || s.fails(fs) {
		return outcome{findings: fs, next: []Next{nextPackTest}}
	}

	// The deterministic write (items 3 to 5).
	plan, err := author.PlanPack(pkg.Tree, pkg.Compiled, pkg.Manifest.Metais, s.inv.Options.Faults)
	if err == nil && plan.Digest != pkg.Digest {
		// The source changed since it was opened, checked and tested.
		err = &author.ReadError{Reason: hostfs.Changed}
	}
	if err != nil {
		return s.packFailed(fs, err)
	}
	name := pkg.Manifest.Name + "-" + pkg.Manifest.Version + ".metais"
	res := &packResult{DryRun: dry, Digest: digest.Format(plan.Digest), Entries: len(plan.Entries)}
	o := outcome{findings: fs, result: res}
	// Every byte is read back as a reader reads a package before any of
	// it is kept: a limit only the bytes show refuses it (ADR-007 D2 step
	// 3, MTS-PKG-038).
	verify := func(r io.ReaderAt, size int64) error {
		vfs, err := author.VerifyPacked(r, size, lim)
		if err == nil && len(vfs) > 0 {
			up := &unreadablePackage{fs: vfs}
			for _, f := range vfs {
				v, _ := f.Param("offset")
				if at, ok := v.Int(); ok && f.Code() == zipfmt.CodeSecondEndRecord {
					up.entry, _ = plan.EntryAt(at)
				}
			}
			err = up
		}
		return err
	}
	inMemory := func() (*bytes.Buffer, error) {
		var buf bytes.Buffer
		err := plan.Write(&buf)
		if err == nil {
			err = verify(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		}
		return &buf, err
	}
	if toStream {
		buf, err := inMemory()
		if err != nil {
			return s.packFailed(fs, err)
		}
		res.File, res.Size = "-", int64(buf.Len())
		if !dry {
			o.stream = func(w io.Writer) { write(w, buf.Bytes()) }
		}
	} else {
		// Without --out the file is name in the working directory, which
		// is never read as a directory to write it in.
		target := "."
		if hasOut {
			target = out
		}
		fo, err := hostfs.OpenFileOut(target, name)
		if err != nil {
			if hostfs.ReasonOf(err) == hostfs.BadName {
				return s.refuseWith(s.badOut(p))
			}
			return s.packFailed(fs, err)
		}
		defer fo.Close()
		held, err := hostfs.Hold(operand, author.LayoutEntries, author.EntryDirs(pkg.Tree.List()))
		if err != nil {
			return s.packFailed(fs, &author.ReadError{Reason: hostfs.ReasonOf(err)})
		}
		entry, err := fo.Within(held)
		switch {
		case err != nil:
			return s.packFailed(fs, err)
		case entry != "":
			return s.packFailed(fs, &inPackageError{entry})
		}
		if fo.InDirectory() {
			// pack chose the name: no --out, or one that names an
			// existing directory.
			res.File = name
		}
		if dry {
			var buf *bytes.Buffer
			buf, err = inMemory()
			res.Size = int64(buf.Len())
		} else {
			res.Size, err = fo.Write(plan.Write, verify)
		}
		if err != nil {
			return s.packFailed(fs, err)
		}
	}
	o.human = packHuman(res, plan.Entries)
	switch {
	case dry:
		o.next = []Next{nextPackWrite}
	case !toStream && lookup(s.table, "inspect") != nil:
		o.next = []Next{nextPackInspect}
	}
	if toStream && !dry {
		// Standard output carries the package and nothing else (ADR-008
		// D11 item 1); what is said about it goes to standard error.
		o.notice, o.human = o.human, nil
	}
	return o
}

// badOut is usage.bad_value for --out's value.
func (s *session) badOut(p args.Parsed) finding.Finding {
	return finding.New(finding.CodeUsageBadValue, invocation, nil, finding.Params{
		"argument": s.flagValueArg(p, "out"),
		"flag":     finding.String("--out"),
		"reason":   finding.String("value_not_allowed"),
	})
}

// unreadablePackage is a package pack built that a reader refuses, with
// the reader's findings, and the entry that holds a second end record's
// signature, when one does.
type unreadablePackage struct {
	fs    []finding.Finding
	entry string
}

func (e *unreadablePackage) Error() string { return "cli: the package built does not read" }

// inPackageError is an output file inside the source directory's entry.
type inPackageError struct{ entry string }

func (e *inPackageError) Error() string { return "cli: the output is in " + e.entry }

// packFailed is pack's refusal of err, after check's and test's findings
// fs, none of them an error: a source entry that could not be read, or
// changed (package.not_readable); an output inside the package; or a
// write that failed, which wrote nothing (ADR-007 D7 item 5).
func (s *session) packFailed(fs []finding.Finding, err error) outcome {
	var re *author.ReadError
	var ip *inPackageError
	var up *unreadablePackage
	var f finding.Finding
	switch {
	case errors.As(err, &up):
		vfs, next := slices.Clone(up.fs), s.nextSteps(up.fs)
		if up.entry != "" {
			// R1-014's refusal stands; the hint and the next step are
			// the author's (ADR-008 D14).
			for i, f := range vfs {
				if f.Code() == packSecondEnd.Code {
					vfs[i] = f.WithHint(packSecondEnd.Hint)
				}
			}
			n := packSecondEnd.Next
			n.Why = strings.Replace(n.Why, "<entry>", shellQuote(up.entry), 1)
			next = []Next{n}
		}
		return outcome{findings: append(slices.Clone(fs), vfs...), next: next}
	case errors.As(err, &re):
		f = finding.New(loader.CodeNotReadable, finding.Subject{Kind: finding.SubjectPackage}, nil,
			finding.Params{"reason": finding.String(string(re.Reason))})
	case errors.As(err, &ip):
		f = finding.New(finding.CodeDestinationInPackage, invocation, nil, finding.Params{"entry": finding.String(ip.entry)})
	case hostfs.ReasonOf(err) == hostfs.NotDirectory:
		f = finding.New(finding.CodeDestinationNotDirectory, invocation, nil, nil)
	default:
		r := hostfs.ReasonOf(err)
		if r == hostfs.Link {
			// A link is never replaced or written through: what stands
			// there is not a regular file.
			r = hostfs.NotRegularFile
		}
		f = nothingWritten(r)
	}
	all := append(slices.Clone(fs), f)
	o := outcome{findings: all, next: s.nextSteps([]finding.Finding{f})}
	return o
}

// packHuman is pack's human result (ADR-007 D7 item 6): the file's name
// when pack chose it, its size, its package digest and its entry count;
// on a dry run, the entries it would pack.
func packHuman(r *packResult, entries []string) []byte {
	what := "the package"
	switch r.File {
	case "":
	case "-":
		what = "the package to standard output"
	default:
		what = r.File
	}
	facts := ": " + plural(int(r.Size), "byte", "bytes") + ", " + plural(r.Entries, "entry", "entries") + ", package digest " + r.Digest + "\n"
	if !r.DryRun {
		return []byte("Packed " + what + facts)
	}
	b := []byte("Would pack " + what + facts + "--dry-run wrote nothing.\n")
	return append(b, listing("Would pack "+strconv.Itoa(len(entries))+" entries:", entries, false)...)
}
