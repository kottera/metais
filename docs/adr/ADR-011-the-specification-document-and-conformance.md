# ADR-011: The specification document and conformance — Markdown chapters under `spec/` whose every normative sentence is an `MTS-` statement, one living text per major released by tag, and a black-box suite of vectors that each cite the statements they prove, run in a sentinel sandbox on every target

**Status:** Accepted 2026-10-18 by Ed Funnekotter; its proposed rulings are ratified in [RULINGS-v1](RULINGS-v1.md).
[Amendment A1](#amendments) (chapter 2's statement area is `PKG`, not `CON`)
is proposed by PLAN-2026-10-metais-v1 Wave 3 and awaits Ed Funnekotter's
ratification; the passages it changes are marked *(A1, proposed)*.
**Date:** 2026-10-18
**Deciders:** Ed Funnekotter
**Designs:** PRD-001 R16 (items 1–3); the document side of R15 (items 1 and
4: how the text labels and grows its revisions); the runs behind SC5, SC6 and
SC7; and the sufficiency SC9 asks of the text.
**Depends on:** ADR-001 (the zip profile's fixed header fields and entry
order, which container comparison checks; the content-defined package
digest); ADR-002 (`metais: "MAJOR.MINOR"`, what a minor may add, one manifest
schema per minor and its `$id`); ADR-003 (the supply arguments a vector
passes); ADR-004 (the dialect a delegated Liquid suite is configured to;
provenance's tool-version field); ADR-005 (container and stream outputs, the
tar profile, the dry-run listing, portable-path rules, staging); ADR-006 (the
finding object, its stable members, codes and order); ADR-007 (the
authoring operations; whether sample data is byte-normative); ADR-008 (the
command line the reference adapter drives, its JSON envelope, the
environment the tool reads); ADR-009 (the hostile corpus's content, limit
names, the host-isolation proof beyond the sentinel run); ADR-010 (where the
runner lives, how the tool is built, the release procedure).

## Context

### What the PRD requires

- **R16.1** — the specification is Markdown under `spec/`, normative in RFC
  2119 terms, with the manifest's JSON Schema published beside it.
- **R16.2** — conformance vectors — a package, its inputs, and the outputs or
  findings a conforming tool must produce — prove every normative statement a
  program can observe.
- **R16.3** — the reference tool passes every vector, and the vectors are
  part of its test suite.
- **R15.1, R15.4** — the specification is versioned (`1.0` for v1), and
  within a major it only adds.
- **SC5** — every vector passes on every supported target, and rendering the
  same vectors on two of them gives byte-identical output.
- **SC6** — a hostile corpus is refused in full, with nothing written outside
  the output directory.
- **SC7** — with sentinels planted in the environment, the working directory
  and the home directory, no sentinel reaches any output, finding or message,
  over every vector and every hostile package; pinned red first.
- **SC9** — an author working from the specification alone, without the
  tool's source, writes a package the tool checks, tests and packs cleanly.

`AGENTS.md` adds two rules that bind the design: a change to what a package
means changes the specification **and the vectors that prove it in the same
change**, and the conformance vectors run as part of `go test`. There is no
CI; `go test -race ./...` on a developer's machine is the gate.

### What the siblings have already fixed

- **ADR-001** compares packages and containers in the vectors **by entries —
  names, order, method and content — never by file bytes**, because a second
  implementation may use another deflate encoder; every other header field is
  a constant of its profile. The **package digest is defined over content**
  (the SHA-256 of the digest list), so every conforming packer produces the
  same one. One build of the tool is byte-identical across platforms.
- **ADR-002** names the specification version in the manifest member
  `metais` as a `MAJOR.MINOR` string; a minor adds only constructs every
  earlier minor refuses; **the specification marks every addition with the
  minor that introduced it** and the tool carries the same table; one
  manifest schema per minor, `$id`
  `https://metais.org/schemas/1.0/manifest.schema.json` (subject to its
  domain ruling), embedded byte-identical in the tool.
- **ADR-008** proposes (its ruling 7) that **the command line and its JSON
  formats are the reference tool's contract, not the format's**: a second
  implementation needs the format, not `metais`'s flags. It promises that
  `validate --json`, `render --out <dir>` and `render --out -` are stable
  within their `/1` formats; that the tool reads only `NO_COLOR` from the
  environment; and that it never prints a path the caller gave, an absolute
  path it constructed, the working directory, the home directory, the user
  or the host name.

### How existing conformance suites are built (verified 2026-10-18)

| Suite | Unit and layout | How a case cites the specification | How the result is matched | Runner |
|---|---|---|---|---|
| JSON Schema Test Suite | one JSON file per keyword per draft directory; each holds test cases (`schema` + `tests[]` of `data`, `valid`) | optional `specification` array naming a section of the core or validation draft, an RFC section or a quote | a boolean: valid or not | left to the implementer; "should require only a JSON parser" |
| CommonMark | examples embedded in `spec.txt` between fences of 32 backticks with the word `example`, input and output separated by `.` | the section the example sits in; examples are numbered by a counter in document order | HTML, normalized before comparison unless `--no-normalize` | `test/spec_tests.py --program $PROG` |
| toml-test | a `.toml` file and its expected `.json` per case, under `valid/` and `invalid/`; `files-toml-1.0.0` and `files-toml-1.1.0` list the cases of each TOML version | none | invalid: a non-zero exit; valid: tagged JSON on stdout | the `toml-test` binary runs any decoder that reads stdin and writes stdout |
| YAML Test Suite | a directory per case with fixed file names (`in.yaml`, `in.json`, `out.yaml`, `test.event`, `error`, `===` for the name); opaque four-character ids | none; a closed tag registry (`doc/tag-usage.md`) | the event stream, or presence of `error` | the implementer's |
| test262 | one JavaScript file per case with YAML frontmatter | `esid`, the anchor of the specification section in the generated HTML | for a negative case, `negative: {phase, type}` — the error's phase and constructor name, never its message | `test262-harness` and others |
| Shopify liquid-spec | YAML behaviour specs in suites (`basics`, `liquid_ruby`, `liquid_ruby_lax`, `partials`, …) with template, environment, filesystem and expected output or error | per-suite, with feature gates (`missing_features`) | output or error | `liquid-spec run`, with a JSON-RPC adapter (`initialize`, `compile`, `render`, `quit`) so any language's engine runs as a subprocess |

Three lessons carry into this design. **Positional identifiers drift**:
CommonMark's example numbers come from a counter, so inserting an example
renumbers every later one. **Citations to the specification make a suite
auditable**: test262's `esid` and the JSON Schema suite's `specification`
field let a reader go from a case to the text it proves. **Errors are matched
by kind, never by wording**: test262 matches the error's type, toml-test the
exit code. And one fact changes scope: **Shopify publishes liquid-spec, an
acceptance suite for Liquid itself** (MIT licence, head
`84bf25e0edbca5f7e530574788b0f875ab331f50` on 2026-10-18), which already
holds thousands of cases for the dialect METAIS pins.

### Facts that constrain a vector tree (verified)

- **git stores neither empty directories nor, portably, symbolic links**, and
  on a Windows checkout converts line endings unless the path's `text`
  attribute is unset (`-text`: "do not attempt any end-of-line conversion").
- **macOS's default volume is case-insensitive**: Apple offers "APFS
  (Case-sensitive)" as a separate format, and on this machine creating `A`
  and then `a` in one directory leaves one file. A committed tree holding two
  names that differ only in case cannot be checked out there.
- **txtar** (Go's text archive) cannot store binary data, file modes or links,
  and a parser treats a missing final newline as present, so it cannot
  express a file that ends without one.
- **GitHub rewrites an HTML `id` to `user-content-` plus the lower-cased
  value** (observed through the GFM rendering API: `<a id="MTS-OUT-012">`
  renders as `id="user-content-mts-out-012"`), so mixed-case anchors are
  fragile and lower-case ones are not.
- **JSON Schema 2020-12**: `$id` must resolve to an absolute URI without a
  non-empty fragment, and "is an identifier and not necessarily a network
  locator".
- **Go**: `//go:embed` patterns may not contain `.` or `..` elements, so a
  package under `internal/` cannot embed `spec/schemas/` directly; the
  language "may combine multiple floating-point operations into a single
  fused operation", so arm64 and amd64 builds of one source can differ in
  float results; a version tag of a module in a subdirectory is prefixed by
  that subdirectory, so a tag beginning `spec/` reads as a version of a module
  at `spec/`; `os.ProcessState` reports user and system CPU time, and
  `SysUsage` returns `getrusage`'s structure, whose `ru_maxrss` is **bytes on
  darwin and KiB on Linux**.
- **The race detector** typically raises memory use 5–10× and execution time
  2–20×.
- **Rosetta 2** runs Intel binaries on Apple silicon, so a darwin/amd64 build
  can be executed on an arm64 Mac.
- **Go's `zipinsecurepath` and `tarinsecurepath` GODEBUG settings** decide
  whether `archive/zip` and `archive/tar` reject insecure names, and an
  environment `GODEBUG` overrides the binary's own setting.

## Decision

### D1 — The document: one Markdown file per chapter under `spec/`

**`spec/` holds the specification as numbered chapter files, each owning one
or more statement areas, plus a front page, a changelog, an errata list and a
generated statement index.**

1. The tree:

   ```
   spec/
     README.md                    front page: version, status, contents (informative)
     01-introduction.md           scope, conformance classes, terminology, notation, BCP 14
     02-container.md
     03-manifest.md
     04-versions.md
     05-data-specification.md
     06-supplying-inputs.md
     07-template-environment.md
     08-filters-and-tags.md
     09-outputs.md
     10-transforms.md
     11-rendering.md
     12-findings.md
     13-authoring.md
     14-security.md
     15-conformance.md
     A-references.md              normative and informative references
     B-worked-package.md          a package from empty directory to packed (informative)
     CHANGELOG.md
     ERRATA.md
     statements.json              generated index of every statement (D3 item 7)
     findings/registry.json       the finding-code registry (ADR-006 D3 item 3)
     schemas/                     published JSON Schemas (D7)
     conformance/                 the vector suite (D9–D16)
   ```

2. **The chapters, their statement areas and the ADR that designs each:**

   | File | Subject | Areas | Designed by |
   |---|---|---|---|
   | `01-introduction.md` | scope, conformance classes, terminology, notation, the BCP 14 statement | `GEN` | ADR-011 |
   | `02-container.md` | the zip profile, entry names, the layout, `.compiled/`, digests and their verification | `CON` *(A1, proposed)* | ADR-001 |
   | `03-manifest.md` | the manifest's structure and fields, the YAML and JSON profiles, the canonical form | `MAN` | ADR-002 |
   | `04-versions.md` | specification and package versions, compatibility, what a tool states | `VER` | ADR-002 |
   | `05-data-specification.md` | the `inputs` section, the JSON Schema dialect and asserted keywords and formats, meaning documents, the combined schema | `DAT` | ADR-003 |
   | `06-supplying-inputs.md` | input forms and their parsing, CSV coercion, collections, the data model templates see, supply, precedence, defaults, absence | `SUP` | ADR-003 |
   | `07-template-environment.md` | the pinned Liquid dialect, variables, partials, truthiness, `blank`/`empty`, undefined names, whitespace, template errors | `TPL` | ADR-004 |
   | `08-filters-and-tags.md` | the METAIS filter catalog; `embed`, `asset` and `provenance` with its per-output syntaxes | `FLT`, `TAG` | ADR-004 |
   | `09-outputs.md` | the `outputs` section, path templates, `for_each`, `when`, assets, the mirror shorthand | `OUT` | ADR-005 |
   | `10-transforms.md` | zip containers with the OOXML, ODF and EPUB settings, CRLF conversion, output validation | `TRN` | ADR-005 |
   | `11-rendering.md` | determinism, staging and all-or-nothing writes, overwrite, dry run, stream output and its tar profile; portable paths | `RND`, `PTH` | ADR-005 |
   | `12-findings.md` | the finding model, severities, the code catalog, locations, the report | `FND` | ADR-006 |
   | `13-authoring.md` | check and the cross-reference, the split-run finding, examples and test, pack, unpack; sample data | `AUT`, `SMP` | ADR-007 |
   | `14-security.md` | the threat model (informative), resource limits and their defaults, confinement, host isolation | `SEC` | ADR-009 |
   | `15-conformance.md` | conformance classes and claims, the operations, the suite and how it is read | `CNF` | ADR-011 |

   **There is no command-line chapter**: the command line is the reference
   tool's contract (ADR-008), documented in its help and in `docs/cli/`. ADR-010
   has no chapter either: the specification names no implementation.

3. **An area code is three upper-case letters, belongs to exactly one
   chapter, and is never reassigned.** The registry is a table in
   `01-introduction.md`. A chapter may gain an area; an area never moves to
   another chapter, because statement identifiers carry it.

4. **Each chapter opens with its title (`# 9. Outputs`), one line naming its
   status and areas (`**Normative.** Areas: OUT.`), and an informative summary
   paragraph.** Sections are numbered headings (`## 9.3 Fan-out`). Section
   numbers and heading anchors may change between revisions; nothing outside
   the chapter cites them except by link text. **The stable citation points
   are statement identifiers (D3).**

5. **The front page (`README.md`) carries a fixed status block** at its top:

   ```
   Version: 1.0
   Status: Draft            (or: Released 2027-01-15, tag spec-1.0)
   Latest release: none     (or: 1.0, tag spec-1.0)
   ```

   followed by the contents (every chapter file, linked) and a paragraph on
   how to read the document. A tripwire (D14) checks the block against
   `CHANGELOG.md`.

6. **Notation** is fixed in `01-introduction.md`: grammars in ABNF (RFC 5234)
   with RFC 7405's `%s` for case-sensitive literals; JSON locations as JSON
   Pointers (RFC 6901); byte values as `0x` hexadecimal; code points as
   `U+XXXX`; JSON per RFC 8259.

### D2 — Normative language

**The BCP 14 statement appears once, in `01-introduction.md`, in RFC 8174's
wording; only five keyword forms are used; and every keyword sits inside a
numbered statement.**

1. The text, verbatim from RFC 8174 §2:

   ```
   The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT",
   "SHOULD", "SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED", "MAY", and
   "OPTIONAL" in this document are to be interpreted as described in
   BCP 14 [RFC2119] [RFC8174] when, and only when, they appear in all
   capitals, as shown here.
   ```

2. **The document uses only `MUST`, `MUST NOT`, `SHOULD`, `SHOULD NOT` and
   `MAY`.** `REQUIRED`, `SHALL`, `SHALL NOT`, `RECOMMENDED`, `NOT
   RECOMMENDED` and `OPTIONAL` are synonyms; one spelling per meaning keeps
   the lint (D14) and the reader's scan simple.
3. **A requirement on a program is `MUST` or `MUST NOT`.** A `SHOULD`
   addressed to a processor would make two behaviours conforming that the
   vectors cannot both accept; R14.3's "refused, not approximated" leaves no
   room for it. **`SHOULD` and `SHOULD NOT` address people** — authors and
   suppliers — and are therefore never observable (D3 item 5). **`MAY`
   grants a processor a freedom** (a compressor, for instance), and every
   vector must pass under each choice it allows.
4. **Every all-capitals keyword in the document is inside a statement**
   (D3); none appears in a note, an example, an informative section, the
   front page, the changelog or the errata. Normative text without a keyword
   — a grammar, a table of fixed values — is normative where a statement
   adopts it ("**MTS-PTH-003** A name MUST NOT match the `reserved-name` rule
   of §11.6").
5. **Informative material is marked**, and only these forms are
   informative: a chapter whose status line says `**Informative.**`; a
   section whose heading ends with `(informative)`; a block quote beginning
   `> **Note.**`; an example (D4). Everything else in a normative chapter is
   normative.

### D3 — Statements and their identifiers

**Every testable rule is one statement with an identifier `MTS-<AREA>-<NNN>`
that is assigned once, never reused and never renumbered.**

1. **Syntax.** A statement begins a paragraph, a list item or a table row
   with an anchor and the bold identifier, optionally followed by an
   attribute list:

   ```
   <a id="mts-out-012"></a>**MTS-OUT-012** An output's path template MUST render to a relative path.

   <a id="mts-flt-031"></a>**MTS-FLT-031** *(since 1.1)* The `wrap` filter MUST …

   | <a id="mts-flt-004"></a>MTS-FLT-004 | `slug` | … |
   ```

   As ABNF:

   ```
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

   The bold markers are omitted inside a table cell; nothing else varies.
2. **A statement's scope** is the paragraph, list item or table row it opens,
   together with any list nested inside that item. A statement states one
   requirement: a sentence holding two requirements that can fail
   independently is two statements.
3. **Numbers are assigned in order of addition within an area**, never by
   position, so inserting a statement mid-section takes the next free number
   and renumbers nothing. A number is never reused, even after its statement
   is withdrawn.
4. **`since`** names the minor that introduced the statement (ADR-002 D7
   item 3); absent, it is `1.0`. Within a major a statement's meaning never
   changes; its text changes only by an erratum (D6), which adds `amended by`.
   A statement an erratum withdraws keeps its text and identifier, marked
   `withdrawn by`.
5. **`not observable`** marks a statement no black-box run can check: every
   `SHOULD` and `SHOULD NOT` statement (they address people), and a
   requirement whose evidence is not in a program's outputs (for example that
   a processor opens no network connection). A `not observable` requirement
   on a processor is followed by a note naming how it is proved for the
   reference tool (ADR-009's host-isolation proofs are the usual answer).
   Everything else is observable, including every rule about a package:
   a package rule is observed through a processor refusing a package that
   breaks it.
6. **Statement kinds follow from the keywords**: a statement containing
   `MUST` or `MUST NOT` is a *requirement*; one containing only `SHOULD` or
   `SHOULD NOT` a *recommendation*; one containing only `MAY` a *permission*.
   Coverage (D14) applies to observable requirements.
7. **`spec/statements.json` is generated** from the chapter files and
   committed: one object per statement, sorted by identifier —
   `{"id", "chapter", "section", "kind", "since", "observable", "status"}`
   with `status` one of `active`, `withdrawn`, and `amended_by` when present.
   A tripwire regenerates it and fails on any difference. Committing it gives
   a second implementation a machine-readable list, and **serializes
   identifier allocation**: two changes that both take `MTS-OUT-013` edit the
   same line of this sorted file and conflict in git, where two paragraphs in
   different hunks would have merged silently.
8. **Prose cites statements by identifier and link**
   (`[MTS-OUT-012](09-outputs.md#mts-out-012)`); vectors cite them by
   identifier alone.
9. **Finding codes link to statements.** `spec/findings/registry.json`
   (ADR-006 D3 item 3) is the one source of the code catalog: each entry's
   `statements` names the statement or statements it reports, and the
   catalog table in `12-findings.md` is generated from the registry and
   committed, so a reader goes from a finding to the rule. A tripwire
   (`TestFindingCatalog`, D14) regenerates it and fails on any difference.

### D4 — Examples and the worked package

**An example that shows a package and what it produces is a vector, included
by reference, so an example in the text cannot drift from the behaviour.**

1. An example block is an HTML comment naming a vector, followed by fenced
   code blocks whose info string carries a `path` attribute:

   ````
   <!-- example: out/for-each-one-file-per-item -->

   ```yaml path=package/metais.yaml
   …
   ```

   ```markdown path=expected/out/articles/first.md
   …
   ```
   ````

2. **A tripwire checks that each fenced block equals the named vector file
   byte for byte** (the fence adds the final newline the file must also end
   with; a file without one cannot be shown and is described instead). An
   example may show a subset of the vector's files; it may not show a file
   the vector lacks.
3. A short fragment that is not a package — one Liquid expression, one ABNF
   production — needs no vector and is marked `> **Example.**`.
4. **`B-worked-package.md`** (informative) builds one small package step by
   step on the authoring golden path — a manifest, one input with its schema
   and meaning document, two templates, one example — using only blocks
   included from vectors under `vectors/gen/`. It is the page SC9's author
   starts from.

### D5 — Revisions of the document

**One living text per major in `spec/`; a revision is released by an
annotated git tag `spec-<MAJOR>.<MINOR>`; no directory per revision.**

1. **The text on `main` is the latest released revision plus any additions
   for the next minor**, each marked `since` that minor. The front page says
   which: `Version: 1.1`, `Status: Draft`, `Latest release: 1.0`. Because a
   minor only adds and every addition is marked, the text of a released
   revision is the current text read without the later `since` statements,
   and the exact released bytes are at its tag.
2. **A revision is released** by one commit that:
   - dates its `CHANGELOG.md` section;
   - sets the front page to `Status: Released <date>, tag spec-<M>.<m>`;
   - writes `spec/conformance/released/<M>.<m>.sha256`, the SHA-256 of every
     file under `spec/conformance/vectors/` and `spec/schemas/` in
     `sha256sum` format, sorted by path;
   - and passes the platform matrix (D15).
   That commit is tagged `spec-<M>.<m>` (for v1, `spec-1.0`). A tag is never
   moved or deleted.
3. **The tag is `spec-1.0`, not `spec/v1.0`**: a tag beginning `spec/` reads
   to Go as a version of a module in the `spec/` directory.
4. **After a release, the first addition opens the next section**
   (`## 1.1 — unreleased`) and sets the front page to `Draft`. Until then the
   front page stays `Released`.
5. **A released revision's statements, schemas and vectors are immutable**
   except by an erratum (D6). The `released/*.sha256` lists make that
   mechanical: a tripwire fails when a listed file's digest changes and no
   erratum in `ERRATA.md` names that file and its new digest.
6. **A tool release advertises only released revisions** (ADR-002 D8's
   `implements`). The tool built from `main` implements the draft so its
   vectors run, but ADR-010's release procedure refuses to ship a tool whose
   `implements` names an unreleased revision; a spec revision is therefore
   released no later than the first tool that implements it.
7. **`CHANGELOG.md`** has one section per revision, newest first:

   ```
   ## 1.1 — unreleased
   ### Added
   - MTS-FLT-031 — the `wrap` filter.
   ### Errata applied
   - E-1.0-002 — MTS-OUT-014: a `when` that renders to whitespace is false.
   ### Editorial
   - Chapter 9: the fan-out section reworded; no statement changed.
   ```

   From `1.1` on, `Added` lists every statement whose `since` is that
   revision and nothing else (a tripwire checks both directions). The `1.0`
   section says "First revision." An **editorial** change alters no
   statement's text, adds none, and changes no vector or schema; it needs no
   new revision and is recorded under the open section, or under the latest
   released one when none is open.
8. **The specification's own version is the manifest's `metais` value**
   (ADR-002 D7): `1.0`, with no patch part. Editorial changes and errata do
   not change it.

### D6 — Errata

**An erratum corrects a released statement that is wrong; it is recorded in
`ERRATA.md` with an identifier, fixed on `main` at once, and published in the
next revision. For a released revision, its vectors are the tie-breaker.**

1. An erratum is one of: a **contradiction** between statements, or between
   a statement and a vector; an **impossibility** (no implementation could
   obey it); an **ambiguity** two implementations have read two ways; a
   **security** defect (the text accepts a package or input that lets a
   package escape R14).
2. Each entry:

   ```
   ## E-1.0-001 — <one-line title>
   - Kind: contradiction | impossibility | ambiguity | security
   - Affects: MTS-OUT-014 (1.0)
   - Reported: 2027-02-01
   - Correction: <the statement's text as corrected>
   - Vectors: added out/when-whitespace-is-false; changed none
   - Files: <path> sha256:<new digest>   (one line per released file it changes)
   - Applied: 2027-02-03; published in 1.1
   ```

   The identifier is `E-<revision it corrects>-<NNN>`, numbered in order of
   report within that revision.
3. **Applying an erratum** edits the statement in place, adds `amended by
   E-…` (or `withdrawn by`), adds or changes the vectors it lists, and records
   it in the open changelog section.
4. **What an erratum may change.** For every kind but `security`, the
   correction is made **toward the released vectors**: the text is rewritten
   to say what they already prove, and no released vector's expected result
   changes. Only a `security` erratum may change a released vector's
   expectation, and only from acceptance to refusal; its entry says so. This
   keeps R15.4 for every package the released vectors describe, and puts the
   one deliberate exception where refusing is the safe direction.
5. A change that neither D5 nor this decision allows — changing what an
   accepted construct means — waits for a new major (ADR-002 D7 item 5).

### D7 — Published schemas

**`spec/schemas/` mirrors `https://metais.org/schemas/`: a file at
`spec/schemas/<path>` has `$id` `https://metais.org/schemas/<path>`.**

1. **The manifest schema** of each minor is
   `spec/schemas/<M>.<m>/manifest.schema.json` — for v1,
   `spec/schemas/1.0/manifest.schema.json` with ADR-002's `$id`
   `https://metais.org/schemas/1.0/manifest.schema.json`. Its content is
   ADR-002's.
2. **Any other schema a format chapter defines** lives beside it under the
   same minor's directory, named `<subject>.schema.json`: for `1.0`,
   ADR-006's `finding.schema.json` and `finding-registry.schema.json`.
3. **The conformance suite's own schemas** are under
   `spec/schemas/conformance/<format-major>/`: `vector.schema.json` (D9),
   `findings.schema.json` (the expected-findings projection, D10),
   `driver-request.schema.json` and `driver-response.schema.json` (D11 item
   6), and `record.schema.json` (D15). The suite format is versioned
   separately from the specification: within its major it only adds members.
4. **Every published schema** declares
   `"$schema": "https://json-schema.org/draft/2020-12/schema"`, validates
   against that meta-schema, has the `$id` this decision's rule gives it,
   contains no external `$ref`, and is written with two-space indentation, LF
   line ends and one final LF. A tripwire checks each.
5. **The command line's JSON formats are not published here**; they stay in
   `docs/cli/schemas/` (ADR-008 D10), because the command line is not part of
   the format.
6. **The tool's embedded copies** live under `internal/` (ADR-010 places
   them), since `//go:embed` cannot reach `spec/`; a test compares each copy
   with its published file byte for byte (ADR-002 D10 item 5).
7. `https://metais.org/` is an identifier space; nothing is hosted there by
   this decision, and no tool or runner fetches it.

### D8 — Conformance classes and the operations a vector uses

**The specification defines three conformance classes and a small set of
abstract operations; vectors are written against the operations, not
against any command line.**

1. **Classes**, defined in `01-introduction.md`:
   - **package** — a packed package or a source directory that satisfies
     every statement addressed to packages;
   - **processor** — a program that opens packages, validates inputs and
     renders outputs;
   - **authoring tool** — a program that checks, tests, packs, unpacks and
     generates sample data.
   The reference tool claims processor and authoring tool. A package's
   conformance is established by a conforming authoring tool's `check`
   reporting no error.
2. **Operations**, defined in the chapter that owns each and listed in
   `15-conformance.md`, with their arguments and results:

   | Operation | Arguments | Results | Owner |
   |---|---|---|---|
   | `open` | `package` | `ok`, findings | ADR-001, ADR-009 |
   | `validate` | `package`, `data`, `set`, `set_json`, `inputs_json`, `limits` | `ok`, findings | ADR-003, ADR-006 |
   | `render` | the supply arguments; one of `out` (a directory) or `stream` (`"archive"` or `{"only": <path>}`); `only`, `overwrite`, `dry_run`, `now`, `sample` (`{"optional", "seed", "items"}`), `limits` | `ok`, findings, the written tree or stream, and with `dry_run` the listing (`path`, `size`, `digest`) | ADR-005 |
   | `check` | `source`, `limits` | `ok`, findings | ADR-007 |
   | `test` | `source`, `examples`, `limits` | `ok`, findings | ADR-007 |
   | `pack` | `source`, `to` (a file path) | `ok`, findings, the written package | ADR-007 |
   | `unpack` | `package`, `to` (a directory) | `ok`, findings, the written tree | ADR-007 |
   | `sample` | `source`, `to` (a directory) or `example` (a name), `optional`, `seed`, `items`, `set`, `set_json`, `inputs_json` | `ok`, findings, the written tree | ADR-007 |

   `ok` is true when the operation completed and no finding is of error
   severity. Argument values are as the owning chapter defines them: `set`
   and `set_json` are lists of strings in ADR-003 D17's `<pointer>=<value>`
   syntax (a text set and a JSON set), `inputs_json` names a vector file
   holding the JSON object, `limits` is an object of ADR-009's limit names
   to values, `now` is an RFC 3339 date-time.
3. **Reading in place** (listing, reading, outlining and searching entries)
   is a property of the reference tool's command line (ADR-008), not an
   operation: its observable content — the entries of a package — is proved
   through `open`, `pack` and `unpack`.
4. **A conformance claim** names the implementation and its version, the
   classes, the highest revision implemented, the suite revision (the
   `spec-<M>.<m>` tag or a commit), and the platforms run (D15). It is
   satisfied when every vector of those classes whose `since` is at or below
   that revision passes on each named platform, with skips only for unmet
   `requires` (D9 item 6).

### D9 — The vector

**A vector is a directory holding `vector.json`, the package, the inputs, an
optional setup tree and the expected results; it runs one or more cases,
each an ordered list of steps.**

1. **Location and identity.** `spec/conformance/vectors/<area>/<slug>/`,
   where `<area>` is the lower-cased area of the vector's first statement and
   `<slug>` matches `[a-z0-9]+(-[a-z0-9]+)*` in at most 64 characters. The
   vector's `id` is `<area>/<slug>`, and equals its path. *(A1, proposed)*
2. **Contents:**

   ```
   vector.json         the vector manifest
   package/            a source directory, or
   package.metais      a packed package (bytes)
   data/ …             any trees or files the steps' arguments name
   setup/              optional: a tree copied into the work directory before the case runs
   expected/           expected trees, findings files, stream bytes and listings
   ```

   **Every file in the vector directory is named by `vector.json` or lies in
   a tree it names**; nothing is dead weight. `expected/` is never copied into
   the work directory, so an implementation cannot read the answer.
3. **`vector.json`** (schema at
   `spec/schemas/conformance/1/vector.schema.json`):

   ```json
   {
     "$schema": "https://metais.org/schemas/conformance/1/vector.schema.json",
     "id": "out/for-each-one-file-per-item",
     "title": "for_each writes one file per collection item, named by the path template",
     "statements": ["MTS-OUT-012", "MTS-OUT-013"],
     "since": "1.0",
     "class": "processor",
     "tags": ["determinism"],
     "requires": [],
     "notes": "",
     "cases": [
       {
         "name": "render",
         "setup": { "copy": null, "dirs": [], "symlinks": [] },
         "steps": [
           {
             "op": "render",
             "args": { "package": "package", "data": "data", "out": "out" },
             "expect": {
               "ok": true,
               "trees": [ { "path": "out", "equals": "expected/out" } ]
             }
           }
         ]
       }
     ]
   }
   ```

   - `statements` — at least one active statement identifier; the vector
     proves each.
   - `since` — equals the highest `since` among its statements.
   - `class` — `processor` or `authoring`.
   - `tags` — from the closed registry in item 5.
   - `requires` — from the closed set in item 6.
   - `notes` — free text; names the program that produced a committed
     `package.metais`, if one did (D16 item 5).
   - `cases` — at least one; `name` matches the slug pattern and is unique in
     the vector.
4. **Setup**, applied to a fresh work directory before the first step:
   `copy` names a vector tree whose contents are copied into the work
   directory's root (a `setup/out/existing.txt` becomes `out/existing.txt`,
   the pre-existing file an overwrite vector needs); `dirs` lists empty
   directories to create; `symlinks` lists `{"path", "target"}` links to
   create; `fifos` lists named pipes to create (ADR-009 D17's input cases).
   These hold what git cannot: empty directories, links and FIFOs. Then the
   runner copies the package and every tree or file the steps' arguments name
   to the same relative paths, and last makes the `unwritable` directories
   unwritable. *(A2, proposed)* Steps of one case share the work directory, in
   order; cases never share one.
5. **Tags** (closed; adding one edits the vector schema):

   | Tag | Means |
   |---|---|
   | `security` | the hostile corpus (D16) |
   | `determinism` | proves a determinism rule (iteration order, number and date formatting, line endings, locale or time-zone independence) |
   | `portable-paths` | proves a portable-path rule (Windows-invalid names, case-only collisions); runs on every platform |
   | `container` | an output or a package compared by entries |
   | `byte-level` | the package is a committed `package.metais` |
   | `limits` | proves a resource limit or its lowering |
   | `regression` | added for a defect found in an implementation |

6. **Requires** (closed): `symlinks` (the setup creates a link), `fifos`
   (the setup creates a named pipe) and `case-sensitive-fs` (the setup or a
   step creates names differing only in case). The runner probes each in the work directory's file system and
   skips the case with the reason when it is absent; a skip is reported, never
   counted as a pass. Linux meets both, so every vector runs on at least one
   platform of the matrix (D15). `unwritable-dirs` (the setup makes a
   directory unwritable) is met where a directory of mode 0555 refuses the
   runner's own process a write. *(A2, proposed)*
7. **Packed packages are committed bytes**, used only where the case is about
   bytes — the zip profile, digests, a hostile container. Everything else uses
   a source directory, so a change to the packer does not churn the suite.
8. **The conformance tree is checkout-safe on every platform**: every path
   under `spec/conformance/` obeys ADR-005's portable-path rules, no two paths
   differ only in case, none exceeds 160 bytes from the repository root
   (Windows' `MAX_PATH` of 260 characters, less room for the clone's own
   location),
   and none is a link. A `.gitattributes` line `spec/conformance/** -text`
   stops line-ending conversion, so CRLF expectations survive a checkout.
   Names that must collide or be invalid are created by `setup`, rendered
   from inputs, or held inside a committed `package.metais`.

### D10 — How results are compared

**Every observed result is compared exactly, except where a statement grants
freedom: container compression, finding wording, and the tool's identity.**

1. **`ok`** — equal.
2. **Findings** — `expect.findings` names a JSON file (schema
   `findings.schema.json`) holding an array of expected findings. **Absent,
   it means no findings at all.** The actual findings, in ADR-006's order,
   must have the same length; each expected object's members must equal the
   actual finding's members of the same name, recursively. Expected objects
   may use only the members ADR-006 D10 item 5 makes stable — `code`,
   `severity`, `subject`, `location`, `schema` (`entry`, `pointer`,
   `keyword`), `field`, `params`, `via`, `alternatives` and `related` —
   never `message`, `hint` or `schema.keyword_location`, whose wording or
   form is the implementation's.
3. **Trees** — `expect.trees` lists `{"path", "equals"}`: the tree at `path`
   in the work directory must hold exactly the regular files of the expected
   tree, byte for byte, and nothing else (no extra file, no link, no empty
   directory, no staging remnant). Options per tree:
   - `containers`: a list of `{"path", "methods"}` — files compared as zips
     **by entries**: names, order and uncompressed content, every header field
     ADR-001 fixes at its fixed value, and the compression method only for the
     entries `methods` lists, as ADR-001 D4 item 6 and ADR-005 D8 item 7
     require. A vector lists only entries whose method the specification
     fixes — those a declaration marks `stored: true`, a profile's
     `mimetype`, and empty entries — since `auto`'s choice for any other
     entry belongs to the encoder; `"all"` is accepted only for a package or
     container in which every entry is one of those. Compressed bytes and
     sizes are never compared.
   - `executable`: paths whose owner-execute bit must be set (ADR-005 D2 item
     6's `executable: true`); every other file's must be clear. Checked only
     where the file system has POSIX modes.
   - `substitute`: files in which `@@TOOL_NAME@@` and `@@TOOL_VERSION@@` are
     replaced, before comparison, by the tool's name and version as the adapter
     reports them — for provenance that records the tool's version (ADR-004
     D14 item 2, which a `render` records as it is; `test` records `test`,
     ADR-004 D15). No other token exists: the package digest a `render`
     records is defined over content (ADR-001 D6), so an expected file holds
     it literally.
4. **Nothing else changes.** The runner records the whole sandbox (D11 item
   4) before and after each step; any file created, changed or removed
   outside a declared tree fails the step. **A step with no `trees` writes
   nothing anywhere**, which is how every refusal is checked.
5. **Stream** — `expect.stream` is `{"equals": <file>}`. An archive stream is
   compared **by entries**, as ADR-005 D17 item 5 states — order, path,
   typeflag, mode, uid, gid, uname, gname, mtime, size and content — because
   ADR-005 leaves the pax extended header's own name and fields to the
   implementation; the reference tool's archive bytes are proved identical
   across platforms by D15's records. A single-output stream is compared
   byte for byte, or, for a container output, adds `"container":
   {"methods": …}` and is compared by entries.
6. **Dry-run listing** — `expect.listing` names a JSON array of
   `{"path", "size", "digest"}` in ADR-005's order, compared exactly, except
   that for a path the step lists in `containers` only `path` is compared.
7. **Repeatable results** — where a statement leaves the bytes to the
   implementation (sample data is such a case, ADR-007 D9 item 3), a step
   sets `"repeatable": true`, its trees name `files` (the paths that must
   exist) instead of `equals`, and the runner runs the case twice in fresh
   sandboxes and requires every observed byte to match between the runs. A
   later step in the same case proves the property the bytes must have — a
   `validate` of the generated data, for instance.

### D11 — The runner

**A Go test that drives the implementation as a black box: through the
reference tool's command line by default, or through a documented driver
protocol for any other implementation.**

1. **Location.** ADR-010 D14 item 2 places it in `internal/conformance`,
   with `TestConformance` running every vector as subtests named
   `TestConformance/<area>/<slug>/<case>`, each vector in parallel, so
   `go test ./internal/conformance -run 'TestConformance/out/'` selects an
   area.
2. **Black box.** The runner never links the implementation. By default its
   `TestMain` builds `./cmd/metais` once into a temporary directory with
   `-trimpath`, adding `-race` when the test binary itself is race-built (a
   build-tagged constant tells it), and runs that binary per step. A second
   process is the only way the environment, working directory and home of a
   run can be controlled and scanned, and the only arrangement a second
   implementation can reproduce.
3. **Flags**, after `-args`:

   | Flag | Default | Meaning |
   |---|---|---|
   | `-conformance.adapter` | `cli` | `cli` drives a `metais`-compatible command line; `exec` drives a driver (item 6) |
   | `-conformance.tool` | the binary built from source | the command-line tool for `cli` |
   | `-conformance.driver` | — | the driver executable for `exec` |
   | `-conformance.suite` | `spec/conformance` in the module | the suite directory |
   | `-conformance.version` | the front page's `Version` | the highest revision claimed; vectors with a later `since` are skipped |
   | `-conformance.classes` | `processor,authoring` | which classes to run |
   | `-conformance.tags`, `-conformance.skip-tags` | all, none | tag selection |
   | `-conformance.record` | — | write the cross-platform record (D15) |
   | `-conformance.compare` | — | compare records instead of running vectors (D15) |
   | `-conformance.keep` | — | keep each failing case's sandbox under this directory |

4. **The sandbox** of each case is a fresh directory
   `<test temp>/MTSSENTINELPATH-<n>/` holding `home/`, `tmp/`, `bin/` (empty)
   and `work/`; `work/` is the working directory of every step. Steps receive
   only relative paths, as the vector wrote them. The runner snapshots `home/`,
   `tmp/` and `work/` (path, type, size, SHA-256, mode, link target) before and
   after each step for D10 item 4.
5. **The `cli` adapter** maps each operation to `metais` with `--json` and
   reads ADR-008's envelope (`ok`, `findings`, `result`):

   | Operation | Command line |
   |---|---|
   | `open` | `inspect <package>` |
   | `validate` | `validate <package> [--data D] [--set S]… [--set-json S]… [--inputs-json -] [--limit N=V]…` (the `inputs_json` file on standard input) |
   | `render` | `render <package> --out <dir>` or `--out -` `[--only P] [--overwrite] [--dry-run] [--now T] [--sample --optional O --seed N --items K]` plus the supply flags; the stream is standard output and the document standard error (ADR-008 D11 item 2) |
   | `check`, `test` | `check <source>`, `test <source> [--example E]…` |
   | `pack`, `unpack` | `pack <source> --out <file>`, `unpack <package> --to <dir>` |
   | `sample` | `sample <source> --to <dir>` or `--example <name>`, with `--optional`, `--seed`, `--items` and the supply flags |
   | the tool's identity | `version` (ADR-002 D8's `tool` object) |

   The adapter also checks ADR-008 D13's mapping onto ADR-006 D12's table —
   exit code 0 exactly when `ok`, and otherwise the code the findings'
   classes give — and fails the step on exit code 3, which means the
   adapter's command line has drifted from the tool's, and on exit code 2, a
   crash, each reported as a harness error apart from conformance
   failures. For an `open` step in a `security`
   vector it runs every package-opening command (D16 item 3).
6. **The driver protocol (`exec` adapter)** is documented in
   `spec/conformance/README.md` and the two `driver-*` schemas. The runner
   starts the driver once per step in the sandbox (same working directory and
   environment as a tool would get) and writes one JSON request to its
   standard input:

   ```json
   {"protocol": "metais.conformance-driver/1", "op": "render",
    "args": {"package": "package", "data": "data", "stream": "archive"},
    "stream_file": ".stream"}
   ```

   The driver performs the operation with its implementation, writes stream
   bytes to `stream_file`, and prints one JSON response on standard output:

   ```json
   {"ok": true, "findings": [], "listing": null,
    "tool": {"name": "othertool", "version": "0.3.0"}}
   ```

   then exits 0. A non-zero exit or an unparsable response is a harness
   error, reported apart from conformance failures. `stream_file` is
   excluded from the nothing-else-changes check. The op `version` returns only
   `tool`.
7. **The protocol is exercised here**: `TestDriverProtocol`, skipped under
   `-short`, runs the whole suite through `exec` with a reference driver that
   wraps the `metais` binary, so the published protocol cannot rot unrun.
8. **Failure output** names the vector, case, step and statement
   identifiers, and shows the first difference: a unified diff of at most 40
   lines for text, a 64-byte hexadecimal window for binary, the expected and
   actual finding projections side by side, or the entry that differs in a
   container.
9. **Budget.** The default run (both environments, D12) takes at most 60
   seconds on an Apple M-series developer machine with the v1 suite; a vector
   that needs more is tagged `limits` and justified in its `notes`.

### D12 — Host isolation: the sentinel environment (SC7)

**Every vector runs in a sentinel environment, and every byte the run
produced is scanned for the sentinel; a second, plain environment must give
byte-identical results.**

1. **Environment A (sentinel)** — the step's environment is exactly:

   | Variable | Value |
   |---|---|
   | `HOME` | `<sandbox>/home/MTSSENTINELHOME` |
   | `TMPDIR` | `<sandbox>/tmp/MTSSENTINELTMP` |
   | `PATH` | `<sandbox>/bin/MTSSENTINELBIN` (empty) |
   | `PWD` | the work directory |
   | `USER`, `LOGNAME` | `MTSSENTINELUSER` |
   | `HOSTNAME` | `MTSSENTINELHOST` |
   | `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_CACHE_HOME` | directories under `HOME`, each holding a `metais/` directory with a file of sentinel content |
   | `TZ` | `Pacific/Chatham` (UTC+12:45) |
   | `LANG`, `LC_ALL` | `tr_TR.UTF-8` (Turkish casing of `i`) |
   | `NO_COLOR`, `TERM` | `MTSSENTINELNOCOLOR`, `MTSSENTINELTERM` (the tool reads `NO_COLOR` only for colour, which human output alone shows; `TERM` it never reads) |
   | `GODEBUG` | `zipinsecurepath=1,tarinsecurepath=1` |
   | `METAIS_SENTINEL` | `MTSSENTINELENV` |

   Nothing is inherited from the test process.
2. **Decoys.** The work directory's root and `HOME` each receive
   `SENTINEL.txt`, and, for every file directly inside a data tree the steps
   name, a file of the same name; each holds one line `MTSSENTINELCWD` or
   `MTSSENTINELHOME`. A tool that reads inputs from the working directory or
   the home directory, rather than from the paths it was given, picks a decoy
   up.
3. **The scan** searches, case-insensitively, for `MTSSENTINEL` in: standard
   output and standard error; every file written in a declared tree; each
   entry of a zip it finds there, inflated, and of zips inside those, to a
   depth of three; each member of a tar stream; and the driver's response. A
   hit fails the step and names the artefact and the sentinel's source
   (`HOME`, `CWD`, `ENV`, `PATH`, …). The token has only letters, so it
   survives HTML, JSON, CSV and URL escaping and the `slug` filter's
   lower-casing; the sandbox path carries it too, so an absolute path in a
   finding is caught.
4. **Environment B (plain)**: `HOME`, `TMPDIR` and `PATH` as plain sandbox
   directories, `TZ=UTC`, `LC_ALL=C`, nothing else, run in other processes
   and started at least two seconds after A, so the process id and the
   clock differ too (ADR-009 D16 item 1, check 3). **Every observed byte of
   A and B must be equal** — standard output, standard error, trees, stream,
   listing. A difference fails the step and names the variables that differ,
   which catches a time-zone, locale or `GODEBUG` dependence the sentinel
   scan cannot see.
5. Under `go test -short` only environment A runs.
6. **Proved red first.** ADR-009 owns a deliberately leaking build (a
   template variable that expands an environment variable); running the suite
   against it must fail on the sentinel scan, and that run is recorded in
   ADR-009's implementation log before the code that makes it pass lands.
   The clock, the process and the host name are outside what a sentinel can
   plant; their proof is ADR-009's.

### D13 — Bounded work in the runner

**Every step runs under a CPU-time, memory and wall-clock bound, measured
from the process's own resource usage.**

1. Defaults: **10 s** of user plus system CPU time, **1024 MiB** peak
   resident memory, **120 s** wall clock, after which the process is killed
   and the step fails. A step may lower them with `"bounds": {"cpu_ms": …,
   "rss_mib": …}`; hostile vectors do (D16 item 4).
2. CPU time comes from `os.ProcessState.UserTime` and `SystemTime`; peak
   memory from `SysUsage`'s `ru_maxrss`, read as bytes on darwin and KiB on
   Linux.
3. When the tool is race-built, the CPU bound is multiplied by 20 and the
   memory bound by 10, the race detector's documented worst cases.
4. A bound is a property of the vector, not of the run: a vector that passes
   only on a fast machine is a defect in the vector.

### D14 — Coverage and the tripwires

**Every observable requirement has at least one vector, every citation
resolves, and the document's mechanics are checked by tests in `go test`.**

The tests, in ADR-010's place for them:

| Test | Fails when |
|---|---|
| `TestStatementsWellFormed` | a statement head does not match D3's grammar; an anchor is not the lower-cased identifier; an identifier is duplicated; an area is not in the registry or not owned by the chapter it appears in |
| `TestStatementIndex` | `spec/statements.json` differs from what the chapters generate |
| `TestKeywordsOnlyInStatements` | an all-capitals BCP 14 keyword appears outside a statement, or a synonym D2 item 2 excludes appears anywhere |
| `TestRecommendationsNotObservable` | a `SHOULD`/`SHOULD NOT` statement lacks `not observable` |
| `TestCoverage` | an active, observable requirement has no vector citing it |
| `TestVectorsWellFormed` | a `vector.json` fails its schema; `id` differs from its path; the area is not its first statement's; `since` is not its statements' highest; a referenced file is missing or a file is unreferenced; a `package.metais` lacks `byte-level` |
| `TestCitationsResolve` | a vector cites an unknown or withdrawn identifier; a prose link to a statement does not resolve |
| `TestFindingCatalog` | the code catalog in `12-findings.md` differs from what `spec/findings/registry.json` generates |
| `TestFindingCodesCovered` | a registered, unretired code appears in no vector's expected findings, or names no statement |
| `TestExamplesMatchVectors` | an example block (D4) differs from its vector file, or names a missing vector or file |
| `TestSuiteTreePortable` | a path under `spec/conformance/` breaks D9 item 8, or `.gitattributes` lacks the `-text` line |
| `TestNoSentinelInSuite` | any suite file contains `MTSSENTINEL` (case-insensitive) |
| `TestReleasedFilesUnchanged` | a file listed in `released/*.sha256` has another digest with no erratum naming it (D5 item 5) |
| `TestChangelog` | a `since` statement is missing from its revision's `Added`, or `Added` lists one that is not; the front page's block disagrees with the changelog |
| `TestSchemasPublished` | a schema under `spec/schemas/` breaks D7 item 4, or an embedded copy differs from it |
| `TestHostileVectorsOpenEverywhere` | a `security` vector with an `open` step is not run through every package-opening command (D16 item 3) |

`TestCoverage` also prints, with `-v`, a table of every statement and the
vectors citing it. The table is printed, not committed: a committed coverage
file would conflict on every pair of changes that each add a vector.

### D15 — Platforms and the cross-platform record (SC5)

**The developer gate runs the suite on the host; a release runs it on darwin
and linux, amd64 and arm64, and requires every pair of records to agree byte
for byte.**

1. **On every change**, `go test -race ./...` runs the suite on the host
   platform, both environments.
2. **At every release** — of a spec revision (D5 item 2) or of the tool
   (ADR-010) — the suite runs on darwin/arm64 (native), darwin/amd64 (under
   Rosetta 2), linux/arm64 and linux/amd64 (native, or a container, emulated
   where the host's architecture differs). For each target the test binary is
   built with `go test -c` and the tool with `go build`, both for that target,
   from one commit with one toolchain; the run passes
   `-conformance.record=<file>`.
3. **The record** (`record.schema.json`) holds the tool's name and version,
   the target, the SHA-256 of the suite tree, and for every step of every case
   in both environments: `ok`, the SHA-256 and length of standard output,
   standard error, the stream and the listing, and every file of every
   declared tree with its path, size, SHA-256 and execute bit. No sandbox path
   appears in it.
4. **`-conformance.compare=a.json,b.json,…`** refuses records with different
   tool versions or suite digests, and otherwise fails on any field that
   differs between any two. A release requires every target's record to pass and
   every pair to agree. The pair darwin/arm64 against linux/amd64 is the minimum
   that may support an SC5 claim, because it differs in both the operating
   system and the architecture — where fused floating-point operations and
   path handling diverge.
5. This catches what the expectations cannot: a container's deflate bytes,
   human messages, and the bytes of repeatable results, which no committed
   file fixes but one build must produce identically everywhere (ADR-001 D4
   item 6).
6. **Windows** is not a v1 target and is not run. `portable-paths` vectors run
   on every platform; the suite tree is checkout-safe on Windows (D9 item 8);
   and the runner assumes nothing POSIX except where `requires` probes for it.

### D16 — The hostile corpus among the vectors (SC6)

**Hostile packages and inputs are ordinary vectors tagged `security`, filed
under the area of the statement each breaks, each expected to be refused with
exact findings and to write nothing anywhere.**

1. **Content** — which hostile packages and inputs exist, and the codes they
   are refused with — is ADR-009's (with ADR-001's container cases and
   ADR-005's path cases). This decision fixes their form.
2. **Placement.** A traversal entry is `con/…`, a reserved output name
   `pth/…`, an exhausted limit `sec/…`: the area of the first statement the
   vector proves. *(A1, proposed)* The corpus is the set selected by `-conformance.tags=security`.
3. **Every opening path refuses.** A vector whose hostility is in the package
   has an `open` step; the `cli` adapter runs it as each of ADR-008's commands
   that opens a package — `inspect`, `ls --recursive`, `schema`, `validate`,
   `render --out out` and `unpack --to unpacked` — and each must give
   `ok: false` and the same findings. Hostility in inputs or in the render
   (an output path built from an input, a YAML alias bomb in a data file, a
   pathological `pattern`) uses `validate` or `render` steps.
4. **Expectations**: `ok: false`; the exact findings; no `trees`, so nothing
   is written in the work directory, the home directory or the temporary
   directory (D10 item 4) — stronger than SC6's "nothing outside the output
   directory"; and `bounds` lowered to what the defence should need (a zip
   bomb's vector sets `rss_mib` well below its expanded size).
5. **One defect per package.** A hostile `package.metais` carries a correct
   `.compiled/` and correct digests around its single defect — unless the
   defect is a digest — so a refusal for the wrong reason fails on the finding
   codes. Its bytes are committed; the program that produced them is named in
   `notes`, and ADR-010 keeps that program and a test that it reproduces the
   committed bytes.
6. **SC6 is claimed** when every `security` vector passes on every matrix
   target; **SC7** when the sentinel scan (D12) passes over every vector,
   these included.

### D17 — Core Liquid is proved by Shopify's liquid-spec

**The dialect's METAIS rules are proved by METAIS vectors; the body of core
Liquid is proved by running a pinned selection of Shopify's liquid-spec
against the reference engine.**

1. `spec/conformance/delegated/liquid-spec.json` pins it. This ADR pins the
   liquid-spec commit (the head of 2026-10-18); ADR-004 D1 pins the dialect
   — gem 5.14.0 under `strict2`, without `include` and `self` — and every
   other difference is a departure. The `missing_features` are liquid-spec's
   own feature gates (`lib/liquid/spec/cli/features.rb` at that commit) for
   what the dialect does not have: inline errors (ADR-004 renders none),
   lax parsing, Ruby types and drops, random inputs (the run is
   deterministic), the `self` drop, Shopify's theme tags, objects, filters
   and error formats, and its resource-limit scoring (ADR-009 counts its
   own). `shopify_blank` and `shopify_string_access` stay enabled: ADR-004
   D5 item 2's `blank` and D4 item 7's `first`/`last` on strings are those
   behaviours, as the gem at 5.14.0 has them. `include` and every other
   case the dialect refuses are departures:

   ```json
   {
     "repository": "https://github.com/Shopify/liquid-spec",
     "commit": "84bf25e0edbca5f7e530574788b0f875ab331f50",
     "suites": ["basics", "liquid_ruby", "partials"],
     "error_mode": "strict2",
     "missing_features": ["inline_errors", "lax_parsing", "ruby_types", "ruby_drops",
                          "drop_class_output", "template_factory", "binary_data", "drops",
                          "randomness", "self_environment_shadowing", "strict2_blank_body_errors",
                          "shopify_tags", "shopify_objects", "shopify_filters",
                          "shopify_error_handling", "shopify_error_format", "shopify_includes",
                          "shopify_resource_limits", "range_resource_limits"],
     "departures": [
       {"spec": "<liquid-spec case name>", "statement": "MTS-TPL-…", "why": "include is not part of the dialect"}
     ]
   }
   ```

2. **Every departure names the METAIS statement that makes the dialect
   differ**, and is an expected failure: a departure that starts passing fails
   the run, because the dialect and the pin have stopped agreeing.
3. **It runs outside `go test`**, because it needs Ruby and Bundler: at every
   tool release, at every change of the Liquid engine or of its version, and
   at every change to `07-template-environment.md` or `08-filters-and-tags.md`.
   The reference engine is reached through a JSON-RPC server implementing
   liquid-spec's `initialize`, `compile`, `render` and `quit`, a test-only
   program ADR-010 places under `internal/`.
4. **What the METAIS vectors still cover**: every statement in chapters 7 and
   8 — the variables, strictness, truthiness and `blank`/`empty`, whitespace,
   partial loading, every METAIS filter and tag — through the full tool. The
   delegated suite proves the inherited core that the text adopts by
   reference to Shopify's implementation.
5. liquid-spec is run, not copied: nothing of it is vendored into this
   repository.

### D18 — The runner's own proof

**The runner is pinned false first: for each check it makes, a fake
implementation that breaks exactly that check must fail, and one that breaks
nothing must pass.**

1. The fake is the test binary re-executed in a fault mode, serving either
   adapter. One fault per check: wrong `ok`; a finding out of order, with a
   wrong member, extra or missing; a tree with an extra, missing or changed
   file, an empty directory or a link; a container entry out of order, with
   other content, with another method on a listed entry, or with a non-fixed
   header field; a write outside a declared tree; a file left in `TMPDIR`; a
   sentinel in standard output, standard error, a written file, an inflated
   zip entry and a tar member; an A/B divergence; CPU, memory and wall bounds
   exceeded; an exit code inconsistent with `ok`; a malformed driver response.
2. These tests run in `go test -race ./...` and need no vector.

## Interfaces with sibling ADRs

**ADR-001 — the container.** I rely on: the list of header fields its profile
fixes, with their values, so container comparison (D10 item 3) checks them;
the package digest being content-defined (so provenance that records it, and
`pack` vectors, are identical across implementations); "method compared only
where the specification fixes it" (its D4 item 6), which D10's `methods`
expresses; the `.compiled/` file names (`digests.json`, `manifest.json`,
`schema.json`, `references.json`); its Conformance list, which becomes
`con/` vectors *(A1, proposed)*. I promise: a vector can say "refused with these findings"
and "packs to these entries, in this order, with these methods and this
package digest" — the digest is inside `digests.json`, compared as entry
content.

**ADR-002 — the manifest and versioning.** I rely on: `metais` as the
specification version and its grammar; the minor rules of D7, which D5 and D6
implement on the document side; the manifest schema's content and its `$id`
(`https://metais.org/schemas/1.0/manifest.schema.json`), which D7 publishes at
`spec/schemas/1.0/manifest.schema.json`; the tool's `version` content (D8),
which the `cli` adapter reads for `@@TOOL_NAME@@` and `@@TOOL_VERSION@@`. I
promise: every addition carries `since`; `statements.json` is the
machine-readable form of ADR-002's "table the tool carries"; the
`https://metais.org/schemas/` tree also holds the suite's schemas, under
`conformance/`, so the domain ruling covers them too.

**ADR-003 — the data specification.** I rely on: the supply arguments of
`validate`, `render` and `sample` (`data`, `set` and `set_json` in ADR-003
D17's pointer syntax, `inputs_json`) as format-level notions, not only
flags; inputs read only from what the caller names, never from the working
directory or the home directory (D12's decoys test exactly this); finding
locations for input files relative to the data directory, never
absolutized. I promise: vectors pass those arguments through unaltered.

**ADR-004 — the template environment.** I rely on: the Liquid dialect being
expressible as liquid-spec's selection — `strict2` and a `missing_features`
list for `include` and `self` — with each other departure tied to a
statement (D17); provenance's tool field recording the tool's version
exactly as `version` reports it under `render`, and `test` under `test`
(ADR-004 D15), so `render` vectors use D10 item 3's tokens and `test`
vectors need none. I pin the liquid-spec commit (D17 item 1). I promise:
chapters 7 and 8 are covered by METAIS vectors in full, besides the
delegated suite.

**ADR-005 — outputs and rendering.** I rely on: a tar profile compared by
entries (ADR-005 D17 item 5; D10 item 5), with the reference tool's bytes
proved identical across platforms by D15; staging that leaves nothing
behind — ADR-005 D14 stages beside the output directory and renames it
into place, so after a step only the declared tree has changed and D10 item
4's check holds; a dry-run listing in a defined order, with ADR-005 D16's
`digest` member; which outputs are containers and which entries are
`stored`; the execute bit, observable through `executable: true`; the
portable-path rules, which D9 item 8 reuses for the suite tree.

**ADR-006 — validation and findings.** I rely on: the stable finding members
of ADR-006 D10 item 5, which D10 item 2 compares, distinct from `message`
and `hint`; a deterministic finding order (ADR-006 D8); the registry in
`spec/findings/registry.json`, each entry naming its `statements`, from
which `12-findings.md`'s catalog is generated (D3 item 9); the exit table of
ADR-006 D12 as ADR-008 D13 maps it, for the `cli` adapter's consistency
check only. I promise: no vector compares a message or a hint.

**ADR-007 — authoring.** I rely on: the arguments and results of `check`,
`test`, `pack`, `unpack` and `sample` in D8's table; the ruling that the
sample generator's bytes are the implementation's (ADR-007 D9 item 3), so
`sample` steps are `repeatable` (D10 item 7) and followed by a `validate`
step. I promise: `B-worked-package.md` follows ADR-007's golden path, built
from vectors.

**ADR-008 — the command line.** I rely on: the command and flag names in D11
item 5's table, as ADR-008 D7 to D9 spell them; the envelope's `ok`,
`findings` and `result` members and `render`'s `result.files` for the
listing; the document on standard error when standard output carries bytes;
`version`'s `tool` object; that the tool reads only `NO_COLOR`; that it
never prints a path the caller gave or one it made absolute. I agree with
its ruling 7: the command line is not part of the format, which is why D8
defines operations and D11 maps them; the `cli` adapter is the only place
the suite knows `metais`'s flags. I note: vectors always pass every operand
explicitly, so the `.` defaults of `check`, `test`, `sample` and `pack` are
never exercised by the suite.

**ADR-009 — security.** I rely on: the hostile corpus's content and codes
(ADR-009 D17), each case committed as bytes with a lowered limit where the
default would need a large file; the limit names a vector's `limits` uses;
the canaries that pin SC7 red (ADR-009 D16 item 2, recorded in its
implementation log); proofs for what a sentinel cannot plant — the clock,
the process, the host name, the network (ADR-009 D16); the statement of why
`GODEBUG` cannot change what the tool accepts (ADR-009 D14 item 2), which
environment A tests with `zipinsecurepath=1,tarinsecurepath=1`. I promise:
the runner and its sandbox, environments A and B, the nothing-written check
and the per-step bounds of D13, which ADR-009's `TestHostIsolation`
extends.

**ADR-010 — the implementation.** I rely on: the runner in
`internal/conformance` (ADR-010 D14 item 2), driving the built binary as a
black box; the embedded schema copies with their equality test; the
liquid-spec JSON-RPC server; the reference driver for
`TestDriverProtocol`; the generators of committed hostile packages with
their reproduction tests; a release procedure that runs D15's matrix and
refuses to advertise an unreleased revision (D5 item 6); `go test -race
./...` staying within D11 item 9's budget. I promise: the runner needs only
the standard library plus the JSON Schema validator ADR-010 already chooses
(for `TestSchemasPublished` and vector validation).

## Conformance

This ADR's rules govern the document and the suite, not what a package means,
so they are proved by the tripwires of D14 and the runner's self-proof of
D18, not by vectors. Chapter 15's statements about claims are `not
observable`. One vector area belongs to this ADR: `gen/`, the vectors behind
`B-worked-package.md` and the introduction's examples, each citing the
statements of the chapters they illustrate.

## Alternatives rejected

### One specification file

A single Markdown file would put every sibling's chapter in one file that
every change edits, and an agent reading one chapter would have to outline a
document many times its size. One file per chapter keeps changes apart and
reads in place.

### Citing sections by number

Section numbers move when a section is inserted, as CommonMark's example
numbers do. An identifier assigned once and never reused survives every
reorganization, and a vector citing it stays true.

### A directory per revision

`spec/1.0/`, `spec/1.1/` as full copies would make every erratum an edit to
several copies and every comparison a diff across directories. Additions are
marked `since`, so one text serves every minor, and a tag holds each released
text exactly.

### Vectors as `metais` command lines

Writing each step as `metais` argv is the shortest path for this repository,
but it makes the reference tool's flags part of the format, which ADR-008
rules out, and it obliges a second implementation to clone the command line.
Operations keep vectors about the format; one adapter holds the mapping.

### txtar vectors

One text file per vector diffs well, but cannot hold a binary package, a
container, or a file that ends without a newline — exactly the bytes several
rules are about.

### YAML vector manifests

YAML reads `1.0` as a number, and a runner would need a YAML parser with
METAIS's own profile to read the suite without surprises. JSON needs only a
JSON parser, as the JSON Schema Test Suite asks of its users.

### An in-process runner

Calling the tool's packages directly is faster, but the run would share the
test process's environment, working directory and home, so the sentinel run
could not control them; it would see internals a second implementation
cannot offer; and it could not drive another implementation at all.

### Comparing containers by bytes against committed files

A second implementation may use another deflate encoder, and a toolchain
upgrade may change the reference encoder's output. By-entries comparison
against committed expectations, plus byte comparison between platforms of one
build (D15), proves both properties without churn.

### Matching findings by message

A message is written for a person and may be reworded or translated; codes
and locations are the contract, as test262 matches an error's type and not
its text.

### Generating hostile packages at run time

The runner would need a zip writer able to produce every malformation, and a
second implementation's runner would need the same. Committed bytes are the
vector; the generator is a convenience with a reproduction test.

### A hand-built Ruby oracle for core Liquid

Running Shopify's gem over each METAIS vector would need translation of
METAIS's variables and would cover only what METAIS authors happened to write.
liquid-spec exists, is Shopify's own, and has an adapter for any language.

### A CI matrix

There is no CI. The matrix is a release-time procedure on machines the
project has; when CI exists it runs the same commands.

## Consequences

### Positive

- Every normative sentence is findable by identifier, and the tripwires
  guarantee each observable one has a vector — R16.2 is a test, not a hope.
- A second implementation runs the suite through a small documented driver
  without adopting `metais`'s command line.
- Examples and the worked package cannot drift from behaviour, which is what
  SC9 needs.
- SC5, SC6 and SC7 are each a named run with a pass/fail result.

### Negative

- Writing a statement costs an identifier, an index line and at least one
  vector; the specification grows slower than prose would.
- Two environments double the suite's run time (D11 item 9 bounds it).
- The full matrix and liquid-spec need machines and Ruby outside `go test`,
  so they are run at releases, not on every change.
- `spec/statements.json` conflicts in git whenever two changes take the same
  identifier; this is intended, and resolved by renumbering one.

### Neutral

- Chapter content is the siblings'; this ADR fixes only where it goes and how
  it is written.

## What this does not decide

- The content of chapters 2 to 14 (ADR-001 to ADR-009).
- Where the specification is published outside this repository, and whether
  anything is hosted under `https://metais.org/`.
- The layout once a second major is opened, beyond: identifiers persist,
  1.x is frozen at its last tag, and 2.x takes `spec/`.
- Whether the PRD's example packages (SC4) also run as vectors; they are
  ADR-007's and ADR-010's.
- A register of third-party conformance claims.
- Running the suite on Windows.

## Proposed rulings

1. **Statement identifiers are `MTS-<AREA>-<NNN>`, assigned by addition,
   never reused, anchored in lower case.** Lean: yes — stable citations that
   render on GitHub. Alternative: section-number citations.
2. **Every BCP 14 keyword sits inside a statement, only `MUST`, `MUST NOT`,
   `SHOULD`, `SHOULD NOT` and `MAY` are used, and processor requirements are
   `MUST`/`MUST NOT` only.** Lean: yes — nothing normative escapes the
   identifier scheme, and vectors never face a `SHOULD`. Alternative: the
   full keyword set.
3. **One living text per major, revisions released by `spec-<M>.<m>` tags,
   additions marked `since`; no directory per revision.** Lean: yes.
   Alternative: frozen copies per revision.
4. **For a released revision the vectors are the tie-breaker; only a
   security erratum may change a released vector's expectation, and only
   toward refusal.** Lean: yes — it keeps R15.4. Alternative: the text always
   wins and released vectors change.
5. **`spec/statements.json` is generated and committed.** Lean: yes — it
   serializes identifier allocation through merge conflicts. Alternative:
   generate on demand only.
6. **Vectors are written against abstract operations; the reference tool is
   driven by a `cli` adapter and others by a driver protocol.** Lean: yes —
   consistent with ADR-008's ruling 7. Alternative: vectors as `metais` argv.
7. **Vector manifests are JSON.** Lean: yes. Alternative: YAML.
8. **Every vector runs in a sentinel environment A and a plain environment B,
   whose results must be identical.** Lean: yes — SC7 plus locale and
   time-zone independence in one mechanism. Alternative: sentinel only.
9. **A step with no declared tree may write nothing anywhere**, including
   `TMPDIR`. Lean: yes — stronger than SC6. Alternative: check only outside
   the output directory.
10. **The release matrix is every supported target, with pairwise-identical records;
    the developer gate runs the host only.** Lean: yes. Alternative: two
    targets.
11. **Core Liquid is proved by a pinned liquid-spec selection run at
    releases, with every departure tied to a statement.** Lean: yes.
    Alternative: METAIS vectors alone.
12. **Spec examples that show a package are vectors included by reference,
    and the worked package is built from them.** Lean: yes.
13. **Hostile packages are committed bytes, one defect each, refused by every
    package-opening command.** Lean: yes. Alternative: generated at run time.
14. **Runner bounds default to 10 s CPU, 1024 MiB, 120 s wall; ×20 and ×10
    under the race detector.** Lean: yes.
15. **No command-line chapter in the specification.** Lean: yes, following
    ADR-008.
16. **The specification text and the vectors are licensed Apache-2.0** like
    the rest of the repository. Lean: yes — implementers can vendor the suite
    under a familiar licence. Alternative: CC BY 4.0 for the text.
17. **Suite paths are checkout-safe on Windows (portable names, no case
    collisions, at most 160 bytes, no links) and `-text`.** Lean: yes.
18. **Archive streams and containers are compared by entries, a container's
    method only where the specification fixes it; the reference tool's own
    bytes are proved by the cross-platform records** (D10 items 3 and 5).
    Lean: yes, following ADR-005 D17 item 5 and ADR-001 D4 item 6.
    Alternative: byte comparison against committed streams, which would bind
    every implementation to one pax header layout and one encoder.
19. **liquid-spec is pinned at commit `84bf25e0`, run with `strict2`,
    excluding `include` and `self`** (D17 item 1). Lean: yes. Alternative:
    track liquid-spec's head at each release.

## References

- RFC 2119, Key words for use in RFCs to Indicate Requirement Levels —
  https://www.rfc-editor.org/rfc/rfc2119
- RFC 8174, Ambiguity of Uppercase vs Lowercase in RFC 2119 Key Words, §2
  (the boilerplate) — https://www.rfc-editor.org/rfc/rfc8174
- RFC 5234, ABNF — https://www.rfc-editor.org/rfc/rfc5234
- RFC 7405, Case-Sensitive String Support in ABNF —
  https://www.rfc-editor.org/rfc/rfc7405
- RFC 6901, JSON Pointer — https://www.rfc-editor.org/rfc/rfc6901
- RFC 8259, JSON — https://www.rfc-editor.org/rfc/rfc8259
- JSON Schema 2020-12 Core, §8.2.1 `$id` —
  https://json-schema.org/draft/2020-12/json-schema-core
- JSON Schema Test Suite, README and `test-schema.json` —
  https://github.com/json-schema-org/JSON-Schema-Test-Suite
- CommonMark spec 0.31.2 (`spec.txt`, `test/spec_tests.py`) —
  https://github.com/commonmark/commonmark-spec
- toml-test — https://github.com/toml-lang/toml-test
- YAML Test Suite and its tag registry `doc/tag-usage.md` —
  https://github.com/yaml/yaml-test-suite
- test262, `CONTRIBUTING.md` (frontmatter `esid`, `negative`) —
  https://github.com/tc39/test262/blob/main/CONTRIBUTING.md
- Shopify liquid-spec (README, `docs/json-rpc-protocol.md`, MIT licence) —
  https://github.com/Shopify/liquid-spec ; feature gates at the pinned
  commit, `lib/liquid/spec/cli/features.rb` (read 2026-10-18) —
  https://github.com/Shopify/liquid-spec/blob/84bf25e0edbca5f7e530574788b0f875ab331f50/lib/liquid/spec/cli/features.rb
- Shopify Liquid v5.14.0 `lib/liquid/variable_lookup.rb` (`first`/`last` on a
  string) — https://github.com/Shopify/liquid/blob/v5.14.0/lib/liquid/variable_lookup.rb
- Go txtar — https://pkg.go.dev/golang.org/x/tools/txtar
- Go `embed` (pattern rules) — https://pkg.go.dev/embed
- The Go Programming Language Specification, floating-point fusion —
  https://go.dev/ref/spec#Arithmetic_operators
- Go Modules Reference, module subdirectories and version tags —
  https://go.dev/ref/mod#vcs-version
- Go `os.ProcessState` — https://pkg.go.dev/os#ProcessState
- getrusage(2), Linux (`ru_maxrss` in KiB) —
  https://man7.org/linux/man-pages/man2/getrusage.2.html ; macOS getrusage(2)
  (`ru_maxrss` in bytes), read on darwin 25.5
- Go race detector, runtime overhead —
  https://go.dev/doc/articles/race_detector
- Go GODEBUG history (`zipinsecurepath`, `tarinsecurepath`) —
  https://go.dev/doc/godebug
- gitattributes, `text` unset — https://git-scm.com/docs/gitattributes
- Git FAQ, "Can I add empty directories?" —
  https://archive.kernel.org/oldwiki/git.wiki.kernel.org/index.php/GitFaq.html
- GitHub REST API, render Markdown (used to observe anchor rewriting) —
  https://docs.github.com/en/rest/markdown/markdown
- Apple, Using Intel-based apps on a Mac with Apple silicon (Rosetta 2) —
  https://support.apple.com/en-us/102527
- Apple, File system formats available in Disk Utility —
  https://support.apple.com/guide/disk-utility/file-system-formats-dsku19ed921c/mac
- Microsoft, Maximum Path Length Limitation (`MAX_PATH`, 260 characters) —
  https://learn.microsoft.com/en-us/windows/win32/fileio/maximum-file-path-limitation

## Amendments

### A1 — Chapter 2's statement area is `PKG`, not `CON`

**Status:** Proposed 2026-10-19 by PLAN-2026-10-metais-v1 Wave 3 (W3-1F,
from W3-1's finding, ruled by the plan orchestrator in plan §9's "Rulings
made during the wave"); awaiting ratification by Ed Funnekotter at the
train's close.
**Amends:** D1 item 2 (the `02-container.md` row's area), D9 item 1 (the
vector directory of a chapter 2 statement), D16 item 2 (`con/…`), and the
promise to ADR-001 under *Interfaces with sibling ADRs* (`con/` vectors).
Cross-referenced from ADR-001's implementation log.

1. **Chapter 2's area is `PKG`.** Its statements are `MTS-PKG-nnn`, and a
   vector whose first statement is one of them lives under
   `spec/conformance/vectors/pkg/`. Wherever D1 item 2's table names
   `CON`, D9 item 1 derives `vectors/con/`, D16 item 2 places a traversal
   entry in `con/…` and *Interfaces* promises ADR-001 `con/` vectors, read
   `PKG`, `vectors/pkg/`, `pkg/…` and `pkg/` vectors.
2. **Why.** D9 item 1 makes a vector's directory the lower-cased area, and
   `con` is a Windows device name, which ADR-005's portable-path rules
   refuse and D9 item 8 applies to every path under `spec/conformance/`,
   because a checkout on Windows cannot create it (`TestSuiteTreePortable`
   holds the tree to that rule). The two items
   cannot both hold for `CON`, and the stricter rule wins (RULINGS-v1's
   convention; PLAN-2026-10-metais-v1 §4). Renaming the area, rather than
   exempting chapter 2 from D9 item 1, keeps one rule for every vector's
   location.
3. **Nothing else moves.** No statement was ever identified as
   `MTS-CON-nnn`: the specification's area registry (chapter 1, §1.8) has
   named `PKG` since chapter 2's first statement, so D3's rule that an
   identifier is assigned once and never renumbered is untouched. `PKG` is
   the area of no other chapter.


### A2 — An unwritable directory in the setup, behind `unwritable-dirs`

**Status:** Proposed 2026-10-19 by PLAN-2026-10-metais-v1 Wave 4 (W4-5,
from Wave 3's closeout, question 3, which the wave 4 brief carries as
runner work); awaiting ratification by Ed Funnekotter at the train's
close.
**Amends:** D9 item 4 (the setup), D9 item 6 (the closed set of
requirements), and the vector schema
`spec/schemas/conformance/1/vector.schema.json`.

1. **The setup gains `unwritable`**, a list of directories, each created
   when the rest of the setup has not made it, and made unwritable to the
   implementation, mode 0555, after everything else D9 item 4 puts in
   place: the package and the trees the steps name included. The
   implementation can list such a directory and cannot create, rename or
   remove anything in it. The runner gives each its mode back when the
   case's steps are done, so the sandbox can be kept or removed.
2. **The requirements gain `unwritable-dirs`.** The runner probes it in
   the work directory's file system as D9 item 6 probes the others: a
   directory below the work directory's base is given mode 0555, and the
   requirement is met when the runner's own process can then create
   neither a file nor a directory in it, nor rename a file into it. It is
   not met when the process runs as root, whom a mode does not bind, nor
   where a directory's mode does not govern writing into it, as on
   Windows. A case whose requirement is not met is skipped with the
   reason and reported, never counted as a pass, as D9 item 6 says. Linux
   and darwin, run as an ordinary user, meet it.
3. **A vector whose setup names an `unwritable` directory requires
   `unwritable-dirs`**; a case that breaks this is a harness error, not a
   pass and not a skip.
4. **Why.** A statement that a failed write is reported, and what it
   leaves, has no vector without a destination the implementation cannot
   write: `output.destination.write_failed` and a render's failed commit
   among them (PLAN-2026-10-metais-v1 §8, "The unwired list"). git cannot
   hold a directory's mode, and D9 item 4 already gives the setup what git
   cannot hold. A probe, rather than an assumption, keeps a run as root
   honest: there no mode stops a write, and a vector that expects a
   failure would otherwise fail, or worse, pass for the wrong reason.
5. **Nothing else moves.** The suite's format only gains a member and a
   value within its major version (`spec/conformance/README.md`), and no
   released vector changes.

## Implementation log

- **2026-10-19, PLAN-2026-10-metais-v1 W2-3.** D14's document tripwires
  landed in `internal/guarantees` (ADR-010 D2's home for tripwires; no
  package was added): `TestStatementsWellFormed`, `TestStatementIndex`,
  `TestKeywordsOnlyInStatements`, `TestRecommendationsNotObservable`,
  `TestCoverage`, `TestCitationsResolve`, `TestFindingCatalog`,
  `TestFindingCodesCovered`, `TestExamplesMatchVectors`,
  `TestReleasedFilesUnchanged`, `TestChangelog` and `TestSchemasPublished`,
  each with a control that breaks a fixture specification once per failure
  condition of its row. D3 item 7's `spec/statements.json` and D3 item 9's
  code catalog in `12-findings.md` are generated by `go generate
  ./internal/guarantees`. Readings the text left open, each enforced by the
  tripwires:
  - `statements.json` is `{"statements": [...]}` with one statement a line,
    in area order and then by number, and `amended_by` an array, since an
    erratum adds one `amended by` attribute each time it amends;
  - a statement holding no key word has no kind under D3 item 6, and
    `TestStatementsWellFormed` refuses it;
  - a key word inside a fenced code block is outside every statement, so
    D2 item 4 refuses it there;
  - every statement whose text says `SHOULD` is marked `not observable`,
    whatever else it says;
  - each chapter's status line names exactly the areas §1.8's registry gives
    it;
  - a `released/*.sha256` list names its files by their path below `spec/`,
    and an erratum's `Files:` lines use the same paths;
  - a fenced block with a `path=` attribute outside an example is refused,
    since nothing would check it.
- **2026-10-19, PLAN-2026-10-metais-v1 W2-2.** D9 to D13, D15 and D18
  landed as `internal/conformance`, with D7 item 3's schemas under
  `spec/schemas/conformance/1/` and the driver protocol in
  `spec/conformance/README.md`. The runner is test code throughout: it
  starts processes, sets their environment and walks the file system,
  which ADR-009 D16 item 3 refuses to every non-test file outside its
  allow-list. Its test closure links no package of the module but
  `internal/testenv` (`TestBlackBox`). D18's fake is the test binary
  re-executed as `metais` in a directory named `metais-fake-<fault>`; the
  reference driver is the test binary re-executed as `metais-refdriver`,
  running the `metais` beside it with the `cli` adapter's command lines.
  Choices the decisions leave open:
  - D16 item 3's opening set is read from `metais help --json`: each
    `result.commands[]` member whose `opens_package` is true, which is
    what the declarations mark (`internal/cli`'s
    `TestOpeningSetAsTheRunnerReadsIt`).
  - The runner reads and writes zips with its own code over
    `compress/flate`: it links no part of the implementation, and
    `layers.json` gives `archive/zip` to `internal/zipfmt` alone.
  - D15 item 3's record also names each step's `invocation`, because a
    hostile open step runs several commands, and lists the cases skipped
    for an unmet `requires`. D15 item 4 compares a pair of records on the
    cases both ran: darwin's default volume folds case, so a
    `case-sensitive-fs` case runs on linux only. The suite's digest is the
    SHA-256 of `<path> <sha256>` lines, one per file, sorted
    (`record.schema.json` states it).
  - D10 item 6's containers, for a dry-run listing, are listed beside it:
    `expect.listing` is the listing's file, or `{"equals", "containers"}`.
  - D10 item 7's `files` are allowed in a repeatable step's trees only; any
    other step's tree names the tree it `equals`.
  - D13's wall clock is the runner's, as D13 item 1 gives vectors only
    `cpu_ms` and `rss_mib`; the self-proof lowers it to catch its fault in
    seconds.
  - The `cli` adapter's exit check reads each code's class from
    `spec/findings/registry.json`, the one source of codes.
- **2026-10-19, PLAN-2026-10-metais-v1 W2-NB1.** D12 item 4's two seconds
  are counted from the end of A's last process, which also satisfies "at
  least two seconds after A" read from A's start: counted from A's start,
  a host slow to run A leaves A's and B's clock readings within one
  second, and a tool that writes the clock at second resolution passes.
  The self-proof's `ab-clock` fault writes the clock in seconds, and
  `TestBStartsTwoSecondsAfterA` holds each of B's clock readings to two
  seconds after every one of A's, through each adapter.
- **2026-10-19, PLAN-2026-10-metais-v1 W3-1.** The suite's first vectors:
  chapters 2 to 4 whole, and the rules of chapters 5, 12 and 14 an open
  makes observable. D9 item 1 and D9 item 8 disagree for chapter 2's area:
  `vectors/con/` is a Windows device name, which item 8 and
  `TestSuiteTreePortable` refuse. The stricter rule stands, and the area is
  `PKG`, provisionally, pending a ruling. The `cli` adapter passes an
  `open` step's `limits` as `--limit`, as it does a `validate` step's. A
  case's setup may make part of a tree a step names — an empty directory or
  a link inside the package — and the runner now copies the tree's files
  around it, as D9 item 4 says, rather than skipping the tree.
- **2026-10-19, PLAN-2026-10-metais-v1 W3-1F.** Amendment A1, proposed,
  records W3-1's `PKG` area, ruled in plan §9. Chapter 3's §3.7 gains
  MTS-MAN-023: the canonical text of an integral number beyond
  ±(2^53−1) is not itself accepted as input, and a package whose schema
  writes one with an exponent opens. `man/large-integer-source`,
  `man/large-integer-packed` and `man/large-integer-canonical-text` cite it.
- **2026-10-19, PLAN-2026-10-metais-v1 W3-NB2.** On linux, D13 item 2's
  `ru_maxrss` is read from a process the runner does not start itself. Go
  starts a child with `CLONE_VFORK|CLONE_VM`, and the kernel's exec folds
  the high-water mark of the address space the child was executed from
  into the child's `ru_maxrss`, so read directly it counts the test
  process's peak: a whole-package `go test` without `-race` drove the
  clean fake past a 32 MiB bound. The runner starts each process through
  a stub, the test binary executed afresh, which starts it, waits for it
  and reports its usage; `TestRSSIsTheProcessAlone` holds 256 MiB in the
  test process and requires the fake's peak to stay below half of it. The
  self-proof's `tmp-refusal` fault writes to `os.TempDir()` on a refusal,
  which the snapshot of the step's `TMPDIR` catches. A tool that clears
  its environment, as `hostenv.Isolate` does, resolves the host's default
  temporary directory instead, which D11 item 4's snapshot does not
  cover; that is open.
- **2026-10-19, PLAN-2026-10-metais-v1 W4-5.** Amendment A2, proposed:
  D9 item 4's setup gains `unwritable`, and item 6's requirements
  `unwritable-dirs`, in the vector schema and `spec/conformance/README.md`.
  The runner makes each such directory mode 0555 after the rest of the
  setup and gives it 0755 back when the case's steps are done; it probes
  the requirement with a file, a directory and a rename into a 0555
  directory, and refuses, as a harness error, a case that locks a
  directory in a vector that does not require it. The self-proof covers
  the pre-state, its control (the same vector without the lock fails the
  clean fake, which writes as it is scripted), the skip a failing probe
  gives, which the record lists and no row counts, and the refusal of an
  undeclared lock. `aut/unpack-write-failed` is the first vector to
  require it. Two mechanisms spent the self-proof's 5 s wall clock on
  processes with nothing wrong with them, and both are held away without
  moving it: the race runtime sleeps a second before every race-built
  process exits (`atexit_sleep_ms`), so the runner gives a race-built
  program `GORACE=atexit_sleep_ms=0` in both environments; and the
  self-proof started every row's processes in one burst, so the runner
  starts at most `GOMAXPROCS` processes at once, each wall clock starting
  with its own process. Under the full parallel `go test -race ./...` on a
  16-core darwin host, the slowest clean self-proof process went from
  1.47 s to under 1 s, and three in four had taken over a second.
