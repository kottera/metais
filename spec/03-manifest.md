# 3. Manifest

**Normative.** Areas: MAN.

The manifest, `metais.yaml`, describes a package: its name and version, the
specification version it targets, a guide for suppliers, its inputs and its
outputs. This chapter specifies the manifest's structure and every field it
holds, the strict YAML and JSON profiles it may be written in, the order in
which it is processed, and the canonical JSON form a packed package carries.

## 3.1 The manifest file

A package holds exactly one manifest, at its top level, and its name is
always `metais.yaml` ([chapter 2](02-container.md)). Its content is YAML or
JSON, decided by one byte, so no file is read under two languages; YAML is the
form an author writes by default, and JSON content is for tooling that emits
JSON. A packed package carries the source manifest byte for byte as the author
wrote it, beside `.compiled/manifest.json`, so comments and formatting survive
unpacking.

<a id="mts-man-001"></a>**MTS-MAN-001** A processor MUST read `metais.yaml`
under the JSON profile (§3.3) when its first byte other than U+0020, U+0009,
U+000A and U+000D is `{`, and under the YAML profile (§3.2) otherwise. Both
profiles give the same data model (§3.4).

<a id="mts-man-002"></a>**MTS-MAN-002** A processor MUST read no manifest but
`metais.yaml`, compared byte for byte, and MUST refuse a package whose top
level holds `metais.json`, `metais.yml`, or `metais.yaml` written in any other
letter case, with `manifest.file.wrong_name` naming it.

## 3.2 The YAML profile

The manifest's YAML is [YAML 1.2.2](A-references.md#yaml) syntax, restricted
so that every accepted document means one thing to every implementation and
maps onto I-JSON ([RFC 7493](A-references.md#rfc7493)). A processor reads the
document as a node tree and applies these rules itself; no library's own type
resolution decides a value.

<a id="mts-man-003"></a>**MTS-MAN-003** A YAML manifest MUST be UTF-8, with no
byte order mark and no character YAML forbids, and a processor MUST refuse one
that is not UTF-8 with `manifest.yaml.encoding`, one with a byte order mark
with `manifest.yaml.bom`, and one holding a forbidden character, a raw U+0085,
U+2028 or U+2029, or a noncharacter with `manifest.yaml.character`.

<a id="mts-man-004"></a>**MTS-MAN-004** A YAML manifest MUST be one document,
with an optional `---` start marker and `...` end marker and no directive, and
its root MUST be a mapping. A processor MUST refuse a second document with
`manifest.yaml.multi_document`, a `%YAML` or `%TAG` directive with
`manifest.yaml.directive`, and a root that is not a mapping with
`manifest.yaml.root_not_mapping`.

<a id="mts-man-005"></a>**MTS-MAN-005** A YAML manifest MUST hold no anchor,
alias, tag or merge key, and a processor MUST refuse each anchor with
`manifest.yaml.anchor`, each alias with `manifest.yaml.alias`, each explicit
tag, `!!str` and the non-specific `!` included, with `manifest.yaml.tag`, and
each plain mapping key `<<` with `manifest.yaml.merge_key`.

<a id="mts-man-006"></a>**MTS-MAN-006** A mapping key MUST be a scalar, taken
as the string its content denotes with no type resolution (`on:` is the key
`"on"`, `012:` the key `"012"`), and keys MUST be unique within their mapping
as those strings. A processor MUST refuse a key that is a mapping or a
sequence with `manifest.yaml.key_not_scalar`, an empty key with
`manifest.yaml.key_empty`, and every repetition of a key in one mapping, so
that `a:` and `"a":` collide, with `manifest.yaml.duplicate_key`.

<a id="mts-man-007"></a>**MTS-MAN-007** A processor MUST resolve a quoted or
block scalar to a string, and a plain scalar by this table and no other, in
order: empty or `null` is null; `true` and `false` are booleans; text matching
`-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][-+]?[0-9]+)?` is a number; `~`, `Null`,
`NULL`, `True`, `TRUE`, `False` and `FALSE`, and every other YAML 1.2.2 Core
Schema integer or float form — a leading `+`, a leading zero, `0o…`, `0x…`,
`.5`, `1.`, and `.inf` or `.nan` in any case or sign — are refused with
`manifest.yaml.scalar_refused`; and anything else is a string. So `no`,
`2026-10-18`, `12:30` and `1.0.0` are strings, and `1.0` is the number 1.

<a id="mts-man-008"></a>**MTS-MAN-008** A processor MUST refuse a YAML
manifest that is not well-formed YAML with one `manifest.yaml.syntax` finding,
at the line where reading stopped, and MUST report every violation of
[MTS-MAN-003](#mts-man-003) to [MTS-MAN-007](#mts-man-007) in a well-formed
document at once, each with its line and column.

<a id="mts-man-024"></a>**MTS-MAN-024** A processor MUST judge
[MTS-MAN-005](#mts-man-005) on the document as it is written, and MUST NOT
expand an alias, apply a merge key, or give a tag a meaning: so a document
whose aliases would expand to an exponential size costs no more to refuse
than to read once, and no tag constructs an object, reads a file or runs
anything.

## 3.3 The JSON profile

<a id="mts-man-009"></a>**MTS-MAN-009** A JSON manifest MUST be one
[RFC 8259](A-references.md#rfc8259) object, with whitespace only around it
and no comment, trailing comma, `NaN` or `Infinity`, restricted to I-JSON. A
processor MUST refuse text that is not with `manifest.json.syntax`, text after
the object with `manifest.json.trailing_content`, invalid UTF-8 with
`manifest.json.encoding`, every repeated member name of one object, compared
after unescaping, with `manifest.json.duplicate_name`, a string holding an
escaped surrogate with no partner with `manifest.json.lone_surrogate`, and a
string holding a noncharacter with `manifest.json.noncharacter`. Every
finding but a syntax or encoding one is reported at once.

## 3.4 The data model and the manifest's members

The manifest's data model is an I-JSON value: objects with unique string
member names, arrays, strings of Unicode scalar values, numbers, `true`,
`false` and `null`. Every rule from here on is stated against the data model,
so it reads the same for YAML and JSON sources.

<a id="mts-man-010"></a>**MTS-MAN-010** A processor MUST convert a manifest
number to the nearest IEEE 754 ([IEEE 754](A-references.md#ieee754)) binary64, and MUST refuse a number that
overflows to an infinity, and one whose value is an integer of magnitude above
2⁵³ − 1, with `manifest.number_out_of_range`.

The root is an object with these members, in no required order:

| Member | Required | Value |
|---|---|---|
| `metais` | yes | the specification version, a string ([chapter 4](04-versions.md)) |
| `name` | yes | the package's name (§3.4.1) |
| `version` | yes | the package's version ([chapter 4](04-versions.md)) |
| `title` | yes | a line of 1 to 200 characters |
| `description` | yes | prose of 1 to 1000 characters, shown verbatim |
| `guide` | yes | prose of 1 to 16,384 characters, CommonMark, for the supplier |
| `authors` | no | an array of 1 to 64 author objects |
| `license` | no | an SPDX license expression of 1 to 512 characters |
| `homepage` | no | an absolute `https` or `http` URI of at most 2048 characters |
| `inputs` | yes | the data specification, possibly empty ([chapter 5](05-data-specification.md)) |
| `input_constraints` | no | the package path of a cross-input schema ([chapter 5](05-data-specification.md)) |
| `outputs` | yes | the output declarations ([chapter 9](09-outputs.md)) |

A *line* is a string with no character of general category Cc that neither
begins nor ends with U+0020. *Prose* is a string whose only Cc characters are
U+000A and U+0009, that neither begins nor ends with U+0020, U+000A or U+0009.
Lengths count Unicode scalar values. An author object has `name`, a line of 1
to 200 characters, and optionally `email`, a line of 3 to 254 characters
holding exactly one `@` with a character on each side and no U+0020, and
`url`, as `homepage`.

> **Note.** In YAML, a `guide` written as a literal block scalar keeps a final
> line break, which prose may not end with; `guide: |-` strips it.

<a id="mts-man-011"></a>**MTS-MAN-011** Every object of the manifest MUST hold
every member this specification requires of it and no member it does not
define — the root, each author, and every object of `inputs` and `outputs` —
and a member this specification defines MUST NOT be `null`. A processor MUST
refuse a missing member with `manifest.required` and an unknown one with
`manifest.additional_property`, and a member of the wrong type, `null`
included, with `manifest.type`.

<a id="mts-man-012"></a>**MTS-MAN-012** `title`, `description`, `guide` and
each author's members MUST meet §3.4's text rules and bounds, and `homepage`
and an author's `url` MUST be absolute URIs ([RFC 3986](A-references.md#rfc3986))
of scheme `https` or `http` in lower case, with a host and no userinfo. A
processor MUST refuse a value shorter or longer than its bounds with
`manifest.min_length` or `manifest.max_length`, and one that breaks a text
rule or is not such a URI with `manifest.pattern`.

<a id="mts-man-013"></a>**MTS-MAN-013** `license` MUST be an SPDX license
expression by the grammar of [SPDX 2.3](A-references.md#spdx) Annex D, with
the operators `AND`, `OR` and `WITH` in upper case, any identifier where a
license or exception identifier stands, and no `DocumentRef-` reference, and a
processor MUST refuse one that is not with `manifest.license.expression`.
Identifiers are not checked against the SPDX License List.

### 3.4.1 The package's name

<a id="mts-man-014"></a>**MTS-MAN-014** `name` MUST be 1 to 128 characters of
one or more segments joined by `.`, each segment 1 to 63 characters matching
`[a-z0-9]+(-[a-z0-9]+)*`, and its first segment MUST NOT be `con`, `prn`,
`aux`, `nul`, `com0` to `com9` or `lpt0` to `lpt9`. A processor MUST refuse a
name that breaks the grammar or a bound with `manifest.pattern` or
`manifest.max_length`, and one whose first segment is a device name with
`manifest.not`.

<a id="mts-man-015"></a>**MTS-MAN-015** *(not observable)* A package published
beyond its author SHOULD be named with a domain its publisher controls,
reversed, as its first segments (`org.example.vpat-acr`).

### 3.4.2 Rules across members

<a id="mts-man-016"></a>**MTS-MAN-016** The inputs' names MUST be unique, and
the outputs' identifiers MUST be unique among the top-level declarations and
among the entries of each container, and a declaration beneath a container
that has `for_each` MUST NOT have `for_each` itself. A processor MUST refuse a
repeated input name with `manifest.input.duplicate_name`, a repeated output
identifier with `manifest.output.duplicate_id`, and a nested `for_each` with
`manifest.output.nested_for_each`.

<a id="mts-man-017"></a>**MTS-MAN-017** Every package path the manifest names
MUST name an entry of the package: an input's `schema` and `meaning` and the
`input_constraints` an entry; an output's `template` an entry beneath
`templates/`; its `asset` an entry, or a directory holding one, beneath
`assets/`, `.` naming `assets/` itself; its `mirror` a directory holding an
entry beneath `templates/`, `.` naming `templates/` itself; and its
`validate` an entry. A processor MUST refuse each that does not with
`manifest.path_not_found`.

<a id="mts-man-018"></a>**MTS-MAN-018** Every output declaration's `path`
MUST parse as a Liquid template of the dialect of
[chapter 7](07-template-environment.md) and
[chapter 8](08-filters-and-tags.md), its `when` as a condition and its
`for_each` as a value expression, and a `path` MUST NOT hold a `render` tag.
A processor MUST refuse each that does not parse with `template.syntax`, an
`include` tag with `template.include`, a filter the dialect does not define
with `template.unknown_filter`, a filter given literal arguments it never
takes with `template.filter_argument`, a division by a literal zero with
`template.division_by_zero`, a number literal beyond what a template number
holds with `template.number_range`, a literal `date` format outside its set
with `template.date_format`, `safe` other than as the last filter of
a `path`'s output expression with `template.safe_position`, and a `render`
in a `path` with `template.tag_context`, each located at the field's manifest
pointer and giving the column within the expression; and it MUST know every
standard filter there, and every filter of
[chapter 8](08-filters-and-tags.md).

## 3.5 The manifest schema

Each specification minor has one manifest schema, a JSON Schema 2020-12
document that accepts exactly the data models this chapter accepts for that
minor; every object it defines is closed. The schema of 1.0 is published as
[schemas/1.0/manifest.schema.json](schemas/1.0/manifest.schema.json), with
the `$id` `https://metais.org/schemas/1.0/manifest.schema.json`, which names
it and is never fetched. Its `inputs` and `outputs` are its `$defs`, and every
property it defines carries a `title` and a `description`.

<a id="mts-man-019"></a>**MTS-MAN-019** A processor MUST validate the
manifest against the manifest schema of the specification version it
declares, and report every failure as a finding of the `manifest.` family of
[chapter 12](12-findings.md)'s keyword codes, whose `schema.entry` is the
schema's `$id`. The rules no schema can express — §3.2, §3.3,
[MTS-MAN-010](#mts-man-010), [MTS-MAN-013](#mts-man-013),
[MTS-MAN-016](#mts-man-016) to [MTS-MAN-018](#mts-man-018) and
[chapter 4](04-versions.md)'s version gate — are judged beside it, each with
its own code.

## 3.6 The order a manifest is processed in

<a id="mts-man-020"></a>**MTS-MAN-020** A processor MUST process a manifest in
this order: locate it (§3.1); bound it by `manifest-size`
([chapter 14](14-security.md)) before reading it; parse it under its profile;
for a packed package, compare its canonical form with
`.compiled/manifest.json` ([MTS-MAN-022](#mts-man-022)); gate on its
specification version ([chapter 4](04-versions.md)); validate it against the
manifest schema and judge the rules beside it in one pass
([MTS-MAN-019](#mts-man-019)); and then judge what depends on its values
([MTS-MAN-017](#mts-man-017), [MTS-MAN-018](#mts-man-018),
[chapter 5](05-data-specification.md)). A step that finds anything ends the
processing with every finding of that step, and no finding of a later step.

## 3.7 The canonical form

<a id="mts-man-021"></a>**MTS-MAN-021** The canonical form of a value of the
data model MUST be its [RFC 8785](A-references.md#rfc8785) serialization:
UTF-8 with no byte order mark, no whitespace between tokens, the members of
every object sorted by their names' UTF-16 code units, array order kept,
strings escaped exactly as RFC 8785 §3.2.2.2 says, numbers as ECMAScript
formats them, and no final line feed. `.compiled/manifest.json` MUST be the
canonical form of the manifest's data model, which holds exactly what the
author wrote: no default is added, and no member the author omitted.

<a id="mts-man-022"></a>**MTS-MAN-022** A processor MUST refuse a packed
package whose `.compiled/manifest.json` is not byte for byte the canonical
form of its `metais.yaml`'s data model with `package.compiled.mismatch`: a
person reading the source manifest and a consumer reading the compiled one
see the same manifest, and a compiled manifest that is not itself canonical is
refused the same way.

<a id="mts-man-023"></a>**MTS-MAN-023** The canonical text of an integral
number whose magnitude is above 2⁵³ − 1 and below 10²¹ is its digits, with no
fraction or exponent, as ECMAScript formats it, and that text is not itself
accepted as input: a schema file that holds it is refused under
[MTS-DAT-002](05-data-specification.md#mts-dat-002) with `schema.syntax`, whose
`rule` is `number_out_of_range`. A processor MUST accept such a number where a
schema file writes it with a fraction or an exponent, as in `"maximum": 11e17`,
and MUST open a package whose compiled file holds its canonical text, because
a compiled file is compared byte for byte and never read back as input. The
canonical text of every other number a processor accepts is accepted as
input.

> **Note.** The canonical form is the one every JSON file under `.compiled/`
> is written in, so each is byte-stable; a consumer that is not a processor of
> this specification needs only a JSON parser to read any of them.
