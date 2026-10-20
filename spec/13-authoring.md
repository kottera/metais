# 13. Authoring

**Normative.** Areas: AUT, SMP.

An author writes a package in a source directory and makes it ready to
publish. This chapter specifies checking a package without data, with its
cross-reference of templates against schemas and its finding for a Liquid
construct split across Office Open XML text runs; examples, and testing a
package against them; packing a source directory into a package and
unpacking one back; and the sample data an authoring tool generates from a
package's schemas.

## 13.1 Checking a package

Checking is how an author hears, before any data exists, everything a
package's supplier and its render would trip over. It reports every finding
of every step in one run, because an author fixing one finding at a time
would run it once per defect. Each step runs whatever the steps before it
found, so a broken manifest does not hide a broken template.

<a id="mts-aut-001"></a>**MTS-AUT-001** An authoring tool's check MUST
accept a source directory ([chapter 2](02-container.md)) or a packed
package, MUST read nothing but the package's own entries, and MUST report
every finding of every step of §13.1 in one report, ordered and
de-duplicated as [chapter 12](12-findings.md) orders a report. It MUST
fail when, and only when, a finding has error severity; a warning does not
fail it.

<a id="mts-aut-002"></a>**MTS-AUT-002** The check MUST run these steps in
this order, each whatever the steps before it found: (1) the source tree;
(2) the entries; (3) the limits; (4) the manifest; (5) the schemas; (6) the
templates and the manifest's expressions; (7) the cross-reference; (8)
Office Open XML text runs; (9) the examples' layout; (10) the compiled form.
A step, or a part of one, whose input could not be made MUST NOT run, and
the check MUST then report one `report.step_skipped` info finding whose
`steps` parameter names every such step or part, in their order, by these
names: `limits of the compiled files`, `manifest paths`, `guide links`,
`combined schema`, `manifest expressions`, `cross-reference` and `compiled
form`. Each of them needs the manifest's model, so they are skipped exactly
when the manifest does not pass §3's schema and rules.

<a id="mts-aut-003"></a>**MTS-AUT-003** A package whose read is refused —
its container, an entry's name, a digest, the walk of a source directory,
or any finding of class `refused` — MUST be refused by the check with the
findings every operation that opens it reports, and with no other finding:
checking runs on a package whose entries were read.

### Steps 1 to 4: the tree, the limits and the manifest

<a id="mts-aut-004"></a>**MTS-AUT-004** In a source directory, a check MUST
report `package.layout.os_litter`, a warning, for each entry beneath a
layout tree (`assets/`, `examples/`, `inputs/`, `references/`,
`templates/`) named `.DS_Store`, `Thumbs.db` or `desktop.ini` or whose name
begins `._`, because pack writes it into the package; and
`package.layout.not_packed`, a warning, for each top-level name that is
not part of the layout and that a source directory does not ignore
silently ([chapter 2](02-container.md)), because pack leaves it out.

<a id="mts-aut-005"></a>**MTS-AUT-005** For a source directory, the check
MUST hold the package pack would write — its source entries, the three
compiled files of §2 and its digest file — to the `entries`, `entry-size`
and `package-uncompressed` limits, and MUST report a breach as
`limit.package`.

<a id="mts-aut-006"></a>**MTS-AUT-006** A check MUST report
`manifest.guide_link_missing`, a warning located at the manifest's
`/guide`, for each distinct relative target of a CommonMark link, image or
link reference definition in the guide, outside code, that names neither
a package entry nor a directory holding one. A target is relative when it
has no URI scheme and does not begin with `/` or `#`; its query and
fragment are removed and it is percent-decoded before it is resolved from
the package root.

<a id="mts-aut-023"></a>**MTS-AUT-023** A check MUST report an output
that names a `validate` schema and whose kind
([chapter 9](09-outputs.md)) is neither `json` nor `yaml` as
`output.declaration.validate_kind`, an error, when the kind is declared or
its path gives it: a path omitted, or one whose text after its last Liquid
construct holds the file's extension. A kind only a render can know is
the render's to judge.

<a id="mts-aut-031"></a>**MTS-AUT-031** A check MUST verify each container
profile ([chapter 10](10-transforms.md)) as far as it can without data, and
report what it finds with the code a render gives it: a container nested
deeper than `container-depth`; and, for each entry whose path is written
without Liquid markup, or is a template's default, the profile's refusals
and its warning on that path. It MUST report `output.container.missing_entry`
for a container whose profile requires an entry when no entry has that path
so, and no entry's path is one only a render can know — a path template, an
asset or a mirror.

<a id="mts-aut-080"></a>**MTS-AUT-080** A check MUST hold every output
path the manifest writes without Liquid markup, and every template's
default path, at every depth of a container to `container-depth`, to the
portable-path rules of [chapter 11](11-rendering.md) for one path
([MTS-PTH-001](11-rendering.md#mts-pth-001) to
[MTS-PTH-005](11-rendering.md#mts-pth-005),
[MTS-PTH-007](11-rendering.md#mts-pth-007) and
[MTS-PTH-010](11-rendering.md#mts-pth-010)), and MUST report a path that
breaks one with the finding a render refuses it with, naming the path;
an asset's or a mirror's destination written so, other than `.`, is held
to them as the directory every file it writes lies beneath. It MUST hold
together, by [MTS-PTH-006](11-rendering.md#mts-pth-006), the paths so
written of the declarations with neither `when` nor `for_each` in the
output directory, and in each container's entries but an entry its
profile reserves, and report each collision as a render reports it. A
path only a render can know, one a path template renders or a file an
asset or a mirror expands into, is the render's to judge.

> **Note.** A path written without Liquid markup is the same whatever the
> data, so every render of the package refuses it. Check exists to tell an
> author whether a package will render, and it says so before any data
> exists.

A package declares the specification minor it is written to
([chapter 4](04-versions.md)), and an author is best served by the lowest
minor whose constructs it uses, which every tool implementing that minor or
a later one renders. A check computes that lowest minor from the constructs
the package uses and the minor that added each. While 1.0 is the only minor
the computation is trivial: every package needs 1.0 and declares it, so a
check reports nothing of it.

### Step 5: the schemas

<a id="mts-aut-033"></a>**MTS-AUT-033** A check MUST compile, without
data, the schema every output's `validate` names, at every depth of a
container, by the rules of [chapter 10](10-transforms.md) §10.4, and MUST
report every finding of every one of them as a render would.

<a id="mts-aut-007"></a>**MTS-AUT-007** For every schema object of every
schema file, a check MUST validate its `default` against that object and
report a failure as `schema.invalid_default`, and validate each member of
its `examples` against that object and report each failure as
`schema.invalid_example`; both are errors, located at the value.

<a id="mts-aut-008"></a>**MTS-AUT-008** A check MUST report a `const` that
fails the schema object holding it as `schema.unsatisfiable`, an error,
and each `enum` member that fails it as `schema.unusable_enum_member`, a
warning, each located at the value.

<a id="mts-aut-009"></a>**MTS-AUT-009** A check MUST report each name in a
schema object's `required` that the object does not declare, as §13.2
defines declared for a property, as `schema.required_not_declared`, a
warning located at the name.

<a id="mts-aut-022"></a>**MTS-AUT-022** With the combined schema made, a
check MUST report a collection input whose schema's declared type set
([chapter 5](05-data-specification.md)) holds no `array`, or that declares
no item schema, as `manifest.input.collection_shape`; and each form an
input declares that disagrees with the declared type set of the schema a
file of it holds — the input's, or a collection's item schema — as
`manifest.input.form_shape`: `csv` needs `array` with an item schema whose
set holds `object`, `text` needs `string` and `markdown` needs `object`.
Both are errors.

### Step 6: the templates

<a id="mts-aut-010"></a>**MTS-AUT-010** A check MUST report every file
under `templates/` that is not valid UTF-8 as `template.not_utf8`, an
error, and MUST parse every other one by the dialect of
[chapter 7](07-template-environment.md), with the same parse rendering
uses, reporting each parse error as [chapter 3](03-manifest.md) reports
one in an expression, located at its line and column. A template holding a
CR LF pair MUST be reported as `template.crlf`, a warning located at the
first of them.

<a id="mts-aut-011"></a>**MTS-AUT-011** A `template.syntax` finding's
`cause` MUST name what the error is taken to come from, by the construct
holding it: document markup, when the template is one §13.3 scans and the
construct holds an XML start or end tag; typographic quotes, when the
construct holds U+2018, U+2019, U+201C or U+201D; an escaped character,
when the template is one §13.3 scans and the construct holds `&lt;`,
`&gt;`, `&amp;`, `&quot;` or `&apos;` outside a string literal; and
otherwise the grammar. The values are the registry's.

## 13.2 The cross-reference

The cross-reference reads every template and manifest expression against the
inputs' schemas without data. It needs no control flow: every branch is read,
and a name assigned anywhere in a template holds everything ever assigned to
it. A name it cannot judge is never an error.

<a id="mts-aut-012"></a>**MTS-AUT-012** A check MUST report a variable
whose root is not `inputs`, `package`, `now`, `sources`, `output` in a
template body, `forloop` inside a `for` body, `tablerowloop` inside
`tablerow`, `item` and `fan_out` in a `for_each` output's context, a name
the template binds, or a parameter its render call passes, as
`template.xref.unknown_variable`, an error naming the nearest known name
within edit distance 2 as its `suggestion`.

<a id="mts-aut-013"></a>**MTS-AUT-013** A property segment resolved against
an input's schema node is declared when a node of its applicable set — the
node and what `$ref`, `$dynamicRef`, `allOf`, `anyOf`, `oneOf`, `if`,
`then`, `else` and `dependentSchemas` reach from it — is open (the schema
`true`, or an object holding annotations only), has the name in
`properties`, has a `patternProperties` pattern matching it, or has an
`additionalProperties` other than `false` — or, with none, an
`unevaluatedProperties` other than `false`. An absent
`additionalProperties` declares nothing. A check MUST report a segment no
alternative declares as `template.xref.unknown_field`, or as
`template.xref.not_an_object` when the node's types exclude object.

<a id="mts-aut-014"></a>**MTS-AUT-014** A check MUST report an integer
index into a node whose types exclude array as
`template.xref.not_an_array`, and an index at or beyond `maxItems`, or
beyond `prefixItems` where `items` is `false`, as
`template.xref.index_out_of_range`, both errors; and a segment computed
from an expression that leaves no checkable candidate, or a computed name
in `inputs`, as `template.xref.dynamic_reference`, a warning.

<a id="mts-aut-015"></a>**MTS-AUT-015** A check MUST report a `render` of
a template that does not exist as `template.xref.missing_partial`, an
error, and a name a partial reads that is neither a global, its own local
nor passed by the call as `template.xref.partial_parameter_not_passed`, a
warning located at the read.

<a id="mts-aut-030"></a>**MTS-AUT-030** A check MUST report an `embed`
whose literal names no input the manifest declares as
`template.xref.unknown_input`, an error naming the nearest declared input
within edit distance 2 as its `suggestion`, and MUST count an `embed` of an
input as a use of it ([MTS-AUT-017](#mts-aut-017)).

<a id="mts-aut-032"></a>**MTS-AUT-032** A check MUST report an `asset` tag
whose path names no entry beneath `assets/`, matched byte for byte, as
`template.xref.missing_asset`, an error.

<a id="mts-aut-016"></a>**MTS-AUT-016** A check MUST analyse each output's
`path`, `when` and `for_each` in that output's context, and MUST report a
`for_each` whose value cannot be an array as
`template.xref.for_each_not_array`, an error.

<a id="mts-aut-017"></a>**MTS-AUT-017** A check MUST report an input that
no reference of a template an output renders, directly or through
`render`, and no manifest expression passes through as
`template.xref.unused_input`, and a template under `templates/` that no
output renders and no `render` names as `template.xref.unused_template`,
both warnings.

<a id="mts-aut-018"></a>**MTS-AUT-018** The cross-reference MUST stop
after 10 000 template-context analyses — a template, a partial in one set
of abstract bindings, or a manifest expression, each analysed in one
context — or 1 000 000 resolved segments, and MUST then report one
`template.xref.budget_exhausted`, a warning naming the bound reached. The
bounds are constants of the analysis, not limits ([chapter 14](14-security.md)):
reaching one refuses nothing.

## 13.3 Office Open XML text runs

<a id="mts-aut-019"></a>**MTS-AUT-019** A check MUST scan every template
whose source declares a WordprocessingML, SpreadsheetML or DrawingML main
namespace, and every template rendered from one, directly or
transitively. It MUST report a Liquid construct that overlaps the content
of a text element and does not lie within one as
`template.ooxml.split_construct`, and a `{{`, `{%`, `}}` or `%}` formed by
the last character of one text element and the first of the next
non-empty one in a paragraph container as
`template.ooxml.split_delimiter`; both are errors.

## 13.4 Examples and the compiled form

<a id="mts-aut-020"></a>**MTS-AUT-020** A check MUST report, as
`example.layout`, an error, an entry directly under `examples/`, an entry
of an example directory other than `inputs/…`, `expected/…` and
`example.yaml` (`reason` `entry`), and an example directory whose name
breaks the rule for an input's name (`reason` `name`, once per example).

<a id="mts-aut-021"></a>**MTS-AUT-021** A check MUST compute the compiled
form ([chapter 2](02-container.md)) of a package whose manifest passed,
and report a failure to compute any part of it; on a packed package, the
stored compiled files MUST equal what is computed.

## 13.5 Unpacking a package

Unpacking turns a packed package back into the source directory an author
edits. It writes what the author wrote and nothing derived: `.compiled/` is
a function of the other entries (§2.5), stale after the first edit, and
packing writes it again. It never writes into a directory that holds
anything of its own, because a file left from an earlier tree would join
the unpacked source and be packed with it.

<a id="mts-aut-024"></a>**MTS-AUT-024** An authoring tool's unpack MUST
take a packed package and a target directory, and MUST verify the package
as every operation that opens one does ([chapter 2](02-container.md)),
before it writes anything: a package that is refused MUST be refused with
the findings every such operation reports, and nothing written. It MUST
NOT run the check of §13.1: a package whose check reports errors is
unpacked as any other.

<a id="mts-aut-025"></a>**MTS-AUT-025** Unpack MUST write every entry of
the package but those beneath `.compiled/`, each at its path beneath the
target, holding the entry's bytes, and nothing else. It MUST write each
file with mode 0644 and each directory with mode 0755, less the process's
umask, whatever the package's zip records for the entry; no file it writes
is executable.

<a id="mts-aut-026"></a>**MTS-AUT-026** When the target does not exist,
unpack MUST create it with its missing parent directories, by writing the
entries into a staging directory it creates beside it, and renaming the
staging directory to the target, so that the target never holds part of
the package. A write that fails MUST leave nothing of the unpack behind, and MUST
be reported as `output.destination.nothing_written`.

<a id="mts-aut-027"></a>**MTS-AUT-027** When the target is an existing
directory that is empty, apart from top-level names beginning with `.` other
than `.compiled` and regular files whose names end `.metais`, unpack MUST
write the entries into it in place, and MUST leave every such name as it
was. A write that fails MUST stop the unpack and MUST remove nothing it
wrote. When it has created a file or a directory in the target, it MUST be
reported as `output.destination.write_failed`, whose `written` lists the
files already written, in the order written; when it has created nothing,
as `output.destination.nothing_written`.

<a id="mts-aut-028"></a>**MTS-AUT-028** Unpack MUST refuse a target
directory that holds any other entry with `output.destination.not_empty`,
listing up to ten of those entries in byte order, and a target that exists
and is not a directory, or lies beneath a file, with
`output.destination.not_directory`; in either case it MUST write nothing.
It has no way to overwrite.

<a id="mts-aut-029"></a>**MTS-AUT-029** The source directory an unpack
writes MUST open as a source directory, and MUST hold the same package:
its entries are the packed package's outside `.compiled/`, and its package
digest is the packed package's (§2.6).

## 13.6 Sample data

An author previews a package before any real data exists, and keeps what
the preview showed as an example to test against. The data an authoring
tool generates for this is the tool's own: this section fixes what the data
must be — a data directory that validates, holding the values the author
pinned — and not how it is made. Two tools may generate different data from
the same package, and one tool's data may change between its versions;
what an author keeps is kept as files, so nothing kept depends on a
generator.

The sample operation takes three options that steer its generator:
`optional`, whether it writes the members a schema leaves optional (`all`,
`none` or `mixed`); `seed`, an unsigned 64-bit integer from which its
choices are drawn; and `items`, the number of items it aims at for every
array and collection. What data they lead to is the tool's own, as the data
is.

<a id="mts-smp-001"></a>**MTS-SMP-001** An authoring tool's sample
operation MUST write the data it generates as a data directory
([§6.9](06-supplying-inputs.md)): one file for each
input it generates a value for, in the first form the input declares and
named as a data directory names a file of that form, and for a collection
input a directory holding one file per item.

<a id="mts-smp-002"></a>**MTS-SMP-002** The sample operation MUST validate
the data it generates as data supplied in a data directory is validated
([chapter 6](06-supplying-inputs.md)), before it writes anything, and MUST
refuse data that fails with one `schema.sample_invalid` error for each
error the validation reports about an input, writing nothing. Data the
operation writes therefore validates.

<a id="mts-smp-003"></a>**MTS-SMP-003** Each value given to the sample
operation with `set`, `set_json` or `inputs_json` MUST be written at its
place as given, the values applied in the order and with the precedence of
[§6.11](06-supplying-inputs.md), and the operation MUST
generate only the values they leave absent, evaluating each conditional of
the schemas against the value that holds the given ones.

<a id="mts-smp-004"></a>**MTS-SMP-004** A field for which the sample
operation can generate no value its schema allows MUST be reported as
`schema.sample_unsatisfiable`, an error about the input whose `instance`
parameter is the field's place in the input's value. Every such field MUST
be reported in one run, and nothing written.

<a id="mts-smp-005"></a>**MTS-SMP-005** When `to` names a directory that
does not exist, the sample operation MUST create it, with its missing
parent directories, holding the data directory; when `to` names an
existing directory with no entry at all, it MUST write the data directory
into it.

<a id="mts-smp-006"></a>**MTS-SMP-006** The sample operation MUST refuse a
`to` directory that holds any entry, a hidden one included, with
`output.destination.not_empty` listing up to ten of the entries in byte
order, and a `to` that exists and is not a directory, or lies beneath a
file, with `output.destination.not_directory`; in either case it MUST write
nothing.

The operation has no way to write into a directory that holds entries. The
reference tool's command line adds one, `--overwrite`, which replaces the
files it writes and leaves every other file alone; it is documented with
the command, as the command line is the tool's own contract.

<a id="mts-smp-007"></a>**MTS-SMP-007** With `example`, the sample
operation MUST write the data directory as `examples/<name>/inputs/` of the
source directory, beside an `examples/<name>/example.yaml`
(§13.4), and nothing else.

<a id="mts-smp-008"></a>**MTS-SMP-008** With `example`, the sample
operation MUST refuse when `examples/<name>/` exists at all, with
`output.destination.nothing_written` whose `reason` is `exists`, and MUST
write nothing: an example is never merged into
or replaced.

<a id="mts-smp-009"></a>**MTS-SMP-009** A render with `sample` MUST render
from data generated as the sample operation generates it with the same
options, in the data directory's place: the values given with `set`,
`set_json` and `inputs_json` held as [MTS-SMP-003](#mts-smp-003) holds
them, the data validated as [MTS-SMP-002](#mts-smp-002) validates it, and a
field that cannot be generated reported as [MTS-SMP-004](#mts-smp-004)
reports it; when any of them fails, nothing is rendered or written.

Since the generated data takes the data directory's place, a render with
`sample` takes no data directory; the reference tool refuses `--data`
given with `--sample` as a usage error.

## 13.7 Testing a package against its examples

An example is a set of inputs and the outputs a render of them should
produce, kept inside the package. Testing renders every example and
compares what it writes with what the example expects, byte for byte or a
container entry for entry, so an author hears of any change to an output
before a supplier does. The expected outputs are part of the package whose
digest a provenance line records, and the tool that renders them changes
version; testing therefore renders under one fixed identity, so that
neither a change to the package nor a new tool changes what an example
expects.

An example is a directory `examples/<name>/` whose name follows the rule
for an input's name ([chapter 3](03-manifest.md)). It holds `inputs/`,
laid out as a data directory ([chapter 6](06-supplying-inputs.md)) and
absent for an example that supplies nothing; `expected/`, optional, laid
out as the output directory a render writes; and `example.yaml`, optional
([MTS-AUT-020](#mts-aut-020) refuses anything else). A packed package
carries its examples, and they are read in place.

<a id="mts-aut-034"></a>**MTS-AUT-034** An authoring tool's test MUST take a
source directory or a packed package and MUST open it as every operation
that opens one does ([chapter 2](02-container.md)), and MUST parse every
template an output reaches. A package that is refused, or a template that
does not parse, MUST be reported with its findings, and then no example
MUST run.

<a id="mts-aut-035"></a>**MTS-AUT-035** Test MUST run every example, or
each one named, in byte order of name, and MUST run each one whatever
another found. It MUST fail when, and only when, a finding has error
severity: an example fails exactly when one of its findings is an error.

<a id="mts-aut-036"></a>**MTS-AUT-036** `example.yaml` MUST be read as
`metais.yaml` is read ([chapter 3](03-manifest.md)): as JSON when its first
byte other than U+0020, U+0009, U+000A and U+000D is `{`, and otherwise as
YAML. It MUST be a mapping whose members are `title`, a string holding no
line break; `description`, a string; and `now`, an RFC 3339 date-time with
an offset; each optional. Test MUST report anything else as
`example.metadata`, an error located in `example.yaml`, and MUST NOT
supply or render that example.

<a id="mts-aut-037"></a>**MTS-AUT-037** Test MUST supply an example's
inputs by reading its `inputs/` as a data directory, applying defaults and
validating them as [chapter 6](06-supplying-inputs.md) does, and MUST
locate each finding about them with the origin `example` at the package
entry it is about, such as `examples/basic/inputs/vendor.yaml`. An example
whose inputs are refused MUST NOT be rendered.

<a id="mts-aut-038"></a>**MTS-AUT-038** Test MUST render an example whose
inputs are valid as a render does ([chapter 11](11-rendering.md)), into
memory, with every finding a render reports, and with the render time
`now` that `example.yaml` gives, or with none; no other value of the host
reaches its templates.

<a id="mts-aut-039"></a>**MTS-AUT-039** Under test, `package.digest`
([chapter 7](07-template-environment.md)) and the `provenance` tag's
`package_digest` ([chapter 8](08-filters-and-tags.md)) MUST be `sha256:`
followed by 64 `0`, and the `provenance` tag's `tool` MUST be `test`. Only
test renders under this identity: a render records the package digest and
the processor.

<a id="mts-aut-040"></a>**MTS-AUT-040** When an example rendered and holds
`expected/`, test MUST compare the files it rendered with the files of
`expected/` by path, ignoring directories: a path in `expected/` the render
did not write MUST be reported as `example.output_missing`, located at the
expected file; a path the render wrote that `expected/` does not hold, as
`example.output_unexpected`, located with the origin `output` at the path;
and a path in both whose bytes differ, as `example.output_differs`, located
at the expected file, except that a container output is compared by its
entries ([MTS-AUT-065](#mts-aut-065)). Each finding's `path` is the path
within the output directory. An example without `expected/` is rendered and
not compared.

<a id="mts-aut-041"></a>**MTS-AUT-041** Two files are text when both are
valid UTF-8 and neither holds a NUL byte, and are otherwise binary, as
`example.output_differs`'s `binary` says. For text, the finding MUST be
located at the line and column of the expected file
([chapter 12](12-findings.md)) where the first code point that differs
starts, and its `diff` MUST be a unified diff from the expected file to the
rendered one; the hunks it shows are the implementation's own. For binary
files, `offset` MUST be the offset of the first byte that differs, or the
shorter file's size when it is a prefix of the other, with each file's size
and its digest, `sha256:` and the lower-case hexadecimal SHA-256 of its
bytes.

<a id="mts-aut-042"></a>**MTS-AUT-042** `example.output_differs`'s `cause`
MUST be the first of these that holds of the expected and the rendered
bytes: `line_endings` when they are equal once every CR LF is replaced by LF,
with `crlf` naming the side that holds CR LF, `expected`, `rendered` or
`both`; `final_newline` when one is the other followed by one LF; `bom` when
they are equal once a leading UTF-8 byte order mark is removed from each;
`trailing_whitespace` when they are equal once the U+0020 and U+0009 ending
each line are removed from each; and otherwise `none`.

A container output ([chapter 9](09-outputs.md)) is compared by its
entries rather than by its zip's bytes, as the conformance suite compares
one: the method of an entry the package leaves to the encoder is the
encoder's choice, so an `expected/` written by another conforming
implementation compares equal.

<a id="mts-aut-065"></a>**MTS-AUT-065** Test MUST compare a container
output with the file of `expected/` at its path by their entries: their
names, in order; each entry's uncompressed content; and the compression
method of each entry whose declaration says `stored: true`, and of a
profile's `mimetype` entry. It MUST NOT compare the method of any other
entry, nor any other byte of the zips, so that two containers whose
entries are equal by these rules are equal whatever their compressed
bytes.

<a id="mts-aut-066"></a>**MTS-AUT-066** Test MUST read an expected
container as it reads a packed package ([chapter 2](02-container.md)),
under the same bounds and the same profile, except that its entries are
in the order they are written rather than in byte order of name, and that
an end-record signature may lie within an entry's data, since an entry may
itself be a stored zip. An
expected container that is not read MUST be reported as
`example.expected_unreadable`, located at the expected file, or, for a
container entry ([MTS-AUT-069](#mts-aut-069)), where that entry is
located, with the code of the first refusal in `reason`; its entries MUST
NOT be compared.

<a id="mts-aut-067"></a>**MTS-AUT-067** An entry the expected container
holds and the rendered one does not MUST be reported as
`example.entry_missing`, located at the expected file's path followed by
`!` and the entry's name; an entry the rendered container holds and the
expected one does not, as `example.entry_unexpected`, located with the
origin `output` at the output's path followed by `!` and the entry's name.
Each finding's `path` is the container output's path and its `entry` the
entry's name.

<a id="mts-aut-068"></a>**MTS-AUT-068** When the entries both containers
hold are in another order in the one than in the other, test MUST report
one `example.entry_order`, at the first index, among those entries, at
which the two sequences of names differ: `index` is that 0-based index,
and `expected` and `rendered` are each container's entry there.

<a id="mts-aut-069"></a>**MTS-AUT-069** An entry both containers hold
whose uncompressed contents differ MUST be reported as
`example.output_differs`, as [MTS-AUT-041](#mts-aut-041) and
[MTS-AUT-042](#mts-aut-042) report a file, located at the expected file's
path followed by `!` and the entry's name, with `entry` naming it. An entry
that is itself a container MUST be compared by its entries in the same
way, each name within it located after its enclosing entry's name and a
further `!`.

<a id="mts-aut-070"></a>**MTS-AUT-070** An entry whose method is compared
([MTS-AUT-065](#mts-aut-065)) and which the expected container deflates
MUST be reported as `example.entry_method`, located as
[MTS-AUT-069](#mts-aut-069) locates an entry, since the rendered entry is
stored.

<a id="mts-aut-071"></a>**MTS-AUT-071** Test on a packed package MUST read
each expected container in place, inside the package, and MUST report what
it reports for the source directory the package was packed from.

A reference tool's command line adds what the operation cannot observe,
such as writing an example's render into its `expected/`; its help
documents it.

## 13.8 Packing a package

Packing is the last step of authoring: it checks a source directory,
tests it, and writes it as one packed package ([chapter 2](02-container.md)).
It has no way to skip a step, so the bytes a supplier receives are the
bytes whose check reported no error and whose examples passed. A source
directory is the author's own tree, and replacing its own previous package
is packing's edit loop, so a package file at the output is replaced whole,
never written into.

<a id="mts-aut-043"></a>**MTS-AUT-043** An authoring tool's pack MUST take
a source directory and an output, and MUST open the source as every
operation that opens a package does ([chapter 2](02-container.md)) before
anything else: a package that is refused MUST be refused with the findings
every such operation reports, and nothing written. A path that opens as a
packed package MUST then be refused with `usage.not_a_source_directory`,
and nothing written.

<a id="mts-aut-044"></a>**MTS-AUT-044** Pack MUST run the check of §13.1
over the source directory. When the check reports an error, pack MUST
report every finding of the check, MUST NOT test the package, and MUST
write nothing.

<a id="mts-aut-045"></a>**MTS-AUT-045** When the check reports no error,
pack MUST run the test of §13.7 over every example. When an example fails,
pack MUST report every finding of the check and of the test, and MUST
write nothing.

<a id="mts-aut-046"></a>**MTS-AUT-046** A warning or a note of the check or
the test MUST be reported and MUST NOT stop pack.

<a id="mts-aut-047"></a>**MTS-AUT-047** Pack MUST write exactly the source
directory's entries ([MTS-PKG-021](02-container.md#mts-pkg-021)) and
`.compiled/`, computed from them: `metais.yaml` and every file beneath
`inputs/`, `templates/`, `references/`, `assets/` and `examples/`, a name
beginning with `.` included. A top-level name beginning with `.`, and a
top-level regular file whose name ends in `.metais`, MUST be left out with
no finding; every other top-level name §2.4's table does not hold MUST be
left out, and reported as the check reports it
([MTS-AUT-004](#mts-aut-004)).

<a id="mts-aut-048"></a>**MTS-AUT-048** When the output names an existing
directory, pack MUST write the package into it as the file
`<name>-<version>.metais`, from the manifest's `name` and `version`;
otherwise it MUST write the package as the file the output names.

<a id="mts-aut-049"></a>**MTS-AUT-049** Pack MUST write the package to a
temporary file in the output file's own directory, named `.metais-tmp-`
followed by the first 16 lower-case hexadecimal digits of the SHA-256 of
the output file's name and created exclusively, and MUST then rename it to
the output file, so that the output file at every instant holds what it
held before or the whole package. A regular file at the output MUST be
replaced in that rename, and never written into.

<a id="mts-aut-050"></a>**MTS-AUT-050** Pack MUST refuse, with
`output.destination.nothing_written`, an output file whose name holds
anything but a regular file, with `reason` `is_directory` for a directory
and `not_regular_file` for anything else, a symbolic link included; an
output whose directory does not exist, with `reason` `not_found`; and an
output whose temporary file's name is already taken, with `reason`
`exists`. It MUST refuse an output beneath a file with
`output.destination.not_directory`. In each case it MUST write nothing.

<a id="mts-aut-051"></a>**MTS-AUT-051** Pack MUST refuse an output file
that lies inside one of the source directory's layout trees, `inputs/`,
`templates/`, `references/`, `assets/` and `examples/`, or that is its
`metais.yaml`, with `output.destination.in_package`, whose `entry` names
that tree or file, and MUST write nothing: the package would join the
source it was packed from, or replace it.

<a id="mts-aut-052"></a>**MTS-AUT-052** Before it writes the package
anywhere, pack MUST read the package it built as a processor verifies a
packed package ([MTS-PKG-032](02-container.md#mts-pkg-032)'s steps 1 to
5), under the limits in force, and MUST refuse one a processor would
refuse with that verification's findings, writing nothing: a limit only
the packed bytes show, such as `entry-ratio` and `package-size`, which
the check cannot judge without the encoder, is pack's to hold.
