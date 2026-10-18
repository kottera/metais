# METAIS specification — outline

**Status:** outline, 2026-10-18. Records the decisions made so far and the
shape of each chapter of the specification. Each section becomes a chapter
of `spec/`; the open questions at the end are settled before that chapter is
written.

## 1. Scope

A METAIS package turns input data into one or more output files. The
specification defines the package, the data specification inside it, the
template environment, how outputs are produced, and what a conforming tool
must do. The reference tool is `metais`, written in Go.

Use cases the format must serve, smallest to largest:

- a Markdown report filled with a few tables;
- a small UI widget, form or dashboard filled by a program or an AI harness;
- a multi-file static website;
- a complete git repository.

Design stance:

- **Strict on the data, loose on where it comes from.** Every input is
  validated against its schema. Where the data comes from — a person, a
  system, a document, an AI writing it — is the supplier's business; the
  package guides with prose, it does not prescribe.
- **Rendering is pure.** No hooks, no code execution, no network. The same
  package and the same inputs produce byte-identical outputs.
- **Usable without unpacking.** Every part of a package can be listed, read
  and searched in place, so a harness loads only what it needs.

## 2. Terminology

- **Package** — the zip a consumer receives (`<name>.metais`).
- **Source directory** — the editable tree an author works in; `pack`
  produces a package from it.
- **Input** — a named slot in the data specification, with a schema and a
  meaning.
- **Template** — a Liquid file under `templates/`.
- **Output** — a file the render writes, declared in the manifest.
- **Supplier** — whoever provides the inputs: a person, a program, an AI
  harness.

## 3. The package container

- A zip archive. Its central directory gives random access, so one file can
  be read without unpacking the others.
- Layout:

  ```
  manifest.yaml        authored manifest (source directory)
  inputs/              JSON Schemas and meaning documents per input
  templates/           Liquid templates and partials
  references/          optional background material
  assets/              optional static files
  examples/<name>/     optional example inputs, and expected outputs
  .compiled/           written by pack, never authored
  ```

- `.compiled/` holds the manifest as canonical JSON, the combined input
  schema with internal `$ref`s inlined, a digest of every file, and a
  references index.
- Packing is deterministic: sorted entries, fixed timestamps, so the same
  source gives the same bytes.
- Limits a tool enforces: total uncompressed size, entry count, compression
  ratio, path length.

## 4. The manifest

- Authored as YAML or JSON; `pack` writes canonical JSON to `.compiled/`, so
  a consumer needs only a JSON parser.
- Carries: the spec version, the package's name, version, title and
  description, a guide for suppliers (prose, Markdown), the inputs, the
  outputs, and any declared defaults.
- A minimal package is small: one manifest, one schema, one template. A UI
  widget must not pay a website's overhead.

## 5. The data specification

- **Inputs are named slots in one shared namespace.** Every template sees
  every input as `inputs.<name>`. No template is bound to a data file: one
  input can feed many templates, and one template can use many inputs.
- **Types are JSON Schema 2020-12.** Each input names its schema file.
- **Meaning is prose.** `title` and `description` in the schema, plus an
  optional Markdown document per input for longer guidance: what the data
  is for, what good data looks like, worked examples. No structured
  "source" keywords.
- **Accepted formats per input:** JSON, YAML, CSV (rows become objects,
  coerced by the schema's types), plain text and Markdown.
- **Collection inputs.** An input may be a directory of files that becomes
  an array, each file validated against the item schema. A Markdown file
  with frontmatter is structured: the frontmatter becomes fields and the
  body becomes a field.
- **Parsed and raw.** Templates can use an input's parsed value, or embed
  its original bytes unchanged.
- **Supplying inputs.** A directory (`<input-name>.<ext>` per slot, or a
  directory per collection), inline values (`--set name=value`,
  `--json '{...}'`), or both. Inline values win over files; the precedence
  is defined exactly.

## 6. The template environment

- **Liquid only.** The specification pins a Liquid dialect and feature set
  so that independent implementations agree.
- Variables: `inputs`, `item` (inside a fan-out), `package` (manifest
  metadata).
- Partials: `{% render %}` from `templates/`.
- Standard filters, at least: `json`, `yaml`, `csv`, `markdown_table`,
  `escape`, `json_script` (safe JSON inside an HTML `<script>`), `slug`,
  date formatting.
- Standard tags:
  - `{% embed "<input>" %}` — an input's original bytes;
  - `{% asset "<path>" %}` — an asset's contents, or its output path;
  - `{% provenance [parts...] %}` — an optional comment in the output's own
    syntax (HTML, Markdown, `#`, …) recording the package id, version and
    digest, input digests and tool version. A render time appears only when
    the caller supplies one, so outputs stay deterministic.
- Escaping is opt-in per output (`escape: html`); the escaping filters are
  always available.

## 7. Outputs

- The manifest lists outputs. Each names a template and an output path; the
  path is itself a template.
- `for_each: <expression>` writes one file per item, with `item` bound.
- `when: <expression>` includes the output only when true.
- `escape: html | none`.
- Assets are copied to declared output paths.
- A shorthand mirrors `templates/` into the output tree for authors who need
  no explicit list.

## 8. Validation

- Reports **every** missing or invalid value in one pass, never only the
  first.
- Each finding carries the input, the JSON path, the rule broken, and the
  field's title and description, so a supplier — often an AI harness — can
  correct it without opening the package.
- Human-readable and JSON (`--json`) reports; a non-zero exit when invalid.

## 9. Rendering

- Validates first; renders nothing if any input is invalid.
- Every output path must resolve inside the output directory (no `..`, no
  absolute paths, no symlinks out).
- `--dry-run` lists the files it would write, with sizes and digests.
- Deterministic: identical package and inputs give identical bytes.

## 10. Authoring

- `init` — a minimal source directory.
- `check` — the package without data: the manifest against the
  specification's own schema; every schema compiles; every template parses;
  every path stays inside the package; and a **cross-reference check** — a
  field a template uses that no schema declares is an error, an input no
  template uses is a warning.
- `test` — renders each example and compares it with its expected outputs.
  Examples double as the best guidance a supplier can have.
- `pack` — `check` and `test`, then the deterministic zip with
  `.compiled/`.
- `unpack` — a package back to an editable source directory.

## 11. The CLI

Every command can print JSON. None unpacks a package to disk unless asked.

| Command | Does |
|---|---|
| `inspect <pkg>` | the manifest and a summary of inputs, outputs, references and assets |
| `ls <pkg> [path] [-r]` | list entries |
| `read <pkg> <path> [--lines a-b]` | print a file, or part of it |
| `outline <pkg> <path>` | the heading tree of a Markdown file |
| `grep <pkg> <regex> [path...] [-C n]` | search, with line numbers and context |
| `schema <pkg> [input]` | one input's schema, or the combined schema |
| `extract <pkg> <glob> --to <dir>` | unpack selected entries |
| `validate <pkg> [--data-dir D] [--set k=v] [--json J]` | check inputs |
| `render <pkg> --out D [inputs...] [--dry-run]` | produce the outputs |
| `init`, `check`, `test`, `pack`, `unpack` | authoring |

An MCP server mode can follow later; it is not in v1.

## 12. Security

- No code execution of any kind: no hooks, no scripts, no custom filters
  shipped inside a package.
- No network access during any command.
- Zip-slip and path-traversal checks on every entry and every output path.
- Size, ratio and entry-count limits against zip bombs.
- Escaping is the author's choice; the specification states plainly that
  embedding untrusted data in HTML without `escape: html` is unsafe.

## 13. Versioning and conformance

- The manifest names the specification version it targets; a tool refuses a
  major version it does not implement.
- The repository ships conformance vectors: a package, its inputs, and the
  expected outputs or expected validation findings. A second implementation
  proves itself against them.

## Decided out of scope for v1

- Structured "source" kinds per input (where the data comes from).
- A gathering plan or interactive question loop; a supplier workflow, such
  as a tera tool-only workflow, populates the data.
- A persisted answers or state file; the data's lifecycle belongs to
  whatever holds it.
- Live or client-refreshed dashboards; at most an iframe preview of a
  rendered output.
- Template engines other than Liquid.
- Hooks.

## Open questions

1. The Liquid dialect to pin, and how far the Go library in use matches it.
2. The manifest's file name and the exact precedence rules for inputs.
3. Writing into a non-empty output directory: refuse, overwrite, or a flag.
4. CSV details: header handling, delimiter, how the schema coerces types.
5. The field name for a Markdown collection item's body.
6. License (probably open source).
