// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

// jsonCodes are the json family's registry entries, in code order.
var jsonCodes = []Entry{
	{
		Code:     "json.duplicate_name",
		Severity: SeverityError,
		Class:    ClassContent,
		Subject:  SubjectInput,
		Params: []ParamSpec{
			{Name: "name", Type: TypeString, Meaning: "The member name that appears more than once in one object, after escapes are decoded.", Always: true},
		},
		Message:    "The member name {name} appears more than once in one object.",
		Hint:       "Keep one member with this name and remove the others. The input's schema was not checked; validate again after the fix, which may report more.",
		Span:       SpanKey,
		Since:      "1.0",
		Statements: []string{"MTS-SUP-003"},
	},
	{
		Code:       "json.lone_surrogate",
		Severity:   SeverityError,
		Class:      ClassContent,
		Subject:    SubjectInput,
		Message:    "{field} is a string with an escaped UTF-16 surrogate that has no partner, which is no character.",
		Hint:       "Write the character it was meant to be, or a surrogate pair of a high and a low surrogate escape. The input's schema was not checked; validate again after the fix, which may report more.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{"MTS-SUP-004"},
	},
	{
		Code:       "json.noncharacter",
		Severity:   SeverityError,
		Class:      ClassContent,
		Subject:    SubjectInput,
		Message:    "{field} is a string holding a Unicode noncharacter, which I-JSON does not allow.",
		Hint:       "Remove the noncharacter (U+FDD0 to U+FDEF, or a code point ending in FFFE or FFFF) from the string. The input's schema was not checked; validate again after the fix, which may report more.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{"MTS-SUP-004"},
	},
	{
		Code:     "json.syntax",
		Severity: SeverityError,
		Class:    ClassContent,
		Subject:  SubjectInput,
		Params: []ParamSpec{
			{Name: "expected", Type: TypeString, Meaning: "What the JSON grammar allows at this place: a value, a member name, a colon, a comma or the closing bracket, a digit, four hexadecimal digits, or a closing quote.", Always: true},
			{Name: "found", Type: TypeString, Meaning: "What is there instead, by its class: end of input, a control character, an invalid escape, a leading zero, or an unexpected character.", Always: true},
		},
		Message:    "The input {input} is not valid JSON here: expected {expected}, found {found}.",
		Hint:       "Correct the JSON at this line and column; a JSON file holds one value, with no comments and no trailing commas. The input's schema was not checked; validate again after the fix, which may report more.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{"MTS-SUP-003"},
	},
	{
		Code:       "json.trailing_content",
		Severity:   SeverityError,
		Class:      ClassContent,
		Subject:    SubjectInput,
		Message:    "The input {input} has text after its one JSON value.",
		Hint:       "Remove what follows the value, or enclose several values in one array. The input's schema was not checked; validate again after the fix, which may report more.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{"MTS-SUP-003"},
	},
}
