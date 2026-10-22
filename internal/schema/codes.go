// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/regexprofile"
)

// The codes the compiler reports about a schema file, ADR-003's and
// ADR-009's (ADR-003 Interfaces, ADR-009 D19). Each is class content and
// about the package entry the schema is written in: a schema file is
// compiled once, whichever inputs use it, so its findings name the entry
// rather than an input.
const (
	CodeKeywordUnknown     finding.Code = "schema.keyword_unknown"
	CodeFormatUnsupported  finding.Code = "schema.format_unsupported"
	CodeDialect            finding.Code = "schema.dialect"
	CodeRefUnresolved      finding.Code = "schema.ref_unresolved"
	CodeRefEscapes         finding.Code = "schema.ref_escapes"
	CodeRefByPath          finding.Code = "schema.ref_by_path"
	CodeIDDuplicate        finding.Code = "schema.id_duplicate"
	CodePatternUnsupported finding.Code = "schema.pattern_unsupported"
	CodeInvalid            finding.Code = "schema.invalid"
)

// The reasons a schema.dialect finding gives.
const (
	dialectSchema     = "schema"
	dialectVocabulary = "vocabulary"
)

var pkg = finding.Subject{Kind: finding.SubjectPackage}

// at is the location of the value at ptr in the schema file entry.
func at(entry, ptr string) *finding.Location {
	l := finding.Location{Origin: finding.OriginPackage, File: entry}.WithPointer(ptr)
	return &l
}

// fstr is a string parameter of a finding.
func fstr(s string) finding.Value { return finding.String(s) }

func report(code finding.Code, entry, ptr string, p finding.Params) finding.Finding {
	if p == nil {
		p = finding.Params{}
	}
	return finding.New(code, pkg, at(entry, ptr), p)
}

// regexFinding is the finding for a pattern the profile refused, or nil
// for an error the profile does not report as a refusal.
func regexFinding(err error, entry, ptr string, lim limits.Set) (finding.Finding, bool) {
	switch e := err.(type) {
	case *regexprofile.UnsupportedError:
		return report(CodePatternUnsupported, entry, ptr, finding.Params{"construct": fstr(string(e.Construct))}), true
	case *regexprofile.LimitError:
		return lim.Breached(e.Limit, *at(entry, ptr), e.Observed, e.HasObserved), true
	}
	return finding.Finding{}, false
}
