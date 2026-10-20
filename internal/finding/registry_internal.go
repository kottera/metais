// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

// The internal codes other packages report by name.
const (
	CodeInternalError  Code = "internal.error"
	CodeInternalMemory Code = "internal.memory"
)

// internalCodes are the internal family's registry entries, in code order.
var internalCodes = []Entry{
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
		Code:     CodeInternalMemory,
		Severity: SeverityError,
		Class:    ClassInternal,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "tool_version", Type: TypeString, Meaning: "The tool's version, as metais version prints it.", Always: true},
		},
		Message:    "metais {tool_version} stopped the command because it held more memory than metais allows itself; this is a defect in metais, not a refusal of the package.",
		Hint:       "Report the command you ran to the metais project; whether it stops here again depends on the machine it runs on.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
}
