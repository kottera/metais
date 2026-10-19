// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import (
	"fmt"
	"slices"
	"strings"
)

// Entry is one code's registry entry (ADR-006 D3 item 3).
type Entry struct {
	Code       Code
	Severity   Severity
	Class      Class
	Subject    SubjectKind
	Params     []ParamSpec
	Message    string // template
	Hint       string // template
	Span       SpanKind
	Since      string
	Statements []string
	Retired    string // the minor that retired the code; empty while it is emitted
}

// ParamSpec declares one parameter of a code.
type ParamSpec struct {
	Name    string
	Type    ParamType
	Items   ParamType // an array's element type; zero when its elements may be any scalar
	Meaning string
	Always  bool
}

// The codes this package, internal/limits and internal/cli emit; the
// package family's are internal/zipfmt's.
const (
	CodeReportTruncated Code = "report.truncated"
	CodeLimitInvalid    Code = "limit.invalid"
	CodeLimitPackage    Code = "limit.package"

	CodeInternalError          Code = "internal.error"
	CodeUsageBadValue          Code = "usage.bad_value"
	CodeUsageMissingCommand    Code = "usage.missing_command"
	CodeUsageRepeatedFlag      Code = "usage.repeated_flag"
	CodeUsageUnexpectedOperand Code = "usage.unexpected_operand"
	CodeUsageUnknownCommand    Code = "usage.unknown_command"
	CodeUsageUnknownFlag       Code = "usage.unknown_flag"
	CodeUsageUnknownTopic      Code = "usage.unknown_topic"
)

// registry is the tool's table of codes, in code order. It is the Go form
// of spec/findings/registry.json, which is the normative text; a test fails
// on any difference between the two (ADR-006 D13 item 4). The JSON Schema
// keyword codes are expanded from one table per family (schemacodes.go).
var registry = withSchemaEntries([]Entry{
	{
		Code:       "csv.after_quote",
		Severity:   SeverityError,
		Class:      ClassContent,
		Subject:    SubjectInput,
		Message:    "The input {input} has text after the closing quote of a CSV field.",
		Hint:       "End a quoted field at its closing quote, followed by the delimiter or the end of the line, and write a quote inside it as two quotes; the rest of this record was not read. The input's schema was not checked; validate again after the fix, which may report more.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:       "csv.bare_quote",
		Severity:   SeverityError,
		Class:      ClassContent,
		Subject:    SubjectInput,
		Message:    "The input {input} has a quote inside a CSV field that does not begin with one.",
		Hint:       "Enclose the whole field in quotes and write each quote inside it as two quotes; the rest of this record was not read. The input's schema was not checked; validate again after the fix, which may report more.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "csv.field_count",
		Severity: SeverityError,
		Class:    ClassContent,
		Subject:  SubjectInput,
		Params: []ParamSpec{
			{Name: "expected", Type: TypeNumber, Meaning: "The number of fields in the file's header record.", Always: true},
			{Name: "found", Type: TypeNumber, Meaning: "The number of fields in this record.", Always: true},
		},
		Message:    "A record of the input {input} has {found} fields where its CSV header has {expected}.",
		Hint:       "Give every record one field per header name; leave a field empty, rather than out, when it has no value. The input's schema was not checked; validate again after the fix, which may report more.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "csv.header_duplicate",
		Severity: SeverityError,
		Class:    ClassContent,
		Subject:  SubjectInput,
		Params: []ParamSpec{
			{Name: "ordinal", Type: TypeNumber, Meaning: "The 1-based position of this field in the header record.", Always: true},
			{Name: "first", Type: TypeNumber, Meaning: "The 1-based position in the header record of the field that first has the same name.", Always: true},
		},
		Message:    "Field {ordinal} of the CSV header of the input {input} repeats the name of field {first}.",
		Hint:       "Give each column its own name; names are compared exactly, with no trimming or case folding. The input's schema was not checked; validate again after the fix, which may report more.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "csv.header_empty",
		Severity: SeverityError,
		Class:    ClassContent,
		Subject:  SubjectInput,
		Params: []ParamSpec{
			{Name: "ordinal", Type: TypeNumber, Meaning: "The 1-based position of the empty field in the header record.", Always: true},
		},
		Message:    "The CSV header of the input {input} has an empty name in field {ordinal}.",
		Hint:       "Name every column in the first record of the file. The input's schema was not checked; validate again after the fix, which may report more.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:       "csv.header_missing",
		Severity:   SeverityError,
		Class:      ClassContent,
		Subject:    SubjectInput,
		Message:    "The input {input} is a CSV file with no header record.",
		Hint:       "Begin the file with a record naming each column, separated by the input's delimiter. The input's schema was not checked; validate again after the fix, which may report more.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:       "csv.unterminated_quote",
		Severity:   SeverityError,
		Class:      ClassContent,
		Subject:    SubjectInput,
		Message:    "The input {input} has a quoted CSV field that is never closed.",
		Hint:       "Close the field with a quote; the rest of the file was read as part of this field, so nothing after it was checked. The input's schema was not checked; validate again after the fix, which may report more.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "input.encoding_invalid",
		Severity: SeverityError,
		Class:    ClassContent,
		Subject:  SubjectInput,
		Params: []ParamSpec{
			{Name: "reason", Type: TypeString, Meaning: "Why the file is not UTF-8: invalid_utf8 (a byte that begins no UTF-8 character, or ends one early) or utf16 (the file begins with a UTF-16 byte order mark).", Always: true},
			{Name: "offset", Type: TypeNumber, Meaning: "The 0-based byte offset of the first byte that is not UTF-8, from the start of the file or argument.", Always: true},
		},
		Message:    "The input {input} is not UTF-8 ({reason}) at byte {offset}.",
		Hint:       "Save the file as UTF-8, with or without a byte order mark. The input's schema was not checked; validate again after the fix, which may report more.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "input.number_out_of_range",
		Severity: SeverityError,
		Class:    ClassContent,
		Subject:  SubjectInput,
		Params: []ParamSpec{
			{Name: "reason", Type: TypeString, Meaning: "Why the number cannot be held: overflow (it rounds to an infinity as a binary64) or unsafe_integer (it is written without a fraction or exponent and its magnitude is above 2^53-1, so it would change silently).", Always: true},
		},
		Message:    "{field} is a number the data model cannot hold exactly ({reason}).",
		Hint:       "Supply it as a string, or as a number of magnitude at most 9007199254740991. The input's schema was not checked; validate again after the fix, which may report more.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
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
		Statements: []string{},
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
		Statements: []string{},
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
		Statements: []string{},
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
		Statements: []string{},
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
		Statements: []string{},
	},
	{
		Code:     "limit.exceeded",
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
		Statements: []string{},
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
		Code:     CodeLimitPackage,
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
		Statements: []string{},
	},
	{
		Code:     "package.container.archive_comment",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "offset", Type: TypeNumber, Meaning: "The byte offset in the package file of the record or field the finding reports.", Always: true},
			{Name: "length", Type: TypeNumber, Meaning: "The length in bytes of the archive comment the end of central directory record declares.", Always: true},
		},
		Message:    "The package's end of central directory record declares an archive comment of {length} bytes; a package has none.",
		Hint:       "Pack the package again with `metais pack` from its source directory, or obtain an intact copy from its publisher.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "package.container.crc_mismatch",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "offset", Type: TypeNumber, Meaning: "The byte offset in the package file of the record or field the finding reports.", Always: true},
		},
		Message:    "The entry's content does not match the CRC-32 its headers record.",
		Hint:       "Pack the package again with `metais pack` from its source directory, or obtain an intact copy from its publisher.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "package.container.data_descriptor",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "offset", Type: TypeNumber, Meaning: "The byte offset in the package file of the record or field the finding reports.", Always: true},
		},
		Message:    "The entry's header sets general purpose bit 3 for a data descriptor; a package's entries carry their CRC-32 and sizes in their headers.",
		Hint:       "Pack the package again with `metais pack` from its source directory, or obtain an intact copy from its publisher.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "package.container.deflate_stream",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "offset", Type: TypeNumber, Meaning: "The byte offset in the package file of the record or field the finding reports.", Always: true},
			{Name: "reason", Type: TypeString, Meaning: "What is wrong with the stream: invalid (it is not a valid DEFLATE stream), truncated (it ends after the entry's data does) or trailing_bytes (the entry's data continues after it ends).", Always: true},
		},
		Message:    "The entry's data is not one DEFLATE stream that fills it exactly ({reason}).",
		Hint:       "Pack the package again with `metais pack` from its source directory, or obtain an intact copy from its publisher.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "package.container.encrypted",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "offset", Type: TypeNumber, Meaning: "The byte offset in the package file of the record or field the finding reports.", Always: true},
		},
		Message:    "The entry is encrypted; a package has no encrypted entries.",
		Hint:       "Pack the package again with `metais pack` from its source directory, or obtain an intact copy from its publisher.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "package.container.entry_order",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "offset", Type: TypeNumber, Meaning: "The byte offset in the package file of the record or field the finding reports.", Always: true},
		},
		Message:    "The entry is out of order; a package's entries are in ascending order of their names' UTF-8 bytes.",
		Hint:       "Pack the package again with `metais pack` from its source directory, or obtain an intact copy from its publisher.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "package.container.extra_field",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "offset", Type: TypeNumber, Meaning: "The byte offset in the package file of the record or field the finding reports.", Always: true},
			{Name: "length", Type: TypeNumber, Meaning: "The length in bytes of the extra field the header declares.", Always: true},
		},
		Message:    "The entry's header carries an extra field of {length} bytes; a package's entries have none.",
		Hint:       "Pack the package again with `metais pack` from its source directory, or obtain an intact copy from its publisher.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "package.container.field_value",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "offset", Type: TypeNumber, Meaning: "The byte offset in the package file of the record or field the finding reports.", Always: true},
			{Name: "record", Type: TypeString, Meaning: "The record holding the field, by its APPNOTE name: central directory header or end of central directory record.", Always: true},
			{Name: "header_field", Type: TypeString, Meaning: "The field, by its APPNOTE name.", Always: true},
			{Name: "value", Type: TypeNumber, Meaning: "The value the field holds.", Always: true},
			{Name: "expected", Type: TypeNumber, Meaning: "The value a package holds in the field.", Always: true},
		},
		Message:    "The {header_field} of the {record} is {value}; a package holds {expected} there.",
		Hint:       "Pack the package again with `metais pack` from its source directory, or obtain an intact copy from its publisher.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "package.container.flags",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "offset", Type: TypeNumber, Meaning: "The byte offset in the package file of the record or field the finding reports.", Always: true},
			{Name: "value", Type: TypeNumber, Meaning: "The general purpose bit flag the header holds.", Always: true},
		},
		Message:    "The entry's general purpose bit flag is {value}; a package's entries set bit 11, for UTF-8 names, and no other bit.",
		Hint:       "Pack the package again with `metais pack` from its source directory, or obtain an intact copy from its publisher.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "package.container.header_mismatch",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "offset", Type: TypeNumber, Meaning: "The byte offset in the package file of the record or field the finding reports.", Always: true},
			{Name: "header_field", Type: TypeString, Meaning: "The field that differs, by its APPNOTE name.", Always: true},
		},
		Message:    "The entry's local file header differs from its central directory header in its {header_field}.",
		Hint:       "Pack the package again with `metais pack` from its source directory, or obtain an intact copy from its publisher.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "package.container.method",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "offset", Type: TypeNumber, Meaning: "The byte offset in the package file of the record or field the finding reports.", Always: true},
			{Name: "method", Type: TypeNumber, Meaning: "The compression method the header declares.", Always: true},
		},
		Message:    "The entry's compression method is {method}; a package's entries are stored (0) or deflated (8).",
		Hint:       "Pack the package again with `metais pack` from its source directory, or obtain an intact copy from its publisher.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:       "package.container.not_zip",
		Severity:   SeverityError,
		Class:      ClassRefused,
		Subject:    SubjectPackage,
		Message:    "The package file is not a zip archive: it ends in no end of central directory record.",
		Hint:       "Pack the package again with `metais pack` from its source directory, or obtain an intact copy from its publisher.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "package.container.overlap_or_gap",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "offset", Type: TypeNumber, Meaning: "The byte offset in the package file of the record or field the finding reports.", Always: true},
			{Name: "declared", Type: TypeNumber, Meaning: "The offset the package's headers declare for the structure, when one does."},
		},
		Message:    "The package's entries, central directory and end record are not contiguous: one overlaps another, or bytes lie between them.",
		Hint:       "Pack the package again with `metais pack` from its source directory, or obtain an intact copy from its publisher.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "package.container.prepended_data",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "offset", Type: TypeNumber, Meaning: "The byte offset in the package file of the record or field the finding reports.", Always: true},
			{Name: "declared", Type: TypeNumber, Meaning: "The offset the package's headers declare for the structure, when one does."},
		},
		Message:    "The package file holds bytes before its first entry; a package's first local file header is at offset 0.",
		Hint:       "Pack the package again with `metais pack` from its source directory, or obtain an intact copy from its publisher.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "package.container.second_end_record",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "offset", Type: TypeNumber, Meaning: "The byte offset in the package file of the record or field the finding reports.", Always: true},
		},
		Message:    "The package file holds a second end of central directory signature in its final 65,557 bytes, so another zip reader could find another central directory.",
		Hint:       "Pack the package again with `metais pack`; an entry that is itself a stored zip near the end of the package can hold the signature.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "package.container.size_mismatch",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "offset", Type: TypeNumber, Meaning: "The byte offset in the package file of the record or field the finding reports.", Always: true},
			{Name: "declared", Type: TypeNumber, Meaning: "The uncompressed size the entry's headers declare.", Always: true},
		},
		Message:    "The entry's content is not the {declared} bytes its headers declare.",
		Hint:       "Pack the package again with `metais pack` from its source directory, or obtain an intact copy from its publisher.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "package.container.trailing_data",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "offset", Type: TypeNumber, Meaning: "The byte offset in the package file of the record or field the finding reports.", Always: true},
		},
		Message:    "The package file holds bytes after its end of central directory record.",
		Hint:       "Pack the package again with `metais pack` from its source directory, or obtain an intact copy from its publisher.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "package.container.zip64",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "offset", Type: TypeNumber, Meaning: "The byte offset in the package file of the record or field the finding reports.", Always: true},
		},
		Message:    "The package uses a zip64 record, field or sentinel value; a package has none.",
		Hint:       "Pack the package again with `metais pack` from its source directory, or obtain an intact copy from its publisher.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "package.name.case_collision",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "offset", Type: TypeNumber, Meaning: "The byte offset in the package file of the record or field the finding reports.", Always: true},
			{Name: "member", Type: TypeString, Meaning: "The entry's name, or the directory prefix of it, that collides.", Always: true},
			{Name: "other", Type: TypeString, Meaning: "The name or directory prefix of the earlier entry it collides with.", Always: true},
		},
		Message:    "The entry's name collides with another's when letter case and normalization are set aside, which one file system would merge into one.",
		Hint:       "Pack the package again with `metais pack` from its source directory, or obtain an intact copy from its publisher.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "package.name.character",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "offset", Type: TypeNumber, Meaning: "The byte offset in the package file of the record or field the finding reports.", Always: true},
			{Name: "code_point", Type: TypeNumber, Meaning: "The first code point the name may not hold, as a number.", Always: true},
		},
		Message:    "The entry's name holds a character a portable name may not: a control, format, private-use, unassigned or separator character, or one Windows reserves.",
		Hint:       "Pack the package again with `metais pack` from its source directory, or obtain an intact copy from its publisher.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "package.name.directory_entry",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "offset", Type: TypeNumber, Meaning: "The byte offset in the package file of the record or field the finding reports.", Always: true},
		},
		Message:    "The entry is a directory entry, its name ending in /; a package's directories exist only as prefixes of its files' names.",
		Hint:       "Pack the package again with `metais pack` from its source directory, or obtain an intact copy from its publisher.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "package.name.duplicate",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "offset", Type: TypeNumber, Meaning: "The byte offset in the package file of the record or field the finding reports.", Always: true},
		},
		Message:    "The entry's name is the name of an earlier entry.",
		Hint:       "Pack the package again with `metais pack` from its source directory, or obtain an intact copy from its publisher.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "package.name.encoding",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "offset", Type: TypeNumber, Meaning: "The byte offset in the package file of the record or field the finding reports.", Always: true},
			{Name: "reason", Type: TypeString, Meaning: "What is wrong: invalid_utf8 (the name is not valid UTF-8) or byte_order_mark (it begins with one).", Always: true},
		},
		Message:    "The entry's name is not UTF-8 without a byte order mark ({reason}).",
		Hint:       "Pack the package again with `metais pack` from its source directory, or obtain an intact copy from its publisher.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "package.name.file_directory_conflict",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "offset", Type: TypeNumber, Meaning: "The byte offset in the package file of the record or field the finding reports.", Always: true},
			{Name: "member", Type: TypeString, Meaning: "The entry's name, or the directory prefix of it, that collides.", Always: true},
			{Name: "other", Type: TypeString, Meaning: "The name or directory prefix of the earlier entry it collides with.", Always: true},
		},
		Message:    "The entry's name is both a file and a directory: one entry's name is, or matches without regard to case, a directory prefix of another's.",
		Hint:       "Pack the package again with `metais pack` from its source directory, or obtain an intact copy from its publisher.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "package.name.not_nfc",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "offset", Type: TypeNumber, Meaning: "The byte offset in the package file of the record or field the finding reports.", Always: true},
		},
		Message:    "The entry's name is not in Unicode Normalization Form C.",
		Hint:       "Pack the package again with `metais pack` from its source directory, or obtain an intact copy from its publisher.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "package.name.not_portable",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "offset", Type: TypeNumber, Meaning: "The byte offset in the package file of the record or field the finding reports.", Always: true},
			{Name: "reason", Type: TypeString, Meaning: "What is not portable: reserved_name (a Windows device name), git (a spelling of .git), segment_end (a segment ending in . or a space) or segment_length (a segment longer than 255 bytes).", Always: true},
		},
		Message:    "The entry's name cannot be written on every platform ({reason}).",
		Hint:       "Pack the package again with `metais pack` from its source directory, or obtain an intact copy from its publisher.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "package.name.shape",
		Severity: SeverityError,
		Class:    ClassRefused,
		Subject:  SubjectPackage,
		Params: []ParamSpec{
			{Name: "offset", Type: TypeNumber, Meaning: "The byte offset in the package file of the record or field the finding reports.", Always: true},
			{Name: "reason", Type: TypeString, Meaning: "What is wrong: empty, leading_slash, empty_segment, dot_segment, backslash or colon.", Always: true},
		},
		Message:    "The entry's name is not a relative path of /-separated segments ({reason}).",
		Hint:       "Pack the package again with `metais pack` from its source directory, or obtain an intact copy from its publisher.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
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
		Statements: []string{},
	},
	{
		Code:     CodeUsageBadValue,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the refused argument among the arguments after metais.", Always: true},
			{Name: "flag", Type: TypeString, Meaning: "The flag whose value is refused, written with its leading --.", Always: true},
			{Name: "reason", Type: TypeString, Meaning: "Why the value is refused: missing_value (the flag takes a value and none follows it), boolean_value (the flag takes no value and was written with =), or value_not_allowed (the value is not one the flag accepts).", Always: true},
			{Name: "allowed", Type: TypeArray, Items: TypeString, Meaning: "Every value the flag accepts, present when the flag accepts a closed set of values."},
		},
		Message:    "The value of {flag} at argument {argument} is refused ({reason}).",
		Hint:       "Give {flag} a value it accepts, as --name value or --name=value; a flag that takes no value is written alone, without =.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:       CodeUsageMissingCommand,
		Severity:   SeverityError,
		Class:      ClassUsage,
		Subject:    SubjectInvocation,
		Message:    "No command was given; metais runs one command, named after any global flags.",
		Hint:       "Run `metais help` for the commands, and run one as metais <command>.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageRepeatedFlag,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position, among the arguments after metais, of the flag's second occurrence.", Always: true},
			{Name: "flag", Type: TypeString, Meaning: "The flag given more than once, written with its leading --.", Always: true},
		},
		Message:    "{flag} is given again at argument {argument}; it may be given only once.",
		Hint:       "Give {flag} once: a repeated flag is refused rather than letting the last one win.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageUnexpectedOperand,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the refused argument among the arguments after metais.", Always: true},
			{Name: "command", Type: TypeString, Meaning: "The command that does not take the operand.", Always: true},
		},
		Message:    "Argument {argument} is an operand that metais {command} does not take.",
		Hint:       "Run `metais {command} --help` for the operands it takes; after --, every argument is an operand.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageUnknownCommand,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the refused argument among the arguments after metais.", Always: true},
		},
		Message:    "Argument {argument} names no metais command; a command is never abbreviated and has no alias.",
		Hint:       "Run `metais help` for the commands.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageUnknownFlag,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the refused argument among the arguments after metais.", Always: true},
			{Name: "reason", Type: TypeString, Meaning: "Why the argument is refused: unknown (it names no flag accepted where it is written; before the command name only the global flags are), abbreviated (it is a prefix of a flag's name, and no flag may be abbreviated), or grouped (one-letter aliases written together, which they never are).", Always: true},
			{Name: "flag", Type: TypeString, Meaning: "The flag the argument abbreviates, or whose one-letter alias begins the group, written with its leading --; present when the reason is abbreviated or grouped."},
			{Name: "command", Type: TypeString, Meaning: "The command whose flags the argument was read against, present when a command was named."},
		},
		Message:    "Argument {argument} is not a flag this command line accepts ({reason}).",
		Hint:       "Run `metais help` for the global flags, or the command's --help for its own; write each flag's name in full, and each one-letter alias on its own.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageUnknownTopic,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the refused argument among the arguments after metais.", Always: true},
		},
		Message:    "Argument {argument} names no command or help topic.",
		Hint:       "Run `metais help` for the commands and the topics.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
})

// Registry returns a copy of the tool's table of codes, in code order.
func Registry() []Entry {
	out := make([]Entry, len(registry))
	for i, e := range registry {
		e.Params = slices.Clone(e.Params)
		e.Statements = slices.Clone(e.Statements)
		out[i] = e
	}
	return out
}

func lookup(code Code) (*Entry, bool) {
	i, ok := slices.BinarySearchFunc(registry, code, func(e Entry, c Code) int {
		return strings.Compare(string(e.Code), string(c))
	})
	if !ok {
		return nil, false
	}
	return &registry[i], true
}

// Lookup returns code's registry entry, and false when code is not
// registered.
func Lookup(code Code) (Entry, bool) {
	e, ok := lookup(code)
	if !ok {
		return Entry{}, false
	}
	c := *e
	c.Params, c.Statements = slices.Clone(e.Params), slices.Clone(e.Statements)
	return c, true
}

// checkParams reports the first way p does not fit e's declared
// parameters.
func (e *Entry) checkParams(p Params) error {
	names := make([]string, 0, len(p))
	for name := range p {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		v := p[name]
		i := slices.IndexFunc(e.Params, func(s ParamSpec) bool { return s.Name == name })
		if i < 0 {
			return fmt.Errorf("parameter %q is not declared", name)
		}
		s := e.Params[i]
		if v.typ != s.Type {
			return fmt.Errorf("parameter %q is a %s, not a %s", name, v.typ, s.Type)
		}
		if s.Type == TypeArray && s.Items != 0 {
			for _, el := range v.list {
				if el.typ != s.Items {
					return fmt.Errorf("parameter %q holds a %s, not a %s", name, el.typ, s.Items)
				}
			}
		}
	}
	for _, s := range e.Params {
		if _, ok := p[s.Name]; s.Always && !ok {
			return fmt.Errorf("parameter %q is always present and was not given", s.Name)
		}
	}
	return nil
}
