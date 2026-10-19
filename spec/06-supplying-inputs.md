# 6. Supplying inputs

**Normative.** Areas: SUP.

A supplier provides each input in a form the manifest accepts for it. This
chapter specifies those forms and how each is parsed, how CSV values are
coerced to the types the schema declares, collection inputs, the data model
templates see, how inputs are supplied from a data directory and as single
values, the precedence between them, how defaults are applied, and what an
input that is absent is.

The data model is JSON's: null, boolean, number, string, array and object.
Objects are unordered and their member names unique, so a value supplied as
JSON, as YAML and as CSV is the same value. A value keeps where it came from:
the file, the line and the column, or the command-line argument, which is
what a finding about it names ([chapter 12](12-findings.md)).

## 6.1 Input files

<a id="mts-sup-001"></a>**MTS-SUP-001** An input file MUST be UTF-8, and a
processor MUST remove a single leading byte order mark (`EF BB BF`) before it
parses a file of any form; it MUST refuse a file that is not UTF-8, a UTF-16
file with or without its byte order mark included, with
`input.encoding_invalid`, naming the line and column of the first byte that
is not.

<a id="mts-sup-002"></a>**MTS-SUP-002** Before it parses a `csv`, `text` or
`markdown` file, a processor MUST normalize each line break, CRLF, CR or LF,
to LF, inside a quoted CSV field too, so that a value never depends on the
line-ending convention of the machine that wrote the file.

## 6.2 JSON

<a id="mts-sup-003"></a>**MTS-SUP-003** A JSON file MUST be one RFC 8259 JSON
text ([RFC 8259](A-references.md#rfc8259)), with any value at the top level and no
member name repeated in one object, names compared after their escapes are
decoded; a processor MUST refuse text that is not one with `json.syntax`,
content after the value with `json.trailing_content`, and a repeated name with
`json.duplicate_name`.

<a id="mts-sup-004"></a>**MTS-SUP-004** A string of an input MUST be a
sequence of Unicode scalar values holding no noncharacter, in every form: a
processor MUST refuse an escaped surrogate with no partner with
`json.lone_surrogate`, an escaped noncharacter with `json.noncharacter`, and a
noncharacter written as itself in a file of any form with
`input.noncharacter`.

<a id="mts-sup-005"></a>**MTS-SUP-005** An input has one number type, IEEE 754
binary64, each literal rounded to the nearest value, ties to even, so that
`1` and `1.0` are one value, equal under `const` and `enum` and valid against
`type: integer`; a processor MUST refuse, with `input.number_out_of_range`, a
literal written without fraction or exponent whose magnitude exceeds 2^53−1,
which would otherwise change silently, and a literal that overflows binary64.

## 6.3 YAML

<a id="mts-sup-006"></a>**MTS-SUP-006** A YAML input MUST be read by the YAML
profile of the manifest ([chapter 3](03-manifest.md)), after a single leading
byte order mark is removed: one document, with no directive, anchor, alias,
merge key or tag, mapping keys that are non-empty scalars, no key repeated,
and plain scalars resolved by the profile's table. A processor MUST refuse each
break of the profile with its code: `yaml.multi_document`, `yaml.directive`,
`yaml.anchor`, `yaml.alias`, `yaml.merge_key`, `yaml.tag`,
`yaml.key_not_scalar`, `yaml.key_empty`, `yaml.duplicate_key`,
`yaml.scalar_refused` for a plain scalar the table refuses, such as `~`,
`.inf`, `0x1F` or `007`, `yaml.character` for a character YAML forbids,
`yaml.bom` for a second byte order mark, and `yaml.syntax` for anything else
that is not YAML.

<a id="mts-sup-007"></a>**MTS-SUP-007** A processor MUST resolve a plain YAML
scalar only to `null`, `true`, `false`, a number of the RFC 8259 grammar, or
else a string, so that a plain `2026-10-18`, `yes` and `on` are strings; MUST
read a key with nothing after it as null; and MUST read an empty file, or one
holding only comments, as null.

## 6.4 CSV

A CSV input is a header record and zero or more data records, its value an
array of objects, one per record, whose members the header names.

<a id="mts-sup-008"></a>**MTS-SUP-008** A processor MUST read a CSV file by RFC
4180 ([RFC 4180](A-references.md#rfc4180)) with these rules: records end in
LF, the last optionally; a field beginning with `"` is quoted, `""` inside it
is a quote, and the delimiter and LF inside it are data; spaces are data; and
an empty line is no record wherever it occurs. It MUST refuse a file with no
header record with `csv.header_missing`, an empty header name with
`csv.header_empty`, a header name given twice, compared exactly, with
`csv.header_duplicate`, a `"` inside an unquoted field with `csv.bare_quote`, a
character after a closing quote other than the delimiter or LF with
`csv.after_quote`, a quoted field that never closes with
`csv.unterminated_quote`, and a record whose field count differs from the
header's with `csv.field_count`, naming the record's line.

<a id="mts-sup-009"></a>**MTS-SUP-009** A processor MUST separate fields by the
delimiter the input's `csv` member declares, one of `,`, `;`, `|` and the tab,
and by `,` when the input declares none, whatever the file's extension.

<a id="mts-sup-010"></a>**MTS-SUP-010** A processor MUST make each data record
an object whose members the header names, MUST leave the member of an unquoted
empty field absent, MUST give the member of a quoted empty field `""` the empty
string, and MUST coerce every other field's text by §6.7 against the declared
type set of the record schema's property, so that a CSV file supplies no null,
array or object.

## 6.5 Plain text

<a id="mts-sup-011"></a>**MTS-SUP-011** A text input's value MUST be the whole
file as one string, after its line breaks are normalized, with one final LF
removed when there is one; an empty file is the empty string. Where an input
does not accept `markdown`, a `.md` or `.markdown` file is text.

## 6.6 Markdown with frontmatter

<a id="mts-sup-012"></a>**MTS-SUP-012** A Markdown input's value MUST be an
object holding the frontmatter's fields and `body`. Frontmatter is present when
the file's first line is exactly `---`, and ends at the next line that is
exactly `---`; between them is YAML of §6.3, which MUST be a mapping or empty.
`body` is the text after the closing line, or the whole file without
frontmatter, its line breaks normalized and one final LF removed, a blank line
after the frontmatter kept. A processor MUST refuse frontmatter with no closing
line with `markdown.frontmatter_unclosed`, frontmatter that is not a mapping
with `markdown.frontmatter_not_mapping`, and a frontmatter field named `body`
with `markdown.frontmatter_body_field`.

## 6.7 Coercion

The static subschema at a path is found by walking from a schema through
`$ref` to its target, `properties` for a member name, `prefixItems` for an
index it covers, and `items` for any other index, and through nothing else:
the walk never evaluates `allOf`, `anyOf`, `oneOf` or a conditional, so its
answer does not depend on the data. Its declared type set is the `type` of the
first schema of the chain from that place through `$ref` targets that declares
one; with no `type`, the set is empty.

<a id="mts-sup-013"></a>**MTS-SUP-013** A processor MUST coerce a CSV field and
the text of a text path set against the declared type set of the static
subschema at its place, and against nothing else.

<a id="mts-sup-014"></a>**MTS-SUP-014** Coercion MUST give, of a text against a
declared type set: the boolean, when the set holds `boolean` and the text is
`true`, `True`, `TRUE`, `false`, `False` or `FALSE`; otherwise the number, when
the set holds `integer` and the text matches `-?(0|[1-9][0-9]*)`; otherwise the
number, when the set holds `number` and the text is an RFC 8259 number;
otherwise the text, as a string. Coercion never fails: a text that fits no
type stays a string, so that ` 42`, `1,5` and `yes` against `integer`,
`number` and `boolean` are each reported by validation, as `input.type`.

## 6.8 Collections

A collection input is an array of items. Supplied from a data directory it is
a directory named for the input, each regular file in it one item, whose form
comes from its last extension: `.json` is `json`, `.yaml` and `.yml` are
`yaml`, `.csv` is `csv`, `.txt` is `text`, and `.md` and `.markdown` are
`markdown` when the input accepts it and otherwise `text`.

<a id="mts-sup-015"></a>**MTS-SUP-015** A processor MUST read a collection
supplied from a data directory as the items of its directory, ignoring each
entry whose name begins with `.`; MUST refuse a file supplying a collection with
`collection.as_file`, a directory supplying it whose name has an extension
with `collection.directory_name`, a subdirectory with
`collection.subdirectory`, and a file
with no extension, or whose extension maps to no form the input accepts, with
`input.extension_not_accepted` listing the extensions it accepts; and MUST read
an empty directory as an empty array.

<a id="mts-sup-016"></a>**MTS-SUP-016** An item's name is its file name in
Unicode NFC and its stem that name without its last extension, and a
processor MUST refuse two items whose stems are equal, or equal but for letter
case, with `collection.stem_duplicate`.

<a id="mts-sup-017"></a>**MTS-SUP-017** A processor MUST order the items of a
collection by their names, comparing the UTF-8 bytes of the NFC names, with no
case folding and no locale.

## 6.9 The data directory

A data directory holds one entry per input: `<name>.<ext>` for an input, and
`<name>/` for a collection. An example's `inputs/` directory has this format.

<a id="mts-sup-018"></a>**MTS-SUP-018** A processor MUST take a data
directory's entry as supplying the input named by the text before its first
`.`, in the form its extension maps to by §6.8, and MUST ignore each entry whose
name begins with `.`. It MUST refuse an entry that names no declared input with
`input.file_unknown`, a second entry for one input with
`input.file_duplicate`, an extension whose form the input does not accept,
`.schema.json` included, with `input.extension_not_accepted`, and a directory
named as an input's file with `input.not_regular_file`.

## 6.10 The command line

Inputs are supplied on the command line whole, by `--inputs-json`, or one
value at a time, by a path set: `--set` and `--set-json`. A path set's
argument is an inputs pointer, a JSON Pointer ([RFC 6901](A-references.md#rfc6901))
whose first token is an input's name, then `=` and the value; the argument is
split at its first `=`.

<a id="mts-sup-019"></a>**MTS-SUP-019** A value supplied on the command line
has no file form: a processor MUST NOT apply an input's `forms` to it, and the
line and column of a finding about it are within its argument.

<a id="mts-sup-020"></a>**MTS-SUP-020** An `--inputs-json` argument MUST be one
JSON object by §6.2 whose members are declared inputs, each giving its input's
whole value. A processor MUST refuse an argument that is not an object with
`input.json_arg_not_object`, a member that names no declared input with
`input.json_arg_unknown`, and an argument that does not parse with the finding
of §6.2 about the member's input when the fault lies inside a member, or
otherwise with `input.json_arg_invalid` about the invocation.

<a id="mts-sup-021"></a>**MTS-SUP-021** A processor MUST refuse, with
`input.set_pointer_invalid`, a path set with no `=`, one whose pointer does
not begin with `/` or has `~` other than in `~0` and `~1`, and one whose first
token names no declared input. A text set's value MUST be coerced by §6.7
against the static subschema at its pointer, an empty text the empty string; a
JSON set's value MUST be one JSON text by §6.2, placed as it is.

<a id="mts-sup-022"></a>**MTS-SUP-022** A processor MUST apply a path set to
the inputs object assembled so far by walking its tokens: each but the last
reaches an existing object member or array element, a missing object member
being created as an empty object; an array index is an existing index written
without a leading zero; `-` is only the last token, and appends; and the last
token's member or element is created or replaced. It MUST refuse a set that
traverses into a string, number, boolean or null, names an index out of
range or written with a leading zero, or has `-` before its last token, with
`input.set_traversal`, and MUST then not apply it. No array is created but by
a JSON set.

## 6.11 Precedence

<a id="mts-sup-023"></a>**MTS-SUP-023** A processor MUST assemble the supplied
inputs object in this order, each step over the result of the one before: each
data-directory entry gives its input's whole value; each `--inputs-json`
member, in command-line order, replaces its input's whole value, never merged
with it; and each path set, text or JSON, in command-line order, replaces the
value at its pointer alone. Every `--inputs-json` applies before every path
set, wherever they stand on the command line.

<a id="mts-sup-024"></a>**MTS-SUP-024** A processor MUST refuse a path set into
an input whose file or argument did not parse with `input.set_into_unparsed`,
and MUST NOT apply it; a file that does not parse is refused even when an
`--inputs-json` member replaces its input.

## 6.12 Defaults

<a id="mts-sup-025"></a>**MTS-SUP-025** After assembling the inputs object and
before validating it, a processor MUST apply defaults to every input, whatever
its source: an absent optional input whose root schema has `default` takes a
copy of it; at an object whose static schema has `properties`, each absent
member with a `default` in its subschema, or in the first `$ref` target in its
chain that has one, takes a copy of it, unless the `required` of the schema
object that holds those `properties` names it; the schema object's own
`properties` are applied before those reached through its `$ref`; each element
of an array is walked with its static subschema, so that a blank CSV field
takes its column's default; and the walk descends into every member and
element, those a default supplied included.

<a id="mts-sup-026"></a>**MTS-SUP-026** A default MUST NOT create a parent
object, replace a present value, `null` included, or be applied from under any
keyword but `properties` and a root `default`: one under `allOf`, `anyOf`,
`oneOf`, a conditional, `dependentSchemas`, `patternProperties` or
`additionalProperties` is an annotation.

## 6.13 Absence and null

<a id="mts-sup-027"></a>**MTS-SUP-027** An input or member is absent when it was
not supplied and no default applied, an unquoted empty CSV field included, and
a processor MUST validate it as missing; `null`, from JSON, from YAML or an
empty YAML value, or from a JSON set, is a supplied value, which a processor
MUST validate against its schema. A processor fills nothing with `""`, `[]` or
`{}` of its own.

## 6.14 One pass

<a id="mts-sup-028"></a>**MTS-SUP-028** A processor MUST read every
data-directory entry a declared input could be and every supply argument,
reporting every finding of each, before it assembles the inputs object; MUST
leave an input whose file or argument did not parse out of the instance,
reporting it neither missing nor `required`; and MUST report every finding of
the supply, its assembly and its validation together, in one report.

<a id="mts-sup-029"></a>**MTS-SUP-029** A processor MUST validate a collection
some of whose items did not parse, and a CSV input some of whose records were
refused alone for their field count or a quote, without those items, reporting
`input.validation.partial` with how many were left out and how many findings
about the array as a whole were therefore not reported; and when any input did
not parse, was validated in part, or an `--inputs-json` argument did not
parse, it MUST NOT evaluate the input constraints, and MUST report
`input.constraints_not_evaluated`.
