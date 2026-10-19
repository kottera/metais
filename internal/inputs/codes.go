// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"fmt"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/inputs/csv"
	"github.com/kottera/metais/internal/jsonx"
)

// The codes the inputs layer reports itself (ADR-003 Interfaces, ADR-009
// D19). Each is class content. The forms' readers report their own:
// jsonx's json family, yamlx's yaml family, csv's, and markdown's
// frontmatter_unclosed.
const (
	// The data directory (ADR-003 D15, D16; ADR-009 D7).
	CodeFileUnknown          finding.Code = "input.file_unknown"
	CodeFileDuplicate        finding.Code = "input.file_duplicate"
	CodeExtensionNotAccepted finding.Code = "input.extension_not_accepted"
	CodeLink                 finding.Code = "input.link"
	CodeNotRegularFile       finding.Code = "input.not_regular_file"
	CodeNotReadable          finding.Code = "input.not_readable"
	CodeSubdirectory         finding.Code = "collection.subdirectory"
	CodeStemDuplicate        finding.Code = "collection.stem_duplicate"
	CodeAsFile               finding.Code = "collection.as_file"
	CodeItemName             finding.Code = "collection.item_name"
	CodeDirectoryName        finding.Code = "collection.directory_name"

	// The forms (ADR-003 D8, D11 to D13).
	CodeEncodingInvalid       finding.Code = jsonx.CodeEncodingInvalid
	CodeNumberOutOfRange      finding.Code = jsonx.CodeNumberOutOfRange
	CodeNoncharacter          finding.Code = "input.noncharacter"
	CodeFrontmatterNotMapping finding.Code = "markdown.frontmatter_not_mapping"
	CodeFrontmatterBodyField  finding.Code = "markdown.frontmatter_body_field"
	CodeLimitExceeded         finding.Code = finding.CodeLimitExceeded
	codeCSVFieldCount         finding.Code = csv.CodeFieldCount
	codeCSVBareQuote          finding.Code = csv.CodeBareQuote
	codeCSVAfterQuote         finding.Code = csv.CodeAfterQuote

	// The command line (ADR-003 D17, D18).
	CodeSetPointerInvalid  finding.Code = "input.set_pointer_invalid"
	CodeSetTraversal       finding.Code = "input.set_traversal"
	CodeSetIntoUnparsed    finding.Code = "input.set_into_unparsed"
	CodeJSONArgNotObject   finding.Code = "input.json_arg_not_object"
	CodeJSONArgUnknown     finding.Code = "input.json_arg_unknown"
	CodeJSONArgInvalid     finding.Code = "input.json_arg_invalid"
	CodeMissing            finding.Code = "input.missing"
	CodeConstraintsSkipped finding.Code = "input.constraints_not_evaluated"
	CodeConstraintsFailed  finding.Code = "input.constraints_failed"
)

// The reasons and kinds the codes above name in their params.
const (
	reasonInvalidUTF8 = "invalid_utf8"
	reasonUTF16       = "utf16"
	reasonUnsafeInt   = "unsafe_integer"

	kindDirectory = "directory"
	kindSpecial   = "special"

	nameEncoding  = "encoding"
	nameCharacter = "character"

	setNotPointer   = "not_a_pointer"
	setMalformed    = "malformed"
	setUnknownInput = "unknown_input"
	setNoValue      = "no_value"

	traverseScalar     = "scalar"
	traverseIndex      = "index_invalid"
	traverseOutOfRange = "index_out_of_range"
	traverseDash       = "dash_not_last"
)

// The related notes (ADR-006 D5 item 9).
const (
	noteCollides   = "collides with"
	noteFirstEntry = "first entry"
)

// rebuild returns f about subj, its location and every related location
// passed through loc. A reader reports a finding about the document it
// was given; the layer places it in the input it is part of, which is a
// collection's item at /<index> or an --inputs-json member, by building
// it again from its own code, params and options, so its message and
// hint are rendered for where it now is.
func rebuild(f finding.Finding, subj finding.Subject, loc func(finding.Location) finding.Location, via ...finding.Via) finding.Finding {
	e, ok := finding.Lookup(f.Code())
	if !ok {
		panic("inputs: a finding of an unregistered code")
	}
	p := finding.Params{}
	for _, ps := range e.Params {
		if v, ok := f.Param(ps.Name); ok {
			p[ps.Name] = v
		}
	}
	var opts []finding.Option
	if s, ok := f.Schema(); ok {
		opts = append(opts, finding.WithSchema(s))
	}
	if fd, ok := f.Field(); ok {
		opts = append(opts, finding.WithField(fd))
	}
	opts = append(opts, finding.WithVia(append(via, f.Via()...)...), finding.WithAlternatives(f.Alternatives()...))
	var related []finding.Related
	for _, r := range f.RelatedAt() {
		related = append(related, finding.Related{Location: loc(r.Location), Note: r.Note})
	}
	opts = append(opts, finding.WithRelated(related...))
	var at *finding.Location
	if l, ok := f.At(); ok {
		l = loc(l)
		at = &l
	}
	return finding.New(f.Code(), subj, at, p, opts...)
}

// keep is the identity of rebuild's loc.
func keep(l finding.Location) finding.Location { return l }

// under is rebuild's loc for a document placed at prefix in its input: a
// pointer the reader gave below its document's root is prefixed. A
// location's pointer is read as a path, so placing it costs its depth, not
// its length.
func under(prefix string) func(finding.Location) finding.Location {
	pp, ok := finding.ParsePath(prefix)
	if !ok {
		panic(fmt.Sprintf("inputs: %q is not a JSON Pointer", prefix))
	}
	return func(l finding.Location) finding.Location {
		if l.HasPointer {
			l = l.WithPath(l.Path().Under(pp))
		}
		return l
	}
}
