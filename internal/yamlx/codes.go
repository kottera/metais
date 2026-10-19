// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package yamlx

import "github.com/kottera/metais/internal/finding"

// rule is one of the profile's refusals. Each has a code in the manifest's
// family and one in an input's (ADR-006 D3 item 2: manifest.yaml.* is
// ADR-002's, yaml.* ADR-003's), registered in spec/findings/registry.json.
type rule uint8

const (
	ruleSyntax rule = iota + 1
	ruleEncoding
	ruleBOM
	ruleCharacter
	ruleDirective
	ruleMultiDocument
	ruleAnchor
	ruleAlias
	ruleTag
	ruleMergeKey
	ruleKeyNotScalar
	ruleKeyEmpty
	ruleDuplicateKey
	ruleScalarRefused
	ruleRootNotMapping
	ruleNumberRange
)

// codes holds each rule's code in the manifest's family, then in an
// input's. The manifest's root is a mapping by ADR-002 D2 item 3; an
// input's may be any value, so that rule has no input code. Encoding and
// number range are common rules of the data, not of YAML: ADR-003 D8
// names input.encoding_invalid and input.number_out_of_range, and ADR-002
// D4 item 3 holds the manifest's numbers whichever profile it is read
// under.
var codes = [...][2]finding.Code{
	ruleSyntax:         {"manifest.yaml.syntax", "yaml.syntax"},
	ruleEncoding:       {"manifest.yaml.encoding", "input.encoding_invalid"},
	ruleBOM:            {"manifest.yaml.bom", "yaml.bom"},
	ruleCharacter:      {"manifest.yaml.character", "yaml.character"},
	ruleDirective:      {"manifest.yaml.directive", "yaml.directive"},
	ruleMultiDocument:  {"manifest.yaml.multi_document", "yaml.multi_document"},
	ruleAnchor:         {"manifest.yaml.anchor", "yaml.anchor"},
	ruleAlias:          {"manifest.yaml.alias", "yaml.alias"},
	ruleTag:            {"manifest.yaml.tag", "yaml.tag"},
	ruleMergeKey:       {"manifest.yaml.merge_key", "yaml.merge_key"},
	ruleKeyNotScalar:   {"manifest.yaml.key_not_scalar", "yaml.key_not_scalar"},
	ruleKeyEmpty:       {"manifest.yaml.key_empty", "yaml.key_empty"},
	ruleDuplicateKey:   {"manifest.yaml.duplicate_key", "yaml.duplicate_key"},
	ruleScalarRefused:  {"manifest.yaml.scalar_refused", "yaml.scalar_refused"},
	ruleRootNotMapping: {"manifest.yaml.root_not_mapping", ""},
	ruleNumberRange:    {"manifest.number_out_of_range", "input.number_out_of_range"},
}

// The values of the parameters with a fixed vocabulary.
const (
	// character's reason.
	reasonForbidden     = "forbidden"
	reasonLineSeparator = "line_separator"
	reasonNoncharacter  = "noncharacter"

	// scalar_refused's form.
	formNull    = "null"
	formBoolean = "boolean"
	formNumber  = "number"

	// encoding's reason, as jsonx gives input.encoding_invalid's.
	reasonInvalidUTF8 = "invalid_utf8"
	reasonUTF16       = "utf16"

	// number_out_of_range's reason, as jsonx gives it.
	reasonOverflow      = "overflow"
	reasonUnsafeInteger = "unsafe_integer"

	// duplicate_key's related note.
	noteFirst = "first occurrence"
)
