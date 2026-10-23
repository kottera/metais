# 5. Data specification

**Normative.** Areas: DAT.

The data specification says which data a package needs and what that data
means. This chapter specifies the `inputs` section of the manifest, the JSON
Schema dialect an input's schema is written in and the keywords and formats
it asserts, how a schema's references stay inside the package, the meaning
documents that explain an input in prose, the combined schema a packed
package carries for all of its inputs, and the two keywords that state rules
across the items of an array.

## 5.1 The inputs section

A package's inputs are declared in the manifest's `inputs` member
([chapter 3](03-manifest.md)), an array of input declarations in the order a
supplier meets them; it may be empty. Each declaration is an object with
these members, in no required order:

| Member | Required | Value |
|---|---|---|
| `name` | yes | the input's name: a lower-case letter, then up to 63 lower-case letters, digits and `_`, and none of `blank`, `empty`, `false`, `first`, `last`, `nil`, `null`, `size` and `true`, which Liquid reserves |
| `title` | yes | what the input is, in brief: a line of 1 to 200 characters ([chapter 3](03-manifest.md) §3.4) |
| `description` | no | what the input means, as prose (§3.4) |
| `schema` | yes | the package path of the input's schema file, beneath `inputs/`, ending `.json`, `.yaml` or `.yml` (§5.2) |
| `meaning` | no | the package path of the input's meaning document, beneath `inputs/`, ending `.md` |
| `forms` | no | the forms the input may be supplied in, each at most once, of `json`, `yaml`, `csv`, `text` and `markdown` ([chapter 6](06-supplying-inputs.md)); omitted, `json` and `yaml` |
| `csv` | no | how a CSV file of the input is read: an object whose one member, `delimiter`, is `,`, `;`, `\|` or a tab; allowed only when `forms` holds `csv` |
| `collection` | no | `true` for a collection input, an array supplied as a directory of item files ([chapter 6](06-supplying-inputs.md) §6.8); omitted, `false` |
| `required` | no | `false` for an input a supplier may leave out; omitted, `true` |

The manifest schema ([chapter 3](03-manifest.md) §3.5) holds a declaration
to this table, and the inputs' names are unique
([MTS-MAN-016](03-manifest.md#mts-man-016)). A name is how a template reads
the input (`inputs.<name>`, [chapter 7](07-template-environment.md)), how a
data directory's file supplies it (§6.9), and the first token of a path set
(§6.10).

An input's schema says what shape its data has; its meaning document says
what the data is for. A meaning document is CommonMark
([CommonMark](A-references.md#commonmark)) prose for the supplier: what the
input describes, and what good data looks like. A processor does not
interpret it. The package's guide says how the inputs relate and in which
order to gather them.

## 5.2 Input schemas at open

A package's inputs are declared in the manifest's `inputs` member, an ordered
list of named slots, each naming the package path of its schema under
`inputs/`. This section states what a processor checks of the schemas when it
opens a package; §5.3 states how supplied inputs are validated against them,
and [chapter 6](06-supplying-inputs.md) how inputs are supplied.

<a id="mts-dat-001"></a>**MTS-DAT-001** Every `.json`, `.yaml` and `.yml` file
under `inputs/` is a schema file, whether or not an input names it, and a
processor MUST compile every one when it opens a package, refusing the package
for any finding about any of them.

<a id="mts-dat-002"></a>**MTS-DAT-002** A schema file MUST be read by the rules
of an input file of its form, JSON for `.json` and YAML for `.yaml` and
`.yml`, after a single leading byte order mark is removed, and a processor MUST
refuse one that breaks a rule of its form with `schema.syntax`, naming the
form and the rule.

<a id="mts-dat-003"></a>**MTS-DAT-003** A schema's dialect is JSON Schema
2020-12 ([JSON Schema](A-references.md#jsonschema)): `$schema`, where a file's
root or an embedded resource has it, MUST be
`https://json-schema.org/draft/2020-12/schema`, and a schema MUST NOT use
`$vocabulary`. A processor MUST refuse another `$schema` and every
`$vocabulary` with `schema.dialect`, and a schema that is not valid under the
2020-12 meta-schemas with `schema.invalid`.

<a id="mts-dat-004"></a>**MTS-DAT-004** A schema MUST use no keyword the
2020-12 core, applicator, unevaluated, validation, meta-data, content and
format vocabularies do not define, but `uniqueKeys` and `coverage`, which
this specification defines (§5.4), and one beginning `x-`, which is the
author's own annotation, and a processor MUST refuse every other with
`schema.keyword_unknown`: a misspelt `requried` is a rule the author believes
is enforced and is not.

<a id="mts-dat-005"></a>**MTS-DAT-005** A schema's `format` MUST be one of
`date-time`, `date`, `time`, `duration`, `email`, `hostname`, `ipv4`, `ipv6`,
`uri`, `uri-reference`, `uuid`, `uri-template`, `json-pointer` and
`relative-json-pointer`, each of which a processor asserts, and a processor
MUST refuse any other with `schema.format_unsupported`.

<a id="mts-dat-006"></a>**MTS-DAT-006** Every `pattern` and
`patternProperties` key MUST be a regular expression of the package
regular-expression profile of [chapter 14](14-security.md), and a processor
MUST refuse one that is not, such as one using lookaround or a backreference,
with `schema.pattern_unsupported`.

<a id="mts-dat-007"></a>**MTS-DAT-007** Each schema file's retrieval URI is
`metais:///` followed by its package path, and a `$ref` or `$dynamicRef` MUST
resolve to a resource of the package by its canonical URI or an embedded
`$id`, an anchor or a JSON Pointer within one, or a 2020-12 meta-schema, and
nothing is ever fetched. A processor MUST refuse a reference that resolves to
nothing of these with `schema.ref_unresolved`, and one whose path climbs above
the package root or resolves outside `inputs/` with `schema.ref_escapes`.

<a id="mts-dat-008"></a>**MTS-DAT-008** A schema file whose root declares an
`$id` MUST be referred to only by that `$id`, and a processor MUST refuse a
reference to its retrieval URI with `schema.ref_by_path`; and two resources
MUST NOT have one canonical URI, which a processor refuses with
`schema.id_duplicate`.

### 5.2.1 The combined schema

<a id="mts-dat-009"></a>**MTS-DAT-009** `.compiled/schema.json` MUST be the
canonical JSON ([chapter 3](03-manifest.md)) of one object schema with a
property per input: `$schema` the 2020-12 dialect; `$id`
`metais:///.compiled/schema.json`; `type` `object`; `properties` holding, for
each input, its `title`, its `description` when it has one, and a `$ref` to
its schema's canonical URI; `required` listing the inputs whose `required` is
not `false`, in manifest order; `additionalProperties` `false`; `allOf`
holding one `$ref` to the `input_constraints` schema's canonical URI, only when
the manifest declares one; and `$defs` embedding every schema file under
`inputs/`, keyed by its package path, each with `$id` set to its canonical URI
and `$schema` set, a YAML file as the JSON of its value and a boolean schema
as `{}` or `{"not": {}}`. No reference inside an embedded resource is
rewritten.

## 5.3 Validating supplied inputs

The inputs a supplier gives are assembled into one inputs object, a member per
input that has a value ([chapter 6](06-supplying-inputs.md)), and that object
is validated against the combined schema of §5.2.1, which names every input,
says which are required, and holds the input constraints. A failure is
reported by the finding [chapter 12](12-findings.md) maps its keyword to.

<a id="mts-dat-010"></a>**MTS-DAT-010** A processor MUST validate the inputs
object against the combined schema, reporting a required input that nothing
supplied with `input.missing` and every failure of an input's value against
its schema by its keyword's finding, and MUST report every failure the
validation meets, not only the first.

<a id="mts-dat-011"></a>**MTS-DAT-011** A processor MUST assert each `format`
of §5.2 on the strings of the inputs, by the definition JSON Schema's
validation §7.3 cites for it, and MUST report a string that is not of its
format with `input.format`; a value that is not a string passes. Where those
definitions leave a choice: `date-time`'s `T` and `Z`, and `time`'s `Z`, may be
lower case; a leap second, second 60, is valid only where the time, moved to
UTC by its offset, is 23:59:60; an `email` is [RFC 5321](A-references.md#rfc5321)'s `Mailbox`, a quoted
local part and an IPv4 or `IPv6:` address literal included; a `hostname` has no
trailing dot; an `ipv4` octet has no leading zero; an `ipv6` address has no
zone; a `uuid` is the 8-4-4-4-12 form of hexadecimal digits of either case; and
a `duration` is RFC 3339 Appendix A's, its units in order with none skipped.

<a id="mts-dat-013"></a>**MTS-DAT-013** A processor MUST report an array that
`uniqueItems` requires to hold distinct items, and that holds two equal ones,
with one `input.unique_items` at the first item, in the order of their
indexes, that equals an earlier one, naming its index and, as
`duplicate_of`, the index of the first item it equals. Two values are equal
when they are the same value of the data model, a number by its mathematical
value, so `1.0` equals `1`.

<a id="mts-dat-014"></a>**MTS-DAT-014** A processor MUST decide `multipleOf`
exactly: a number is a multiple of the divisor when their quotient, computed
from their decimal values without rounding, is an integer, however large or
small either is, and MUST report one that is not with `input.multiple_of`.

<a id="mts-dat-012"></a>**MTS-DAT-012** A processor MUST evaluate the input
constraints over the whole inputs object, reporting a failure inside one
input's value about that input, with the conditional that selected it, a
required input the constraints require with `input.missing`, and a failure of a
rule over the inputs object as a whole, which names no one input, with
`input.constraints_failed` about the invocation, naming the keyword.

## 5.4 Rules across the items of an array

JSON Schema judges each item of an array alone, and `uniqueItems` compares
whole items. A table whose rows each describe one thing needs two rules more:
that no two rows describe the same thing, and that every thing the package
requires has a row. `uniqueKeys` and `coverage` state them. Each is an
assertion keyword of this dialect, written in a schema object beside
`uniqueItems`, and each judges an array.

A *key pointer* is a JSON Pointer ([RFC 6901](A-references.md#rfc6901)) other
than the empty one, evaluated against one item of an array. An item's value
at a key pointer is the value the pointer resolves to within the item, and
an item where it resolves to nothing has no value there. Values are compared
as [MTS-DAT-013](#mts-dat-013) compares them, so `1.0` equals `1`, and `null`
is a value, equal to `null`.

```json
{
  "type": "array",
  "uniqueKeys": ["/criterion"],
  "coverage": { "key": "/criterion", "values": ["1.1.1", "1.2.1", "1.3.1"] },
  "items": {
    "type": "object",
    "required": ["criterion", "conformance"],
    "properties": {
      "criterion": { "type": "string" },
      "conformance": { "type": "string" }
    }
  }
}
```

This array refuses a second row whose `criterion` is `1.1.1`, and reports
each of `1.1.1`, `1.2.1` and `1.3.1` that no row has. Its rows may hold other
criteria too: `coverage` names what must be there, not all that may be.

<a id="mts-dat-015"></a>**MTS-DAT-015** `uniqueKeys` MUST be an array of 1 to
8 key pointers, no two of them equal, and a processor MUST refuse one that is
not with `schema.invalid`, naming the keyword.

<a id="mts-dat-016"></a>**MTS-DAT-016** `coverage` MUST be an object with
exactly two members, `key`, a key pointer, and `values`, an array of one or
more values no two of which are equal, and a processor MUST refuse one that
is not with `schema.invalid`, naming the keyword.

<a id="mts-dat-017"></a>**MTS-DAT-017** A processor MUST evaluate
`uniqueKeys` and `coverage` wherever the schema object holding them applies
to an array, through `$ref`, `allOf`, a conditional and every other
applicator, as it evaluates `uniqueItems` there, and MUST NOT evaluate either
against an instance that is not an array.

<a id="mts-dat-018"></a>**MTS-DAT-018** An item's key under `uniqueKeys` is
the list of its values at the keyword's pointers, in their order, and an item
with no value at any one of them has no key. A processor MUST report each
item whose key equals an earlier item's key, pointer by pointer, with one
`input.unique_keys` naming its index and, as `duplicate_of`, the index of the
first item with that key, located at its value at the pointer when the
keyword has one pointer and at the item when it has several, with the first
item's same place in `related`; an item with no key is never reported.

<a id="mts-dat-019"></a>**MTS-DAT-019** A processor MUST report each member of
`coverage`'s `values` that no item's value at `key` equals with one
`input.coverage` located at the array, whose `missing` parameter is that
member; an item with no value at `key` covers nothing.

A failure of either keyword in an output's value is reported by the `output.`
family, as every keyword's is ([chapter 12](12-findings.md) §12.3). A
duplicate key names two indexes and no value, so a finding never repeats what
a supplier wrote ([MTS-FND-011](12-findings.md#mts-fnd-011)); the value
`input.coverage` names is the package's own, written in its schema.

> **Note.** A validator that implements JSON Schema 2020-12 and not this
> specification treats `uniqueKeys` and `coverage` as unknown keywords, which
> 2020-12 lets it collect as annotations, and so accepts a table this
> specification refuses. The keywords have plain names because no minor of
> this specification ever gives meaning to a keyword beginning `x-`
> ([chapter 4](04-versions.md) §4.1.2). The combined schema carries them as
> the author wrote them, so a supplier that reads it reads the rules.
