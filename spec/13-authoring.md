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

### Step 5: the schemas

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
be reported as `output.destination.write_failed` with an empty `written`.

<a id="mts-aut-027"></a>**MTS-AUT-027** When the target is an existing
directory that is empty, apart from top-level names beginning with `.` other
than `.compiled` and regular files whose names end `.metais`, unpack MUST
write the entries into it in place, and MUST leave every such name as it
was. A write that fails MUST stop the unpack, MUST remove nothing it wrote,
and MUST be reported as `output.destination.write_failed`, whose `written`
lists the files already written, in the order written.

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
