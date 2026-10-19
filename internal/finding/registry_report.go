// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

// The report codes other packages report by name.
const (
	CodeReportTruncated Code = "report.truncated"
)

// reportCodes are the report family's registry entries, in code order.
var reportCodes = []Entry{
	{
		Code:     "report.step_skipped",
		Severity: SeverityInfo,
		Class:    ClassContent,
		Subject:  SubjectReport,
		Params: []ParamSpec{
			{Name: "steps", Type: TypeArray, Items: TypeString, Meaning: "The steps of check that did not run because their input could not be made, each by its name in chapter 13, in their order.", Always: true},
		},
		Message:    "Some steps of check did not run, because what they read could not be made: {steps}.",
		Hint:       "Fix the errors reported and check again, which runs these steps and may report more.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{"MTS-AUT-002"},
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
		Statements: []string{"MTS-FND-010"},
	},
}
