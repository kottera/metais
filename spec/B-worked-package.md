# Appendix B. A worked package

**Informative.** No statement area.

This appendix builds one small package step by step on the authoring path of
chapter 13, from an empty directory to a packed package: a manifest, one input
with its schema and its meaning document, two templates, and one example. It
is the page an author new to METAIS starts from, and it needs nothing beyond
this specification and an authoring tool.

Every file it shows is included from a vector of the conformance suite under
[`conformance/vectors/gen/`](conformance/vectors/gen/), so what it shows is
what a conforming authoring tool accepts, byte for byte:

- [`gen/worked-package`](conformance/vectors/gen/worked-package/vector.json)
  is the finished package, which check, test and pack accept;
- [`gen/worked-package-check`](conformance/vectors/gen/worked-package-check/vector.json)
  is its first draft, which check refuses;
- [`gen/worked-package-test`](conformance/vectors/gen/worked-package-test/vector.json)
  is its second draft, which test refuses.

Each block is introduced by a comment naming its vector, and its `path` is the
file's path within that vector. A vector holds the source directory as
`package/`; in this appendix the same directory is called `release-notes/`, so
`package/metais.yaml` is the file `release-notes/metais.yaml`.

The commands are those of `metais`, the reference tool. This specification
defines what each operation does, not how a tool spells it, so another
authoring tool runs the same operations under its own names.
`metais help <command>` prints every operand and flag of a command.

## B.1 What the package does

The package writes two files for one release of a product: release notes in
Markdown, named after the version, and a plain-text announcement. Whoever
supplies the data — a person, a program or an AI harness — supplies one
input, the release: its product, version and date, an optional highlight,
and the list of its changes.

## B.2 An empty directory

A source directory is an ordinary directory
([chapter 2](02-container.md)). Make one:

```sh
mkdir release-notes
```

Its top level holds only the names of §2.4's layout. This package uses four of
them: `metais.yaml`, `inputs/`, `templates/` and `examples/`. A top-level name
beginning with `.`, such as `.git`, is no part of the package
([MTS-PKG-021](02-container.md#mts-pkg-021)).

> **Note.** `metais init release-notes` writes a smaller package of the same
> shape, with a greeting in place of the release, for an author who would
> rather edit than start from nothing. `metais init release-notes --minimal`
> writes only the layout, one input and one template to replace, and
> `--html` makes either one's output an HTML page that declares
> `escape: html` and writes its data into a `<script>` with `json_script`.
> This appendix writes every file itself, so that each byte it shows is one
> a vector proves.

## B.3 The manifest

The manifest, `metais.yaml`, says what the package is, what it asks for and
what it writes ([chapter 3](03-manifest.md)):

<!-- example: gen/worked-package -->

```yaml path=package/metais.yaml
metais: "1.0"
name: "release-notes"
version: "1.0.0"
title: "Release Notes"
description: "Writes the release notes and a short announcement for one release of a product."
guide: |-
  This package writes two files for one release of a product: release
  notes in Markdown, named after the version, and a plain-text
  announcement to post where the product's users will see it.

  Supply one input, the release. Read
  [what the release means](inputs/release.md) before you write it.
inputs:
  - name: release
    title: "Release"
    schema: inputs/release.schema.json
    meaning: inputs/release.md
outputs:
  - id: notes
    template: notes.md.liquid
    path: "release-{{ inputs.release.version }}.md"
  - id: announcement
    template: announcement.txt.liquid
```

- `metais` is the specification version the package targets
  ([chapter 4](04-versions.md)), and `version` is the package's own.
- `name` is lower-case letters, digits and hyphens
  ([MTS-MAN-014](03-manifest.md#mts-man-014)). It names the packed file,
  `release-notes-1.0.0.metais`.
- `title` and `description` are what a supplier sees first. `guide` is the
  supplier's instructions, in CommonMark; its link to the meaning document is
  a package path. `guide: |-` keeps the block's final line break out of it,
  as prose requires (§3.4).
- `inputs` declares the one input, `release`, with its schema and its meaning
  document, each a package path under `inputs/`
  ([MTS-MAN-017](03-manifest.md#mts-man-017)). With no `forms`, it may be
  supplied as JSON or YAML.
- `outputs` declares the two files. `notes` has a `path`, which is itself a
  Liquid template, so the file is named from the version: `release-2.4.0.md`
  ([MTS-OUT-003](09-outputs.md#mts-out-003)). `announcement` has none, so its
  file is its template's path with `.liquid` removed, `announcement.txt`
  ([MTS-OUT-001](09-outputs.md#mts-out-001)). In YAML, a `path` that begins
  with `{` must be quoted; quoting every one costs nothing.

The manifest names files that do not exist yet. Each of the next sections
writes one of them.

## B.4 The input

An input is a schema, which a tool enforces, and a meaning document, which a
supplier reads ([chapter 5](05-data-specification.md)). Both live under
`inputs/`.

### The schema

The schema is JSON Schema 2020-12 ([MTS-DAT-003](05-data-specification.md#mts-dat-003)):

<!-- example: gen/worked-package -->

```json path=package/inputs/release.schema.json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Release",
  "description": "One release of a product: what it is, when it shipped, and what changed.",
  "type": "object",
  "additionalProperties": false,
  "required": ["product", "version", "date", "changes"],
  "properties": {
    "product": {
      "type": "string",
      "minLength": 1,
      "maxLength": 80,
      "title": "Product",
      "description": "The product's name, as its users know it."
    },
    "version": {
      "type": "string",
      "pattern": "^[0-9]+\\.[0-9]+\\.[0-9]+$",
      "title": "Version",
      "description": "The release's version: three numbers joined by dots.",
      "examples": ["2.4.0"]
    },
    "date": {
      "type": "string",
      "format": "date",
      "title": "Date",
      "description": "The day the release shipped, as YYYY-MM-DD."
    },
    "highlight": {
      "type": "string",
      "minLength": 1,
      "maxLength": 400,
      "title": "Highlight",
      "description": "One or two sentences on what matters most in this release."
    },
    "changes": {
      "type": "array",
      "minItems": 1,
      "title": "Changes",
      "description": "Every change a user would notice, in the order to list them.",
      "items": {
        "type": "object",
        "additionalProperties": false,
        "required": ["kind", "text"],
        "properties": {
          "kind": {
            "enum": ["added", "changed", "fixed"],
            "title": "Kind",
            "description": "Whether the change adds, changes or fixes something."
          },
          "text": {
            "type": "string",
            "minLength": 1,
            "maxLength": 400,
            "title": "Text",
            "description": "The change, in one sentence a user understands."
          }
        }
      }
    }
  }
}
```

- Every property has a `title` and a `description`, which a supplier's tools
  show beside the field.
- `"additionalProperties": false` refuses a field the schema does not name.
  It also lets the check tell a misspelt field in a template from a real one
  (B.6): a schema without it declares every name.
- `format: date` is asserted, not only annotated
  ([MTS-DAT-005](05-data-specification.md#mts-dat-005)), and `pattern` is a
  regular expression of the profile of [chapter 14](14-security.md).
- `highlight` is not in `required`, so a supplier may leave it out.

### The meaning document

The meaning document says what the data is for and what good data looks
like, which a schema cannot:

<!-- example: gen/worked-package -->

```markdown path=package/inputs/release.md
# Release

The release is one version of a product, as its users receive it. It
gives the heading and the list of the release notes, and every line of
the announcement.

- **product** is the product's name as its users know it, such as
  "Lantern". It is not the name of a repository or a team.
- **version** is the version the users install, such as "2.4.0". It names
  the release notes' file, `release-2.4.0.md`.
- **date** is the day the release shipped, written YYYY-MM-DD. It is not
  the day the notes were written.
- **highlight** is optional: one or two sentences on what matters most,
  written for a user deciding whether to upgrade. Leave it out when no
  change stands above the others.
- **changes** lists every change a user would notice, at least one, in
  the order the notes list them. Each has a **kind**, `added`, `changed`
  or `fixed`, and a **text**: one sentence saying what is different, in
  the user's words rather than the code's. "Exports keep their column
  order" is a good text; "Refactor the CSV writer" is not.
```

Write it for someone who has never seen the package: what each field is, an
example of a good value, and the mistake a supplier is likely to make. The
guide in the manifest points at it, so a supplier reads it before writing
any data.

## B.5 Two templates

Each output renders one template under `templates/`, in the Liquid dialect of
[chapter 7](07-template-environment.md). Every supplied input is a variable
under `inputs`, by its name.

The release notes:

<!-- example: gen/worked-package -->

```liquid path=package/templates/notes.md.liquid
# {{ inputs.release.product }} {{ inputs.release.version }}

Released on {{ inputs.release.date }}.
{%- if inputs.release.highlight %}

{{ inputs.release.highlight }}
{%- endif %}

## Changes
{% for change in inputs.release.changes %}
- {{ change.kind | capitalize }}: {{ change.text }}
{%- endfor %}
```

The announcement:

<!-- example: gen/worked-package -->

```liquid path=package/templates/announcement.txt.liquid
{{ inputs.release.product }} {{ inputs.release.version }} shipped on {{ inputs.release.date }}, with {{ inputs.release.changes | size }} changes.
{%- if inputs.release.highlight %} {{ inputs.release.highlight }}{% endif %}
Read the release notes: release-{{ inputs.release.version }}.md
```

- `{%-` removes the whitespace before the tag, the line break included, and
  nothing else is ever trimmed
  ([MTS-TPL-014](07-template-environment.md#mts-tpl-014),
  [MTS-TPL-015](07-template-environment.md#mts-tpl-015)). That is how the
  highlight's blank line and paragraph appear only when the highlight is
  supplied.
- An omitted optional field is nil, which is false, so
  `{% if inputs.release.highlight %}` asks whether it was supplied
  ([MTS-TPL-025](07-template-environment.md#mts-tpl-025)).
- `capitalize` and `size` are standard filters; [chapter 8](08-filters-and-tags.md)
  lists the filters METAIS adds.
- Neither output sets `escape`, so nothing is escaped: these are Markdown and
  plain text, not HTML (§7.10).

## B.6 Checking the package

The package is complete enough to check. Checking needs no data: it reads
every template against the schemas, and reports every finding of every step
in one run ([MTS-AUT-001](13-authoring.md#mts-aut-001)).

```sh
metais check release-notes
```

In the first draft, the announcement's first line read the release's `data`
rather than its `date`:

<!-- example: gen/worked-package-check -->

```liquid path=package/templates/announcement.txt.liquid
{{ inputs.release.product }} {{ inputs.release.version }} shipped on {{ inputs.release.data }}, with {{ inputs.release.changes | size }} changes.
{%- if inputs.release.highlight %} {{ inputs.release.highlight }}{% endif %}
Read the release notes: release-{{ inputs.release.version }}.md
```

The check reports it as an error, before any data exists
([MTS-AUT-013](13-authoring.md#mts-aut-013)). This is the finding as the
vector expects it, with the members every conforming tool gives; a tool adds
its own message and hint, and `metais check --json` prints all of them:

<!-- example: gen/worked-package-check -->

```json path=expected/check.json
[
  {
    "code": "template.xref.unknown_field",
    "severity": "error",
    "subject": {
      "kind": "template",
      "name": "templates/announcement.txt.liquid"
    },
    "location": {
      "origin": "package",
      "file": "templates/announcement.txt.liquid",
      "line": 1,
      "column": 87
    },
    "params": {
      "property": "data",
      "declared": [
        "changes",
        "date",
        "highlight",
        "product",
        "version"
      ],
      "declared_count": 5,
      "suggestion": "date",
      "input": "release",
      "output": "announcement"
    },
    "related": [
      {
        "origin": "package",
        "file": "inputs/release.schema.json",
        "pointer": "",
        "note": "where the schema declares it"
      }
    ]
  }
]
```

Its location is the line and column of the reference, `declared` lists the
names the schema does declare, and `suggestion` is the nearest of them.
While the check reports an error the package does not conform
([MTS-GEN-001](01-introduction.md#mts-gen-001)), and pack refuses it with the
same finding and writes nothing
([MTS-AUT-044](13-authoring.md#mts-aut-044)).

Correcting `data` to `date` gives the template of B.5, and the check then
reports no finding at all. A warning would not fail it, but is worth reading:
an unused input, or a template no output renders, is usually a mistake
([MTS-AUT-017](13-authoring.md#mts-aut-017)).

## B.7 An example

An example is a set of inputs and the outputs they should produce, kept
inside the package, so a change to an output is caught before a supplier
sees it ([§13.7](13-authoring.md#137-testing-a-package-against-its-examples)).
It is a directory `examples/<name>/`, whose name follows the rule for an
input's name, so `lantern` and not `first-release`
([MTS-AUT-020](13-authoring.md#mts-aut-020)).

`example.yaml` is optional, and gives the example a title and a description
([MTS-AUT-036](13-authoring.md#mts-aut-036)):

<!-- example: gen/worked-package -->

```yaml path=package/examples/lantern/example.yaml
title: Lantern 2.4.0
description: A release with a highlight and one change of each kind.
```

`inputs/` is a data directory: one file per input, named after it, in a form
the input accepts ([MTS-SUP-018](06-supplying-inputs.md#mts-sup-018)). This
one is YAML:

<!-- example: gen/worked-package -->

```yaml path=package/examples/lantern/inputs/release.yaml
product: Lantern
version: 2.4.0
date: 2026-10-20
highlight: Lantern now opens shared boards without an account.
changes:
  - kind: added
    text: Anyone with a board's link can open it read-only.
  - kind: changed
    text: Exports keep the column order you see on screen.
  - kind: fixed
    text: Cards dragged between boards no longer lose their labels.
```

A plain `2026-10-20` and `2.4.0` are strings in METAIS's YAML, never a date or
a number ([MTS-SUP-007](06-supplying-inputs.md#mts-sup-007)), so neither needs
quotes.

`expected/` holds the files the render should write, laid out as the output
directory. A directory is part of a package only through the files beneath
it, so an empty `expected/` is refused
([MTS-PKG-023](02-container.md#mts-pkg-023)): write it with its files.

## B.8 Testing the package

Test renders every example and compares what it writes with `expected/`,
byte for byte ([MTS-AUT-040](13-authoring.md#mts-aut-040)):

```sh
metais test release-notes
```

In the second draft, the expected release notes were written by hand, and
their third line left out a word:

<!-- example: gen/worked-package-test -->

```markdown path=package/examples/lantern/expected/release-2.4.0.md
# Lantern 2.4.0

Released 2026-10-20.

Lantern now opens shared boards without an account.

## Changes

- Added: Anyone with a board's link can open it read-only.
- Changed: Exports keep the column order you see on screen.
- Fixed: Cards dragged between boards no longer lose their labels.
```

The check still reports nothing: an example is the author's own statement of
what the package should write, and the package conforms. Test, though, fails,
locating the first character that differs, line 3, column 10
([MTS-AUT-041](13-authoring.md#mts-aut-041)):

<!-- example: gen/worked-package-test -->

```json path=expected/test.json
[
  {
    "code": "example.output_differs",
    "severity": "error",
    "subject": {
      "kind": "example",
      "name": "lantern"
    },
    "location": {
      "origin": "package",
      "file": "examples/lantern/expected/release-2.4.0.md",
      "line": 3,
      "column": 10
    },
    "params": {
      "path": "release-2.4.0.md",
      "binary": false,
      "cause": "none"
    }
  }
]
```

A tool adds a unified diff from the expected file to the rendered one, and
`metais test` prints it. When a test fails, decide which side is right. Here
the template is, so the expected file is corrected. To see what the
templates write before writing `expected/` at all, render the example's
inputs into a directory outside the package and read the result:

```sh
metais render release-notes --data release-notes/examples/lantern/inputs --out preview
```

Once the render is what the package should write, `metais test release-notes
--update` records it as the example's expected files; read them before you
keep them, because an expected file nobody read proves nothing.

These are the expected files of the finished package:

<!-- example: gen/worked-package -->

```markdown path=package/examples/lantern/expected/release-2.4.0.md
# Lantern 2.4.0

Released on 2026-10-20.

Lantern now opens shared boards without an account.

## Changes

- Added: Anyone with a board's link can open it read-only.
- Changed: Exports keep the column order you see on screen.
- Fixed: Cards dragged between boards no longer lose their labels.
```

```text path=package/examples/lantern/expected/announcement.txt
Lantern 2.4.0 shipped on 2026-10-20, with 3 changes. Lantern now opens shared boards without an account.
Read the release notes: release-2.4.0.md
```

With them, test passes.

## B.9 Packing the package

Pack checks the source directory, tests it, and only then writes it as one
file ([§13.8](13-authoring.md#138-packing-a-package)). There is no way to
skip a step, so the package a supplier receives is one whose check reported
no error and whose examples passed
([MTS-AUT-044](13-authoring.md#mts-aut-044),
[MTS-AUT-045](13-authoring.md#mts-aut-045)).

```sh
mkdir dist
metais pack release-notes --out dist
```

Given an existing directory, pack writes `<name>-<version>.metais` into it,
here `dist/release-notes-1.0.0.metais`
([MTS-AUT-048](13-authoring.md#mts-aut-048)). The packed package begins
with `README.md`, stored, holding the specification's README text
([§2.4.1](02-container.md#241-the-readme),
[MTS-PKG-071](02-container.md#mts-pkg-071)). No author writes it and the
source directory does not hold it; it is there so that a reader given the
file with nothing else sees, in its first bytes, what the file is and where
to find the tool. After it come `.compiled/`, which pack computes from the
files written above, and every one of those files, byte for byte, in
ascending order of their names
([MTS-AUT-047](13-authoring.md#mts-aut-047),
[MTS-PKG-010](02-container.md#mts-pkg-010)). It is a zip archive, so it is
not shown here; the vector `gen/worked-package` holds it as
`expected/out/release-notes-1.0.0.metais`, and a processor opens it.

A supplier reads the packed package without unpacking it:
`metais inspect dist/release-notes-1.0.0.metais` prints its identity, its
guide, its input and its outputs, and the commands to run next.

## B.10 Where to go next

- A package with more than one input, a collection input, outputs repeated
  over a collection or written only when a condition holds:
  [chapter 6](06-supplying-inputs.md) and [chapter 9](09-outputs.md).
- A Word, OpenDocument or EPUB file, or CRLF line endings:
  [chapter 10](10-transforms.md).
- Background material for the supplier, in `references/`, and static files in
  `assets/`: [chapter 2](02-container.md).
- Data to start an example from, generated from the schemas:
  [§13.6](13-authoring.md#136-sample-data).
- Every finding a tool can report, with the rule it reports:
  [chapter 12](12-findings.md).
