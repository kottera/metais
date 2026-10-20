# 14. Security

**Normative.** Areas: SEC.

A package and its inputs are untrusted, and nothing of the host on which they
are rendered reaches them. This chapter describes the threat model, and
specifies the resource limits and their defaults, the confinement of every
entry path and output path, and the isolation of a render from its host.

## 14.1 The threat model (informative)

What this chapter protects is the host a package is rendered on: its CPU,
memory and disk, and the program that called the processor; its secrets,
the environment's variables, the files the calling user can read and the
facts about the machine — its user, its host name, its paths, its time zone
and its processes; and its file system, of which nothing is written,
replaced or removed outside the output the caller named. It protects the
caller's outputs too: a render writes exactly what the package and the
inputs determine, or nothing.

Two parties are hostile. A package's author controls every byte of the
package — its container, its entry names, its manifest, schemas, templates,
assets, references and examples — and of a source directory handed to
someone else to check, test or pack. An input's supplier controls the data
directory and the values given with the operation: file contents, names,
types and sizes. A supplier is often less privileged than the caller, an
agent in a sandbox or the author of a downloaded archive, so a supplier who
plants a link to a file only the caller can read, hoping a render copies it
into an output the supplier sees, is within the model.

The caller is trusted. It names the package, the inputs, the output and the
limits, sets the render time and asks for files to be replaced, and its
choices are honoured. Data crosses into a template only as the validated
inputs, the manifest's own members and the render time; nothing else of the
host reaches a template, an output or a finding (§7.7.1, §14.5,
[MTS-RND-005](11-rendering.md#mts-rnd-005)).

These are outside the model: a malicious caller, who can run any program it
likes; the operating system, the file system, the hardware and other
processes racing a processor on the directories it uses; whether what a
render writes is safe to run, open or publish, which is the caller's
decision, with the one exception of a path that runs code merely by
existing, `.git` ([chapter 2](02-container.md)); the authenticity of a
package, since a digest detects corruption and an author who changes an
entry can change its digest; side channels, as the time a render takes and
the size of what it writes; and resource use up to the limits, which a
render within every limit may spend.

## 14.2 Package limits

Every size, count and ratio a processor reads from a package is bounded, and
every bound has a name and a default. A package within every package limit is
portable: every conforming processor accepts it at the defaults. A caller may
lower a limit for one operation; nothing raises one.

| Limit | Default | Bounds |
|---|---|---|
| `package-size` | 256 MiB | the packed package file |
| `package-uncompressed` | 512 MiB | the sum of every entry's uncompressed size, declared and actual |
| `entries` | 10,000 | the entries of a package |
| `entry-size` | 64 MiB | one entry, uncompressed |
| `entry-ratio` | 200 | an entry's uncompressed size divided by its compressed size, for an entry larger than 1 MiB uncompressed |
| `path-length` | 512 bytes | an entry name, and an output path |
| `path-depth` | 32 segments | an entry name, and an output path |
| `manifest-size` | 1 MiB | `metais.yaml`, and `.compiled/manifest.json` |
| `schema-size` | 4 MiB | one schema file |
| `combined-schema-size` | 16 MiB | the combined schema, stored or derived |
| `schema-depth` | 64 | the nesting of subschemas |
| `schema-subschemas` | 20,000 | the subschemas of every schema file of a package together, and of the combined schema |
| `schema-location-length` | 512 bytes | the location of one subschema of a schema file |
| `ref-depth` | 32 | the references one chain of subschemas follows at one instance location |
| `pattern-length` | 4,096 bytes | one `pattern` or `patternProperties` key |
| `regex-program` | 4,096 instructions | one compiled regular expression |
| `template-size` | 8 MiB | one template, and one `path` of the manifest |
| `template-nesting` | 100 | the block tags open at once in one template or `path` |
| `value-depth` | 64 | the nesting of arrays and objects in the manifest and in a schema file |
| `number-length` | 1,024 bytes | one number literal |

KiB, MiB and GiB are powers of 1024.

<a id="mts-sec-001"></a>**MTS-SEC-001** A processor MUST enforce every limit of
§14.2 at its default, or at the lower value a caller gives for one operation,
and MUST refuse a package whose own content breaches one with `limit.package`,
naming the limit and its bound, before it reads the bytes the limit bounds.

A schema's cost to compile grows with how many subschemas it has and how long
their locations are, which its size in bytes does not measure, so two limits
bound them. A subschema is the root of a schema file and every value a
keyword of [chapter 5](05-data-specification.md) gives as a schema, as an
array of schemas or as an object of them, a boolean schema included.

<a id="mts-sec-020"></a>**MTS-SEC-020** A processor MUST count the subschemas
of every schema file of a package together, file by file in the byte order of
their names and within a file root first, then each object's members in the
byte order of their names, and MUST refuse a package whose count exceeds
`schema-subschemas` with `limit.package`, located at the file and JSON Pointer
of the first subschema past the limit, before it compiles any schema.

<a id="mts-sec-021"></a>**MTS-SEC-021** A subschema's location is the URI of
its resource, which is the longer of the canonical URI of the nearest `$id`
at or above it and its file's retrieval URI, then `#`, then its JSON Pointer
within its file as RFC 6901 escapes it. A processor MUST refuse a schema file
holding a subschema whose location is longer, in bytes, than
`schema-location-length` with `limit.package`, located at the file and JSON
Pointer of the first such subschema in the order of MTS-SEC-020, before it
compiles any schema.

<a id="mts-sec-022"></a>**MTS-SEC-022** A processor MUST count the subschemas
of the combined schema ([chapter 5](05-data-specification.md)) as a whole,
apart from the schema files', and MUST refuse a package whose combined
schema, stored or derived, has more than `schema-subschemas` with
`limit.package`, located at `.compiled/schema.json` and the JSON Pointer of
its first subschema past the limit, before it compiles the combined schema.
A location in the combined schema is not bounded by `schema-location-length`.

<a id="mts-sec-058"></a>**MTS-SEC-058** A limit's bound is the most an
operation may use: a processor MUST accept a size, a count or a depth equal
to the bound of the limit that bounds it, at the default or at a lower value
a caller gives, and only one greater than the bound breaches the limit.

<a id="mts-sec-043"></a>**MTS-SEC-043** A processor MUST refuse a schema file
longer, in bytes, than `schema-size` with `limit.package` located at the file,
observing its length, before it reads the file; and a package whose combined
schema, stored or derived, is longer in its canonical JSON than
`combined-schema-size` with `limit.package` located at
`.compiled/schema.json`, observing that length, before it compiles the
combined schema.

<a id="mts-sec-044"></a>**MTS-SEC-044** A subschema's depth is 1 at its schema
file's root, and one more than the depth of the subschema holding it. A
processor MUST refuse a schema file holding a subschema that is an object and
is deeper than `schema-depth` with `limit.package`, located at the file and
JSON Pointer of the first such subschema in the order of MTS-SEC-020,
observing its depth, before it compiles any schema.

A reference that consumes nothing of the instance can be followed by another,
and a cycle of them never ends, so the references followed at one place in
the instance are bounded too. A chain is a sequence of subschemas each of
which applies where the one before it applies: as a subschema of `allOf`,
`anyOf`, `oneOf`, `not`, `if`, `then`, `else` or `dependentSchemas`, or as
the target of a `$ref` or a `$dynamicRef`. A `$dynamicRef` whose target
declares the `$dynamicAnchor` it names may reach any subschema declaring that
anchor, and a reference to a meta-schema ends a chain.

<a id="mts-sec-045"></a>**MTS-SEC-045** A processor MUST refuse a package one
of whose schema files holds a `$ref` or `$dynamicRef` that begins a chain
following more than `ref-depth` references, itself included, or a chain that
never ends, with `limit.package` located at the file and JSON Pointer of the
first such reference in the order of MTS-SEC-020, observing the number of
references when the chain ends, before it compiles any schema.

<a id="mts-sec-035"></a>**MTS-SEC-035** A processor MUST enforce
`value-depth` and `number-length` while it parses the manifest and each schema
file: it MUST refuse each array or object at the first depth beyond
`value-depth` with one `limit.package` located at it, reporting nothing nested
beneath it, and each number literal longer than `number-length` bytes with
`limit.package` located at it, before converting it to a number.

## 14.3 Entry confinement

<a id="mts-sec-002"></a>**MTS-SEC-002** A processor MUST refuse every entry
name [chapter 2](02-container.md)'s rules refuse before it reads any entry's
content, so that no name of a package, however it is written, can name
anything outside the package; and it MUST look an entry up by comparing its
name byte for byte with the package's own names, with no case folding,
normalization or resolution against a file system.

<a id="mts-sec-003"></a>**MTS-SEC-003** A processor MUST NOT follow a symbolic
link, or open a named pipe, socket or device, beneath a source directory's
`metais.yaml`, `inputs/`, `templates/`, `references/`, `assets/` or
`examples/`, and MUST read nothing of a source directory outside them.

<a id="mts-sec-004"></a>**MTS-SEC-004** A processor MUST NOT fetch anything a
package names — a schema's `$ref`, `$schema` or `$id`, a manifest's
`homepage` — and MUST resolve a schema reference only within the package and
the 2020-12 meta-schemas.

<a id="mts-sec-028"></a>**MTS-SEC-028** A processor MUST treat an entry's
content as bytes whatever it holds: it MUST NOT open, list or decompress an
entry that is itself a zip, a packed package or another archive, so that a
package can carry such an entry, however far it would expand, at the cost of
its own bytes.

## 14.4 Data limits

Every size and count a processor reads from the inputs of one operation is
bounded too. A caller may lower a data limit for one operation; nothing raises
one. A limit of §14.2 that bounds a parsed document, `value-depth` and
`number-length`, bounds every input as it bounds the manifest.

| Limit | Default | Bounds |
|---|---|---|
| `input-size` | 32 MiB | one input file, one collection item, one `--inputs-json`, `--set` or `--set-json` value |
| `inputs-total` | 128 MiB | all the input bytes one operation reads |
| `input-values` | 1,000,000 | the values all the inputs of one operation decode to |
| `yaml-size` | 1 MiB | one YAML input file or collection item |
| `csv-rows` | 1,000,000 | the data records of one CSV input |
| `csv-columns` | 1,024 | the fields of one record |
| `csv-cell` | 1 MiB | one field |
| `collection-items` | 100,000 | the files of one collection, and the entries of a data directory |
| `frontmatter-size` | 1 MiB | the frontmatter of one Markdown file |
| `findings` | 100,000 | the findings one operation lists ([chapter 12](12-findings.md)) |

<a id="mts-sec-005"></a>**MTS-SEC-005** A processor MUST enforce every limit
of §14.4, and `value-depth` and `number-length` on every input, at its default
or at the lower value a caller gives for one operation, and MUST refuse an
input that breaches one with `limit.exceeded` about that input, naming the
limit and its bound, before it reads the bytes the limit bounds; the input
then gives no value.

<a id="mts-sec-059"></a>**MTS-SEC-059** A processor MUST count against
`input-values` every value of the data model it decodes from the inputs of one
operation, summed over them, and MUST refuse with `limit.exceeded` naming
`input-values` the input whose value passes the bound, at that value. A JSON or
YAML file, and an `--inputs-json` or `--set-json` text, counts every value it
decodes to, its root included; a CSV file counts its array, each data record's
object and each field the record holds, an unquoted empty field being absent;
a plain text file and a `--set` value count one; a Markdown file counts its
object, its `body` and every value of its frontmatter's fields, its
frontmatter's mapping being the object; and a collection counts one for its
array, and each item as its form counts. A value counts when it is decoded,
though precedence later discards it; a value a path set's intermediate step
creates, or a schema default supplies, does not count.

<a id="mts-sec-060"></a>**MTS-SEC-060** A processor MUST refuse a YAML input
file or collection item whose bytes number more than `yaml-size` with
`limit.exceeded` naming `yaml-size`, before it parses the document.

## 14.5 Reading the host

<a id="mts-sec-006"></a>**MTS-SEC-006** Inside a data directory a processor
MUST open only the entries a declared input could be and the items of those
collections; MUST NOT follow a symbolic link, whatever it points at, refusing
it with `input.link`; and MUST NOT open a named pipe, socket or device,
refusing it with `input.not_regular_file` from what the directory says it is.
The data directory itself is the caller's own name, and a processor follows it
when it is a link.

> **Note.** A hard link is not detected. A file hard-linked to one outside
> the data directory cannot be told from an ordinary file without refusing
> every file that has more than one link, which breaks common tooling. A
> hard link cannot cross machines in a downloaded archive's data, and making
> one needs access to its target, so what remains is a supplier who already
> shares the caller's machine and can read the target. On the writing side
> nothing remains: a render replaces an existing file by renaming over its
> directory entry, so another link to it keeps the old bytes
> ([chapter 11](11-rendering.md)).

<a id="mts-sec-007"></a>**MTS-SEC-007** A processor MUST take every supplied
text, a file's content, an `--inputs-json` argument and a path set's value,
as data: `$VAR`, `${VAR}`, `%VAR%`, `~` and `@path` in it are those characters,
and nothing in it is looked up in the environment or read from a file.

<a id="mts-sec-034"></a>**MTS-SEC-034** A processor MUST take every string a
package holds, each member of its manifest, each string of its schema files and
each entry name, as data: `$VAR`, `${VAR}`, `%VAR%` and `~` in it are those
characters wherever the string appears, in a template's view of it, in an
output path and in an output's bytes, and nothing in it is looked up in the
environment or expanded to a home directory.

## 14.6 Work and output limits

What a render computes and writes is bounded by counters, never by a clock,
so a render is refused at the same point on every host. Each bound is a
limit with a name and a default, lowered for one operation as a package
limit is.

| Limit | Default | Bounds |
|---|---|---|
| `partial-depth` | 32 | the `render` calls active at once |
| `render-work` | 2³² units | the template evaluation of one render |
| `loop-iterations` | 10,000,000 | the iterations of every `for`, `tablerow` and `render … for` of one render, summed |
| `value-size` | 64 MiB | a string a template builds: a filter's result, a `capture` |
| `array-size` | 1,000,000 elements | an array a filter or a range builds |
| `assign-bytes` | 512 MiB | the bytes `assign` and `capture` store in one render, summed |
| `schema-work` | 450,000 units | the validations of one operation, summed |
| `outputs` | 100,000 | the files one render plans |
| `container-depth` | 3 | the containers an output nests one inside another, the outermost counting as the first ([chapter 10](10-transforms.md)) |
| `output-size` | 128 MiB | one file a render writes |
| `output-total` | 512 MiB | every byte one render writes |

<a id="mts-sec-008"></a>**MTS-SEC-008** A processor MUST bound a render's
template work by `render-work`, counted in units — 16 for every node it
evaluates (an output, a tag, a filter application, an operator), and, for
every filter application and operator, units for the size of its input, its
arguments and its result — and MUST refuse a render whose units exceed the
limit with `limit.exceeded.template`, naming the limit, at the token whose
evaluation exceeded it.

<a id="mts-sec-009"></a>**MTS-SEC-009** A processor MUST refuse a render whose
loop iterations, summed over the render, exceed `loop-iterations`, a range
whose span exceeds what remains refused before its first iteration; a
string a template builds longer than `value-size`; an array a filter or a
range builds longer than `array-size`; or bytes stored by `assign` and
`capture` beyond `assign-bytes`, each with `limit.exceeded.template` naming
the limit, at the token that exceeded it.

<a id="mts-sec-010"></a>**MTS-SEC-010** A processor MUST refuse an output whose
bytes exceed `output-size`, and the output whose bytes take a render's total
beyond `output-total`, with `limit.exceeded.output`, naming the output and the
limit.

<a id="mts-sec-011"></a>**MTS-SEC-011** A processor MUST refuse a render whose
manifest plans more files than `outputs` with `limit.package`, located at the
first declaration beyond the limit.

Nested `anyOf` over references makes a small schema's validation exponential
in its depth, so a validation's work is counted before anything of it is
evaluated, and counted whatever any evaluation's outcome, so that every
processor counts the same units.

<a id="mts-sec-046"></a>**MTS-SEC-046** A processor MUST count, before it
evaluates anything of a validation, the units it costs: one for each
subschema applied to a place in the instance, and one more for each member
of the object or item of the array there. At a place apply its `$ref`; its
`$dynamicRef`, resolved by the dynamic scope; every subschema of `allOf`,
`anyOf` and `oneOf`; `not`; `if`; `then` unless `if` is `false`; `else`
unless `if` is `true`; and the `dependentSchemas` subschema of each member
the object has. To a member apply its `properties` subschema; each
`patternProperties` subschema whose pattern matches its name;
`additionalProperties` when neither applies; `unevaluatedProperties` when
none of those three applies; and `propertyNames`, to its name, in a dynamic
scope of its own. To an item apply `prefixItems` by its index; `items` beyond
them; `contains`; and `unevaluatedItems` when neither `prefixItems` nor
`items` applies. A processor MUST refuse a validation whose units would take
the operation's total beyond `schema-work`, giving no other finding of that
validation: of the inputs, with `limit.exceeded` naming the limit about each
input the inputs object holds; of an output, with `limit.exceeded.output`.

## 14.7 The package regular-expression profile

A schema's `pattern` and the names of its `patternProperties` are regular
expressions of one profile: ECMA-262 patterns
([ECMA-262](A-references.md#ecma262)) in `u` mode, unanchored, without the
constructs no automaton can match. A pattern is matched without
backtracking, so a hostile pattern costs what a plain one costs.

<a id="mts-sec-040"></a>**MTS-SEC-040** A processor MUST give every pattern of
the profile the answer ECMA-262 gives it, and MUST decide each match in time
proportional to the size of the pattern's program times the length of the
string, whatever the pattern. In particular, `.` matches every code point
but U+000A, U+000D, U+2028 and U+2029; `\s` matches U+0009 to U+000D,
U+0020, U+00A0, U+1680, U+2000 to U+200A, U+2028, U+2029, U+202F, U+205F,
U+3000 and U+FEFF, and `\S` every other code point; `[]` matches nothing; and
`[^]` matches every code point.

<a id="mts-sec-041"></a>**MTS-SEC-041** A processor MUST refuse a pattern
holding a lookahead, a lookbehind, a numbered or named backreference, or a
quantifier whose bound is above 1000 with `schema.pattern_unsupported`,
naming the construct: `lookahead`, `lookbehind`, `backreference` or
`repeat_bound`.

<a id="mts-sec-042"></a>**MTS-SEC-042** A processor MUST refuse a pattern
longer, in bytes, than `pattern-length`, or whose program has more
instructions than `regex-program`, with `limit.package` located at the file
and JSON Pointer of its `pattern` or `patternProperties` member, observing
the length or the count, before it matches anything. A pattern's program is
the one Go's `regexp/syntax` package
([Go regexp/syntax](A-references.md#go-regexp-syntax)) compiles from the
pattern translated to its syntax, simplified.

## 14.8 Output confinement

<a id="mts-sec-012"></a>**MTS-SEC-012** A processor MUST write nothing outside
a render's output directory: it MUST refuse every output path that could
name anything outside it ([chapter 11](11-rendering.md)) before it writes
anything, and MUST make every write beneath the output directory without
following a symbolic link.
