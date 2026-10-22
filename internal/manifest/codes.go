// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package manifest

import "metais.org/metais/internal/finding"

// The version gate's codes (ADR-002 D7 items 1, 7, 8 and 11; ADR-006's
// package.version family). Each is class refused, about the package.
const (
	CodeVersionMissing   finding.Code = "package.version.missing"
	CodeVersionNotString finding.Code = "package.version.not_string"
	CodeVersionMalformed finding.Code = "package.version.malformed"
	CodeVersionMajor     finding.Code = "package.version.unsupported_major"
	CodeVersionMinor     finding.Code = "package.version.unsupported_minor"
)

// The codes of the manifest rules the manifest schema does not express
// (ADR-002 D10 item 6). Each is class content, about the manifest.
const (
	CodeLicenseExpression  finding.Code = "manifest.license.expression"
	CodeInputDuplicateName finding.Code = "manifest.input.duplicate_name"
	CodeOutputDuplicateID  finding.Code = "manifest.output.duplicate_id"
	CodeOutputNestedEach   finding.Code = "manifest.output.nested_for_each"
)

// The reasons of manifest.license.expression.
const (
	reasonSyntax        = "syntax"
	reasonOperatorCase  = "operator_case"
	reasonDocumentRef   = "document_ref"
	reasonParentheses   = "parentheses"
	reasonLicenseRefAdd = "license_ref_plus"
)

// noteFirst is the related note of a duplicate's first occurrence (ADR-006
// D5 item 9).
const noteFirst = "first occurrence"

// ManifestFile is the one name of a source manifest (ADR-002 D1 item 1),
// the file a finding about it names.
const ManifestFile = "metais.yaml"

// Source is where the manifest's value tree was read from, so a finding
// can say where a value is: its file, relative to the package root, and
// the positions its reader kept. Positions is nil when the reader kept
// none, and then a finding has a pointer and no line.
type Source struct {
	File      string
	Positions finding.Positions
}

// location returns the location of the value at ptr. Every code here
// marks the value's first character: for package.version.missing, whose
// span is the parent, that value is the root object lacking the member.
func (s Source) location(ptr string) *finding.Location {
	l := finding.Location{Origin: finding.OriginPackage, File: s.File}.WithPointer(ptr)
	if s.Positions != nil {
		if p, ok := s.Positions.Value(ptr); ok {
			l = l.At(p)
		}
	}
	return &l
}

// finding builds the finding for code about kind, at the value at ptr.
func (s Source) finding(code finding.Code, kind finding.SubjectKind, ptr string, p finding.Params, opts ...finding.Option) finding.Finding {
	return finding.New(code, finding.Subject{Kind: kind}, s.location(ptr), p, opts...)
}
