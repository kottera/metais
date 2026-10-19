# 10. Transforms

**Normative.** Areas: TRN.

An output may be transformed after it renders, and only by the transforms this
specification defines. This chapter specifies zip container outputs, with the
settings the Office Open XML, OpenDocument and EPUB formats require of them,
conversion of line endings to CRLF, and validation of an output against a
schema the package carries.

## 10.1 Zip containers

A container output is one zip file whose entries are declarations of the
forms an output takes, written inline beneath it. A part that must also be a
file of its own is declared again at the top level: an entry is written only
into its container.

<a id="mts-trn-001"></a>**MTS-TRN-001** A processor MUST write a container
output as one zip file, at the container's `path` in its enclosing tree,
whose entries are the files of the container's `entries` declarations and
nothing else: each entry's file rendered exactly as a top-level output with
that declaration is, at its `path` relative to the root of the container,
and none of them written on its own. A container MUST hold file entries
only, never an entry for a directory. Within an entry's template,
`output.path` is the entry's path within its container and `output.root`
leads from the entry back to the container's root.

<a id="mts-trn-002"></a>**MTS-TRN-002** A container MUST be its own tree for
§11.2's portable-path rules: a processor MUST hold each entry's path to them
within its container, MUST look for collisions and file–directory conflicts
among the container's entries only, and MUST report each with the code
§11.2 gives it, located at the entry's declaration. Entry ids MUST be
unique within their container, and an entry MUST NOT declare `executable`,
since a container records no file mode; a processor refuses both as it
refuses any other manifest that breaks its rules ([chapter 3](03-manifest.md)).

<a id="mts-trn-003"></a>**MTS-TRN-003** A container MAY hold a container as
an entry, to the depth of the `container-depth` limit
([chapter 14](14-security.md)), counting the outermost container as the
first. A processor MUST refuse a container nested deeper with
`limit.package` naming `container-depth`, located at the deeper container's
`container` member.

<a id="mts-trn-004"></a>**MTS-TRN-004** A processor MUST write a container's
entries in this order, whatever the order of their declarations: first the
entries its profile places first (§10.2), in the profile's order; then the
files of every entry declaration with `first: true`, in declaration order,
each declaration's own files in byte order of their entry paths; then every
other file, in byte order of its entry path, comparing the paths' UTF-8
bytes. Each file MUST appear once, at the earliest of these places.

<a id="mts-trn-005"></a>**MTS-TRN-005** A processor MUST store, with
compression method 0, every file of an entry declaration with
`stored: true` and the `mimetype` entry a profile writes; every other entry
it MUST deflate (method 8) when that is strictly shorter than its content
and store otherwise, so that an empty entry is always stored. A conforming
processor's choice between the two for an entry without `stored: true` is
its encoder's, and is not compared.

<a id="mts-trn-006"></a>**MTS-TRN-006** A processor MUST write each
container's zip with every field a package's zip fixes
([chapter 2](02-container.md)): the fixed version, flags, time and date and
external attributes; no extra field and no comment on any entry; sizes and
CRC in each local header, with no data descriptor; no zip64 record; and an
end record that is the file's last 22 bytes. The entries follow one another
with nothing between them, in the order of MTS-TRN-004.

<a id="mts-trn-007"></a>**MTS-TRN-007** A processor MUST refuse a container
that plans more than 65,534 entries, a profile's `mimetype` among them, with
`output.container.too_many_entries`, since a zip without zip64 holds no
more.

<a id="mts-trn-008"></a>**MTS-TRN-008** A container without a `profile`
MUST be written exactly as declared: no entry is added to it, none is
required, and an entry may have any portable path, `mimetype` included.

<a id="mts-trn-012"></a>**MTS-TRN-012** A container with `for_each`
MUST be written once for each evaluation whose `when` holds, as one zip at
the path that evaluation renders ([chapter 9](09-outputs.md)), with its
entries planned and rendered in that evaluation: every entry's `when`,
`path` and template MUST see the evaluation's `item` and `fan_out`. An
entry of such a container MUST NOT have a `for_each` of its own
(MTS-MAN-016).

## 10.2 Container profiles

A profile states a format's packaging rules once, so an author who names it
gets them right without knowing them. It applies the settings the format
requires, and verifies what can be seen without reading a part's content;
it never validates a part's XML. Each of a profile's refusals and its
warning is located at the entry's declaration, and a missing entry at the
container's `profile`.

<a id="mts-trn-009"></a>**MTS-TRN-009** A container with `profile: ooxml`
MUST have an entry whose path is `[Content_Types].xml`, which a processor
MUST write first; a processor MUST refuse a container without one with
`output.container.missing_entry`, and an entry whose path holds a
character outside ASCII with `output.container.non_ascii`.

<a id="mts-trn-010"></a>**MTS-TRN-010** For a container with
`profile: odf`, a processor MUST write an entry `mimetype` itself, first,
whose content is exactly the container's `mimetype` with no line ending,
stored, with no extra field and no data descriptor. It MUST refuse an entry
of the author's whose path is `mimetype` with
`output.container.reserved_entry`, and a container with no entry
`META-INF/manifest.xml` with `output.container.missing_entry`.

<a id="mts-trn-011"></a>**MTS-TRN-011** For a container with
`profile: epub`, a processor MUST write an entry `mimetype` itself, first,
whose content is exactly `application/epub+zip` with no line ending,
stored, with no extra field and no data descriptor. It MUST refuse an entry
of the author's whose path is `mimetype` with
`output.container.reserved_entry`, and a container with no entry
`META-INF/container.xml` with `output.container.missing_entry`; and it MUST
report `output.container.space`, a warning, for each entry whose path holds
U+0020.

## 10.3 CRLF conversion

The transforms run on an output's bytes in memory, after the output renders
and before anything is written: conversion of line endings first, then
validation, so that a schema judges exactly the bytes that will be written.

A template is best kept with LF line endings, since a checkout that converts
them would change the package; an output a platform needs with CR LF gets
them from the declaration instead.

<a id="mts-trn-013"></a>**MTS-TRN-013** When a template output, a mirror,
or a container's template entry declares `newline: crlf`, a processor MUST
convert the bytes of its file after it renders and before it is validated
(§10.4), and before a container holding it is assembled: every LF (U+000A)
not immediately preceded by CR (U+000D) becomes CR LF, and a CR LF and a lone
CR are left as they are, so the conversion is idempotent and a template
written with CR LF converts to the same bytes. The conversion applies to the
whole file, the bytes a value or a tag placed in it included. Under `newline:
as-written`, the default, a processor MUST write the bytes the template
produced.

## 10.4 Output validation

An output may name a schema its file must satisfy, so that a package whose
output feeds another program can promise that the data it writes is valid,
and a supplier whose data would make it invalid hears so before anything is
written.

<a id="mts-trn-014"></a>**MTS-TRN-014** When a template output, or a
container's template entry, names a schema with `validate`, a processor MUST
parse the final bytes of each of its files, after §10.3, by its kind
([chapter 9](09-outputs.md)): a `json` file by the rules of a JSON input file
and a `yaml` file by those of a YAML input file ([chapter
6](06-supplying-inputs.md)), after a single leading byte order mark is
removed; and MUST validate the value against the schema by the dialect,
keywords, formats and reference rules of [chapter
5](05-data-specification.md), with `inputs/` read as the top-level directory
that holds the `validate` file. The schema files compiled with it are the
`validate` file and each JSON or YAML file beneath that directory that its
references reach by path, transitively; beneath `inputs/`, every schema file
there. A reference into `.compiled/` escapes the package's content. A schema
that does not compile MUST refuse the render with the findings chapter 5
gives it, and an output whose kind is neither `json` nor `yaml` with
`output.declaration.validate_kind`, located at its `validate`.

<a id="mts-trn-015"></a>**MTS-TRN-015** An output whose text does not parse
by its kind's rules, or whose value fails its schema, MUST refuse the render,
and the processor MUST write nothing. It MUST report every finding of every
validated file: a rule of the form the text breaks as `output.parse`, whose
`rule` names the code an input file breaking the same rule gets; a limit the
text breaches as `limit.exceeded.output`; and each failure of the schema by
the `output.` family of §12.3. Each is about the output, named by its `id`
(an entry without one by the nearest enclosing declaration's), and located
with origin `output`, its file the output's path (for a container's entry,
the container's path and the entry's, joined by `/`), its pointer the JSON
Pointer into the output's value, and its line and column where the parser
gives them.