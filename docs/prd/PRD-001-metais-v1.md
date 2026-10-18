# PRD-001: METAIS v1 — the package format and its reference tool

**Status:** Draft, for review.
**Date:** 2026-10-18
**Author:** drafted in the 2026-10-18 design session with Ed Funnekotter
**Deciders:** Ed Funnekotter
**Feature:** `metais-v1`
**Design input:** [docs/spec-outline.md](../spec-outline.md)
**Designed by:** the ADRs listed in [§26](#26-design--adrs)

## 1. Overview

A METAIS package is a single file that turns input data into finished files.
It carries the templates that become the files, a data specification saying
exactly which data the templates need and what that data means, and,
optionally, background references and static assets. Whoever supplies the
data — a person, a program, or an AI harness — can open the package, learn
what it needs, gather or write that data however suits them, have it
validated, and render the result.

`metais` is the reference tool. It reads a package without unpacking it,
validates inputs, renders outputs, and gives authors what they need to
write, check, test and pack a package of their own.

The format is specified in a document of its own, with conformance vectors,
so that a second implementation can be written from the specification alone
and proved against it.

Because a package fully describes what must be supplied and what is
produced, it is also a way to define a submission: a standards body or a
regulator can publish the package, and each member or regulated entity
renders its report from it (§4.1, §4.2).

**Why it is needed.** Template tools exist (Cookiecutter, Copier, Helm,
Backstage templates, `dotnet new`), and AI-oriented scaffolders are
appearing, but none combines the four properties this product is built on
(survey of 2026-10-18 in [docs/spec-outline.md](../spec-outline.md)):

1. **A data specification written for whoever supplies the data**, saying
   what each input means and not only its type, because the supplier is
   often an AI that must find or write the data rather than a person
   answering a prompt.
2. **One portable, self-describing file**, read in place and selectively, so
   a harness loads only what it needs.
3. **Background references and assets inside the package**, for the
   supplier and for the render.
4. **Pure rendering**: a package cannot run code or read anything of the
   machine it renders on — not even an environment variable — so an
   unfamiliar package is safe to instantiate.

## 2. Goals

- A supplier that has never seen a package learns, from the package alone,
  what data it needs, what that data means and how it will be used.
- Every input is validated strictly, and every problem is reported in one
  pass, worded so the supplier can correct it.
- The same package and inputs give byte-identical outputs on every platform.
- A package is safe to render whoever wrote it.
- An author goes from an empty directory to a tested, packed package on one
  documented path.
- The format outlives the reference tool: specified, versioned and proved by
  conformance vectors.

## 3. Non-goals

- **Running code from a package**: hooks, scripts, custom filters, post-render
  tasks. Anything that must run after rendering is the caller's.
- **Template languages other than Liquid.**
- **Prescribing where data comes from.** No structured "source" kinds, no
  question plans, no interactive gathering loop. The package explains what
  the data means; how it is gathered is the supplier's affair.
- **Persisted answers or state.** The tool is stateless: inputs in, files
  out. The lifecycle of the data belongs to whatever holds it.
- **Updating previously rendered output** by re-rendering onto it and
  merging changes.
- **Live or client-refreshed output.** Refreshing means rendering again; at
  most a caller shows a rendered output in a frame.
- **Templated binary output.** Templates produce text; binary files reach the
  output only as assets.
- **Fetching packages or inputs over the network**, and a package registry or
  discovery service.
- **A public Go API.** The command line is v1's contract; the Go packages are
  internal.
- **Signing packages.** v1 carries digests, not signatures.
- **An MCP server mode**, and any integration with tera, which is tera's to
  build later.

## 4. Use cases

### 4.1 A conformance report a governing body defines

A body that sets a standard publishes the report its members must submit as a
package. The worked case is the accessibility conformance report (ACR), which
vendors produce from ITI's Voluntary Product Accessibility Template (VPAT)
and buyers require under Section 508 and EN 301 549. The package's inputs
are the product and vendor, the evaluation methods (prose), the edition being
reported (WCAG, Revised 508, EN 301 549 or International), and one table per
standard: a row per criterion, each with a conformance level from a fixed set
(Supports, Partially Supports, Does Not Support, Not Applicable, Not
Evaluated) and remarks. Its references hold the text of every criterion.

An AI harness is asked to produce the ACR for a product. It inspects the
package, reads the guide, and for each criterion reads that criterion's
reference and evaluates the product against it. It supplies the tables as
CSV, writes the remarks, validates, and renders the report. The edition
decides which tables appear. The same inputs also render a machine-readable
copy for buyers that compare reports.

### 4.2 A regulated incident report, filed in stages

The EU's Digital Operational Resilience Act requires a financial entity to
report a major ICT-related incident three times — an initial notification,
an intermediate report and a final report — on one template whose glossary
(Implementing Regulation (EU) 2025/302, Annex II) numbers every field, gives
its type, and says whether it is mandatory at each stage. A package carries
that template with the glossary as its schemas. The stage is an input, and
which fields are required follows from it: an initial notification that
omits a field required only in the final report is valid, and a final report
that omits it is a finding. Each stage's report updates what the previous
one said, from the same input set amended.

### 4.3 A small UI widget filled by a program

A status widget package renders one HTML fragment from a JSON object. No AI
is involved: a scheduled workflow fetches the data, calls `metais validate`
and `metais render`, and publishes the fragment. A minimal package — one
manifest, one schema, one template — must cost no more than this.

### 4.4 A multi-file static website

A site package renders one page per article from a directory of Markdown
files with frontmatter (a collection input), plus an index page and a feed
that both read the same collection and a shared site configuration. Its
stylesheet and scripts ship as assets.

### 4.5 A git repository scaffold

A package renders a complete repository — source files, configuration, a
README, CI workflow — from a project description. Some files appear only when
an option is chosen; file and directory names are derived from the inputs.

### 4.6 An agent exploring an unfamiliar package

A harness is handed a large package it has never seen. It reads the summary,
then the outline of one reference, then twenty lines of it found by a search,
and the schema of the one input it is asked to supply — never more of the
package than the task needs.

### 4.7 Rendering into a pipeline

A caller wants the outputs as a stream rather than a directory — to hand
them to another program, or to store them as one artifact. It renders to
standard output and receives a deterministic archive, or the bytes of the one
output it named.

### 4.8 Authoring a package

An author starts from `metais init`, writes schemas and templates, and adds
two examples, each with inputs and the outputs they should produce. `metais
check` finds a template that uses a field no schema declares; `metais test`
renders the examples and compares them; `metais pack` writes the package.

## 5. Concepts and terminology

- **Package** — the single file a consumer receives, `<name>.metais`.
- **Source directory** — the editable tree an author works in; packing it
  produces a package.
- **Manifest** — the package's description: identity, versions, the guide
  for suppliers, its inputs and its outputs.
- **Input** — a named slot in the data specification, with a schema and a
  meaning.
- **Collection input** — an input supplied as a directory of files, which
  becomes an array.
- **Template** — a Liquid file that renders an output. **Partial** — a
  template rendered from inside another.
- **Output** — a file a render produces, declared by the manifest.
- **Reference** — background material in the package, for the supplier.
- **Asset** — a static file the render copies to the output.
- **Example** — a set of inputs bundled with a package, optionally with the
  outputs they must produce.
- **Supplier** — whoever provides the inputs. **Author** — whoever writes the
  package.
- **Finding** — one problem validation or checking reports.
- **Specification version** — the version of the METAIS format a package
  targets. **Package version** — the version of one package.

## 6. Requirement R1 — one file carries everything

1. **A package is a zip archive**, named `<name>.metais`, holding the
   manifest, the data specification, the templates, and optionally
   references, assets and examples, in a layout the specification fixes.
2. **A minimal package is small**: a manifest, one schema and one template
   are enough.
3. **A package is self-describing.** Nothing outside it is needed to read,
   validate or render it.
4. **A packed package carries what consumers need precomputed**: the
   manifest in canonical JSON, one self-contained schema combining every
   input, a SHA-256 digest of every entry and of the package, and an index of
   the references.
5. **The tool verifies those digests** when it reads a packed package, and
   refuses one whose content does not match them.

## 7. Requirement R2 — a package is read in place

1. **Every part of a package can be listed, read, outlined and searched
   without unpacking it to disk**: a summary of the package; the entries
   under a path; a file, or a range of its lines; the heading tree of a
   Markdown file; a regular-expression search with line numbers and context;
   the schema of one input or of all of them.
2. **Selected entries can be extracted** to a directory on request.
3. **Every command can print JSON** in a documented, versioned shape, so a
   harness or a program never parses prose.
4. **What a supplier reads first is designed**: the summary names the
   inputs with their titles, the outputs, the guide for suppliers, and what
   to read next.

## 8. Requirement R3 — the data specification says what is needed and what it means

1. **Each input has a JSON Schema (draft 2020-12).** Types, required and
   optional fields, formats, enumerations and constraints are all expressed
   there.
2. **Each input explains its meaning in prose**: the schema's titles and
   descriptions, and optionally a longer Markdown document per input saying
   what the data is for, what good data looks like, and how it is used.
3. **The manifest carries a guide for suppliers**: prose about the package
   as a whole and how its inputs relate.
4. **The package does not prescribe where data comes from.** Guidance about
   sources, if any, is prose.
5. **Examples are part of the specification of the data.** A filled example
   is the most concrete statement of what an input expects, so the summary
   points suppliers to them.

## 9. Requirement R4 — inputs in the forms data already takes

1. **An input may be supplied as JSON, YAML, CSV, plain text or Markdown.**
   The manifest says which forms each input accepts.
2. **CSV follows RFC 4180**: UTF-8, a required header row, a comma unless the
   manifest declares another delimiter, and each value converted to the type
   the schema declares for its column.
3. **A collection input is a directory of files** that becomes an array, each
   file validated against the item schema. A Markdown file with frontmatter is
   structured data: its frontmatter fields become the item's fields and its
   body becomes `body`; a frontmatter field named `body` is a finding.
4. **Templates can use an input parsed, or embed its original bytes
   unchanged.**

## 10. Requirement R5 — supplying inputs

1. **Inputs come from a data directory, from the command line, or both.** A
   data directory holds one file per input, named after it, or one directory
   per collection input. On the command line, a single value is set by path
   (`--set`) and a whole input or several inputs as JSON (`--json`).
2. **The precedence is defined exactly**, and a value given on the command
   line wins over a file.
3. **A schema's `default` is applied** to an optional field the supplier
   left out, before validation and rendering.
4. **An optional field that was not supplied and has no default is absent**,
   which a template sees as `nil`, never as an empty string or an empty array.

## 11. Requirement R6 — validation reports everything, usefully

1. **Validation reports every finding in one pass**, never only the first.
2. **Each finding says where and why**: the input, the location in it (file,
   line where there is one, and JSON path), the rule broken, and the title
   and description of the field, so the supplier can correct it without
   opening the package.
3. **Findings are printed for a person, or as JSON.** Validation exits
   non-zero when there is any finding of error severity.
4. **Rendering validates first** and writes nothing if the inputs are
   invalid.

## 12. Requirement R7 — the template language

1. **Templates are Liquid**, in a dialect the specification pins: Liquid's
   core tags and filters as the Shopify reference implementation defines
   them, without the deprecated `include`, plus the METAIS filters and tags
   below. Conditionals (`if`/`elsif`/`else`, `unless`, `case`/`when`), loops
   (`for` with its `else` and modifiers), `assign`, `capture` and whitespace
   control are all available.
2. **Every template sees every input**, as `inputs.<name>`. One input can feed
   many templates, and one template can use many inputs.
3. **Partials** are rendered from the package's templates only.
4. **Standard filters** serve the common cases: at least serializing to JSON,
   YAML and CSV, a Markdown table from an array of objects, HTML escaping,
   JSON safe to place inside an HTML `<script>` element, a URL-safe slug, and
   date formatting.
5. **Standard tags**: embed an input's original bytes; include an asset's
   contents or its output path; and emit provenance (R10).
6. **Escaping is chosen per output.** An output may declare HTML escaping;
   otherwise values render as they are. The escaping filters are always
   available.

## 13. Requirement R8 — outputs

1. **The manifest declares the outputs.** Each names a template and an output
   path, and the path is itself a template.
2. **An output may be repeated over a collection** (`for_each`), writing one
   file per item.
3. **An output may be conditional** (`when`), written only when its
   condition holds.
4. **Assets are copied to declared output paths.**
5. **A shorthand mirrors the templates tree into the output** for authors who
   need no explicit list.

## 14. Requirement R9 — rendering is pure and deterministic

1. **No command runs code from a package**, no command uses the network, and
   a render reads nothing from its host but the package and the supplied
   inputs (R13 item 4).
2. **The same package and the same inputs give byte-identical outputs** on
   every supported platform and in every run.
3. **Text is UTF-8**, line endings are written exactly as the template has
   them, iteration over objects is in a defined order, and nothing depends on
   the locale or the time zone.
4. **The current time is available only when the caller supplies it**, so
   that a render is reproducible.

## 15. Requirement R10 — provenance, on request

1. **A template may emit a provenance comment** in the syntax of its output
   (HTML, Markdown, `#`-comment and others the specification lists).
2. **The author chooses what it records**: any of the package's identity,
   version and digest, the digest of each input, and the tool's version; and
   the render time only when the caller supplied one. The list is closed: no
   path, user name, host name or other fact about the rendering machine is
   ever recorded (R13 item 4).

## 16. Requirement R11 — rendering is safe, to a directory or a stream

1. **Every output path stays inside the output directory**: no absolute path,
   no `..`, no symbolic link out.
2. **Output paths are portable.** Nothing in the format or the tool precludes
   Windows: a path whose name Windows cannot hold (reserved device names,
   reserved characters, a trailing dot or space), or two outputs whose paths
   differ only in letter case, is refused on every platform.
3. **The tool refuses to render into a non-empty directory** unless the
   caller allows replacing existing files. It never deletes a file.
4. **A dry run lists every file the render would write**, with its size and
   digest, and writes nothing.
5. **Outputs can be written to standard output**: as a deterministic archive
   of every output, or, when the caller names exactly one output, as that
   file's bytes alone.

## 17. Requirement R12 — authoring

1. **An author starts from a generated minimal package** and works in a
   source directory, with the manifest in YAML or JSON.
2. **Checking a package needs no data.** It verifies the manifest against the
   specification, compiles every schema, parses every template, confines
   every path, and cross-references the templates against the schemas: a
   field a template uses that no schema declares is an error, and an input no
   template uses is a warning.
3. **Examples are tests.** Each example's inputs are validated and rendered,
   and its outputs, where given, are compared byte for byte.
4. **Packing runs the check and the tests**, then writes a package whose
   bytes depend only on the source directory.
5. **A package can be unpacked** back into an editable source directory.

## 18. Requirement R13 — a package is untrusted input

1. **Every entry is confined**: no absolute path, no `..`, no link, before it
   is read.
2. **Every resource is bounded**: total and per-entry uncompressed size,
   entry count, compression ratio, path length, template recursion and
   output count. The limits have defaults and the caller may lower them.
3. **A package outside the specification is refused, not approximated**, with
   a finding that names what is wrong.
4. **Nothing of the host reaches a template.** There is no way to read an
   environment variable from a template: no variable, filter, tag or
   manifest field exposes the environment, and the tool never expands one
   into an input, a path or a default. The same holds for everything else
   the host knows — files outside the package and the inputs the caller
   named, the working directory, the user, the host name, the process, and
   the clock (R9 item 4). The only data a template sees is the package's own
   content and the inputs the caller supplied, so a package cannot read a
   secret from the machine that renders it, nor write one into its output.
   A caller who wants a value from the environment puts it in an input
   explicitly, where it is validated and visible like any other.

## 19. Requirement R14 — versions

1. **The specification is versioned** (`1.0` for v1). A package's manifest
   names the specification version it targets.
2. **A package carries its own version**, a semantic version its author
   assigns.
3. **The tool states which specification versions it implements**, accepts a
   package of any of them, and refuses a package of a major version it does
   not implement, saying so.
4. **Within a major version the specification only adds.** A package valid
   under `1.0` stays valid, and renders the same, under every later `1.x`.

## 20. Requirement R15 — the specification is a document with a proof

1. **The specification is Markdown under `spec/`**, normative in RFC 2119
   terms, with the manifest's JSON Schema published beside it.
2. **Conformance vectors** — a package, its inputs, and the outputs or the
   findings a conforming tool must produce — prove every normative statement
   that a program can observe.
3. **The reference tool passes every vector**, and the vectors are part of
   its test suite.

## 21. Requirement R16 — the golden paths

1. **Consuming a package has one documented path**: inspect, read what the
   summary points to, get the schema, supply the data, validate, render.
2. **Authoring a package has one documented path**: init, edit, check, test,
   pack.
3. **The tool's help text is authoritative** for every command and flag, and
   every refusal names the next step.

## 22. Requirement R17 — portable and independent

1. **The tool is one static binary.** darwin and linux on amd64 and arm64 are
   v1's supported targets; nothing in the format or the tool precludes
   Windows (R11 item 2).
2. **METAIS depends on nothing of tera's.** The format and the tool know
   nothing about it.

## 23. Dependencies

- JSON Schema draft 2020-12.
- Liquid, as defined by the Shopify reference implementation.
- RFC 4180 (CSV), YAML 1.2, CommonMark with YAML frontmatter.
- The zip file format (PKWARE APPNOTE) and the POSIX tar format.
- RFC 2119 for the specification's normative language.
- The Go libraries the implementation ADR names, each only if it passes the
  conformance vectors; where none does, METAIS implements its own.

## 24. Risks

- **Liquid implementations disagree at the edges** (truthiness, filters,
  whitespace). Mitigation: the dialect is pinned and every observable rule
  has a vector.
- **JSON Schema validators disagree**, particularly on `format`. Mitigation:
  the specification says which keywords are asserted, and vectors cover each.
- **Determinism leaks**: archive metadata, map order, float and date
  formatting, YAML's implicit typing. Mitigation: each is fixed by the
  specification and proved by a vector run on every supported platform.
- **Liquid's truthiness surprises authors**: an empty string is true.
  Mitigation: absence is `nil` (R5 item 4), and the authoring documentation
  states the rule with the idiom to use.
- **Scope creep toward a programming language.** Mitigation: the non-goals;
  computation belongs in the data.
- **A governing body's template is its copyright.** ITI's VPAT is ITI's, and
  regulators publish under their own reuse terms. Mitigation: the example
  packages are checked against each source's terms before they ship, and are
  written from material that may be reused (W3C's criterion text, GSA's
  OpenACR catalog, EU legislation under its reuse policy) or in this
  project's own words.

## 25. Success criteria

1. **SC1** — An AI harness given only the binary, the conformance report
   package of §4.1 and access to a product produces a valid ACR for it,
   reading nothing of the package beyond what the summary and the commands it
   points to show.
2. **SC2** — The incident report package of §4.2 accepts each stage's report
   with exactly that stage's mandatory fields, and reports every mandatory
   field a stage omits.
3. **SC3** — A program renders the widget package of §4.3 from JSON with no
   harness, in a single validate and render.
4. **SC4** — The packages of §4.1 to §4.5 ship in this repository as
   examples, and pack with their tests green.
5. **SC5** — Every conformance vector passes on every supported target, and
   rendering the same vectors on two of them gives byte-identical output.
6. **SC6** — A corpus of hostile packages (path traversal, links, zip bombs,
   Windows-invalid and case-colliding paths, digest mismatches) is refused
   in full, with nothing written outside the output directory.
7. **SC7** — Rendering every conformance vector and every hostile package
   with sentinel values planted in the environment, the working directory
   and the home directory never puts a sentinel into any output, finding or
   message. The test is pinned red first, against a render that does read
   the environment.
8. **SC8** — Validating an input set with errors planted throughout reports
   every planted error in one run.
9. **SC9** — An author working only from the specification, without the
   tool's source, writes a package that the tool checks, tests and packs
   cleanly.

## 26. Design → ADRs

| Requirements | ADR |
|---|---|
| R1 | ADR-001 — the container and its layout |
| R1, R14 | ADR-002 — the manifest and versioning |
| R3, R4, R5 | ADR-003 — the data specification |
| R7, R10 | ADR-004 — the template environment |
| R8, R9, R11 | ADR-005 — outputs and rendering |
| R6 | ADR-006 — validation and findings |
| R12 | ADR-007 — authoring |
| R2, R16 | ADR-008 — the command line and its machine output |
| R13 | ADR-009 — the security model |
| R17 | ADR-010 — the implementation |
| R15 | ADR-011 — the specification document and conformance |
