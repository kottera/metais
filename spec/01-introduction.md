# 1. Introduction

**Normative.** Areas: GEN.

This chapter says what the METAIS specification covers, states how its
normative language and its statements are read, defines the conformance
classes and the terms the other chapters use, fixes the notation, and
registers the statement areas each chapter owns.

## 1.1 Scope (informative)

A METAIS package is a single file that turns input data into finished files.
It carries the templates that become the files, a data specification saying
which data the templates need and what that data means, and, optionally,
background references, static assets and examples. Whoever supplies the
data — a person, a program or an AI harness — can open the package, learn what
it needs, gather or write that data however suits them, have it validated, and
render the result.

This specification defines:

- the package: its container and layout (chapter 2), its manifest (chapter 3)
  and the versions it names (chapter 4);
- the data a package asks for, and how that data is supplied and validated
  (chapters 5 and 6);
- the template environment, its filters and tags, the outputs a package
  declares, the transforms applied to them, and how a render produces them
  (chapters 7 to 11);
- the findings through which every problem is reported (chapter 12);
- the authoring operations that check, test, pack and unpack a package and
  generate sample data for it (chapter 13);
- the security model: resource limits, confinement and host isolation
  (chapter 14);
- what a conformance claim is, and the suite of vectors that proves one
  (chapter 15).

It does not define the command line of any tool, where a package or its data
comes from, or how packages are distributed. `metais`, the reference tool,
documents its own command line.

## 1.2 Revisions of this document (informative)

This document is revision 1.0 of the specification. A revision is named by the
specification version a package's manifest targets (chapter 4): a major and a
minor, with no patch part.

There is one text per major. The text on the development branch is the latest
released revision plus any additions for the next minor, each addition marked
with the minor that introduced it. A revision is released by an annotated git
tag `spec-<MAJOR>.<MINOR>`, which holds its exact bytes. [CHANGELOG.md](CHANGELOG.md)
records what each revision changes, and [ERRATA.md](ERRATA.md) records the
corrections made to released revisions.

## 1.3 Normative language

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT",
"SHOULD", "SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED", "MAY", and
"OPTIONAL" in this document are to be interpreted as described in
BCP 14 [RFC2119] [RFC8174] when, and only when, they appear in all
capitals, as shown here.

Of those key words, this document uses only the five that mean *must*,
*must not*, *should*, *should not* and *may*; the others are synonyms of these
and do not appear in its text. A requirement on a program is stated with
*must* or *must not*. *Should* and *should not* address people — authors and
suppliers — and never a program. *May* grants a processor a freedom, and every
vector passes under each choice that freedom allows.

Every key word written in capitals is inside a statement (§1.4). Informative
material is marked, and only these forms are informative:

- a chapter whose status line says `**Informative.**`;
- a section whose heading ends with `(informative)`;
- a block quote beginning `> **Note.**`;
- an example: a block quote beginning `> **Example.**`, or an example block
  included from a vector.

Everything else in a normative chapter is normative. Normative text that holds
no key word, such as a grammar or a table of fixed values, is normative where a
statement adopts it.

## 1.4 Statements

Every testable rule is one statement, with an identifier of the form
`MTS-<AREA>-<NNN>`. The area is one of those registered in §1.8 and owned by
the chapter in which the statement appears; the number is assigned in order of
addition within the area, and is never reused or renumbered. A statement
begins a paragraph, a list item or a table row, and its scope is that
paragraph, item or row together with any list nested inside the item.

A statement opens with an anchor, its identifier in bold (bare inside a table
cell) and, optionally, a list of attributes:

```abnf
statement-head = %s"<a id=\"" anchor %s"\"></a>" [ %s"**" ] id [ %s"**" ] [ SP attributes ] SP
id             = %s"MTS-" area "-" number
area           = 3%x41-5A                 ; A-Z
number         = 3*DIGIT                   ; zero-padded to at least three digits
anchor         = the id in lower case
attributes     = %s"*(" attribute *( %s"; " attribute ) %s")*"
attribute      = %s"since " version / %s"not observable"
               / %s"amended by " erratum / %s"withdrawn by " erratum
version        = 1*DIGIT "." 1*DIGIT
erratum        = %s"E-" version "-" 3*DIGIT
```

- `since` names the minor revision that introduced the statement; without it,
  the statement belongs to 1.0.
- `not observable` marks a statement that no black-box run of a program can
  check: every statement addressed to people, and a requirement whose evidence
  is not in a program's outputs.
- `amended by` and `withdrawn by` name the erratum that corrected or withdrew
  the statement. A withdrawn statement keeps its text and its identifier.

A statement that uses *must* or *must not* is a requirement; one that uses only
*should* or *should not* is a recommendation; one that uses only *may* is a
permission. Prose cites a statement by its identifier, linked to its anchor;
vectors cite it by identifier alone. Section numbers and headings may change
between revisions, and statement identifiers are the stable points of
citation.

## 1.5 Conformance classes

The conformance classes are:

- **package** — a packed package or a source directory that satisfies every
  statement addressed to packages;
- **processor** — a program that opens packages, validates inputs and renders
  outputs;
- **authoring tool** — a program that checks, tests, packs and unpacks
  packages, and generates sample data for them.

A program conforms to a class when it satisfies every statement addressed to
that class. A package's conformance is established by a conforming authoring
tool's check reporting no finding of error severity. Chapter 15 defines the
operations each class performs and what a conformance claim names.

> **Note.** `metais`, the reference tool, claims the processor and authoring
> tool classes.

## 1.6 Terminology

- **Package** — the single file a consumer receives, named `<name>.metais`: a
  zip archive in the profile of chapter 2. Where the distinction matters, a
  *packed package*.
- **Source directory** — the editable tree an author works in; packing it
  produces a package.
- **Entry** — one file inside a package or a source directory, named by its
  path from the package's root.
- **Manifest** — the package's description: its identity, its versions, the
  guide for suppliers, its inputs and its outputs.
- **Input** — a named slot in the data specification, with a schema and a
  meaning.
- **Collection input** — an input supplied as a directory of files, which
  becomes an array.
- **Data specification** — a package's inputs, with their schemas and the
  prose that explains them.
- **Template** — a Liquid file in the package that renders an output.
- **Partial** — a template rendered from inside another template.
- **Output** — a file a render produces, as the manifest declares it.
- **Reference** — background material in a package, for the supplier.
- **Asset** — a static file in a package that a render copies to the output, or
  that a template includes.
- **Example** — a set of inputs bundled with a package, optionally with the
  outputs they produce.
- **Supplier** — whoever provides the inputs.
- **Author** — whoever writes the package.
- **Render** — the operation that produces a package's outputs from its
  inputs.
- **Finding** — one problem an operation reports.
- **Specification version** — the version of this specification a package
  targets.
- **Package version** — the version of one package, assigned by its author.
- **Host** — the machine on which a processor or an authoring tool runs, and
  everything it holds beyond the package and the inputs the caller names.
- **Statement** — one numbered, testable rule of this specification (§1.4).
- **Vector** — one case of the conformance suite: a package, the operations
  run on it with their arguments, and the results a conforming program
  produces.

## 1.7 Notation

- Grammars are written in ABNF ([RFC 5234](A-references.md#rfc5234)), with
  [RFC 7405](A-references.md#rfc7405)'s `%s` prefix for a case-sensitive
  literal string.
- A location inside a JSON value is a JSON Pointer
  ([RFC 6901](A-references.md#rfc6901)).
- JSON is as [RFC 8259](A-references.md#rfc8259) defines it.
- A byte value is written in hexadecimal with the prefix `0x`, as in `0x0A`.
- A Unicode code point is written `U+` followed by at least four upper-case
  hexadecimal digits, as in `U+000A`.
- Text in `monospace` is literal: a name, a value, or the content of a file.
- `§9.3` refers to section 3 of chapter 9.

## 1.8 Area registry

An area code is three upper-case letters and belongs to exactly one chapter.
A chapter may gain an area; an area never moves to another chapter and is
never reassigned, because statement identifiers carry it.

| Area | Chapter | Subject |
|---|---|---|
| `GEN` | [1. Introduction](01-introduction.md) | scope, conformance classes, terminology, notation, normative language |
| `CON` | [2. Container](02-container.md) | the zip profile, entry names, the layout, `.compiled/`, digests and their verification |
| `MAN` | [3. Manifest](03-manifest.md) | the manifest's structure and fields, the YAML and JSON profiles, the canonical form |
| `VER` | [4. Versions](04-versions.md) | specification and package versions, compatibility, what a tool states |
| `DAT` | [5. Data specification](05-data-specification.md) | the `inputs` section, the JSON Schema dialect and its asserted keywords and formats, meaning documents, the combined schema |
| `SUP` | [6. Supplying inputs](06-supplying-inputs.md) | input forms and their parsing, CSV coercion, collections, the data model templates see, supply, precedence, defaults, absence |
| `TPL` | [7. Template environment](07-template-environment.md) | the pinned Liquid dialect, variables, partials, truthiness, `blank` and `empty`, undefined names, whitespace, template errors |
| `FLT` | [8. Filters and tags](08-filters-and-tags.md) | the METAIS filter catalog |
| `TAG` | [8. Filters and tags](08-filters-and-tags.md) | the `embed`, `asset` and `provenance` tags, and provenance's syntax for each output |
| `OUT` | [9. Outputs](09-outputs.md) | the `outputs` section, path templates, `for_each`, `when`, assets, the mirror shorthand |
| `TRN` | [10. Transforms](10-transforms.md) | zip containers with the OOXML, ODF and EPUB settings, CRLF conversion, output validation |
| `RND` | [11. Rendering](11-rendering.md) | determinism, staging and all-or-nothing writes, overwrite, dry run, stream output and its tar profile |
| `PTH` | [11. Rendering](11-rendering.md) | portable paths |
| `FND` | [12. Findings](12-findings.md) | the finding model, severities, the code catalog, locations, the report |
| `AUT` | [13. Authoring](13-authoring.md) | check and the cross-reference, the split-run finding, examples and test, pack, unpack |
| `SMP` | [13. Authoring](13-authoring.md) | sample data |
| `SEC` | [14. Security](14-security.md) | resource limits and their defaults, confinement, host isolation |
| `CNF` | [15. Conformance](15-conformance.md) | conformance classes and claims, the operations, the suite and how it is read |
