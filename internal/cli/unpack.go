// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"strconv"

	"github.com/kottera/metais/internal/author"
	"github.com/kottera/metais/internal/cli/args"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/render/sink"
)

// unpackResult is metais.unpack/1's result: the files unpack wrote, or
// with --dry-run would write, relative to the directory, in the order
// written (ADR-008 D9).
type unpackResult struct {
	DryRun bool     `json:"dry_run"`
	Files  []string `json:"files"`
}

// The steps unpack's refusals point to. No path the caller gave is
// echoed: the directory is <dir> (ADR-008 D14 item 2).
var (
	nextUnpackEmpty = Next{Run: "metais unpack <package> --out <dir>",
		Why: "Unpacks into a directory that is empty or does not exist yet; nothing in a directory is ever overwritten."}
	nextUnpackAgain = Next{Run: "metais unpack <package> --out <dir>",
		Why: "Unpacks again, into an empty directory, once the cause is fixed."}
	nextUnpackPacked = Next{Run: "metais inspect <package>",
		Why: "Reads the source directory as it is: it is already the editable tree unpack writes."}
	nextUnpackWrite = Next{Run: "metais unpack <package> --out <dir>",
		Why: "Writes the files listed; the dry run wrote nothing."}
)

// unpackDeclaration is metais unpack (ADR-007 D8, ADR-008 D9).
func unpackDeclaration() *Command {
	refusals := packageRefusals("unpack")
	for i := range refusals {
		if refusals[i].Next == nextInspectAgain {
			refusals[i].Next = Next{Run: "metais unpack <package> --out <dir>", Why: "Unpacks the package once it is fixed or replaced; nothing of a refused package is written."}
		}
	}
	refusals = append(refusals,
		Refusal{Code: finding.CodeUsageNotPackedPackage, Cause: "The package operand is a source directory, which is already what unpack writes.", Next: nextUnpackPacked},
		Refusal{Code: finding.CodeDestinationNotEmpty, Cause: "<dir> holds an entry other than the quiet ones.", Next: nextUnpackEmpty},
		Refusal{Code: finding.CodeDestinationNotDirectory, Cause: "<dir>, or a directory above it, exists and is not a directory.", Next: nextUnpackEmpty},
		Refusal{Code: finding.CodeDestinationWriteFailed, Cause: "A write into an existing directory failed after it had created something there; the files already written are listed and left in place.", Next: nextUnpackAgain},
		Refusal{Code: finding.CodeDestinationNothingWritten, Cause: "A write failed and nothing was left: into a new directory, at any point; into an existing one, before it created anything.", Next: nextUnpackAgain},
	)
	return &Command{
		Name: "unpack", Group: "Authoring",
		Synopsis:    "unpack <package> --out <dir> [--dry-run] [--limit <name>=<value>]... [--json]",
		Description: "Write a packed package's source directory, to edit it.",
		Detail: "Writes every entry of the packed package into <dir>, as the source directory it was " +
			"packed from: the manifest as its author wrote it, the inputs, templates, references, " +
			"assets and examples. .compiled/ is not written: it is derived from the rest, and " +
			"metais pack writes it again; nor is README.md, the specification's text, which pack " +
			"also writes. Files are written with mode 0644 and directories 0755, " +
			"less the umask, whatever the package records.\n\n" +
			"Before it writes anything, unpack verifies the whole package as inspect does, its " +
			"digests included, and refuses a package that fails, writing nothing. It does not run " +
			"check: a package with check errors is the one an author needs to unpack and fix. A " +
			"source directory is refused: it is already what unpack would write.\n\n" +
			"--out is required. A <dir> that does not exist is created, with its missing parents, " +
			"by writing beside it and renaming the whole directory into place, so a failed write " +
			"leaves nothing. An existing <dir> must be empty, apart from names beginning with . " +
			"(such as .git) other than .compiled, and files ending .metais; it is written in place, " +
			"and a failed write stops unpack, which lists the files it wrote and removes none of " +
			"them. There is no --overwrite: a file left from an earlier tree would join the " +
			"unpacked source and be packed with it.\n\n" +
			"--dry-run lists the files unpack would write, and writes nothing anywhere. It is " +
			"refused exactly when unpack would be refused before writing, with the same findings: " +
			"a package that fails to open, a source directory, a <dir> that is not empty or not a " +
			"directory, and a staging directory left beside a <dir> that does not exist. What only " +
			"writing finds out, such as a full disk, it cannot report.",
		Role:     finding.RoleAuthoring,
		Operands: []args.Operand{{Name: "<package>", Required: true, Description: "A packed package file; - is refused, since a package is read with random access."}},
		Flags: []args.Flag{
			{Name: "out", Value: "<dir>", Required: true, Description: "The directory to write the source directory into: one that does not exist yet, or an empty one."},
			{Name: "dry-run", Description: "List the files unpack would write, and write nothing."},
			limitFlag,
			failOnFlag,
		},
		Format:   "metais.unpack/1",
		Prints:   "The files written, or with --dry-run that would be, relative to <dir>, in the order written.",
		Refusals: refusals,
		Examples: []Example{
			{"metais unpack report-1.0.0.metais --out report", "Writes the package's source directory into the new directory report."},
			{"metais unpack report-1.0.0.metais --out . --json", "Writes it into the current directory, which must be empty, and lists the files as one metais.unpack/1 document."},
			{"metais unpack report-1.0.0.metais --out report --dry-run", "Lists the files it would write into report, and writes nothing."},
		},
		run: runUnpack,
	}
}

// runUnpack is metais unpack: it opens the package as every command that
// opens one does, so a package refused anywhere is refused here with the
// same findings and nothing written, then writes its source directory.
func runUnpack(s *session, p args.Parsed) outcome {
	operand := p.Operands[0]
	if operand == "-" {
		return s.refuseWith(finding.New(finding.CodeUsagePackageFromStdin, invocation, nil,
			finding.Params{"argument": finding.Int(int64(s.line.at[p.OperandArgs[0]] + 1))}))
	}
	to, _ := p.Value("out")
	if to == "" {
		// An empty name names no directory; it is never read as the
		// working directory.
		i, _ := p.Arg("out")
		return s.refuseWith(finding.New(finding.CodeUsageBadValue, invocation, nil, finding.Params{
			"argument": finding.Int(int64(s.line.at[i] + 1)),
			"flag":     finding.String("--out"),
			"reason":   finding.String("value_not_allowed"),
		}))
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
	if pkg.Form != loader.Packed {
		// The open runs first, so a hostile source directory is refused
		// as every opening command refuses it (ADR-011 D16 item 3).
		return s.refuseWith(finding.New(finding.CodeUsageNotPackedPackage, invocation, nil,
			finding.Params{"argument": finding.Int(int64(s.line.at[p.OperandArgs[0]] + 1))}))
	}
	dry := p.Has("dry-run")
	var files []string
	var err error
	if dry {
		files, err = author.UnpackDryRun(pkg.Tree, to)
	} else {
		files, err = author.Unpack(pkg.Tree, to, s.inv.Options.Faults)
	}
	if err != nil {
		return s.unpackFailed(err)
	}
	head := "Wrote " + strconv.Itoa(len(files)) + " files:"
	var next []Next
	switch {
	case dry:
		head = "Would write " + strconv.Itoa(len(files)) + " files; --dry-run wrote nothing:"
		next = append(next, nextUnpackWrite)
	case lookup(s.table, "check") != nil:
		next = append(next, Next{Run: "metais check <dir>", Why: "Checks the unpacked source directory, which you can now edit."})
	}
	b := listing(head, files, false)[1:]
	return outcome{result: &unpackResult{DryRun: dry, Files: files}, human: b, next: next}
}

// unpackFailed is the outcome of err from author.Unpack. The entries found
// in a directory that is not empty, and the files written before a failed
// write, are listed in the human report after the finding.
func (s *session) unpackFailed(err error) outcome {
	var ne *author.NotEmptyError
	var we *author.WriteError
	switch {
	case errors.As(err, &ne):
		o := s.refuseWith(finding.New(finding.CodeDestinationNotEmpty, invocation, nil, finding.Params{
			"entries": stringList(ne.Entries),
			"more":    finding.Bool(ne.More),
		}))
		o.human = listing("Found in the directory:", ne.Entries, ne.More)
		return o
	case errors.As(err, &we) && we.Op == "open" && we.Reason == hostfs.NotDirectory:
		return s.refuseWith(finding.New(finding.CodeDestinationNotDirectory, invocation, nil, nil))
	case errors.As(err, &we) && !we.Left && len(we.Written) == 0:
		return s.refuseWith(sink.NothingWritten(we.Reason))
	case errors.As(err, &we):
		o := s.refuseWith(finding.New(finding.CodeDestinationWriteFailed, invocation, nil, finding.Params{
			"reason":  finding.String(sink.WriteReason(we.Reason)),
			"written": stringList(we.Written),
		}))
		if len(we.Written) > 0 {
			o.human = listing("Written before the failure, and left in place:", we.Written, false)
		}
		return o
	}
	panic("cli: unpack failed with " + err.Error())
}
