// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"strconv"
	"strings"

	"github.com/kottera/metais/internal/author"
	"github.com/kottera/metais/internal/cli/args"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/render"
)

// initResult is metais.init/1's result: the files init wrote, or with
// --dry-run would write, relative to the directory, in the order written
// (ADR-007 D1 item 5).
type initResult struct {
	DryRun bool     `json:"dry_run"`
	Files  []string `json:"files"`
}

// The steps init's refusals point to. No path the caller gave is echoed:
// the directory is <dir> (ADR-008 D14 item 2).
var (
	nextInitName = Next{Run: "metais init --name <name>",
		Why: "Gives the package's name, which init takes only from the command line, never from the working directory."}
	nextInitValidName = Next{Run: "metais init <dir> --name <name>",
		Why: "Gives a name of lower-case letters, digits, single hyphens and dots, which the package name rule accepts."}
	nextInitEmpty = Next{Run: "metais init <dir>",
		Why: "Writes the scaffold into a directory that is empty or does not exist yet."}
	nextInitAgain = Next{Run: "metais init <dir>",
		Why: "Writes the scaffold again, into an empty directory, once the cause is fixed."}
)

// runInit is metais init (ADR-007 D1, ADR-008 D9): it takes the name from
// --name or from the last element of <dir> as written, and writes the
// scaffold into <dir>.
func runInit(s *session, p args.Parsed) outcome {
	dir, dirGiven := "", len(p.Operands) > 0
	if dirGiven {
		dir = p.Operands[0]
	}
	name, named := p.Value("name")
	source := "name"
	if !named {
		var ok bool
		if name, ok = author.NameFromDir(dir); !ok {
			return s.refuseWith(finding.New(finding.CodeUsageNameRequired, invocation, nil, nil))
		}
		source = "directory"
	}
	if !author.ValidName(name) {
		return s.refuseInvalidName(p, name, source)
	}
	dryRun := p.Has("dry-run")
	files, err := author.Init(dir, name, dryRun)
	if err != nil {
		return s.initFailed(err)
	}
	var next []Next
	if lookup(s.table, "check") != nil {
		// ADR-008 D14 item 3: init is followed by check, once metais has
		// it.
		run := "metais check"
		if dirGiven {
			run += " <dir>"
		}
		next = append(next, Next{Run: run, Why: "Checks the package, which reports no finding until you edit it."})
	}
	return outcome{
		result: &initResult{DryRun: dryRun, Files: files},
		human:  initHuman(dryRun, files),
		next:   next,
	}
}

// refuseWith is the outcome of a refusal with f, and its declared next
// step.
func (s *session) refuseWith(f finding.Finding) outcome {
	fs := []finding.Finding{f}
	return outcome{findings: fs, next: s.nextSteps(fs)}
}

// refuseInvalidName refuses name, taken from source. A name taken from
// --name gets a suggestion, written into the next step as well; a name
// taken from the directory is part of a path the caller gave, and no part
// of one is printed (ADR-008 D11 item 5, R1-011).
func (s *session) refuseInvalidName(p args.Parsed, name, source string) outcome {
	var i int
	if source == "name" {
		i, _ = p.Arg("name")
	} else {
		i = p.OperandArgs[0]
	}
	params := finding.Params{
		"argument": finding.Int(int64(s.line.at[i] + 1)),
		"source":   finding.String(source),
	}
	suggestion, ok := "", false
	if source == "name" {
		suggestion, ok = author.SuggestName(name)
	}
	if ok {
		params["suggestion"] = finding.String(suggestion)
	}
	o := s.refuseWith(finding.New(finding.CodeUsageNameInvalid, invocation, nil, params))
	if ok {
		for i, n := range o.next {
			if n == nextInitValidName {
				o.next[i].Run = strings.Replace(n.Run, "<name>", suggestion, 1)
				o.next[i].Why = "Gives the package a name the rule accepts, made from the one refused."
			}
		}
	}
	return o
}

// initFailed is the outcome of err from author.Init. The entries found in
// a directory that is not empty, and the files written before a failed
// write, are listed in the human report after the finding.
func (s *session) initFailed(err error) outcome {
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
		return s.refuseWith(render.NothingWritten(we.Reason))
	case errors.As(err, &we):
		o := s.refuseWith(finding.New(finding.CodeDestinationWriteFailed, invocation, nil, finding.Params{
			"reason":  finding.String(render.WriteReason(we.Reason)),
			"written": stringList(we.Written),
		}))
		if len(we.Written) > 0 {
			o.human = listing("Written before the failure, and left in place:", we.Written, false)
		}
		return o
	}
	panic("cli: init failed with " + err.Error())
}

func stringList(ss []string) finding.Value {
	vs := make([]finding.Value, len(ss))
	for i, s := range ss {
		vs[i] = finding.String(s)
	}
	return finding.List(vs...)
}

// listing is a heading and names, one to a line, each written as human
// output writes a host's file name.
func listing(heading string, names []string, more bool) []byte {
	b := []byte("\n" + heading + "\n")
	for _, n := range names {
		b = append(b, "  "...)
		b = finding.AppendText(b, n)
		b = append(b, '\n')
	}
	if more {
		b = append(b, "  and more\n"...)
	}
	return b
}

// initHuman is init's human result: the files written, or that would be,
// and what to do next (ADR-007 D1 item 5).
func initHuman(dryRun bool, files []string) []byte {
	head := "Wrote " + strconv.Itoa(len(files)) + " files:"
	if dryRun {
		head = "Would write " + strconv.Itoa(len(files)) + " files; --dry-run wrote nothing:"
	}
	b := listing(head, files, false)[1:]
	return append(b, "\nEdit inputs/ and templates/ to make the package yours.\n"...)
}
