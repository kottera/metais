// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import (
	"fmt"
	"slices"
	"strings"
)

// Entry is one code's registry entry (ADR-006 D3 item 3).
type Entry struct {
	Code       Code
	Severity   Severity
	Class      Class
	Subject    SubjectKind
	Params     []ParamSpec
	Message    string // template
	Hint       string // template
	Span       SpanKind
	Since      string
	Statements []string
	Retired    string // the minor that retired the code; empty while it is emitted
}

// ParamSpec declares one parameter of a code.
type ParamSpec struct {
	Name    string
	Type    ParamType
	Items   ParamType // an array's element type; zero when its elements may be any scalar
	Meaning string
	Always  bool
}

// The codes this package, internal/limits and internal/cli emit.
const (
	CodeReportTruncated Code = "report.truncated"
	CodeLimitInvalid    Code = "limit.invalid"

	CodeInternalError          Code = "internal.error"
	CodeUsageBadValue          Code = "usage.bad_value"
	CodeUsageMissingCommand    Code = "usage.missing_command"
	CodeUsageRepeatedFlag      Code = "usage.repeated_flag"
	CodeUsageUnexpectedOperand Code = "usage.unexpected_operand"
	CodeUsageUnknownCommand    Code = "usage.unknown_command"
	CodeUsageUnknownFlag       Code = "usage.unknown_flag"
	CodeUsageUnknownTopic      Code = "usage.unknown_topic"
)

// registry is the tool's table of codes, in code order. It is the Go form
// of spec/findings/registry.json, which is the normative text; a test fails
// on any difference between the two (ADR-006 D13 item 4).
var registry = []Entry{
	{
		Code:     CodeInternalError,
		Severity: SeverityError,
		Class:    ClassInternal,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "tool_version", Type: TypeString, Meaning: "The tool's version, as metais version prints it.", Always: true},
			{Name: "phase", Type: TypeString, Meaning: "Where the tool met the defect: parse (reading the command line), run (running the command) or output (writing its result).", Always: true},
		},
		Message:    "metais {tool_version} met a defect in itself ({phase}); neither the command line nor the package caused it.",
		Hint:       "Report the command you ran to the metais project; running it again will meet the same defect.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeLimitInvalid,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "reason", Type: TypeString, Meaning: "Why the argument is refused: unknown_name (no limit has that name), malformed_value (the value is not a non-negative whole number with a unit the limit allows), or above_default (the value is higher than the limit's default, and a limit is never raised).", Always: true},
			{Name: "name", Type: TypeString, Meaning: "The limit's name, present when the argument names a limit."},
			{Name: "default", Type: TypeNumber, Meaning: "The limit's default, present when the argument names a limit."},
			{Name: "unit", Type: TypeString, Meaning: "The limit's unit, present when the argument names a limit: bytes, count or ratio."},
			{Name: "value", Type: TypeNumber, Meaning: "The value asked for, in the limit's unit, present when the reason is above_default and the value is at most 2^53."},
		},
		Message:    "A --limit argument is refused ({reason}); a limit can only be lowered, to a whole number no higher than its default.",
		Hint:       "Run `metais limits` to see every limit's name, unit and default, and give --limit NAME=VALUE with VALUE at most that default.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeReportTruncated,
		Severity: SeverityInfo,
		Class:    ClassContent,
		Subject:  SubjectReport,
		Params: []ParamSpec{
			{Name: "listed", Type: TypeNumber, Meaning: "How many findings the report lists before this one.", Always: true},
			{Name: "omitted", Type: TypeNumber, Meaning: "How many more findings were found and not listed.", Always: true},
		},
		Message:    "The report lists the first {listed} findings; {omitted} more were found and are counted in the summary but not listed.",
		Hint:       "Fix the findings listed and run the command again, or read the summary's totals by code.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageBadValue,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the refused argument among the arguments after metais.", Always: true},
			{Name: "flag", Type: TypeString, Meaning: "The flag whose value is refused, written with its leading --.", Always: true},
			{Name: "reason", Type: TypeString, Meaning: "Why the value is refused: missing_value (the flag takes a value and none follows it), boolean_value (the flag takes no value and was written with =), or value_not_allowed (the value is not one the flag accepts).", Always: true},
			{Name: "allowed", Type: TypeArray, Items: TypeString, Meaning: "Every value the flag accepts, present when the flag accepts a closed set of values."},
		},
		Message:    "The value of {flag} at argument {argument} is refused ({reason}).",
		Hint:       "Give {flag} a value it accepts, as --name value or --name=value; a flag that takes no value is written alone, without =.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:       CodeUsageMissingCommand,
		Severity:   SeverityError,
		Class:      ClassUsage,
		Subject:    SubjectInvocation,
		Message:    "No command was given; metais runs one command, named after any global flags.",
		Hint:       "Run `metais help` for the commands, and run one as metais <command>.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageRepeatedFlag,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position, among the arguments after metais, of the flag's second occurrence.", Always: true},
			{Name: "flag", Type: TypeString, Meaning: "The flag given more than once, written with its leading --.", Always: true},
		},
		Message:    "{flag} is given again at argument {argument}; it may be given only once.",
		Hint:       "Give {flag} once: a repeated flag is refused rather than letting the last one win.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageUnexpectedOperand,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the refused argument among the arguments after metais.", Always: true},
			{Name: "command", Type: TypeString, Meaning: "The command that does not take the operand.", Always: true},
		},
		Message:    "Argument {argument} is an operand that metais {command} does not take.",
		Hint:       "Run `metais {command} --help` for the operands it takes; after --, every argument is an operand.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageUnknownCommand,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the refused argument among the arguments after metais.", Always: true},
		},
		Message:    "Argument {argument} names no metais command; a command is never abbreviated and has no alias.",
		Hint:       "Run `metais help` for the commands.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageUnknownFlag,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the refused argument among the arguments after metais.", Always: true},
			{Name: "reason", Type: TypeString, Meaning: "Why the argument is refused: unknown (it names no flag accepted where it is written; before the command name only the global flags are), abbreviated (it is a prefix of a flag's name, and no flag may be abbreviated), or grouped (one-letter aliases written together, which they never are).", Always: true},
			{Name: "flag", Type: TypeString, Meaning: "The flag the argument abbreviates, or whose one-letter alias begins the group, written with its leading --; present when the reason is abbreviated or grouped."},
			{Name: "command", Type: TypeString, Meaning: "The command whose flags the argument was read against, present when a command was named."},
		},
		Message:    "Argument {argument} is not a flag this command line accepts ({reason}).",
		Hint:       "Run `metais help` for the global flags, or the command's --help for its own; write each flag's name in full, and each one-letter alias on its own.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageUnknownTopic,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the refused argument among the arguments after metais.", Always: true},
		},
		Message:    "Argument {argument} names no command or help topic.",
		Hint:       "Run `metais help` for the commands and the topics.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
}

// Registry returns a copy of the tool's table of codes, in code order.
func Registry() []Entry {
	out := make([]Entry, len(registry))
	for i, e := range registry {
		e.Params = slices.Clone(e.Params)
		e.Statements = slices.Clone(e.Statements)
		out[i] = e
	}
	return out
}

func lookup(code Code) (*Entry, bool) {
	i, ok := slices.BinarySearchFunc(registry, code, func(e Entry, c Code) int {
		return strings.Compare(string(e.Code), string(c))
	})
	if !ok {
		return nil, false
	}
	return &registry[i], true
}

// Lookup returns code's registry entry, and false when code is not
// registered.
func Lookup(code Code) (Entry, bool) {
	e, ok := lookup(code)
	if !ok {
		return Entry{}, false
	}
	c := *e
	c.Params, c.Statements = slices.Clone(e.Params), slices.Clone(e.Statements)
	return c, true
}

// checkParams reports the first way p does not fit e's declared
// parameters.
func (e *Entry) checkParams(p Params) error {
	names := make([]string, 0, len(p))
	for name := range p {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		v := p[name]
		i := slices.IndexFunc(e.Params, func(s ParamSpec) bool { return s.Name == name })
		if i < 0 {
			return fmt.Errorf("parameter %q is not declared", name)
		}
		s := e.Params[i]
		if v.typ != s.Type {
			return fmt.Errorf("parameter %q is a %s, not a %s", name, v.typ, s.Type)
		}
		if s.Type == TypeArray && s.Items != 0 {
			for _, el := range v.list {
				if el.typ != s.Items {
					return fmt.Errorf("parameter %q holds a %s, not a %s", name, el.typ, s.Items)
				}
			}
		}
	}
	for _, s := range e.Params {
		if _, ok := p[s.Name]; s.Always && !ok {
			return fmt.Errorf("parameter %q is always present and was not given", s.Name)
		}
	}
	return nil
}
