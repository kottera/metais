// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package zipfmt

import (
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/unitext"
)

// The codes the reader reports, ADR-001's refusals (ADR-001 Interfaces,
// ADR-009 D19). Each is class refused, about the package.
const (
	CodeNotZip          finding.Code = "package.container.not_zip"
	CodePrependedData   finding.Code = "package.container.prepended_data"
	CodeTrailingData    finding.Code = "package.container.trailing_data"
	CodeArchiveComment  finding.Code = "package.container.archive_comment"
	CodeSecondEndRecord finding.Code = "package.container.second_end_record"
	CodeZip64           finding.Code = "package.container.zip64"
	CodeEncrypted       finding.Code = "package.container.encrypted"
	CodeMethod          finding.Code = "package.container.method"
	CodeFlags           finding.Code = "package.container.flags"
	CodeDataDescriptor  finding.Code = "package.container.data_descriptor"
	CodeExtraField      finding.Code = "package.container.extra_field"
	CodeFieldValue      finding.Code = "package.container.field_value"
	CodeHeaderMismatch  finding.Code = "package.container.header_mismatch"
	CodeOverlapOrGap    finding.Code = "package.container.overlap_or_gap"
	CodeEntryOrder      finding.Code = "package.container.entry_order"
	CodeCRCMismatch     finding.Code = "package.container.crc_mismatch"
	CodeSizeMismatch    finding.Code = "package.container.size_mismatch"
	CodeDeflateStream   finding.Code = "package.container.deflate_stream"

	CodeNameEncoding       finding.Code = "package.name.encoding"
	CodeNameNotNFC         finding.Code = "package.name.not_nfc"
	CodeNameShape          finding.Code = "package.name.shape"
	CodeNameCharacter      finding.Code = "package.name.character"
	CodeNameNotPortable    finding.Code = "package.name.not_portable"
	CodeNameDuplicate      finding.Code = "package.name.duplicate"
	CodeNameCaseCollision  finding.Code = "package.name.case_collision"
	CodeNameFileDirectory  finding.Code = "package.name.file_directory_conflict"
	CodeNameDirectoryEntry finding.Code = "package.name.directory_entry"
)

// The records a field_value finding names, by their APPNOTE names.
const (
	recordLocal   = "local file header"
	recordCentral = "central directory header"
	recordEOCD    = "end of central directory record"
)

// The reasons a deflate_stream finding gives.
const (
	streamInvalid   = "invalid"
	streamTruncated = "truncated"
	streamTrailing  = "trailing_bytes"
)

// at is the location of a finding about the entry name, or about the file
// as a whole when name is empty.
func at(name string) *finding.Location {
	return &finding.Location{Origin: finding.OriginPackage, File: name}
}

var pkg = finding.Subject{Kind: finding.SubjectPackage}

func num(n int64) finding.Value { return finding.Int(n) }

// container builds a package.container finding about name (empty for the
// file as a whole) at offset, with params beside offset.
func container(code finding.Code, name string, offset int64, p finding.Params) finding.Finding {
	if p == nil {
		p = finding.Params{}
	}
	p["offset"] = num(offset)
	return finding.New(code, pkg, at(name), p)
}

func fieldValue(name, record, field string, offset, value, expected int64) finding.Finding {
	return container(CodeFieldValue, name, offset, finding.Params{
		"record": finding.String(record), "header_field": finding.String(field),
		"value": num(value), "expected": num(expected),
	})
}

// nameFindings maps the predicate's problems with the entry name at the
// central directory header at offset to package.name findings, and to
// limit.package for path-depth. A name ending in "/" is a directory entry
// (ADR-001 D2 item 6).
func nameFindings(name string, offset int64, problems []unitext.Problem, lim limits.Set) []finding.Finding {
	var out []finding.Finding
	for _, pr := range problems {
		var f finding.Finding
		switch pr.Kind {
		case unitext.Encoding:
			f = container(CodeNameEncoding, name, offset, finding.Params{"reason": finding.String(pr.Reason)})
		case unitext.NotNFC:
			f = container(CodeNameNotNFC, name, offset, nil)
		case unitext.Shape:
			if pr.Reason == unitext.ReasonTrailingSlash {
				f = container(CodeNameDirectoryEntry, name, offset, nil)
			} else {
				f = container(CodeNameShape, name, offset, finding.Params{"reason": finding.String(pr.Reason)})
			}
		case unitext.Character:
			f = container(CodeNameCharacter, name, offset, finding.Params{"code_point": num(int64(pr.CodePoint))})
		case unitext.NotPortable:
			f = container(CodeNameNotPortable, name, offset, finding.Params{"reason": finding.String(pr.Reason)})
		case unitext.Length:
			f = lim.Breached(limits.PathLength, *at(name), pr.Observed, true)
		case unitext.Depth:
			f = lim.Breached(limits.PathDepth, *at(name), pr.Observed, true)
		default:
			continue
		}
		out = append(out, f)
	}
	return out
}
