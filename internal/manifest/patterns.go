// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package manifest

import "regexp"

// The patterns of the field rules, written once and used twice: Build
// matches them with Go's regexp, and spec/schemas/1.0/manifest.schema.json
// carries each as the pattern of the field it judges, which
// TestSchemaPatterns holds byte for byte. So where the schema and the model
// judge a field, they judge it with the same expression.
//
// Each is written in the subset ECMA-262 and RE2 read alike (ADR-002 D10
// item 7): no lookaround, no `.`, and every control character a literal
// character in a class, which the JSON text of the schema writes as a
// \u escape.
const (
	// linePattern is ADR-002 D4 item 6's line: no character of general
	// category Cc, and neither beginning nor ending with U+0020. Cc is
	// U+0000 to U+001F and U+007F to U+009F.
	linePattern = "^[^\u0000- \u007f-\u009f]([^\u0000-\u001f\u007f-\u009f]*[^\u0000- \u007f-\u009f])?$"

	// prosePattern is D4 item 6's prose: the only Cc characters are U+000A
	// and U+0009, and it neither begins nor ends with U+0020, U+000A or
	// U+0009.
	prosePattern = "^[^\u0000- \u007f-\u009f]([^\u0000-\u0008\u000b-\u001f\u007f-\u009f]*[^\u0000- \u007f-\u009f])?$"

	// namePattern is D5 item 1's name: segments of [a-z0-9] runs joined by
	// single hyphens, the segments joined by `.`.
	namePattern = `^[a-z0-9]+(-[a-z0-9]+)*(\.[a-z0-9]+(-[a-z0-9]+)*)*$`

	// nameDevicePattern matches a name whose leading segment is a Windows
	// device name (D5 item 2, ADR-002 D10 item 7's expression).
	nameDevicePattern = `^(con|prn|aux|nul|com[0-9]|lpt[0-9])(\.|$)`

	// nameLongSegmentPattern matches a name with a segment longer than 63
	// characters (D5 item 1): 64 characters with no `.` among them.
	nameLongSegmentPattern = `[a-z0-9-]{64}`

	// semverPattern is the grammar semver.org publishes for Semantic
	// Versioning 2.0.0 (D6 item 1), in its ECMA-262 form.
	semverPattern = `^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)` +
		`(-((0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*)(\.(0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*))*))?` +
		`(\+([0-9a-zA-Z-]+(\.[0-9a-zA-Z-]+)*))?$`

	// specPattern is D7 item 1's specification version.
	specPattern = `^(0|[1-9][0-9]{0,3})\.(0|[1-9][0-9]{0,3})$`

	// emailPattern is D4 item 10's email: exactly one `@`, with at least
	// one character on each side, no U+0020 and no Cc character.
	emailPattern = "^[^@ \u0000-\u001f\u007f-\u009f]+@[^@ \u0000-\u001f\u007f-\u009f]+$"

	// identifierPattern is an input's name (ADR-003 D1 item 3) and an
	// output's id (ADR-005 D2 item 1).
	identifierPattern = `^[a-z][a-z0-9_]{0,63}$`

	// schemaPathPattern is the package path of a schema file: under
	// `inputs/`, with no empty segment, ending .json, .yaml or .yml
	// (ADR-003 D1 item 4, D3 item 1).
	schemaPathPattern = `^inputs/([^/]+/)*[^/]+\.(json|yaml|yml)$`

	// meaningPathPattern is the package path of a meaning document: under
	// `inputs/`, ending .md (ADR-003 D2 item 2).
	meaningPathPattern = `^inputs/([^/]+/)*[^/]+\.md$`

	// relativePathPattern is a path relative to `templates/` or `assets/`,
	// or the package path of an output's validate schema: segments with
	// none empty, so neither a leading nor a trailing `/` (ADR-005 D1, D5
	// item 1, D11 item 1).
	relativePathPattern = `^([^/]+/)*[^/]+$`

	// compiledPattern matches a package path under `.compiled/`, which an
	// output's validate cannot name (ADR-005 D11 item 1).
	compiledPattern = `^\.compiled/`

	// mimetypePattern is an odf container's media type: `type/subtype` in
	// printable ASCII, with no whitespace (ADR-005 D9 item 2).
	mimetypePattern = `^[!-.0-~]+/[!-.0-~]+$`
)

// The parts of httpURIPattern, RFC 3986's grammar (§3, Appendix A).
const (
	uriUnreserved = `-A-Za-z0-9._~`
	uriSubDelims  = `!$&'()*+,;=`
	uriPct        = `%[0-9A-Fa-f]{2}`
	uriPchar      = `([` + uriUnreserved + uriSubDelims + `:@]|` + uriPct + `)`
	uriRegName    = `([` + uriUnreserved + uriSubDelims + `]|` + uriPct + `)+`
	uriDecOctet   = `(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])`
	uriIPv4       = uriDecOctet + `\.` + uriDecOctet + `\.` + uriDecOctet + `\.` + uriDecOctet
	uriH16        = `[0-9A-Fa-f]{1,4}`
	uriLs32       = `(` + uriH16 + `:` + uriH16 + `|` + uriIPv4 + `)`
	uriIPv6       = `((` + uriH16 + `:){6}` + uriLs32 +
		`|::(` + uriH16 + `:){5}` + uriLs32 +
		`|(` + uriH16 + `)?::(` + uriH16 + `:){4}` + uriLs32 +
		`|((` + uriH16 + `:){0,1}` + uriH16 + `)?::(` + uriH16 + `:){3}` + uriLs32 +
		`|((` + uriH16 + `:){0,2}` + uriH16 + `)?::(` + uriH16 + `:){2}` + uriLs32 +
		`|((` + uriH16 + `:){0,3}` + uriH16 + `)?::` + uriH16 + `:` + uriLs32 +
		`|((` + uriH16 + `:){0,4}` + uriH16 + `)?::` + uriLs32 +
		`|((` + uriH16 + `:){0,5}` + uriH16 + `)?::` + uriH16 +
		`|((` + uriH16 + `:){0,6}` + uriH16 + `)?::)`
	uriIPvFuture = `v[0-9A-Fa-f]+\.[` + uriUnreserved + uriSubDelims + `:]+`
)

// httpURIPattern is D4 item 12's homepage, and an author's url: an RFC
// 3986 absolute-URI whose scheme is `https` or `http` in lower case, whose
// authority has a non-empty host and no userinfo, and which therefore has
// no fragment (absolute-URI has none). The host is a reg-name, which
// covers an IPv4 address, or an IP-literal. Its length is bounded by the
// schema's maxLength, not here.
const httpURIPattern = `^https?://(` + uriRegName + `|\[(` + uriIPv6 + `|` + uriIPvFuture + `)\])` +
	`(:[0-9]*)?(/` + uriPchar + `*)*(\?(` + uriPchar + `|[/?])*)?$`

var (
	lineRE            = regexp.MustCompile(linePattern)
	proseRE           = regexp.MustCompile(prosePattern)
	nameRE            = regexp.MustCompile(namePattern)
	nameDeviceRE      = regexp.MustCompile(nameDevicePattern)
	nameLongSegmentRE = regexp.MustCompile(nameLongSegmentPattern)
	semverRE          = regexp.MustCompile(semverPattern)
	specRE            = regexp.MustCompile(specPattern)
	emailRE           = regexp.MustCompile(emailPattern)
	identifierRE      = regexp.MustCompile(identifierPattern)
	schemaPathRE      = regexp.MustCompile(schemaPathPattern)
	meaningPathRE     = regexp.MustCompile(meaningPathPattern)
	relativePathRE    = regexp.MustCompile(relativePathPattern)
	compiledRE        = regexp.MustCompile(compiledPattern)
	mimetypeRE        = regexp.MustCompile(mimetypePattern)
	httpURIRE         = regexp.MustCompile(httpURIPattern)
)
