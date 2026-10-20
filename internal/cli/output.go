// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"slices"

	"github.com/kottera/metais/internal/cli/args"
	"github.com/kottera/metais/internal/fault"
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
	// notice is said about the result in human mode, on standard error
	// (ADR-008 D11 item 3): a grep truncation, the entries it skipped, an
	// outline with no headings. The JSON document carries the same in its
	// result.
	notice []byte
	next   []Next
	// report is set when the outcome is the report of a command whose
	// findings are its result, and nothing else is: validate's. Its
	// summary line follows its human findings (ADR-006 D11 item 3), and
	// its result is printed with errors among its findings, since the
	// command produced it (ADR-008 D10 item 1).
	report bool
	// dataDir is the --data argument the command read, for the path
	// canary alone.
	dataDir string
	// stream writes the bytes standard output carries (ADR-008 D11 item
	// 1): render --out -'s archive, or the one file --only selects. It is
	// set only on a render that succeeded, and runs before anything else
	// is written.
	stream func(w io.Writer)
}

// canaryLeak is nil in every release build. The path canary of ADR-009
// D16 item 2 sets it, under its build tag, to put the data directory's
// absolute path into each finding's message, so the SC7 harness can be
// shown to see a host path in a finding.
var canaryLeak func(b []byte, dataDir string) []byte

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
// ADR-008 D13 maps from o's findings, the command's role and --fail-on.
// The report lists at most the findings limit of the command's limits.
func (s *session) emit(o outcome) int {
	listed := limits.Get(limits.Findings).Default
	if s.lim != nil {
		listed = s.lim.Value(limits.Findings)
	}
	r := finding.NewReportFaults(int(min(listed, int64(^uint(0)>>1))), s.inv.Options.Faults)
	for _, f := range o.findings {
		r.Add(f)
	}
	role := finding.RoleNoPackage
	if s.line.cmd != nil {
		role = s.line.cmd.Role
	}
	if s.role != 0 {
		role = s.role
	}
	status := r.Exit(role, s.failOn())
	fs, sum := r.Findings(), r.Summary()
	next := o.next
	if next == nil {
		next = []Next{}
	}
	// Standard output carries the bytes and nothing else when the command
	// writes bytes there, a refusal's included, so that a refused render's
	// standard output is empty (ADR-008 D11 items 2 and 4).
	stdout := s.inv.Stdout
	if s.line.streamsBytes() {
		if s.inv.Options.Faults.Has(fault.StdoutNotice) {
			write(stdout, []byte("metais: writing the render to standard output\n"))
		}
		if o.stream != nil {
			o.stream(stdout)
		}
		stdout = s.inv.Stderr
	}
	if s.line.json {
		result := o.result
		if sum.Errors > 0 && !(o.report && s.reports(fs)) {
			result = nil
		}
		leak := func(b []byte) []byte { return b }
		if canaryLeak != nil && o.dataDir != "" {
			leak = func(b []byte) []byte { return canaryLeak(b, o.dataDir) }
		}
		if s.inv.Options.Faults.Has(fault.ColourJSON) && isTerminal(stdout) {
			write(stdout, []byte("\x1b[1m"))
			defer write(stdout, []byte("\x1b[0m"))
		}
		writeEnvelope(stdout, envelope{
			Format:  s.line.format(),
			OK:      status == finding.ExitOK && sum.Errors == 0,
			Summary: sum.AppendJSON(nil),
			Result:  result,
			Next:    next,
		}, fs, leak)
		return status
	}
	// Human mode (ADR-008 D11 item 3): a refusal, its findings and its
	// next steps go to standard error and nothing to standard output;
	// otherwise the result goes to standard output, and any finding and
	// the Next: block to standard error. The findings of a command whose
	// findings are its result go to standard output when they do not
	// refuse it, with its summary line, its human result and the Next:
	// block (ADR-006 D11 item 3).
	if s.reports(fs) {
		b := finding.AppendHuman(nil, s.colour(s.inv.Stdout), fs...)
		if canaryLeak != nil && o.dataDir != "" {
			b = canaryLeak(b, o.dataDir)
		}
		if o.report {
			b = sum.AppendHuman(b)
		}
		write(s.inv.Stdout, appendNext(append(b, o.human...), next))
		return status
	}
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
	write(s.inv.Stderr, appendNext(append(b, o.notice...), next))
	return status
}

// streamsBytes reports whether standard output carries bytes on the line
// l, render --out -'s or pack --out -'s, from before the command runs, so
// that a refusal of the line, its own or the command's, writes nothing
// there: render or pack --out - with no --dry-run, which writes none.
func (l line) streamsBytes() bool {
	out, _ := l.parsed.Value("out")
	return l.cmd != nil && (l.cmd.Name == "render" || l.cmd.Name == "pack") && out == "-" && !l.parsed.Has("dry-run")
}

// reports reports whether fs are the command's result rather than its
// refusal: the command's findings are its result, and fs do not refuse
// it, nor, under a consumer command, hold an error about the package's
// own content, which refuses it as the exit status does (ADR-006 D12 item
// 2).
func (s *session) reports(fs []finding.Finding) bool {
	if s.line.cmd == nil || !s.line.cmd.FindingsAreResult || refusal(fs) {
		return false
	}
	return s.line.cmd.Role != finding.RoleConsumer || !slices.ContainsFunc(fs, func(f finding.Finding) bool {
		return f.Severity() == finding.SeverityError && packageContent(f.Subject().Kind)
	})
}

// refusal reports whether fs refuse the command rather than report on its
// package: a finding of class usage, refused, destination or internal.
func refusal(fs []finding.Finding) bool {
	return slices.ContainsFunc(fs, func(f finding.Finding) bool { return f.Class() != finding.ClassContent })
}

// packageContent reports whether a finding about k is about the package's
// own content.
func packageContent(k finding.SubjectKind) bool {
	switch k {
	case finding.SubjectPackage, finding.SubjectManifest, finding.SubjectSchema, finding.SubjectTemplate, finding.SubjectAsset, finding.SubjectReference:
		return true
	}
	return false
}

// failOn is the severity that fails the command (ADR-006 D2 item 5): a
// warning when the line gave --fail-on warning, else an error.
func (s *session) failOn() finding.Severity {
	if v, _ := s.line.parsed.Value(failOnFlag.Name); v == "warning" {
		return finding.SeverityWarning
	}
	return finding.SeverityError
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

// nextSteps is the declared next step of each of fs's causes, each once,
// in the order of fs.
func (s *session) nextSteps(fs []finding.Finding) []Next {
	var out []Next
	for _, f := range fs {
		if n := declaredCause(s.refusals(), f).Next; !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	return out
}

// declaredCause is the refusal of rs that f is a finding of: the row of
// its code whose match f satisfies, or else the row of its code with no
// match (ADR-008 D14 item 4).
func declaredCause(rs []Refusal, f finding.Finding) Refusal {
	if i := slices.IndexFunc(rs, func(r Refusal) bool { return r.Code == f.Code() && r.match != nil && r.match(f) }); i >= 0 {
		return rs[i]
	}
	if i := slices.IndexFunc(rs, func(r Refusal) bool { return r.Code == f.Code() && r.match == nil }); i >= 0 {
		return rs[i]
	}
	panic("cli: " + string(f.Code()) + " is not among the refusals declared")
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

// writeEnvelope writes e as writeJSON would with fs as its findings, and
// writes the findings one at a time, each through leak, so that a report
// of the findings limit's findings is never held as one encoding beside
// the findings it encodes.
func writeEnvelope(w io.Writer, e envelope, fs []finding.Finding, leak func([]byte) []byte) {
	bw := bufio.NewWriter(w)
	e.Findings = json.RawMessage("[]")
	b := marshal(e)
	at := bytes.Index(b, []byte(`,"findings":[],`)) + len(`,"findings":`)
	bw.Write(b[:at])
	bw.WriteByte('[')
	var f []byte
	for i := range fs {
		if i > 0 {
			bw.WriteByte(',')
		}
		f = finding.AppendJSON(f[:0], fs[i])
		bw.Write(leak(f[1 : len(f)-1]))
	}
	bw.WriteByte(']')
	bw.Write(b[at+len("[]"):])
	_ = bw.Flush()
}

// marshal is v's JSON document, as writeJSON writes it.
func marshal(v any) []byte {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		panic("cli: encoding a document: " + err.Error())
	}
	return b.Bytes()
}

// writeJSON writes v as one compact JSON document ending in one LF, with
// HTML escaping off (ADR-008 D10 item 2).
func writeJSON(w io.Writer, v any) { write(w, marshal(v)) }

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
