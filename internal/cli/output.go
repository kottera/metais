// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"slices"

	"github.com/kottera/metais/internal/cli/args"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostenv"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/version"
)

// outcome is what a command produced, before it is written.
type outcome struct {
	findings []finding.Finding
	// result is the JSON document's result, a pointer to one of the
	// result types; nil is null.
	result any
	// human is the human result, written to standard output.
	human []byte
	next  []Next
}

// envelope is the JSON document every run prints with --json (ADR-008
// D10), its members in their declared order: format, ok, findings,
// summary, result, next. summary is ADR-006 D10 item 4's, which every
// envelope carries beside findings.
type envelope struct {
	Format   string          `json:"format"`
	OK       bool            `json:"ok"`
	Findings json.RawMessage `json:"findings"`
	Summary  json.RawMessage `json:"summary"`
	Result   any             `json:"result"`
	Next     []Next          `json:"next"`
}

// emit writes o for the line s read and returns the exit status, which
// ADR-008 D13 maps from o's findings and the command's role.
func (s *session) emit(o outcome) int {
	r := finding.NewReport(int(limits.Get(limits.Findings).Default))
	for _, f := range o.findings {
		r.Add(f)
	}
	role := finding.RoleNoPackage
	if s.line.cmd != nil {
		role = s.line.cmd.Role
	}
	status := r.Exit(role, finding.SeverityError)
	fs, sum := r.Findings(), r.Summary()
	next := o.next
	if next == nil {
		next = []Next{}
	}
	if s.line.json {
		result := o.result
		if sum.Errors > 0 {
			result = nil
		}
		writeJSON(s.inv.Stdout, envelope{
			Format:   s.line.format(),
			OK:       status == finding.ExitOK && sum.Errors == 0,
			Findings: finding.AppendJSON(nil, fs...),
			Summary:  sum.AppendJSON(nil),
			Result:   result,
			Next:     next,
		})
		return status
	}
	// Human mode (ADR-008 D11 item 3): a refusal, its findings and its
	// next steps go to standard error and nothing to standard output;
	// otherwise the result goes to standard output, and any finding and
	// the Next: block to standard error.
	b := finding.AppendHuman(nil, s.colour(s.inv.Stderr), fs...)
	if sum.Errors > 0 {
		b = append(b, o.human...)
		write(s.inv.Stderr, appendNext(b, next))
		return status
	}
	if s.line.cmd != nil && s.line.cmd.NextToStdout {
		write(s.inv.Stdout, appendNext(o.human, next))
		write(s.inv.Stderr, b)
		return status
	}
	write(s.inv.Stdout, o.human)
	write(s.inv.Stderr, appendNext(b, next))
	return status
}

// refuse is the outcome of a line refused before any command ran: its
// findings, and the steps their declarations name. In human mode the
// command's usage line goes to standard error with them, or, when no
// command was named, the overview (ADR-008 D1 item 4, D15 item 6).
func (s *session) refuse(fs []finding.Finding) int {
	o := outcome{findings: fs, next: s.nextSteps(fs)}
	switch {
	case s.line.cmd != nil:
		o.human = []byte("\nUsage:\n  metais " + s.line.cmd.Synopsis + "\n")
	case slices.ContainsFunc(fs, func(f finding.Finding) bool { return f.Code() == finding.CodeUsageMissingCommand }):
		o.human = append([]byte("\n"), overviewText(s.table)...)
	}
	return s.emit(o)
}

// refusals is what the line's command declares it may refuse with, or
// the tool's own when the line names no command.
func (s *session) refusals() []Refusal {
	if s.line.cmd != nil {
		return s.line.cmd.Refusals
	}
	return toolRefusals
}

// nextSteps is the declared next step of each of fs's codes, each once,
// in the order of fs.
func (s *session) nextSteps(fs []finding.Finding) []Next {
	var out []Next
	for _, f := range fs {
		i := slices.IndexFunc(s.refusals(), func(r Refusal) bool { return r.Code == f.Code() })
		if i < 0 {
			panic("cli: " + string(f.Code()) + " is not among the refusals declared")
		}
		if n := s.refusals()[i].Next; !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	return out
}

// internalError reports a defect met in phase (ADR-006 D12 item 4): one
// internal.error finding, carrying the tool's version and the phase and
// nothing of the panic, and status 2.
func (s *session) internalError(phase string) int {
	f := finding.New(finding.CodeInternalError, invocation, nil, finding.Params{
		"tool_version": finding.String(version.String()),
		"phase":        finding.String(phase),
	})
	if !s.line.json {
		s.line.json = args.RequestsJSON(s.inv.Args)
	}
	return s.emit(outcome{findings: []finding.Finding{f}, next: []Next{
		{Run: "metais version", Why: "Prints the version to report with the command that met the defect."},
	}})
}

// writeJSON writes v as one compact JSON document ending in one LF, with
// HTML escaping off (ADR-008 D10 item 2).
func writeJSON(w io.Writer, v any) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		panic("cli: encoding a document: " + err.Error())
	}
	write(w, b.Bytes())
}

// write writes b to w. A failed write is not reported: the stream it
// would be reported on is the one that failed, and a closed standard
// output ends the process as Go programs end (ADR-006 D12 item 5).
func write(w io.Writer, b []byte) {
	if len(b) > 0 {
		_, _ = w.Write(b)
	}
}

// appendNext appends the human Next: block (ADR-008 D14 item 1).
func appendNext(b []byte, next []Next) []byte {
	if len(next) == 0 {
		return b
	}
	b = append(b, "\nNext:\n"...)
	for _, n := range next {
		b = append(b, "  "+n.Run+"  — "+n.Why+"\n"...)
	}
	return b
}

// isTerminal is hostenv.IsTerminal; a test replaces it to drive the
// colour policy's terminal branch, which a test's streams never reach.
var isTerminal = hostenv.IsTerminal

// colour reports whether human output to stream is coloured (ADR-008
// D12 item 1): always and never decide alone; auto colours a terminal
// when NO_COLOR is unset or empty.
func (s *session) colour(stream io.Writer) bool {
	switch s.line.colour {
	case "always":
		return true
	case "never":
		return false
	}
	return !s.inv.Options.Display.NoColor && isTerminal(stream)
}
