// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import (
	"slices"
	"strings"
	"testing"
)

// Codes the tests report with, beside the registered ones. They stand for
// the codes later waves register, so the report can be proved over every
// subject, origin and span before those codes exist.
const (
	codeInputMinLength Code = "input.fixture_min_length"
	codeInputRequired  Code = "input.fixture_required"
	codeInputType      Code = "input.fixture_type"
	codeManifestField  Code = "manifest.unknown_member"
	codeOutputPath     Code = "output.path.traversal"
	codePackageName    Code = "package.fixture.name"
	codeTemplateParse  Code = "template.parse"
	codeUsageFlag      Code = "usage.test_flag"
	codeWarnUnused     Code = "template.xref.unused_input"
	codeDestNonEmpty   Code = "output.destination.not_empty"
	codeInternalError  Code = "internal.test_error"
)

var testEntries = []Entry{
	{Code: codeInputMinLength, Severity: SeverityError, Class: ClassContent, Subject: SubjectInput,
		Params:  []ParamSpec{{Name: "limit", Type: TypeNumber, Always: true}, {Name: "length", Type: TypeNumber, Always: true}},
		Message: "{field} must be at least {limit} characters long; it has {length}.", Hint: "Write at least {limit} characters.", Span: SpanValue, Since: "1.0"},
	{Code: codeInputRequired, Severity: SeverityError, Class: ClassContent, Subject: SubjectInput,
		Params:  []ParamSpec{{Name: "property", Type: TypeString, Always: true}},
		Message: "{field} is required.", Hint: "Add `{property}` to the input {input}.", Span: SpanParent, Since: "1.0"},
	{Code: codeInputType, Severity: SeverityError, Class: ClassContent, Subject: SubjectInput,
		Params:  []ParamSpec{{Name: "expected", Type: TypeArray, Items: TypeString, Always: true}, {Name: "actual", Type: TypeString, Always: true}},
		Message: "{field} must be {expected}; it is {actual}.", Hint: "Write a value of type {expected}.", Span: SpanValue, Since: "1.0"},
	{Code: codeInternalError, Severity: SeverityError, Class: ClassInternal, Subject: SubjectInvocation,
		Message: "metais failed.", Hint: "Report it.", Span: SpanNone, Since: "1.0"},
	{Code: codeManifestField, Severity: SeverityError, Class: ClassContent, Subject: SubjectManifest,
		Params:  []ParamSpec{{Name: "member", Type: TypeString, Always: true}, {Name: "known", Type: TypeBoolean}},
		Message: "The manifest has a member `{member}` it does not define.", Hint: "Remove it from {package}.", Span: SpanKey, Since: "1.0"},
	{Code: codeDestNonEmpty, Severity: SeverityError, Class: ClassDestination, Subject: SubjectOutput,
		Message: "The output directory is not empty.", Hint: "Use --overwrite.", Span: SpanValue, Since: "1.0"},
	{Code: codeOutputPath, Severity: SeverityError, Class: ClassContent, Subject: SubjectOutput,
		Message: "The output {output} has a path outside the output directory.", Hint: "Fix the path template.", Span: SpanValue, Since: "1.0"},
	{Code: codePackageName, Severity: SeverityError, Class: ClassRefused, Subject: SubjectPackage,
		Message: "An entry name is not a valid package path.", Hint: "Rename the entry in {package}.", Span: SpanValue, Since: "1.0"},
	{Code: codeTemplateParse, Severity: SeverityError, Class: ClassContent, Subject: SubjectTemplate,
		Message: "The template does not parse.", Hint: "Fix the tag.", Span: SpanValue, Since: "1.0"},
	{Code: codeWarnUnused, Severity: SeverityWarning, Class: ClassContent, Subject: SubjectTemplate,
		Message: "No template uses the input.", Hint: "Use it or remove it {{literally}}.", Span: SpanValue, Since: "1.0"},
	{Code: codeUsageFlag, Severity: SeverityError, Class: ClassUsage, Subject: SubjectInvocation,
		Message: "The flag is not known.", Hint: "Run `metais help`.", Span: SpanNone, Since: "1.0"},
}

// withTestCodes registers testEntries beside the real registry until t
// ends. It changes a package variable, so a test that calls it must not
// call t.Parallel.
func withTestCodes(t testing.TB) {
	t.Helper()
	saved := registry
	for _, e := range testEntries {
		if _, ok := lookup(e.Code); ok {
			t.Fatalf("test code %s is also a registered code; give the test another", e.Code)
		}
	}
	all := append(slices.Clone(saved), testEntries...)
	slices.SortFunc(all, func(a, b Entry) int { return strings.Compare(string(a.Code), string(b.Code)) })
	registry = all
	t.Cleanup(func() { registry = saved })
}

func input(name string) Subject {
	return Subject{Kind: SubjectInput, Name: name, Title: strings.ToUpper(name)}
}

func dataDir(file string, line, col int, ptr string) *Location {
	l := Location{Origin: OriginDataDir, File: file, Line: line, Column: col}.WithPointer(ptr)
	return &l
}

func minLength(name, file string, line int, ptr string, length int64) Finding {
	col := 0
	if line > 0 {
		col = 1
	}
	return New(codeInputMinLength, input(name), dataDir(file, line, col, ptr), Params{"limit": Int(10), "length": Int(length)})
}
