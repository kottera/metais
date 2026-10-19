// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

// markdownCodes are the markdown family's registry entries, in code order.
var markdownCodes = []Entry{
	{
		Code:       "markdown.frontmatter_body_field",
		Severity:   SeverityError,
		Class:      ClassContent,
		Subject:    SubjectInput,
		Message:    "The frontmatter of the input {input} has a field named body, which is the text after the frontmatter.",
		Hint:       "Rename the field. The input's schema was not checked; validate again after the fix, which may report more.",
		Span:       SpanKey,
		Since:      "1.0",
		Statements: []string{"MTS-SUP-012"},
	},
	{
		Code:     "markdown.frontmatter_not_mapping",
		Severity: SeverityError,
		Class:    ClassContent,
		Subject:  SubjectInput,
		Params: []ParamSpec{
			{Name: "actual", Type: TypeString, Meaning: "The frontmatter's YAML type: array, boolean, number or string.", Always: true},
		},
		Message:    "The frontmatter of the input {input} is a YAML {actual}, not a mapping.",
		Hint:       "Write the frontmatter as key: value lines, or leave it empty. The input's schema was not checked; validate again after the fix, which may report more.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{"MTS-SUP-012"},
	},
	{
		Code:       "markdown.frontmatter_unclosed",
		Severity:   SeverityError,
		Class:      ClassContent,
		Subject:    SubjectInput,
		Message:    "The input {input} begins with a frontmatter line of --- and has no later line of --- to close it.",
		Hint:       "End the frontmatter with a line that is exactly ---; a file meant to begin with a thematic break begins with *** instead. The input's schema was not checked; validate again after the fix, which may report more.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{"MTS-SUP-012"},
	},
}
