// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

// The limit codes other packages report by name.
const (
	CodeLimitInvalid          Code = "limit.invalid"
	CodeLimitPackage          Code = "limit.package"
	CodeLimitExceeded         Code = "limit.exceeded"
	CodeLimitExceededOutput   Code = "limit.exceeded.output"
	CodeLimitExceededTemplate Code = "limit.exceeded.template"
)

// limitCodes are the limit family's registry entries, in code order.
var limitCodes = []Entry{
	{
		Code:     CodeLimitExceeded,
		Severity: SeverityError,
		Class:    ClassContent,
		Subject:  SubjectInput,
		Params: []ParamSpec{
			{Name: "limit", Type: TypeString, Meaning: "The limit's name, as metais limits lists it.", Always: true},
			{Name: "bound", Type: TypeNumber, Meaning: "The limit's value in this command, its default or the lower value given with --limit.", Always: true},
			{Name: "lowered", Type: TypeBoolean, Meaning: "Whether the bound was lowered with --limit rather than being the default.", Always: true},
			{Name: "observed", Type: TypeNumber, Meaning: "The value found, when the reading measured it; absent when the reading stopped at the bound."},
		},
		Message:    "The input {input} breaches the {limit} limit of {bound}.",
		Hint:       "Supply a smaller or shallower input; a limit can be lowered with --limit but never raised. Run `metais limits` to see every limit.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{"MTS-SEC-005"},
	},
	{
		Code:     "limit.exceeded.output",
		Severity: SeverityError,
		Class:    ClassContent,
		Subject:  SubjectOutput,
		Params: []ParamSpec{
			{Name: "limit", Type: TypeString, Meaning: "The limit's name, as `metais limits` lists it.", Always: true},
			{Name: "bound", Type: TypeNumber, Meaning: "The limit's bound in this command, in the limit's unit.", Always: true},
			{Name: "lowered", Type: TypeBoolean, Meaning: "Whether the caller lowered the limit with --limit; false when the bound is the specification's default.", Always: true},
			{Name: "observed", Type: TypeNumber, Meaning: "The value found, in the limit's unit; absent when the render stopped at the bound."},
		},
		Message:    "The output {output} breaches the {limit} limit of {bound}.",
		Hint:       "Supply less data, or render fewer or smaller files; a limit can be lowered with --limit and never raised. Run `metais limits` to see every limit.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{"MTS-OUT-011", "MTS-PTH-009", "MTS-SEC-010"},
	},
	{
		Code:     "limit.exceeded.template",
		Severity: SeverityError,
		Class:    ClassContent,
		Subject:  SubjectTemplate,
		Params: []ParamSpec{
			{Name: "limit", Type: TypeString, Meaning: "The limit's name, as `metais limits` lists it.", Always: true},
			{Name: "bound", Type: TypeNumber, Meaning: "The limit's bound in this command, in the limit's unit.", Always: true},
			{Name: "lowered", Type: TypeBoolean, Meaning: "Whether the caller lowered the limit with --limit; false when the bound is the specification's default.", Always: true},
			{Name: "output_path", Type: TypeString, Meaning: "The path of the output being rendered, relative to the root of its tree; present when the error was met rendering an output."},
		},
		Message:    "Rendering the template breaches the {limit} limit of {bound}.",
		Hint:       "Render less: fewer iterations, partials nested less deeply, smaller strings and arrays; a limit can be lowered with --limit and never raised. Run `metais limits` to see every limit.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{"MTS-OUT-010", "MTS-SEC-008", "MTS-SEC-009", "MTS-TPL-046"},
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
		Code:     "limit.package",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "limit", Type: TypeString, Meaning: "The limit's name, as `metais limits` lists it.", Always: true},
			{Name: "bound", Type: TypeNumber, Meaning: "The limit's bound in this command, in the limit's unit.", Always: true},
			{Name: "lowered", Type: TypeBoolean, Meaning: "Whether the caller lowered the limit with --limit; false when the bound is the specification's default.", Always: true},
			{Name: "observed", Type: TypeNumber, Meaning: "The value found, in the limit's unit; absent when reading stopped at the bound."},
		},
		Message:    "The package exceeds the {limit} limit of {bound}.",
		Hint:       "Make the package smaller than the limit; run `metais limits` to see every limit, which can be lowered and never raised.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{"MTS-AUT-052", "MTS-PTH-007", "MTS-SEC-001", "MTS-SEC-011", "MTS-SEC-020", "MTS-SEC-021", "MTS-SEC-022"},
	},
}
