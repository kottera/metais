# ADR-003: The data specification — named inputs declared in the manifest, JSON Schema 2020-12 with every used format asserted and every reference confined to the package, one parse rule per form, schema-directed coercion, and a data directory overridden by whole-input JSON and then by JSON Pointer path sets

**Status:** Accepted 2026-10-18 by Ed Funnekotter; its proposed rulings are ratified in [RULINGS-v1](RULINGS-v1.md).
**Date:** 2026-10-18
**Deciders:** Ed Funnekotter
**Designs:** PRD-001 R3 (items 1 to 5), R4 (items 1 to 4), R5 (items 1 to 4), and
R1 item 4 for the combined input schema
**Depends on:** ADR-001 (the `inputs/`, `examples/` and `.compiled/` layout, and
entry-name rules); ADR-002 (the manifest's top-level structure, the YAML profile,
the canonical JSON form); ADR-006 (the finding model this ADR's rules report
through); ADR-009 (every resource limit and the regular-expression profile)

## Context

PRD-001 asks for a data specification written for whoever supplies the data:
each input has a JSON Schema (draft 2020-12) and a prose meaning (R3); inputs
arrive as JSON, YAML, CSV, plain text or Markdown, singly or as a directory of
files (R4); they are supplied from a data directory, `--set` and an inline JSON
object (`--inputs-json`, ADR-008 D7) with an exact precedence, defaults applied and absence kept distinct from emptiness
(R5). A packed package carries one self-contained schema combining every input
(R1 item 4). The use cases put concrete pressure on the design:

- **§4.1, the conformance report.** One CSV table per standard; the edition
  input decides which tables appear, so which *inputs* are required depends on
  another input's value. An empty remarks cell is common.
- **§4.2, the staged incident report.** The stage is an input, and which
  fields are required follows from it. Each stage amends the previous input
  set, so a supplier changes a few nested values and re-validates.
- **§4.3, the widget.** One schema, one JSON object, no ceremony.
- **§4.4, the website.** A directory of Markdown articles with frontmatter;
  output paths are commonly derived from file names.
- **§4.8, authoring.** Sample data must be writable back as an example's
  inputs, so every form needs a parse rule with a known inverse.

The facts that constrain the design, verified against the primary sources
listed under References:

1. **`format` is an annotation by default in 2020-12.** The standard
   meta-schema declares the Format-Annotation vocabulary (validation §7.2.1),
   under which assertion "MUST be disabled by default". The optional
   Format-Assertion vocabulary requires "full validation support for all of the
   formats defined" and that implementations "MUST fail upon encountering
   unknown formats" (validation §7.2.2, §7.2.3). Validators disagree most on
   `format`, which PRD-001 §25 names as a risk.
2. **The 2020-12 meta-schema** declares the core, applicator, unevaluated,
   validation, meta-data, format-annotation and content vocabularies, all
   required.
3. **`contentEncoding`, `contentMediaType` and `contentSchema` are
   annotations**, "not assertions" (validation §8).
4. **`default` is an annotation** with no defined application behaviour
   (validation §9.2). Applying it is METAIS's own rule and must be specified
   completely.
5. **The core specification discourages network retrieval**:
   implementations "SHOULD NOT assume they should perform a network
   operation" and "SHOULD understand ahead of time which schemas they will be
   using" (core §8.2.3, §9.1.2). When no base URI is known, "a suitable
   implementation-specific default URI MAY be used" (core §9.1.1). `$id` "MUST
   NOT contain a non-empty fragment" (core §8.2.1). Bundling embeds referenced
   resources so that references need not change, and must not replace the
   referencing schema object (core §9.3.1). Unknown keywords "SHOULD be treated
   as annotations" (core §4.3.1, §6.5). Regular expressions follow ECMA-262 and
   "MUST NOT" be taken as anchored (core §6.4).
6. **RFC 4180** delimits records with CRLF, makes the header optional, counts
   spaces as part of a field, quotes fields containing a delimiter, a double
   quote or a line break, and escapes a quote by doubling it. Its `TEXTDATA` is
   printable ASCII; UTF-8 is an extension METAIS makes. **CSV has no standard
   way to tell null from an empty string**; PostgreSQL's `COPY` reads an
   unquoted empty field as null and `""` as the empty string.
7. **YAML 1.2.2's core schema** resolves `null`, `Null`, `NULL`, `~` and an
   empty value to null; `true`/`True`/`TRUE`/`false`/`False`/`FALSE` to
   booleans; base-10, `0o` and `0x` integers; floats including `.inf` and
   `.nan`; and everything else, including `2026-10-18`, to a string. It has no
   timestamp tag. Mapping keys are unique. Line breaks inside scalars are
   normalized. A byte order mark may start a document.
8. **JSON (RFC 8259)** leaves duplicate member names' behaviour
   "unpredictable"; only integers in [-(2^53)+1, (2^53)-1] are interoperable;
   a parser "MAY ignore" a byte order mark. **I-JSON (RFC 7493)** forbids
   duplicate names, unpaired surrogates and noncharacters, and numbers beyond
   binary64. **RFC 8785** canonical JSON requires I-JSON data, serializes
   numbers as ECMAScript does, and sorts member names by UTF-16 code units.
9. **JSON Pointer (RFC 6901)** escapes `~` as `~0` and `/` as `~1`, forbids
   leading zeros in array indices, and gives `-` as the member after an
   array's last element.
10. **Go's `encoding/csv`** ignores blank lines, converts `\r\n` to `\n`
    including inside quoted fields, refuses a bare quote unless `LazyQuotes`,
    and does not mention a byte order mark (a library choice is ADR-010's;
    the rules below are stated so a library can be checked against them).
11. **Frontmatter is a convention, not CommonMark.** Jekyll's rule is that it
    "must be the first thing in the file" and be YAML "set between
    triple-dashed lines". In CommonMark a `---` line is a thematic break.
12. **Helm's `--set`** uses dotted paths with backslash escapes and infers
    types from the text, which is why Helm also has `--set-string` and
    `--set-json`.

## Decision

### D1 — Inputs are declared in the manifest as an ordered list of named slots

1. **The manifest field `inputs` is a list**, in the order the author wants a
   supplier to meet them; `inspect` and the combined schema keep that order.
   The member is required (ADR-002 D4) and may be empty (`inputs: []`, a
   package with no inputs). A list, not an
   object keyed by name, because the order is part of the guidance and JSON
   object order is not significant.
2. **Each entry is an object with these fields, and no others** (an unknown
   field is a finding):

   | Field | Required | Value | Default |
   |---|---|---|---|
   | `name` | yes | the input's name (item 3) | — |
   | `title` | yes | plain text, 1 to 200 characters, no line break | — |
   | `description` | no | plain text, line breaks allowed | absent |
   | `schema` | yes | package path of the input's schema file (D3) | — |
   | `meaning` | no | package path of a Markdown meaning document (D2) | absent |
   | `forms` | no | non-empty list, without repeats, of `json`, `yaml`, `csv`, `text`, `markdown` | `[json, yaml]` |
   | `csv` | no | object with one field, `delimiter` (D11 item 2); only when `forms` contains `csv` | `{delimiter: ","}` when `forms` contains `csv` |
   | `collection` | no | boolean (D15) | `false` |
   | `required` | no | boolean: whether the input must be supplied (D6) | `true` |

3. **A name matches `^[a-z][a-z0-9_]{0,63}$`** and is unique among the
   package's inputs. Lower case only, so that two data-directory files can
   never differ only in letter case; no hyphen and no dot, so that the name
   is a Liquid identifier in every implementation, a file stem with one
   meaning, and a JSON Pointer token needing no escape. These names are
   **reserved and refused**: `blank`, `empty`, `false`, `first`, `last`,
   `nil`, `null`, `size`, `true` — Liquid gives each a meaning as a literal or
   as a property of a collection, and `inputs.size` must never be ambiguous.
4. **Package paths** (`schema`, `meaning`, and `input_constraints` in D6) are
   relative to the package root, use `/`, and lie under `inputs/`. A path that
   names no entry, or an entry outside `inputs/`, is a finding.
5. **`forms` governs files only.** A value given on the command line (D17) is
   a JSON value or a path set and is accepted whatever `forms` says.
6. **Form and schema shape must agree**, judged on the root schema's
   statically declared `type` (D14 item 1) — for a collection, on its item
   schema: `csv` needs a type set that includes `array`, with an item schema
   whose type set includes `object`; `text` needs `string`; `markdown` needs
   `object`; `json` and `yaml` take anything. A disagreement is a `check`
   finding (ADR-007 runs it).
7. **Defaults of these fields are the specification's, not the canonical
   manifest's.** `.compiled/manifest.json` carries exactly what the author
   wrote (ADR-002 D9 item 2); a consumer applies this table's defaults.

### D2 — Meaning is prose, attached to the slot

1. **The slot's `title` and `description` are the input's meaning in brief**
   and live in the manifest, not the schema. One schema file may serve two
   slots — `billing_address` and `shipping_address` sharing
   `inputs/address.schema.json` — and the slot means something the type does
   not. The schema's own `title` and `description` describe the type and its
   fields; findings use the slot's for a root-level finding and the schema's
   at deeper locations (ADR-006).
2. **The meaning document** is CommonMark, UTF-8, extension `.md`, under
   `inputs/`. It says what the data is for, what good data looks like, and how
   the package uses it. METAIS gives it no structure; one document may serve
   several inputs. `init` scaffolds it as `inputs/<name>.md` (ADR-007); the
   path is always declared, never discovered.
3. **Schema annotations are part of the meaning**: `title`, `description`,
   `examples`, `deprecated`, `readOnly` and `writeOnly` are carried into the
   combined schema unchanged. None of them changes validation.
4. **Nothing names a source.** There is no field for where data comes from;
   guidance about sources is prose in the meaning document or the
   manifest-level supplier guide (ADR-002).

### D3 — The schema dialect is JSON Schema 2020-12, strict about what it does not know

1. **Every schema file is JSON (`.json`) or YAML (`.yaml`, `.yml`)** under
   `inputs/`, read with D9 or D10's rules. Every such file under `inputs/` is
   loaded as a schema resource, whether or not an input names it, so that
   `$id` lookup (D5) is complete and deterministic, and each must compile.
   `inputs/` holds nothing but schema files and `.md` documents; any other
   entry there is a `check` finding.
2. **`$schema`, where present, is exactly
   `https://json-schema.org/draft/2020-12/schema`**, in a file's root and in
   any embedded resource. Absent, that dialect is assumed. Any other value,
   and any `$vocabulary` keyword, is a finding.
3. **Asserted:** every keyword of the core, applicator, unevaluated and
   validation vocabularies, among them `$ref`, `$dynamicRef`, `allOf`,
   `anyOf`, `oneOf`, `not`, `if`/`then`/`else`, `dependentSchemas`,
   `dependentRequired`, `properties`, `patternProperties`,
   `additionalProperties`, `propertyNames`, `prefixItems`, `items`,
   `contains`/`minContains`/`maxContains`, `unevaluatedProperties`,
   `unevaluatedItems`, `type`, `enum`, `const`, the numeric, string, array
   and object constraints, and `format` under D4.
4. **Annotations only:** the meta-data vocabulary, with `default` given
   METAIS's application rule (D19); the content vocabulary
   (`contentEncoding`, `contentMediaType`, `contentSchema`), never asserted,
   as 2020-12 defines them.
5. **An unknown keyword is a finding, except one beginning `x-`.** The core
   specification lets an unknown keyword pass as an annotation; METAIS does
   not, because a misspelt `requried` or `maxLenght` is a rule the author
   believes is enforced and is not. `x-` keywords are the author's own
   annotations, carried unchanged and never interpreted.
6. **`pattern` and `patternProperties`** use the package regular-expression
   profile ADR-009 fixes for every regular expression a package supplies,
   unanchored as core §6.4 requires. A pattern outside the profile is a
   finding.
7. **Validation collects every error that can be a finding.** Every
   assertion whose failure ADR-006 D6 reports is evaluated to completion:
   `allOf`, `properties`, `patternProperties`, `additionalProperties`,
   `items`, `prefixItems`, `$ref` and the `then`/`else` a condition selects
   collect every failure, and an `anyOf` or `oneOf` with no matching branch
   collects every branch's failures (ADR-006 D6 item 6). Three contexts are
   evaluated only for their yes or no, because none of their inner failures
   is ever a finding: the subschema of `not` (its finding is `not` itself),
   the `if` subschema, and the remaining branches of an `anyOf` or `oneOf`
   once a branch has matched (a second match is `one_of.ambiguous`, with no
   inner failures). So every failing location is reported (R6 item 1);
   ADR-006 de-duplicates and orders them.

### D4 — Every `format` a package uses is asserted; a format METAIS does not assert cannot be used

1. **These formats are asserted**, with the definitions validation §7.3 cites:
   `date-time`, `date`, `time` and `duration` (RFC 3339 §5.6 and Appendix A),
   `email` (RFC 5321 §4.1.2 `Mailbox`), `hostname` (RFC 1123 §2.1), `ipv4`,
   `ipv6` (RFC 4291 §2.2), `uri` and `uri-reference` (RFC 3986), `uuid` (the
   RFC 4122 string form), `uri-template` (RFC 6570), `json-pointer` (RFC 6901
   §5) and `relative-json-pointer`. The specification carries each one's exact
   grammar, including the cases validators split on: lower-case `t`/`z` and
   leap seconds in `date-time`, address literals and quoted local parts in
   `email`, and trailing dots in `hostname`.
2. **Any other `format` value is a finding**: `idn-email`, `idn-hostname`,
   `iri`, `iri-reference`, `regex`, and any name outside §7.3. The IDN and IRI
   formats need Unicode mapping tables on which implementations disagree, and
   `regex` names ECMA-262, which ADR-009's profile deliberately is not. A
   format left unasserted would let a supplier believe a value had been checked
   when it had not; refusing it matches what §7.2.3 requires of an asserting
   implementation. An author who needs one of them writes a `pattern` or
   states the expectation in prose.
3. **`format` applies to strings only**; on any other instance type it passes,
   as §7.3 states.
4. **This is METAIS's own rule over the 2020-12 dialect**, stated in the
   specification, not a claim that METAIS implements the Format-Assertion
   vocabulary in full. Packages keep the standard `$schema` URI.

### D5 — References resolve inside the package, never over the network

1. **Each schema file's retrieval URI is `metais:///` followed by its package
   path**, its characters percent-encoded where RFC 3986 requires: the file
   `inputs/vendor.schema.json` is `metais:///inputs/vendor.schema.json`. The
   scheme names nothing fetchable, so no library mistakes a reference for a
   download, and relative references resolve by RFC 3986 §5.2 against the
   file's place in the package.
2. **A file's canonical URI is its root `$id`, resolved against its retrieval
   URI, or its retrieval URI when it has none.** `$id` may be any absolute or
   relative URI without a non-empty fragment (core §8.2.1). A published schema
   copied into a package keeps its `https://…` `$id`.
3. **A `$ref` or `$dynamicRef` resolves to one of**: a resource in the
   package by canonical URI or by an embedded `$id`; an anchor (`$anchor`,
   `$dynamicAnchor`) or JSON Pointer fragment within one; the retrieval URI of
   a package schema file whose root declares no `$id`; or the 2020-12
   meta-schemas, which every implementation carries built in. Anything else is
   a finding, and nothing is ever fetched.
4. **A file whose root declares an `$id` is reached only by that `$id`.** A
   reference to its retrieval path is a finding naming the `$id` to use, so
   that every resource has exactly one name and bundling (D7) rewrites no
   reference.
5. **A reference whose path would climb above the package root**, before
   RFC 3986's dot-segment removal clamps it, is a finding; so is one
   resolving outside `inputs/`. (An output validation schema, ADR-005 D11,
   follows these rules with "`inputs/`" read as the top-level directory
   that holds the `validate` file.) Path components are percent-decoded as UTF-8
   to find the entry.
6. **Two resources with one canonical URI** (two `$id`s alike, or an `$id`
   equal to another file's retrieval URI) are a finding.

### D6 — Static requiredness in the manifest, conditional requiredness in a cross-input schema

1. **`required: true` makes an input mandatory unconditionally**; an absent
   required input is a finding at the input's root.
2. **The optional top-level manifest field `input_constraints`** names one
   schema file under `inputs/` that is evaluated against the object holding
   every input (D7). It is how a package says that one input's value decides
   what another must contain or whether it must be present — the §4.1
   edition deciding which tables are required, the §4.2 stage deciding which
   fields of the report are mandatory:

   ```yaml
   # inputs/constraints.schema.yaml
   allOf:
     - if:   { properties: { stage: { const: final } }, required: [stage] }
       then:
         required: [incident]
         properties:
           incident: { required: [root_cause, resolution_date] }
   ```

   Its instance locations are inputs-object pointers, so a finding at
   `/incident/root_cause` is reported against input `incident` at
   `/root_cause`.
3. **A constraint that depends on a single input belongs in that input's own
   schema**; `check` warns when an `input_constraints` subschema reads one
   input only (ADR-007).

### D7 — The combined schema is one object schema with a property per input

1. **`pack` writes `.compiled/schema.json`** (the name ADR-001 D7 reserves),
   a Compound Schema Document (core §9.3.1), in the canonical JSON form
   (ADR-002 D9):

   ```json
   {
     "$schema": "https://json-schema.org/draft/2020-12/schema",
     "$id": "metais:///.compiled/schema.json",
     "type": "object",
     "properties": {
       "vendor": {
         "title": "Vendor",
         "description": "The organisation that makes the product.",
         "$ref": "metais:///inputs/vendor.schema.json"
       }
     },
     "required": ["vendor"],
     "additionalProperties": false,
     "allOf": [{ "$ref": "metais:///inputs/constraints.schema.json" }],
     "$defs": {
       "inputs/vendor.schema.json": { "$id": "metais:///inputs/vendor.schema.json", "…": "…" },
       "inputs/constraints.schema.json": { "$id": "metais:///inputs/constraints.schema.json", "…": "…" }
     }
   }
   ```

2. **`properties`** holds one member per input, carrying the slot's `title`
   and `description` beside a `$ref` to the input's canonical URI (2020-12
   allows siblings of `$ref`). `required` lists the inputs with
   `required: true`, in manifest order. `allOf` is present only when
   `input_constraints` is declared.
3. **`$defs` embeds every schema file under `inputs/`**, keyed by package
   path, each as its own resource with `$id` set to its canonical URI and
   `$schema` set, and YAML files converted to JSON. Because D5 item 4 gives
   each resource one name, no reference inside an embedded resource changes.
4. **This is what "internal `$ref`s inlined" means here: the document is
   self-contained, not reference-free.** Substituting each `$ref` with its
   target cannot express a recursive schema, changes the base URI that
   relative references inside the target resolve against, and breaks
   `$dynamicRef`. Bundling is the form 2020-12 defines for this.
5. **Validation evaluates exactly this document** against the inputs object
   (D22). A tool reading a packed package compiles the combined schema from
   `inputs/` and the manifest and refuses the package when it differs from
   `.compiled/schema.json`, so a supplier reading the source schemas
   and a validator using the compiled one can never disagree.

### D8 — Common rules for every input file, and the data model

1. **Every input file is UTF-8**; invalid UTF-8 is a finding at the line and
   byte where it occurs. A single leading byte order mark (`EF BB BF`) is
   removed before parsing, in every form — spreadsheet programs write one when
   exporting UTF-8 CSV. Any other encoding, UTF-16 included, is a finding.
2. **For `csv`, `text` and `markdown`, line breaks — CRLF, CR and LF — are
   normalized to LF before parsing**, inside quoted CSV fields too. YAML
   normalizes its own (YAML §5.4); a JSON string cannot hold a raw line break.
   A parsed value therefore never depends on the line-ending convention of
   the machine that wrote the file, and the same input checked out on two
   platforms renders the same bytes (R10 item 2). The raw bytes (D21) keep
   the original line breaks.
3. **The data model is JSON's**, in the I-JSON profile: null, boolean,
   number, string, array, and object. Objects are unordered and their member
   names unique; the data model gives member order no meaning, so a value
   supplied as JSON, as YAML and as CSV is the same value. The order a
   template iterates an object's members in is ADR-004's.
4. **Strings** are sequences of Unicode scalar values; an escaped unpaired
   surrogate or a noncharacter is a finding (RFC 7493 §2.1).
5. **There is one number type, IEEE 754 binary64**, as JSON Schema's data
   model has one: `1` and `1.0` are the same value, equal under `const` and
   `enum`, both valid against `type: integer` (validation §6.1.1), with the
   same digest. A number with zero fractional part and magnitude at most
   2^53−1 is presented to templates as an integer, any other as a float
   (ADR-004 renders them). A numeric literal is rounded to the nearest
   binary64, ties to even; one written without fraction or exponent whose
   magnitude exceeds 2^53−1 is a finding ("supply it as a string"), because it
   would otherwise change silently; one that overflows to infinity is a
   finding; YAML's `.inf` and `.nan` are findings.
6. **Every parsed value keeps its source position** — file, line and column
   for JSON, YAML, frontmatter and text; file, record line and column header
   for CSV; the argument's position for the command line — so that a finding
   names the line where there is one (R6 item 2).

### D9 — JSON

1. **A file is one RFC 8259 JSON text**, any value at the top level;
   whitespace-only or trailing content is a finding.
2. **A duplicate member name in any object is a finding**, every occurrence
   reported. Names are compared after escapes are decoded (RFC 7493 §2.3).
3. Numbers follow D8 item 5.

### D10 — YAML

1. **Input YAML is the manifest's YAML profile (ADR-002 D2), unchanged**,
   applied after D8 item 1 has removed a leading byte order mark: YAML
   1.2.2 syntax; a single document; no directives; no anchors, aliases,
   merge keys or explicit tags of any kind; mapping keys taken as their
   scalar text; duplicate keys a finding; and plain scalars resolved only by
   ADR-002 D2 item 9's table — `null`, `true`, `false`, the RFC 8259 number
   grammar, and everything else a string, with `~`, `Null`, `True`, a
   leading zero, `0o`, `0x`, `.inf` and `.nan` refused.
2. **On top of that profile**, numbers follow D8 item 5 (so an integer
   written without fraction or exponent beyond 2^53−1 is a finding).
3. **A plain `2026-10-18` is a string** (the core schema has no timestamp);
   `key:` with nothing after it is null, not absent (D20). A plain `yes` or
   `on` is a string.
4. An empty file, or one holding only comments, is null.

### D11 — CSV

1. **A CSV file is a header record and zero or more data records**, parsed
   by RFC 4180 with these definitions:
   - records end in LF after D8 item 2, the last optionally;
   - fields are separated by the input's delimiter (item 2);
   - a field is quoted when it begins with `"`; inside it, `""` is a quote and
     the delimiter and LF are data; after the closing quote the next
     character is the delimiter, LF or end of file, and anything else is a
     finding;
   - a `"` inside an unquoted field is a finding; an unterminated quoted
     field is a finding;
   - spaces are data, never trimmed (RFC 4180 §2 rule 4);
   - **an empty line is not a record** and is ignored wherever it occurs, as
     Go's reader does; a single-column CSV writes an empty string as `""`.
2. **`delimiter` is one of `,` `;` `|` and the tab character** (in YAML and
   JSON, `"\t"`). A closed set covers comma-separated, the semicolon files
   spreadsheet programs write in locales with a decimal comma, pipe-separated
   and tab-separated exports, and lets every implementation agree; the file
   extension stays `.csv` whatever the delimiter.
3. **The header is required.** Each name is non-empty, unique by exact
   comparison, and taken as written (no trimming, no case folding). A file
   with no header record is a finding.
4. **Each data record becomes an object**, member names from the header.
   A record whose field count differs from the header's is a finding naming
   its line.
5. **An unquoted empty field is absent; a quoted empty field `""` is the
   empty string.** This is PostgreSQL's convention, and R5 item 4 requires
   it: an optional remarks column left blank in a spreadsheet reaches the
   template as `nil`, not as an empty string that Liquid treats as true.
   An absent member then takes its schema default (D19).
6. **Every other field is text, coerced by D14** against the row schema's
   declared type for its column. A header naming no declared property yields
   strings; whether such a column is allowed is the schema's question
   (`additionalProperties`).
7. **CSV cannot carry null, an array or an object.** A column whose declared
   type set holds only `array`, `object` or `null` types is a `check`
   finding; a supplier who needs those uses JSON or YAML.
8. **A CSV file's value is an array of objects**, wherever it is supplied —
   as an input, or as one item of a collection (an item that is itself a
   table). Its row schema is the `items` subschema of the schema at that
   place (D14 item 1).
9. Any structural finding (items 1, 3, 4) means the file contributes no
   value (D22 item 3); every structural finding the parser can reach is
   still reported.

### D12 — Plain text

1. **The value is the whole file as one string**, after D8 items 1 and 2,
   **with one final LF removed** when there is one. A text file's last line
   ends in a line terminator that is not part of its content, and with this
   rule a paragraph supplied in `summary.txt` and the same paragraph given on
   the command line are the same value. An empty file is the empty string —
   present, not absent.

### D13 — Markdown with frontmatter

1. **The value is an object**: the frontmatter's fields, plus `body`.
2. **Frontmatter is present when the file's first line is exactly `---`.**
   It ends at the next line that is exactly `---`; no closing line is a
   finding. Between them is YAML in D10's profile, which must be a mapping
   or empty. A sequence or scalar there is a finding. A frontmatter field
   named `body` is a finding (R4 item 3). A file whose first line is anything
   else has no frontmatter; an author wanting a thematic break first writes
   `***`.
3. **`body` is the text after the closing line** (or the whole file without
   frontmatter), under D12's rule: line breaks normalized, one final LF
   removed, nothing else changed. A blank line after the frontmatter stays
   part of the body. The body is not parsed: every string is valid CommonMark.
4. **A Markdown item schema must declare `body`** when it forbids other
   members (`additionalProperties: false` or `unevaluatedProperties: false`);
   otherwise every item fails, and `check` reports it.

### D14 — Schema-directed coercion, and the static walk that finds a schema

1. **The static subschema at a path** is found by walking from a schema
   through `$ref` (to its target), `properties` (for a member name),
   `prefixItems` (for an index it covers) and `items` (for any other index),
   and through nothing else. Its **declared type set** is the `type` value of
   the first schema in the chain from that location through `$ref` targets
   that declares one, as a set; with no `type`, the set is empty. The walk
   never evaluates `allOf`, `anyOf`, `oneOf` or conditionals, so its answer
   does not depend on the data; an author who wants coercion declares `type`
   on the property itself.
2. **A text value is coerced to the declared type set T** — used for CSV
   fields (D11) and text path sets (D17):
   - `boolean` in T and the text is one of YAML's core forms `true`, `True`,
     `TRUE`, `false`, `False`, `FALSE` → that boolean;
   - otherwise `integer` in T and the text matches `-?(0|[1-9][0-9]*)` → that
     number;
   - otherwise `number` in T and the text matches the JSON number grammar
     (RFC 8259 §6) → that number;
   - otherwise → the text, as a string.
3. **Coercion never fails.** Text that fits no listed type stays a string,
   and validation then reports the `type` failure with the field's title and
   description, so `" 42"`, `007`, `1,5` and `yes` in an integer column
   each produce the same useful finding rather than a parse error.
4. **Dates and times stay strings**, as JSON Schema has them (`type: string`
   with `format: date`); there is no date coercion.

### D15 — Collections

1. **`collection: true` makes the input an array of items**, its `schema`
   the schema of the whole array: its root type set includes `array`, its
   `items` is the item schema, and `minItems`, `maxItems` and `uniqueItems`
   bound the collection. A collection whose schema declares no array type
   or no `items` is a `check` finding. `forms` lists the forms an item file
   may take.
2. **Supplied from a data directory, a collection is a directory**
   `<data-dir>/<name>/`; a file `<name>.<ext>` for a collection input is a
   finding naming the directory form. On the command line it is a JSON array
   (D17).
3. **The items are the directory's regular files**, one item each:
   - an entry whose name begins with `.` is ignored, file or directory
     (`.DS_Store`, `.gitkeep`);
   - a subdirectory is a finding — collections are flat;
   - a file with no extension, or one whose extension (item 4) maps to no
     form in `forms`, is a finding listing the accepted extensions;
   - a symbolic link or special file is a finding and is not followed (D16
     item 6).
4. **An item's form comes from its last extension**, matched exactly in
   lower case: `.json` → `json`; `.yaml`, `.yml` → `yaml`; `.csv` → `csv`;
   `.txt` → `text`; `.md`, `.markdown` → `markdown` when `forms` contains
   it, otherwise `text` when `forms` contains that. One collection may mix
   forms.
5. **An item's name is its file name normalized to Unicode NFC; its stem is
   the name without its last extension.** Two items with equal stems
   (`intro.md` and `intro.json`), or stems that differ only in letter case
   (by ADR-005's case comparison), are a finding: the stem is the item's
   identity, it is commonly an output path, and a data directory holding both
   cannot exist on a case-insensitive file system.
6. **Items are ordered by name, comparing the UTF-8 bytes of the NFC names**
   — code point order, case-sensitive, no locale. NFC first, because one name
   arrives as different bytes from different file systems.
7. **An empty directory is an empty array**, present; a missing directory is
   an absent input (D20).

### D16 — The data directory

1. **One directory holds the supplied files**: `<name>.<ext>` per input,
   `<name>/` per collection input. `examples/<example>/inputs/` inside a
   package has exactly this format (ADR-001, ADR-007).
2. **A top-level entry's input name is the text before its first `.`**, and
   the rest is its extension, mapped to a form as in D15 item 4:
   `vendor.json` is `vendor` as JSON; `vendor.schema.json` has the unknown
   extension `.schema.json` and is a finding.
3. **An entry that names no declared input is a finding** — a misspelt
   `vendr.json` must not silently leave an optional input absent.
   Names beginning with `.` are ignored.
4. **Two entries for one input** (`vendor.json` and `vendor.yaml`, or
   `articles.json` and `articles/`) are a finding.
5. **An extension whose form the input does not accept** is a finding
   listing the accepted extensions.
6. **Symbolic links and special files in the data directory are findings and
   are never followed or read.** The data directory is often written by a
   program, and following a link is how a file the caller never named would
   reach a template (R14 item 4). A caller who means to supply a file copies
   it in. The data directory path itself is the caller's argument and is
   resolved as given.

### D17 — The command line: whole inputs as JSON, single values by JSON Pointer

1. **`--inputs-json` supplies one JSON object whose members are input names**
   and whose values are those inputs' whole values (a collection's is an
   array). It follows D9's rules. A non-object, or a member naming no
   declared input, is a finding. It may be given more than once. Its
   argument is inline JSON text, or `-` for standard input, at most once
   (ADR-008 D7).
2. **A path set addresses one value by an inputs pointer**: a JSON Pointer
   (RFC 6901) whose first token is an input name — `/vendor/name`,
   `/criteria/3/remarks`, `/articles/0/title`. The argument is split at its
   first `=`: the pointer before, the value after. A member name containing
   `=` is unreachable this way and is supplied with `--inputs-json` or a file. A
   pointer not beginning with `/`, or whose first token names no declared
   input, is a finding.
3. **There are two kinds of path set**: a **text set** (`--set
   /vendor/name=Acme Ltd`), whose value is coerced by D14 against the static
   subschema at the pointer, and a **JSON set**, whose value is a JSON text
   (D9) placed as it is — the only way to set null, an array or an object at
   a path — given with `--set-json /criteria/3=null`, as Helm names the same
   thing (ADR-008 D7). A text set with an empty value sets the empty string,
   then coerced.
4. **Applying a path set** to the inputs object assembled so far: each token
   but the last must reach an existing object or array, except that a
   missing object member is created as an empty object; an array index token
   must be an existing index without leading zeros; `-` is accepted only as
   the last token and appends; the last token's member or element is created
   or replaced. Traversing into a string, number, boolean or null, an index
   out of range, or `-` before the last token is a finding, and the set is
   not applied. Arrays are never created implicitly; a JSON set creates one.
5. **Command-line values have no file form, so `forms` does not apply**
   (D1 item 5), and they have no raw bytes (D21).

### D18 — Precedence

1. **The supplied inputs object is assembled in exactly this order**, each
   step applied to the result of the one before:
   1. the data directory: each file or directory gives its input's whole
      value;
   2. each `--inputs-json` object in command-line order: each member **replaces**
      its input's whole value — never merged;
   3. each path set, text or JSON, in command-line order: it replaces only
      the value at its pointer.
2. **The order is fixed by kind, not by position between kinds**: every
   `--inputs-json` applies before every path set, wherever they appear on the line;
   among one kind, the later wins.
3. **Partial overrides of nested values are path sets.** With `vendor.json`
   holding `{"name": "Acme", "address": {"city": "Lyon", "zip": "69001"}}`,
   `--set /vendor/address/city=Paris` changes the city and keeps the rest;
   `--inputs-json '{"vendor": {"address": {"city": "Paris"}}}'` replaces the whole
   vendor and drops `name` and `zip`. Whole replacement is chosen over merging
   because a merge has to give some value the meaning "delete", and in the
   data model null is a value.
4. **A path set into an input whose file did not parse** is a finding and is
   not applied. A file that did not parse stays a finding even when `--inputs-json`
   replaces its input: every supplied file must parse.

### D19 — Defaults

1. **Defaults are applied after D18 and before validation**, to inputs from
   every source alike, by a walk over the instance:
   - **an absent optional input** whose root schema has `default` takes a
     copy of it;
   - **at an object** whose static schema (D14 item 1) has `properties`,
     each member that is absent, has a `default` in its subschema (on the
     subschema itself, else on the first `$ref` target in its chain that has
     one), and is not named in the `required` list of the same schema object
     that holds that `properties`, takes a copy of the default; the schema
     object's own `properties` are applied before those reached through its
     `$ref`, and a member already present is left alone;
   - **at an array**, each existing element is walked with its static
     subschema (`prefixItems`, then `items`) — so a blank CSV cell takes the
     column's default;
   - the walk then descends into every member and element, including the
     ones a default just supplied, so a default object's own members take
     theirs.
2. **A default never creates a parent.** A member default applies only where
   the enclosing object exists; a missing parent object stays missing unless
   it has a default of its own.
3. **A present value is never replaced**; `null` is present (D20).
4. **Defaults under any other keyword** — `allOf`, `anyOf`, `oneOf`,
   `if`/`then`/`else`, `dependentSchemas`, `patternProperties`,
   `additionalProperties` — are annotations and never applied; `check` warns
   at each one. A required member with a default is likewise warned: the
   default is never used.
5. **A default is validated with the rest of the instance.** A default that
   fails its schema is the package's error; `check` validates every applied
   default (ADR-007), so it is caught before a supplier meets it.
6. Because a default is inserted only through `properties`, an inserted
   member is always evaluated by that keyword and never trips
   `additionalProperties: false` or `unevaluatedProperties: false`.

### D20 — Absence and null

1. **Absent is not supplied, and no default applied**: an input not given,
   an object member not given, an unquoted empty CSV field. Validation sees
   the member missing (so `required` reports it). A template sees an absent
   input as `nil`, and an absent member as `nil` when the schema declares it
   (ADR-004 D6 items 2 and 3); reading a member neither declared nor
   supplied is a template error (ADR-004 D6 item 4).
2. **Null is a supplied value**: JSON `null`, YAML `null` or an empty YAML
   value, a JSON set of `null`. A template also sees `nil`, but validation
   checks it against its schema, and a default never replaces it. A
   frontmatter line `subtitle:` is null, and fails a `type: string` schema.
3. **Nothing is ever filled with `""`, `[]` or `{}`** by METAIS: those values
   exist only when supplied, created by a path set's intermediate step (D17
   item 4), or given by a default.

### D21 — What a render is given for each input

For every declared input the data layer hands the template environment
(ADR-004) an **input record**; ADR-004 decides every syntax through which a
template reaches it.

1. **`value`** — the validated value after defaults, or absent; bound to
   `inputs.<name>`.
2. **`raw`** — the original bytes, exactly as read, BOM and line breaks
   included, or none. An input has raw bytes when its whole value came from
   one data-directory file and no `--inputs-json` member or path set touched it.
   A collection has no raw form of its own; **each item has one** under the
   same rule, applied per item (`--set /articles/0/title=…` removes item 0's
   raw form and no other's). A render whose templates need a raw form that
   does not exist is refused with a finding naming the input or item and the
   template, before anything renders.
3. **`source`** — for an input: `file` with its data-directory path, or
   `command-line`, or `default`; for each collection item: its `name` and
   `stem` (D15 item 5) and its form, or none for an item supplied on the
   command line. Values have no identity in Liquid, so these are reached by
   input name and item index, never through the value.
4. **`digest`** — SHA-256 over the canonical JSON serialization (RFC 8785,
   ADR-002 D9) of `value`, for the input and for each item;
   none when absent. Computed after defaults, because it identifies the data
   a render used: an omitted field and the same field supplied with its
   default value render alike and digest alike. This is the input digest the
   provenance tag records (R11).

### D22 — The order of a validation

1. **Package**: the manifest, every schema and the combined schema are read
   and compiled; a package finding refuses the package (R14 item 3).
2. **Supply**: the data directory is scanned and every file parsed; every
   `--inputs-json` and path-set argument parsed. Every supply finding is collected.
3. **Assembly**: D18, then D19. An input whose file did not parse is
   **unparsed**: it is left out of the instance and no `required` finding is
   reported for it, since its parse findings already are.
4. **Validation**: the instance is evaluated against the combined schema.
   When any input is unparsed, the `input_constraints` subschema is not
   evaluated, and an `info` finding says so: cross-input rules over an
   incomplete instance would report errors that are not there.
5. **Every finding from steps 2 to 4 is reported together** (R6 item 1);
   ADR-006 orders and de-duplicates them. Render proceeds only when none is
   an error (R6 item 4).

## Interfaces with sibling ADRs

**ADR-001 (container and layout).** Relied on: `inputs/` holds the schema
files and meaning documents; `examples/<example>/inputs/` is a data directory
in D16's format; `.compiled/schema.json` is the name ADR-001 D7 reserves for
the combined schema, covered by the digest file and re-derived at open
(ADR-001 D8 step 7); entry names are NFC UTF-8 (ADR-001 D2), and this ADR
NFC-normalizes collection item names, which come from the host, regardless.
Promised: D3 item 1's rule for what `inputs/` may hold.

**ADR-002 (manifest and versioning).** Relied on: the YAML profile of
ADR-002 D2, which D10 adopts unchanged; RFC 8785 as the canonical JSON form,
which D7 and D21 item 4 use; the manifest's JSON Schema including `inputs`
(D1) and the root member `input_constraints` (D6), both in ADR-002 D4's
table; an unknown field refused relative to the package's specification
version; the canonical manifest materializing no default (ADR-002 D9 item
2), so D1's defaults are stated by the specification and applied by every
reader. Promised: the field definitions, with types and defaults, for that
schema.

**ADR-004 (template environment).** Relied on: `inputs.<name>` bound to D21
item 1's `value`, absent and null both `nil`, with declared-absent members
materialized as `nil` and undeclared absent members an error by ADR-004 D6
items 2 to 4 (D20 item 1 cites them); D8 item 5's number rule (integer when
the fractional part is zero and the magnitude is within 2^53−1, float
otherwise), restated by ADR-004 D4 item 2; object members iterated in RFC
8785's canonical order, by UTF-16 code units (ADR-004 D4 item 5) — collection
items and every other ordering in this ADR stay in UTF-8 byte order, since
they order names, not object members; `embed` reading D21 item 2's `raw` by
input name and, for collections, by item index (ADR-004 D12); D21 item 3's
`source` exposed as the `sources` global (ADR-004 D7 item 9); the provenance
tag recording D21 item 4's digest. Promised: the input record of D21; the
finding for a missing raw form; that every value in the data model is one of
D8 item 3's kinds.

**ADR-005 (outputs and rendering).** Relied on: `for_each` over a collection
input exposing the item index (`fan_out.index0`, ADR-004 D7 item 5), so the
item's record is reachable; ADR-005 D12 item 6's canonical caseless match,
which D15 item 5 reuses for stems; output validation schemas (ADR-005 D11)
following D3 to D5 in full, with D5 item 5's directory read as the
`validate` file's top-level directory. Promised: stems are unique and
case-distinct within a collection.

**ADR-006 (findings).** Relied on: a finding carries input name, file, line
and column where there is one, and a JSON Pointer within the input; the
human rendering's `at` line prints the inputs pointer
(`/vendor/address/city`), which is a valid `--set` path (ADR-006 D11 item
1). Severities `error`, `warning` and `info` exist. Promised: D8 item 6's
source positions; the slot title and description for root-level findings
(D2 item 1); these codes, registered in ADR-006's families:
`input.file_unknown`, `input.file_duplicate`, `input.extension_not_accepted`,
`input.link`, `input.not_regular_file`, `input.encoding_invalid`,
`input.number_out_of_range`, `input.missing` (a required input not
supplied), `input.set_pointer_invalid`, `input.set_traversal`,
`input.set_into_unparsed`, `input.json_arg_not_object`,
`input.raw_unavailable`, `input.constraints_not_evaluated` (info),
`collection.subdirectory`, `collection.stem_duplicate`,
`collection.as_file`, `json.syntax`, `json.trailing_content`,
`json.duplicate_name`, `json.lone_surrogate`, `yaml.syntax` and one
`yaml.<rule>` per ADR-002 D2 rule (`yaml.alias`, `yaml.tag`,
`yaml.multi_document`, `yaml.duplicate_key`, `yaml.scalar_refused`, …),
`csv.header_missing`, `csv.header_duplicate`, `csv.header_empty`,
`csv.field_count`, `csv.bare_quote`, `csv.unterminated_quote`,
`csv.after_quote`, `markdown.frontmatter_unclosed`,
`markdown.frontmatter_not_mapping`, `markdown.frontmatter_body_field`,
`schema.keyword_unknown`, `schema.format_unsupported`, `schema.dialect`,
`schema.ref_unresolved`, `schema.ref_escapes`, `schema.ref_by_path`,
`schema.id_duplicate`, `package.compiled.mismatch` (D7 item 5), plus every
JSON Schema keyword failure under ADR-006 D6's `input.<keyword>` codes. Each
parse code's hint says the input's schema was not checked.

**ADR-007 (authoring).** Promised, for `check` to run: D1 items 2 to 6, D3
items 1, 2, 5 and 6, D4 item 2, D5, D6 item 3, D11 item 7, D13 item 4, D15
item 1, D19 items 4 and 5. Relied on: the sample-data generator writes its
data in D16's format so it can be saved as an example, every collection item
named with the extension of its form (`001.json`); what it writes
round-trips through D9 to D13 — in particular a CSV it writes quotes an
empty string as `""`, holds no null, array or object, and a text file it
writes ends in one LF.

**ADR-008 (command line).** Relied on: ADR-008 D7's supply flags — `--data
<dir>`, `--set <pointer>=<text>` (the text set of D17 item 3, repeatable),
`--set-json <pointer>=<json>` (the JSON set, repeatable) and `--inputs-json
<json>|-` (D17 item 1, repeatable, `-` at most once), each kind applied in
command-line order; the `schema` command printing one input's schema or the
combined schema of D7. Promised: D17 and D18's semantics.

**ADR-009 (security).** Relied on: `input-size`, `inputs-total`,
`collection-items`, `csv-rows`, `csv-columns`, `csv-cell`,
`frontmatter-size`, `value-depth`, `number-length`, `schema-size`,
`schema-depth`, `ref-depth`, `schema-work` and the `entries` limit, each
with its default; the regular-expression profile D3 item 6 uses (ADR-009
D10 item 6); the confinement of D5 under the `metais:///` base URIs (ADR-009
D9 item 6). Promised: no schema reference is fetched (D5); no link in a data
directory is followed (D16 item 6); nothing of the host — no environment
variable, path or clock — is a source of any input value or default.

**ADR-010 (implementation).** Relied on, for the libraries it verifies:
`santhosh-tekuri/jsonschema/v6` configured with a loader serving only
`metais:///` resources, METAIS's own format validators for D4's list,
unknown keywords detected before compilation, and every error returned with
instance and keyword locations (ADR-010 D5); METAIS's own JSON reader with
duplicate names, line and column and each number's literal text (ADR-010
D7); `internal/yamlx` enforcing ADR-002's profile with positions (ADR-010
D6); METAIS's own CSV reader meeting D11, including BOM removal and the
quoted/unquoted empty distinction (ADR-010 D8); Unicode NFC normalization
(`golang.org/x/text/unicode/norm`).

**ADR-011 (conformance).** Promised: the observable rules below, each to be
traced to a normative statement in `05-data-specification.md` (`DAT`) or
`06-supplying-inputs.md` (`SUP`).

## Conformance

Each of these is observable and needs vectors — valid and invalid inputs with
the expected value or findings:

- Manifest `inputs`: the name pattern and reserved names; unknown fields;
  defaults of `forms`, `required`, `collection`, `csv`; form/shape
  agreement.
- Dialect: an unknown keyword refused, an `x-` keyword accepted; a foreign
  `$schema`; every asserted format with valid and invalid strings, including
  the edge cases D4 item 1 names; each refused format.
- References: relative resolution between files; an `$id`-named file reached
  by `$id` and refused by path; climbing above the root; an `https` reference
  with no matching resource; a recursive schema; `$dynamicRef`.
- The combined schema: byte-exact `.compiled/schema.json` for a
  package with shared, YAML, `$id`-carrying and constraint schemas; a tampered
  combined schema refused.
- Cross-input constraints: §4.1's edition-dependent tables and §4.2's
  stage-dependent fields (SC2).
- Each form: BOM removal; every line-break convention giving one value; JSON
  duplicates, big integers, `1` versus `1.0`; YAML's null forms, `.inf`,
  `2026-10-18`; CSV headers, quoting, empty lines, field counts, each
  delimiter, quoted versus unquoted empty, every coercion row of D14 item 2
  with type sets in different orders; text's final LF; frontmatter
  delimiters, an unclosed block, a `body` field, the body's leading blank
  line.
- Collections: ordering with mixed case and non-ASCII names in NFC and NFD;
  dot entries ignored; subdirectory, duplicate stem and case-only stems
  refused; empty versus missing directory.
- Supply: unknown and duplicate data-directory entries; links refused; every
  step of D18 including the nested-override example of D18 item 3; path-set
  traversal errors and `-`; `--set` coercion; `--set-json` null.
- Defaults: optional members, required members (not applied), inside array
  items and CSV rows, inside a default object, under `$ref`, under `allOf`
  (not applied), parent not created, null not replaced.
- The input record: raw available and removed by a path set, per item;
  `input.raw_unavailable` refusal; the input digest of a value supplied by file and
  by `--inputs-json` being equal.
- One-pass reporting: parse findings across several files reported together
  with schema findings for the parsed inputs, and the constraints-not-evaluated
  `info` finding (SC8).

## Alternatives rejected

### Dotted `--set` paths

`--set vendor.address.city=Paris` reads well, but a member name containing a
dot needs an escape syntax of METAIS's own, and a numeric segment is
ambiguous between an index and a member name. JSON Pointer already has both
answers, and it is the form findings carry, so a supplier pastes a finding's
location straight into the fix.

### Typing `--set` values by their text

Reading `true`, `42` or `[1]` as JSON whatever the field is gives a ZIP code
`02134` that is a string and a version `1.10` that becomes the number 1.1 —
the surprise that made Helm add `--set-string`. The schema knows the type;
D14 asks it, and the JSON set covers the cases where text cannot.

### Merging `--inputs-json` into the file's value

A deep merge, or RFC 7386 JSON Merge Patch, must reserve some value for
"delete this member", and Merge Patch reserves null — which in METAIS is
data. Whole-input replacement plus single-value path sets says exactly what
happens with no value given a second meaning.

### An empty CSV field as the empty string

Simpler to describe, but every blank spreadsheet cell would reach the
template as `""`, which Liquid treats as true, and no default would ever
apply to a CSV column. The quoted/unquoted rule keeps both meanings
expressible.

### Leaving `format` as an annotation, or adopting the full Format-Assertion vocabulary

Annotation-only `format` validates nothing that the author wrote down, which
contradicts strict data. The full vocabulary requires the IDN and IRI formats
and ECMA-262 `regex`, where implementations disagree; asserting a named
subset and refusing the rest gives every conforming tool one answer.

### Allowing unknown keywords as annotations

It is what 2020-12 recommends, and it lets a misspelt constraint pass
silently. The `x-` prefix keeps a channel for the author's own annotations.

### The collection's `schema` as the item schema

It would need manifest fields for `minItems` and the like, and the schema
would no longer describe the value a template sees at `inputs.<name>`.

### Substituting every `$ref` in the combined schema

It cannot represent a recursive schema and changes the base URI of whatever
is substituted; bundling is 2020-12's own answer.

### Fetching or cataloguing external schemas

Fetching is ruled out. A catalogue of well-known schemas built into the tool
would make a package's meaning depend on the tool's version; a package that
needs a published schema carries a copy with its `$id`.

### Requiredness only in the manifest, or only inside each input's schema

Neither expresses §4.1's edition deciding which tables are required or §4.2's
stage, given that the stage is its own input. Folding every input into one
object input would express it, at the cost of the many-to-many slot model.
`input_constraints` adds one optional file and no new concept: it is a schema
over the object the combined schema already describes.

### Objects that keep their member order

Order would then depend on how the supplier wrote the file, so the same data
as JSON and as YAML could render differently, and CSV-derived rows would
carry an order other forms cannot express.

### Keeping a text file's final newline, and keeping line endings as written

Both make the parsed value depend on the editor or the platform that last
wrote the file; the raw form keeps the exact bytes for a template that wants
them.

### Supplying a collection as a single file

Equivalent to `--inputs-json` with an array, and it gives every collection two data
directory spellings. One spelling per input keeps the data directory
unambiguous; `--inputs-json` remains the whole-array route.

## Consequences

### Positive

- A supplier reads one manifest list and one combined schema and knows every
  input, its meaning, its forms and its rules, including the cross-input ones.
- The same data in any form, on any platform, gives the same value, the same
  digest and the same output.
- Every validation problem — malformed files, wrong types, missing
  conditionally required fields — is reported in one run with a location the
  supplier can act on, and a finding's inputs pointer (its input name and
  pointer, as ADR-006 D11's `at` line prints them) is a valid `--set`
  path.
- Validators cannot disagree on `format`: what is used is asserted, and what
  could disagree is refused.

### Negative

- Authors who use `idn-hostname`, `iri` or an unknown keyword, routine in
  published schemas, must edit those schemas before they compile.
- An integer beyond 2^53−1 must be a string, and CSV cannot express null.
- `1.0` supplied in a file renders as `1`; a supplier who needs the spelling
  supplies a string.
- Requiring the raw form refuses a render whose input came from the command
  line, which a caller only learns at render time unless `check` lists the
  inputs a package embeds (ADR-007 may).
- METAIS needs a CSV reader that distinguishes `""` from an empty field and
  a JSON parser that reports duplicates — neither is Go's standard library
  default.

### Neutral

- The combined schema is larger than the sum of its files by one wrapper per
  input.
- The `metais:` URI scheme appears in the combined schema and in findings
  about references; it is never dereferenced.

## What this does not decide

- Liquid's view of the data: truthiness, iteration order, number printing,
  the `embed` syntax and how item names are reached (ADR-004).
- Every resource limit and its default (ADR-009), and the regular-expression
  profile's contents (ADR-009).
- Finding codes, severities, wording and the report shape (ADR-006).
- Flag spelling, help text and the command-line envelope (ADR-008 D7).
- The sample-data generator's algorithm (ADR-007).
- Output validation schemas' location (ADR-005); they reuse D3 to D5 if
  ADR-005 adopts this dialect.
- Locale-aware input such as decimal commas or localized booleans: not in v1.

## Proposed rulings

1. **`inputs` is an ordered list of slots, `title` required, `schema`
   required, `forms` defaulting to `[json, yaml]`, `required` to `true`.**
   Lean: yes. The order is guidance and an unnamed slot helps no supplier.
   Alternative: an object keyed by name, title optional.
2. **Input names `^[a-z][a-z0-9_]{0,63}$` with Liquid's special words
   reserved.** Lean: yes; it removes case collisions and dialect edges.
   Alternative: allow upper case and hyphens.
3. **The slot's title and description live in the manifest, not the schema
   root.** Lean: yes; one schema can serve several slots. Alternative: take
   them from the schema's root.
4. **`input_constraints`, a cross-input schema over the inputs object.**
   Lean: yes; §4.1 and §4.2 need it. Alternative: put stage-dependent data in
   one input.
5. **Every used `format` asserted, from a closed list; IDN, IRI, `regex` and
   unknown formats refused.** Lean: yes. Alternative: annotation-only, or
   assert-known and ignore the rest.
6. **Unknown schema keywords refused except `x-`.** Lean: yes. Alternative:
   annotations, as 2020-12 recommends, with a `check` warning.
7. **"Internal `$ref`s inlined" read as bundling into one self-contained
   document, not substitution.** Lean: yes; substitution cannot express
   recursion. Alternative: substitute and refuse recursive schemas.
8. **`metais:///<path>` as retrieval URI; a file with an `$id` reachable only
   by it.** Lean: yes. Alternative: `https://*.invalid/` base URIs.
9. **One number type; integers beyond 2^53−1 written as integers refused.**
   Lean: yes (I-JSON, JCS). Alternative: separate integer and float kinds by
   lexical form.
10. **Line breaks normalized to LF in parsed CSV, text and Markdown; BOM
    removed everywhere; a text file's final LF removed.** Lean: yes, for
    cross-platform identity. Alternative: values exactly as written.
11. **CSV: unquoted empty → absent, `""` → empty string; empty lines ignored;
    delimiter from `,` `;` `|` tab; no null, array or object.** Lean: yes.
    Alternative: empty → empty string, any single-character delimiter.
12. **Coercion by declared `type` through a static walk; failing text stays
    a string for validation to report.** Lean: yes. Alternative: coercion
    failures as parse findings.
13. **Markdown form is always an object; a prose `.md` file for a `text`-only
    input is read as text.** Lean: yes. Alternative: a Markdown form that is
    the body string when the schema is a string.
14. **Collections: flat directory, dot entries ignored, NFC names sorted by
    byte order, stems unique and case-distinct; never a single file.** Lean:
    yes. Alternative: recursive directories; a whole-array file accepted.
15. **A data directory entry naming no input is an error; links and special
    files refused.** Lean: yes. Alternative: a warning; follow links.
16. **`--set` takes an inputs JSON Pointer and a schema-coerced text; a
    second kind, `--set-json`, takes JSON; whole inputs come from the
    repeatable `--inputs-json`** (flag names ADR-008 D7's). Lean: yes.
    Alternative: dotted paths; JSON-literal inference.
17. **Precedence: data directory, then every `--inputs-json` (whole-input
    replacement), then every path set, each in command-line order.** Lean:
    yes. Alternative: deep merge; pure command-line order.
18. **Defaults through `properties`/`items`/`prefixItems`/`$ref` only, never
    to required members, never creating parents.** Lean: yes. Alternative:
    also through `allOf`.
19. **Raw bytes only for an input or item supplied by an untouched file; an
    absent raw form refuses the render.** Lean: yes; synthesizing bytes would
    not be the original. Alternative: canonical JSON as a stand-in.
20. **The input digest is SHA-256 of the canonical JSON of the value after
    defaults.** Lean: yes. Alternative: digest of the raw bytes, or of the
    value before defaults.
21. **When any input fails to parse, cross-input constraints are not
    evaluated, with an `info` finding.** Lean: yes. Alternative: evaluate
    and risk false findings.
22. **"Every error" means every failure that can be a finding** (D3 item 7):
    `not`'s subschema, `if`, and the untried branches of a satisfied
    `anyOf`/`oneOf` are evaluated for their yes or no only. Lean: yes. Why:
    none of their inner failures is ever reported (ADR-006 D6), and it is
    what the chosen validator does (ADR-010 D5 item 6). Alternative: full
    evaluation everywhere, which no finding would use.

## References

- JSON Schema 2020-12 Core (draft-bhutton-json-schema-01): base URI §9.1.1,
  `$id` §8.2.1, `$schema` §8.1.1, network §8.2.3 and §9.1.2, bundling §9.3.1,
  unknown keywords §4.3.1 and §6.5, regular expressions §6.4 —
  https://json-schema.org/draft/2020-12/json-schema-core
- JSON Schema 2020-12 Validation: `type`/`integer` §6.1.1, format
  vocabularies §7.2, defined formats §7.3, content §8, `default` §9.2 —
  https://json-schema.org/draft/2020-12/json-schema-validation
- The 2020-12 meta-schema and its `$vocabulary` —
  https://json-schema.org/draft/2020-12/schema
- RFC 4180, CSV — https://www.rfc-editor.org/rfc/rfc4180
- PostgreSQL `COPY`, CSV format (null versus empty string) —
  https://www.postgresql.org/docs/current/sql-copy.html
- Go `encoding/csv` — https://pkg.go.dev/encoding/csv
- YAML 1.2.2: line breaks §5.4, byte order mark §5.2, unique keys §3.2.1.1,
  core schema §10.3 — https://yaml.org/spec/1.2.2/
- RFC 8259, JSON — https://www.rfc-editor.org/rfc/rfc8259
- RFC 7493, I-JSON — https://www.rfc-editor.org/rfc/rfc7493
- RFC 8785, JSON Canonicalization Scheme — https://www.rfc-editor.org/rfc/rfc8785
- RFC 6901, JSON Pointer — https://www.rfc-editor.org/rfc/rfc6901
- RFC 3986, URI generic syntax (resolution §5.2, dot segments §5.2.4) —
  https://www.rfc-editor.org/rfc/rfc3986
- RFC 3339, date and time — https://www.rfc-editor.org/rfc/rfc3339
- RFC 3629, UTF-8 — https://www.rfc-editor.org/rfc/rfc3629
- RFC 5321 (`email`) — https://www.rfc-editor.org/rfc/rfc5321
- RFC 1123 (`hostname`) — https://www.rfc-editor.org/rfc/rfc1123
- RFC 4291 (`ipv6`) — https://www.rfc-editor.org/rfc/rfc4291
- RFC 4122 (`uuid`) — https://www.rfc-editor.org/rfc/rfc4122
- RFC 6570 (`uri-template`) — https://www.rfc-editor.org/rfc/rfc6570
- RFC 7386, JSON Merge Patch — https://www.rfc-editor.org/rfc/rfc7386
- Unicode Standard Annex #15, normalization forms —
  https://unicode.org/reports/tr15/
- CommonMark specification (thematic breaks) — https://spec.commonmark.org/
- Jekyll front matter — https://jekyllrb.com/docs/front-matter/
- Helm `install` flags (`--set`, `--set-string`, `--set-json`) —
  https://helm.sh/docs/helm/helm_install

## Implementation log

This log is this section together with the entry files in
`docs/adr/logs/ADR-003/`. Each work item writes its entry as a file of its
own there, named as AGENTS.md's "Writing documentation" says. The entries
below are the ones written in this file before that rule, and stay where
they were written.

- **2026-10-19, PLAN-2026-10-metais-v1 W2-12.** D7 items 1 to 4 landed as
  `schema.Bundle` in `internal/schema`, building the combined schema as an
  `internal/liquid/value` tree from the inputs' declarations and their
  parsed schema files, with D5 items 1 and 2's `RetrievalURI` and
  `CanonicalURI`. Bundling rewrites only each resource's `$id` and
  `$schema`. A boolean schema file has no members to hold an `$id`, so it
  is embedded as the object schema that means the same: `true` as `{}`,
  `false` as `{"not": {}}`. D14 item 1 landed as `schema.Walker`, which
  indexes a document's resources and anchors and walks an instance
  location through `$ref`, `properties`, `prefixItems` and `items` only.
  It does not follow a `$ref` to a 2020-12 meta-schema, and reports one
  unresolved. D14 item 2 landed as `schema.Coerce`, with `CoerceField` for
  D11 item 5's empty fields. A coerced number keeps its text as its
  literal. The refusal of a large integer (D8 item 5) stays with the reader
  that coerces.
- **2026-10-19, PLAN-2026-10-metais-v1 W3-2.** D8 to D22 landed as
  `internal/inputs`, a library Wave 4's `validate` and `render` wire. A
  `Layer` is built from what D22 step 1 produced: the manifest, the
  `schema.Compiler` and the combined schema with its compiled form. A
  `Supply` takes one command's flags through one entry point each,
  `ReadDataDir`, `AddInputsJSON`, `AddSet` and `AddSetJSON`, each parsing
  what it is given at once (D22 step 2), and `Validate` assembles (D18),
  applies the defaults (D19) and validates the inputs object against the
  combined schema (D22 steps 3 and 4), returning every finding and one
  `Record` per input (D21). The forms are Wave 2's readers; coercion and
  defaults walk the combined schema statically, through `schema.Walker`
  and its new `Chain`, the `$ref` chain D19 reads defaults along; an item
  name is normalized by the new `unitext.NFC`. The data directory is read
  through `hostfs`'s read side alone, in `Supply.ReadDataDir`, which a
  layering rule holds every reference to `hostfs` to. The combined
  schema's failures are split by D6 item 2's rule, each input's subtree
  mapped by `schemafail.Map` with that input as its target, its
  `keyword_location` taken from the input's own root schema (the combined
  schema's `/properties/<name>` step removed, so ADR-006 D7 item 4's
  suppression reads it), the constraints' kept from the combined root.
  Where the text is silent, these readings were taken:
  - a failure at the root of the inputs object other than `required`,
    which only the input constraints can give (a `not`, an `anyOf` over
    inputs), names no input, and ADR-006 D4 has no subject for the inputs
    object: it is `input.constraints_failed`, about the invocation, with
    its schema location. A root `required` or `dependentRequired` is
    `input.missing` for each member, with the conditions above it as
    `via`;
  - a finding of an `--inputs-json` argument inside a member is about that
    member's input, at its pointer within the input; one outside every
    member is `input.json_arg_invalid`, about the invocation. An argument
    that does not parse, or is not an object, leaves the inputs it would
    have supplied unknown, so no input is reported missing and the
    constraints are not evaluated;
  - an input any file or argument of which does not parse is unparsed and
    left out (D22 item 3), and no path set is applied into it, which is
    D18 item 4 read for arguments as for files; an `--inputs-json` member
    that replaces an input whose file did not parse is validated, and the
    set is still refused;
  - D11 item 9 says a CSV file with a structural finding contributes no
    value, and ADR-006 D7 item 4 says a CSV input with a record rejected
    alone is validated in part. The layer applies D7 item 4 to an input's
    own CSV file (a wrong field count, a quote finding, and a number the
    data model or `number-length` refuses, each rejects its record alone)
    and D11 item 9 to a collection's item that is a table, which is left
    out whole; a header finding, an unterminated quote and a limit refuse
    the file whole. A path set into an array validated in part is refused,
    since its indexes would not name the items they say;
  - `number-length` holds a number coerced from a CSV field or a text set
    as it holds a literal in JSON or YAML;
  - `collection-items` bounds every entry of a collection's directory, dot
    entries included, and the data directory's own listing, whose breach
    is the caller's argument refused (`DataDirError`, `too_large`), as is a
    data directory that cannot be opened;
  - when the input constraints are not evaluated, the combined schema,
    which is compiled once, is evaluated whole and the constraints'
    failures are discarded; once `regex-work` is spent, each input present
    is `limit.exceeded`;
  - the codes beyond the Interfaces list: `input.not_readable` (ADR-006 D9
    item 8's reasons), `input.noncharacter` (D8 item 4 in a CSV, text or
    Markdown file), `collection.item_name` (a name that is not UTF-8 or
    holds `\` or `:`), `collection.directory_name` (a collection's
    directory named with an extension, which D15 item 2's `<name>/` is
    not), `input.json_arg_unknown`, `input.json_arg_invalid` and
    `input.constraints_failed`. `input.raw_unavailable` is a render's
    refusal and is Wave 4's to register.
- **2026-10-19, PLAN-2026-10-metais-v1 W4-1.** `metais validate` wires the
  inputs layer, and the specification states what it makes observable:
  §5.2 (D4's formats on supplied strings, D6's constraints, D22's
  validation) and chapter 6 (D8 to D20 and D22). Every row of the
  *Conformance* list has a vector under `spec/conformance/vectors/sup/` or
  `dat/` but these:
  - an item name in NFD, which the suite cannot carry: ADR-011 D9 item 8
    holds every file name of the suite to the portable-path predicate, so
    the NFC half is a vector and the NFD half is `internal/inputs`'
    `TestCollectionItems`;
  - D21's input record, whose raw form and digest are what a render is
    given: `input.raw_unavailable` and the digests are the render's to
    observe;
  - the data directory given as a link, which a vector cannot name (its
    data path must be a tree of the vector): `internal/inputs`'
    `TestDataDirLinksNotFollowed` and `internal/cli`'s
    `TestValidateLinkedDataDirectory` prove D16 item 6's last sentence.
  - `collection.item_name` and `input.not_readable` are listed W7 in
    `unwired.txt`, with the codes no vector can produce: the first needs an
    item name the suite's portable-path rule refuses, the second a file the
    runner cannot make unreadable or take away between listing and reading.
