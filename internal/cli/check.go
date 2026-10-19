// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"slices"
	"strings"

	"github.com/kottera/metais/internal/author"
	"github.com/kottera/metais/internal/cli/args"
	"github.com/kottera/metais/internal/finding"
	templates "github.com/kottera/metais/internal/template"
)

var (
	nextCheckAgain = Next{Run: "metais check <package>", Why: "Checks the package again once the errors are fixed; every step runs again and may report more."}
	nextTest       = Next{Run: "metais test <package>", Why: "Renders every example and compares it with the outputs it should produce."}
)

// checkDeclaration is metais check (ADR-008 D9).
func checkDeclaration() *Command {
	return &Command{
		Name: "check", Group: "Authoring",
		Synopsis:    "check [<package>] [--limit <name>=<value>]... [--json]",
		Description: "Check a package without data, and report every finding of every step at once.",
		Detail: "The package is a source directory, the current directory when omitted, or a packed package " +
			"file. check reads nothing but the package and writes nothing. It runs every step in order: the " +
			"source tree, the entries, the limits of the package pack would write, the manifest and its " +
			"guide's links, every schema with its defaults, examples, consts, enums and required names, " +
			"every template, the cross-reference of templates against schemas, Office Open XML text runs, " +
			"the examples' layout, and the compiled form. Each step runs whatever an earlier one found; a " +
			"step whose input could not be made is named in one report.step_skipped finding.\n\n" +
			"A package that cannot be read at all — its container, its entry names, its digests or its " +
			"source tree — is refused with the findings every command gives it.\n\n" +
			"check exits 1 when any finding is an error; warnings do not fail it unless --fail-on warning is given.",
		Role: finding.RoleAuthoring,
		Operands: []args.Operand{{Name: "<package>",
			Description: "A source directory, or a packed package file; the current directory when omitted. - is refused, since a package is read with random access."}},
		Flags:             []args.Flag{limitFlag, failOnFlag},
		Format:            "metais.check/1",
		FindingsAreResult: true,
		Prints:            "Every finding, then a line counting the errors, warnings and notes, then the Next: block.",
		Refusals:          checkRefusals(),
		Examples: []Example{
			{"metais check", "Checks the source directory in the working directory."},
			{"metais check report-1.0.0.metais", "Checks a packed package."},
			{"metais check --json", "Prints every finding as one metais.check/1 document."},
		},
		run: runCheck,
	}
}

// checkRefusals are the codes check declares (ADR-008 D16 item 2): the
// operand's and the limits', every code a package's own read refuses it
// with, and every code its steps report, warnings and notes included,
// each pointing back at check.
func checkRefusals() []Refusal {
	var out []Refusal
	for _, r := range packageRefusals("check") {
		if r.Code == finding.CodeUsageMissingOperand {
			continue
		}
		r.Next = nextCheckAgain
		if r.Code == finding.CodeLimitInvalid || r.Code == finding.CodeUsagePackageFromStdin {
			r.Next = nextCommandHelp("check")
		}
		out = append(out, r)
	}
	for _, code := range templates.ParseCodes() {
		if !slices.ContainsFunc(out, func(r Refusal) bool { return r.Code == code }) {
			out = append(out, Refusal{Code: code, Cause: "A template does not parse (" + strings.ReplaceAll(strings.TrimPrefix(string(code), "template."), "_", " ") + "), an error.", Next: nextCheckAgain})
		}
	}
	what := map[string]string{
		"package":  "The package's tree",
		"manifest": "The manifest",
		"schema":   "A schema",
		"template": "A template or a manifest expression",
		"example":  "An example's layout",
		"report":   "The report",
	}
	for _, e := range finding.Registry() {
		family, rest, _ := strings.Cut(string(e.Code), ".")
		declared := slices.ContainsFunc(out, func(r Refusal) bool { return r.Code == e.Code })
		if declared || e.Code == "schema.sample_unsatisfiable" || e.Code == finding.CodeReportTruncated {
			continue
		}
		profile := slices.Contains(author.ContainerCodes(), e.Code)
		if !slices.Contains(author.CheckCodes(), e.Code) && !profile && (what[family] == "" || e.Severity == finding.SeverityError) {
			continue
		}
		subject := what[family]
		if profile {
			subject = "A container's profile"
		}
		cause := subject + " (" + strings.ReplaceAll(strings.ReplaceAll(rest, "_", " "), ".", ": ") + "), a " + e.Severity.String() + "."
		out = append(out, Refusal{Code: e.Code, Cause: cause, Next: nextCheckAgain})
	}
	return out
}

// checkResult is metais.check/1's result: the number of findings of each
// severity (ADR-008 D9).
type checkResult struct {
	Counts checkCounts `json:"counts"`
}

type checkCounts struct {
	Error   int `json:"error"`
	Warning int `json:"warning"`
	Info    int `json:"info"`
}

// runCheck is metais check.
func runCheck(s *session, p args.Parsed) outcome {
	operand := "."
	if len(p.Operands) > 0 {
		operand = p.Operands[0]
		if operand == "-" {
			f := finding.New(finding.CodeUsagePackageFromStdin, invocation, nil,
				finding.Params{"argument": finding.Int(int64(s.line.at[p.OperandArgs[0]] + 1))})
			return outcome{findings: []finding.Finding{f}, next: s.nextSteps([]finding.Finding{f})}
		}
	}
	lim, fs := s.limits(p)
	if len(fs) > 0 {
		return outcome{findings: fs, next: s.nextSteps(fs)}
	}
	fs = author.Check(operand, lim, s.inv.Options.Faults)
	if refusal(fs) {
		return outcome{findings: fs, next: s.nextSteps(fs)}
	}
	r := finding.NewReport(len(fs) + 1)
	for _, f := range fs {
		r.Add(f)
	}
	sum := r.Summary()
	res := &checkResult{Counts: checkCounts{Error: sum.Errors, Warning: sum.Warnings, Info: sum.Info}}
	human := "check: " + plural(sum.Errors, "error", "errors") + ", " + plural(sum.Warnings, "warning", "warnings") + ", " + plural(sum.Info, "note", "notes") + "\n"
	o := outcome{findings: fs, result: res, human: []byte(human)}
	if sum.Errors > 0 || s.failOn() == finding.SeverityWarning && sum.Warnings > 0 {
		o.next = []Next{nextCheckAgain}
	} else if lookup(s.table, "test") != nil {
		o.next = []Next{nextTest}
	}
	return o
}
