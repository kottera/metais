// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package xref

import "metais.org/metais/internal/finding"

// The cross-reference's codes (ADR-007 D3, ADR-006's template family).
// Each is class content. A finding about a template, a partial or a
// manifest expression is about a template: one in a package entry names
// that entry, and one in a manifest expression names the manifest file,
// with the expression's pointer.
const (
	CodeUnknownVariable    finding.Code = "template.xref.unknown_variable"             // D3 item 3
	CodeDynamicReference   finding.Code = "template.xref.dynamic_reference"            // D3 item 6, a warning
	CodeUnknownField       finding.Code = "template.xref.unknown_field"                // D3 item 4.6
	CodeNotAnObject        finding.Code = "template.xref.not_an_object"                // D3 item 4.6
	CodeNotAnArray         finding.Code = "template.xref.not_an_array"                 // D3 item 6
	CodeIndexOutOfRange    finding.Code = "template.xref.index_out_of_range"           // D3 item 6
	CodeMissingPartial     finding.Code = "template.xref.missing_partial"              // D3 item 8
	CodeParameterNotPassed finding.Code = "template.xref.partial_parameter_not_passed" // D3 item 8, a warning
	CodeForEachNotArray    finding.Code = "template.xref.for_each_not_array"           // D3 item 9, about an output
	CodeUnusedInput        finding.Code = "template.xref.unused_input"                 // D3 item 11, a warning about an input
	CodeUnusedTemplate     finding.Code = "template.xref.unused_template"              // D3 item 11, a warning
	CodeUnknownInput       finding.Code = "template.xref.unknown_input"                // D3 item 12
	CodeMissingAsset       finding.Code = "template.xref.missing_asset"                // D3 item 12
	CodeBudgetExhausted    finding.Code = "template.xref.budget_exhausted"             // D3 item 13, a warning about the package
)

// The notes of the related locations the analysis gives (ADR-006 D5 item
// 9).
const (
	noteSchema   = "where the schema declares it"
	noteCallSite = "the render tag that does not pass it"
)

// Codes returns every code the cross-reference reports.
func Codes() []finding.Code {
	return []finding.Code{CodeUnknownVariable, CodeDynamicReference, CodeUnknownField, CodeNotAnObject,
		CodeNotAnArray, CodeIndexOutOfRange, CodeMissingPartial, CodeParameterNotPassed, CodeForEachNotArray,
		CodeUnusedInput, CodeUnusedTemplate, CodeUnknownInput, CodeMissingAsset, CodeBudgetExhausted}
}
