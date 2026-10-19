// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package cli is the metais command line: its command table and the
// dispatch from an invocation to a command (ADR-008).
//
// The whole tool runs in-process through Run, so a test drives a command
// exactly as cmd/metais does, with its own streams and, where it proves a
// guarantee, its own faults (ADR-010 D13 item 3).
package cli

import (
	"context"
	"io"
	"slices"
	"strings"

	"github.com/kottera/metais/internal/cli/args"
	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostenv"
	"github.com/kottera/metais/internal/limits"
)

// Options is what an invocation carries besides its arguments and streams.
type Options struct {
	// Display is the host's display setting, read by hostenv.Isolate.
	Display hostenv.Display
	// Faults are the faults a test injects. Main leaves it zero, and
	// nothing reachable from the command line sets it.
	Faults fault.Set
}

// Invocation is one run of the tool.
type Invocation struct {
	Args    []string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	Options Options
}

// Main runs the tool for cmd/metais: args are the arguments after the
// program name, and d is what hostenv.Isolate returned. It builds the
// invocation with a zero fault set and returns the exit status.
func Main(args []string, stdin io.Reader, stdout, stderr io.Writer, d hostenv.Display) int {
	return Run(context.Background(), Invocation{
		Args:    args,
		Stdin:   stdin,
		Stdout:  stdout,
		Stderr:  stderr,
		Options: Options{Display: d},
	})
}

// session is one invocation on its way through the tool.
type session struct {
	ctx   context.Context
	inv   Invocation
	table []*Command
	line  line
	// lim is the command's limits once --limit has been read, nil before
	// and on a line that gives none a command reads.
	lim *limits.Set
	// role, when set, is what the run is to the package in place of its
	// command's role: render --sample is an authoring command (ADR-008 D8
	// item 8).
	role finding.Role
}

// Run dispatches inv to the command its arguments name and returns the
// exit status, which ADR-008 D13 maps from the findings: every invocation
// is either run or refused with a usage finding, and a panic anywhere in
// the tool is reported as internal.error with status 2 (ADR-006 D12 item
// 4).
func Run(ctx context.Context, inv Invocation) int {
	return run(ctx, inv, table)
}

// run is Run over the command table t.
func run(ctx context.Context, inv Invocation, t []*Command) (status int) {
	s := &session{ctx: ctx, inv: inv, table: t}
	phase := "parse"
	defer func() {
		if r := recover(); r != nil {
			status = s.internalError(phase)
		}
	}()
	s.line = readLine(t, inv.Args)
	l := s.line
	if len(l.findings) > 0 {
		phase = "output"
		return s.refuse(l.findings)
	}
	phase = "run"
	var o outcome
	switch {
	case l.help && l.cmd == nil:
		o = s.overview()
	case l.help:
		o = s.commandHelp(l.cmd)
	default:
		o = l.cmd.run(s, l.parsed)
	}
	phase = "output"
	return s.emit(o)
}

// line is what a command line asks for, read against the table.
type line struct {
	// cmd is the command the line names, or nil when it names none or
	// one the table does not have.
	cmd    *Command
	parsed args.Parsed
	// at maps an index into the slice parsed was read from to its index
	// in the whole line.
	at []int
	// json is whether the line asked for JSON. On a line that does not
	// parse it is whether --json appears before the first --.
	json bool
	// colour is --color's value, or auto when it is absent or refused.
	colour string
	// help is whether --help or -h was given.
	help bool
	// findings are the line's refusals, in no particular order.
	findings []finding.Finding
}

// format is the name of the document the line prints (ADR-008 D10 item 1):
// metais.help/1 for a line whose --help runs (D4 item 2), else the
// command's, or metais.usage/1 when the line names no command.
func (l line) format() string {
	switch {
	case l.help && len(l.findings) == 0:
		return "metais.help/1"
	case l.cmd != nil:
		return l.cmd.Format
	}
	return usageFormat
}

// readLine reads argv against the table t. Before the command's name only
// the global flags and --version are read (ADR-008 D3 item 3); the
// command's name is the first argument that is not one of them, and the
// rest of the line is read against the command's declaration, with the
// flags before its name counted with the flags after it, so a flag given
// on both sides is a repetition.
func readLine(t []*Command, argv []string) line {
	l := line{json: args.RequestsJSON(argv), colour: "auto"}
	n := leadLength(argv)
	lead, leadErrs := args.Parse(leadSpec, argv[:n])
	for _, e := range leadErrs {
		l.findings = append(l.findings, usageFinding(e, e.Arg, ""))
	}
	version := lead.Has(versionFlag.Name)
	var rest []string
	var at []int // at[i] is the index in argv of rest[i]
	for i, a := range argv[:n] {
		if a != "--"+versionFlag.Name {
			rest, at = append(rest, a), append(at, i)
		}
	}
	switch {
	case version:
		l.cmd = lookup(t, "version")
	case n < len(argv):
		l.cmd = lookup(t, argv[n])
		if l.cmd == nil {
			l.findings = append(l.findings, finding.New(finding.CodeUsageUnknownCommand, invocation, nil,
				finding.Params{"argument": finding.Int(int64(n + 1))}))
		}
		n++
	}
	if l.cmd == nil {
		l.parsed, l.at = lead, nil
		for i := range n {
			l.at = append(l.at, i)
		}
		l.help = lead.Has("help")
		if n == len(argv) && !l.help && len(l.findings) == 0 {
			l.findings = append(l.findings, finding.New(finding.CodeUsageMissingCommand, invocation, nil, nil))
		}
		l.setDisplay(lead, len(l.findings) == 0)
		return l
	}
	for i := n; i < len(argv); i++ {
		rest, at = append(rest, argv[i]), append(at, i)
	}
	p, errs := args.Parse(l.cmd.spec(), rest)
	if p.Has("help") {
		// Help needs no operand: a line that asks for it is refused only
		// for an argument it does not accept (ADR-008 D4 item 2).
		errs = slices.DeleteFunc(errs, func(e args.Error) bool { return e.Kind == args.MissingOperand })
	}
	for _, e := range errs {
		i := len(argv)
		if e.Arg < len(at) {
			i = at[e.Arg]
		}
		if i < n {
			// Refused already, against the global flags: a flag given
			// twice before the command's name is given twice there.
			continue
		}
		l.findings = append(l.findings, usageFinding(e, i, l.cmd.Name))
	}
	l.parsed, l.at = p, at
	l.help = p.Has("help")
	l.setDisplay(p, len(l.findings) == 0)
	return l
}

// setDisplay takes --json and --color from p, which parsed cleanly when
// ok is set.
func (l *line) setDisplay(p args.Parsed, ok bool) {
	if ok {
		l.json = p.Has("json")
	}
	if v, given := p.Value("color"); given {
		l.colour = v
	}
}

// leadLength is the number of arguments before the command's name: every
// argument from the first that begins with - and is longer than one byte,
// the value of a global flag written --name value or -x value included,
// up to the first that does not or is --.
func leadLength(argv []string) int {
	i := 0
	for i < len(argv) {
		a := argv[i]
		if a == "--" || a == "-" || !strings.HasPrefix(a, "-") {
			break
		}
		i++
		for _, f := range leadSpec.Flags {
			short := f.Short != 0 && a == "-"+string(f.Short)
			if !f.Boolean() && (a == "--"+f.Name || short) && i < len(argv) {
				i++
			}
		}
	}
	return i
}

// invocation is the subject of every finding about the command line.
var invocation = finding.Subject{Kind: finding.SubjectInvocation}

// usageFinding is the finding for e, an argument refused at index i of
// the line, read against the command called cmd, or before any command
// when cmd is empty. Its argument parameter is 1-based.
func usageFinding(e args.Error, i int, cmd string) finding.Finding {
	p := finding.Params{"argument": finding.Int(int64(i + 1))}
	flag := "--" + e.Flag
	code := finding.CodeUsageBadValue
	switch e.Kind {
	case args.UnknownFlag, args.AbbreviatedFlag, args.GroupedFlags:
		code = finding.CodeUsageUnknownFlag
		p["reason"] = finding.String(map[args.Kind]string{args.UnknownFlag: "unknown", args.AbbreviatedFlag: "abbreviated", args.GroupedFlags: "grouped"}[e.Kind])
		if e.Flag != "" {
			p["flag"] = finding.String(flag)
		}
		if cmd != "" {
			p["command"] = finding.String(cmd)
		}
	case args.MissingValue, args.BooleanValue, args.ValueNotAllowed:
		p["flag"] = finding.String(flag)
		p["reason"] = finding.String(map[args.Kind]string{args.MissingValue: "missing_value", args.BooleanValue: "boolean_value", args.ValueNotAllowed: "value_not_allowed"}[e.Kind])
		if f, ok := declaredFlag(e.Flag); ok && f.Values != nil {
			vs := make([]finding.Value, len(f.Values))
			for j, v := range f.Values {
				vs[j] = finding.String(v)
			}
			p["allowed"] = finding.List(vs...)
		}
	case args.RepeatedFlag:
		code = finding.CodeUsageRepeatedFlag
		p["flag"] = finding.String(flag)
	case args.ExtraOperand:
		code = finding.CodeUsageUnexpectedOperand
		p["command"] = finding.String(cmd)
	case args.MissingOperand:
		code = finding.CodeUsageMissingOperand
		delete(p, "argument")
		p["command"] = finding.String(cmd)
		p["operand"] = finding.String(e.Operand)
	default:
		panic("cli: no usage code for " + e.Kind.String())
	}
	return finding.New(code, invocation, nil, p)
}

// declaredFlag returns the declaration of the flag called name among the
// global flags and every command's own.
func declaredFlag(name string) (args.Flag, bool) {
	all := slices.Clone(leadSpec.Flags)
	for _, c := range table {
		all = append(all, c.Flags...)
	}
	for _, f := range all {
		if f.Name == name {
			return f, true
		}
	}
	return args.Flag{}, false
}
