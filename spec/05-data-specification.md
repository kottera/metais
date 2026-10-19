# 5. Data specification

**Normative.** Areas: DAT.

The data specification says which data a package needs and what that data
means. This chapter specifies the `inputs` section of the manifest, the JSON
Schema dialect an input's schema is written in and the keywords and formats
it asserts, how a schema's references stay inside the package, the meaning
documents that explain an input in prose, and the combined schema a packed
package carries for all of its inputs.

## 5.1 Input schemas at open

A package's inputs are declared in the manifest's `inputs` member, an ordered
list of named slots, each naming the package path of its schema under
`inputs/`. This section states what a processor checks of the schemas when it
opens a package; §5.2 states how supplied inputs are validated against them,
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
format vocabularies do not define, but one beginning `x-`, which is the
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

### 5.1.1 The combined schema

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

## 5.2 Validating supplied inputs

The inputs a supplier gives are assembled into one inputs object, a member per
input that has a value ([chapter 6](06-supplying-inputs.md)), and that object
is validated against the combined schema of §5.1.1, which names every input,
says which are required, and holds the input constraints. A failure is
reported by the finding [chapter 12](12-findings.md) maps its keyword to.

<a id="mts-dat-010"></a>**MTS-DAT-010** A processor MUST validate the inputs
object against the combined schema, reporting a required input that nothing
supplied with `input.missing` and every failure of an input's value against
its schema by its keyword's finding, and MUST report every failure the
validation meets, not only the first.

<a id="mts-dat-011"></a>**MTS-DAT-011** A processor MUST assert each `format`
of §5.1 on the strings of the inputs, by the definition JSON Schema's
validation §7.3 cites for it, and MUST report a string that is not of its
format with `input.format`; a value that is not a string passes. Where those
definitions leave a choice: `date-time`'s `T` and `Z`, and `time`'s `Z`, may be
lower case; a leap second, second 60, is valid only where the time, moved to
UTC by its offset, is 23:59:60; an `email` is RFC 5321's `Mailbox`, a quoted
local part and an IPv4 or `IPv6:` address literal included; a `hostname` has no
trailing dot; an `ipv4` octet has no leading zero; an `ipv6` address has no
zone; a `uuid` is the 8-4-4-4-12 form of hexadecimal digits of either case; and
a `duration` is RFC 3339 Appendix A's, its units in order with none skipped.

<a id="mts-dat-012"></a>**MTS-DAT-012** A processor MUST evaluate the input
constraints over the whole inputs object, reporting a failure inside one
input's value about that input, with the conditional that selected it, a
required input the constraints require with `input.missing`, and a failure of a
rule over the inputs object as a whole, which names no one input, with
`input.constraints_failed` about the invocation, naming the keyword.
