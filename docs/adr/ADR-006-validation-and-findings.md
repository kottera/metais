# ADR-006: Validation and findings — one finding model for every command, with registered dotted codes whose meaning never changes within a major, locations as JSON Pointer plus line and column, every finding in one deterministic pass, and a fixed exit-code table

**Status:** Accepted 2026-10-18 by Ed Funnekotter; its proposed rulings are ratified in [RULINGS-v1](RULINGS-v1.md).
[Amendment A1](#amendments) (the `manifest` family registers only the
keyword codes the manifest schema can produce) is accepted by Ed
Funnekotter on 2026-10-19. [Amendment A2](#amendments) (the registry file
is generated from the tool's table) records his ruling of 2026-10-19 and is
accepted on merge. [Amendment A3](#amendments) (A1's tripwire reads each
keyword where it stands, and the train's reporting readings) is accepted
by Ed Funnekotter on 2026-10-22, at PLAN-2026-10-metais-v1's ratification;
the passages it changes are marked *(A3)*.
[ADR-003 Amendment A2](ADR-003-the-data-specification.md#amendments)
(rules across the items of an array: `uniqueKeys` and `coverage`) is
proposed on 2026-10-23 and amends D6 (two keyword codes in each of the `input.` and `output.` families), D7 item 4 and D8 item 3 (the duplicate key compares the schema location).
**Date:** 2026-10-18
**Deciders:** Ed Funnekotter
**Designs:** PRD-001 R6 (items 1 to 4); the finding half of R14 item 3 (a
refusal names what is wrong), R17 item 3 (every refusal names the next step)
and R2 item 3 (findings in JSON); the finding side of SC7 (no host value in a
finding) and SC8 (every planted error in one run).
**Depends on:** ADR-002 (canonical JSON numbers; the manifest's titles and
descriptions; the manifest checked as a document against its own schema);
ADR-003 (parsers that keep positions; how a value's origin is tracked through
the precedence merge; collection order; the input schemas); ADR-008 (the JSON
envelope that carries the findings; which stream the human report goes to; the
spelling of the flags named here); ADR-009 (the finding-count limit and every
other limit whose breach is a finding); ADR-010 (the JSON Schema library, and
the adapter from its error tree); ADR-011 (the vector format that compares
findings; where the registry file lives in `spec/`).

## Context

### What the PRD requires

R6 asks for validation that reports **every** finding in one pass, never only
the first; each finding naming the input, the location in it (file, line where
there is one, JSON path), the rule broken, and the title and description of
the field, so the supplier corrects it without opening the package; findings
for a person or as JSON; a non-zero exit when any finding is an error; and a
render that validates first and writes nothing if the inputs are invalid.

The same word, *finding*, is used by every other part of the PRD: a package
outside the specification is refused "with a finding that names what is
wrong" (R14 item 3); `check` reports findings about a package with no data
(R13 item 2); an output that fails its schema refuses the render "with
findings" (R9 item 4); a field the sample generator cannot satisfy "is a
finding" (R13 item 6); conformance vectors give "the outputs or the findings
a conforming tool must produce" (R16 item 2). So the model this ADR fixes is
not validation's alone: it is the one shape in which **every** command reports
**every** problem, and the conformance vectors compare it.

Two success criteria constrain it directly. SC7 plants sentinel values in the
environment, the working directory and the home directory and requires that
none reaches "any output, finding or message". SC8 plants errors throughout an
input set and requires every one in a single run.

### Prior art, checked against the primary sources

- **JSON Schema 2020-12, Core §12 "Output Formatting"** defines four output
  formats: *flag* (a boolean), *basic* ("a flat list of output units"),
  *detailed* ("a condensed hierarchical structure based on the structure of
  the schema") and *verbose*. An output unit carries `valid`,
  `keywordLocation` (a JSON Pointer to the keyword that "MUST include any
  by-reference applicators such as `$ref` or `$dynamicRef`", and so "may not
  be resolvable by the normal JSON Pointer process"), `absoluteKeywordLocation`
  (the dereferenced location as a full URI), `instanceLocation` (a JSON
  Pointer into the instance), and `error`. The specification **does not
  define the wording of an error**, and an output unit carries no file, no
  line, no stable code, no title and no description. The detailed format's
  condensation rules: every applicator (`*Of`, `$ref`, `if`/`then`/`else`)
  gets a node, childless nodes are removed, single-child nodes are replaced
  by their child.
- **RFC 6901 (JSON Pointer)**: zero or more reference tokens each prefixed by
  `/`; `~` is written `~0` and `/` is written `~1`; decoding replaces `~1`
  before `~0`; an array index is a non-negative decimal without leading zeros,
  and `-` names the position after the last element, which never exists. A
  pointer may be carried in a URI fragment, percent-encoded (§6).
- **SARIF 2.1.0** (OASIS, errata 01) is the established interchange format for
  analysis results. Its `ruleId` is "a hierarchical string … [that specifies]
  the stable identifier of the rule", which "SHOULD be opaque" (§3.27.5); its
  `level` is one of `error`, `warning`, `note`, `none` (§3.27.10); a region's
  `startLine` and `startColumn` are positive integers, a column defaulting to
  1 (§3.30.5, §3.30.6); a run states its `columnKind`, either
  `utf16CodeUnits` or `unicodeCodePoints` (§3.14.27); a plain-text message's
  first sentence "SHOULD provide a useful summary" (§3.11.3).
- **`github.com/santhosh-tekuri/jsonschema/v6`** (v6.0.3, released
  2026-08-06), a Go validator that passes the JSON-Schema-Test-Suite for
  2020-12 by its README, and the library whose error model is studied here
  (ADR-010 chooses the library). Its `ValidationError` is a **tree**:
  `SchemaURL` (the absolute, dereferenced location of the schema that holds
  the keyword), `InstanceLocation []string` (pointer tokens), `ErrorKind`, and
  `Causes []*ValidationError`. Each `ErrorKind` is a struct with structured
  fields — `kind.Required{Missing []string}` reported **once, at the parent
  object**, for every missing property; `kind.AdditionalProperties{Properties}`;
  `kind.Type{Got, Want}`; `kind.Enum{Got, Want}`; `kind.UniqueItems{Duplicates
  [2]int}`; `kind.AdditionalItems{Count}`; `kind.OneOf{Subschemas}` (the two
  indexes that both matched, or nil when none did); `kind.Reference{Keyword,
  URL}` wrapping a `$ref` failure. The validator **collects every error**,
  except inside contexts that need only a yes or no (`not`, `if`, and the
  branches of `oneOf` after one has matched). Its **messages echo instance
  values** (`"%s does not match pattern %s"`, `"%s is not valid %s"`). It
  reports `unevaluatedProperties` by ranging over a Go map, so the order of
  those errors differs between runs. It offers the flag, basic and detailed
  outputs; its `jv` command exits 1 on validation errors and 2 on usage
  errors.
- **`python-jsonschema`'s `best_match`** ranks errors by instance depth and
  treats `anyOf`/`oneOf` as weak, and its documentation says "This function is
  a heuristic. Its return value may change for a given set of inputs from
  version to version."
- **Positions from Go parsers**: `encoding/csv`'s `Reader.FieldPos` gives the
  line and column of a field's start, 1-based, "columns are counted in bytes,
  not runes", and `csv.ParseError` carries `StartLine`, `Line`, `Column`;
  `encoding/json`'s `Decoder.InputOffset` gives the byte offset after the most
  recent token; the YAML organisation's maintained `go-yaml` v3 (the original
  `go-yaml/yaml` repository is archived) gives every `Node` a `Line` and
  `Column`, and its scanner advances the column once per character, not per
  byte.
- **The Go runtime** exits with status 2 on an unrecovered panic or an
  unexpected runtime failure (`runtime` package documentation, GOTRACEBACK).
- **GNU Coding Standards §4.4** formats a compiler message as
  `sourcefile:lineno:column: message`, lines and columns from 1, and counts
  columns by display width with tab stops every 8.
- **JSON Schema 2020-12, Validation §9.4**: `writeOnly: true` marks a value
  that is sent but never retrieved; the example given is "a password input
  field".
- **Bidirectional controls** (UAX #9: U+202A–U+202E, U+2066–U+2069) reorder
  how text is displayed without changing its logical content; CVE-2021-42574
  ("Trojan Source") is the attack built on them.

What follows from these facts:

1. JSON Schema's own output formats are a starting point, not an answer: they
   lack the file, the line, a stable code, the field's title and description,
   and any wording — exactly what R6 item 2 requires.
2. A validator's tree must be **mapped**, not passed through: its messages
   echo values and are not stable; its `required` failure sits at the parent
   with a list; its order is not deterministic.
3. Positions exist in every parser the PRD's forms need, but in different
   units (bytes for CSV, characters for YAML, offsets for JSON). The model has
   to name one unit and say how the others convert.
4. Status 2 already means "the Go program crashed" for any Go binary, whatever
   the program itself chooses.

## Decision

### D1 — One finding model, for every command

**Every problem any command detects is reported as a finding of one shape,
and nothing is reported any other way.**

1. A finding has these members. "Always" means present in every finding;
   "when known" means present exactly when the rule in the cited decision
   gives it a value, and absent otherwise (never `null`, never `""`).

   | Member | Presence | Content |
   |---|---|---|
   | `code` | always | The registered code (D3), e.g. `input.min_length`. |
   | `severity` | always | `error`, `warning` or `info` (D2); fixed by the code. |
   | `message` | always | One sentence saying what is wrong (D9). |
   | `hint` | always | The next step, naming a value, an edit or a command (D9). |
   | `subject` | always | What the finding is about: `kind`, and `name` and `title` when known (D4). |
   | `location` | when known | Where: `origin`, `file`, `argument`, `line`, `column`, `pointer` (D5). |
   | `schema` | when known | The schema keyword broken: `entry`, `pointer`, `keyword`, `keyword_location` (D6). |
   | `field` | when known | The `title` and `description` of the field, from the schema or the manifest's own schema (D6 item 9). |
   | `params` | always | The code's registered parameters, an object (possibly empty) (D3 item 6). |
   | `via` | when known | The conditional or alternative that made the rule apply (D6 items 6 and 7). |
   | `alternatives` | when known | For `any_of`/`one_of`, each branch and its findings (D6 item 6). |
   | `related` | when known | Other locations the finding concerns: the earlier duplicate, the colliding path (D5 item 9). |

2. **`location` is absent only when the subject has no place**: a usage
   finding, a whole-package finding such as an unreadable zip, the report's
   own `report.truncated`. Every finding about a value in an input, an output
   or the manifest has a `location` with a `pointer`.
3. **The model is the same for every command**: `validate`, `render`,
   `check`, `test`, `pack`, `unpack`, the sample preview, every reading
   command (`inspect`, `ls`, `read`, `outline`, `grep`, `schema`, `extract`),
   a usage error, a refused package, a security refusal and an internal
   error. A command that succeeds reports an empty list.
4. **A finding is constructed only from a registered code and its
   parameters.** The message and hint are produced from the code's templates
   (D3 item 7); no code path accepts a free-text message. This is what makes
   D9's echo rule structural rather than a matter of care.

### D2 — Severities

**There are exactly three severities, each with a fixed meaning for exit codes
and for `pack`.**

1. **`error`** — the command cannot give a correct result. Inputs with an error
   are not rendered; a package with an error is not packed; any error makes
   the exit status non-zero (D12).
2. **`warning`** — something is probably wrong but the result is still
   correct: an input no template uses (R13 item 2), a declared minor version
   higher than the package needs (ADR-002). A warning never changes the exit
   status unless the caller asks (item 5), never stops a render and never
   stops `pack`.
3. **`info`** — a fact the reader needs to interpret the report, not a
   problem: that an input was validated only in part (D7 item 4), that the
   report was truncated (D8 item 6). Info never changes the exit status,
   whatever the caller asks.
4. **A code's severity is part of its meaning** (D3 item 4). Neither a package
   nor a caller changes the severity of a finding; the JSON and human reports
   always show the registered severity.
5. **A caller may make warnings fail the command**: with ADR-008 D4's
   `--fail-on warning` (default `--fail-on error`),
   a warning sets exit 1 exactly as an error would. The finding's `severity`
   in the report is unchanged, and a render or a `pack` that would otherwise
   proceed still does not write anything when the flag turns a warning into a
   failure — the command fails before writing.
6. **`pack` refuses on any error** from its `check` and `test` steps and writes
   no package; warnings are reported and the package is written.

### D3 — Codes

**A code is a dotted lowercase name, registered in the specification, whose
meaning, severity and failure class never change within a major version.**

1. **Syntax**: `^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*){1,3}$`, at most 64 bytes:
   between two and four segments, each starting with a letter and holding
   only `a`–`z`, `0`–`9` and `_`. Examples: `csv.header_missing`,
   `package.digest.mismatch`, `input.required`.
2. **The first segment is the family**, from a closed list the specification
   holds; a family is added only by a new minor version of the specification.
   Each family has an owning ADR, which registers its members:

   | Family | Owner | Covers |
   |---|---|---|
   | `usage` | ADR-006 (members ADR-008 adds) | the command line: unknown command or flag, a missing or extra argument, a bad flag value, conflicting flags, an operand naming no entry, input or output (`usage.entry_not_found`, `usage.unknown_output`, …), a stream the result cannot be written to (`usage.archive_to_terminal`) |
   | `internal` | ADR-006 | a defect in the tool itself: `internal.error` |
   | `report` | ADR-006, ADR-007 | the report itself: `report.truncated`, `report.step_skipped` (a `check` step that could not run) |
   | `package` | ADR-001, ADR-002, ADR-007, ADR-009 | the package as a file and a tree: `package.not_found`, `package.not_readable`; the container (`package.container.*`); entry names (`package.name.*`); the layout and a source directory's tree (`package.layout.*`); `.compiled/` (`package.compiled.*`); digests (`package.digest.*`); the specification version (`package.version.*`) |
   | `limit` | ADR-009 | every resource limit: `limit.package` (class `refused`, the package's own content breaches it), `limit.exceeded` (class `content`, inputs, work or outputs do), and `limit.invalid` for a malformed or raising `--limit` (class `usage`) |
   | `manifest` | ADR-002, ADR-003, ADR-005, ADR-007 | the manifest, its file and profiles (`manifest.file.*`, `manifest.yaml.*`, `manifest.json.*`), its fields and semantic rules, and its validation against the manifest schema |
   | `schema` | ADR-003, ADR-007, ADR-009 | an input's or an output's schema as a document: does not compile, an unknown keyword or format, an unresolved or escaping `$ref`, a pattern outside the profile (`schema.pattern_unsupported`), a static defect `check` finds, a field the sample generator cannot satisfy (`schema.sample_unsatisfiable`) |
   | `input` | ADR-006 (keyword codes), ADR-003 (supply), ADR-009 (links and special files) | a supplied value and the input schema it fails; supplying inputs |
   | `json`, `yaml`, `csv`, `text`, `markdown`, `collection` | ADR-003 | parsing an input in each form, and a collection directory |
   | `template` | ADR-004, ADR-007 | template parsing, rendering errors, the cross-reference (`template.xref.*`), the OOXML split run (`template.ooxml.*`), a manifest expression's syntax |
   | `output` | ADR-005, ADR-006 (keyword codes), ADR-009 (links, reserved names) | output declarations, paths, transforms, the destination (`output.destination.*`), output schema validation |
   | `asset`, `reference` | ADR-001, ADR-005 | assets and references as entries and as outputs |
   | `example` | ADR-007 | examples, `test`'s comparison and `test --update` |

   Every code any ADR names obeys item 1 and belongs to a family of this
   table.

3. **The registry is part of the specification**, as a machine-readable file,
   `spec/findings/registry.json`, validated against
   `spec/schemas/1.0/finding-registry.schema.json` (placement ADR-011 D1 and
   D7 item 2), from which the code catalog in `12-findings.md` is generated
   (ADR-011 D3 item 9), with a chapter of prose that states the model. Each registry entry has:
   `code`; `severity`; `class` (item 5); `subject` (the subject kind it is
   always reported with); `params` (an ordered list of `name`, JSON type,
   meaning, and whether always present); `message` and `hint` templates
   (item 7); `location` (which span its line and column mark, D5 item 6);
   `since` (the specification minor that added it, `"1.0"` for v1); and
   `statements` (the identifiers, `MTS-<AREA>-<NNN>` per ADR-011 D3, of the
   normative statements it reports, which is how ADR-011 traces a vector to
   its rule).
4. **Within a major version, a registered code never changes its meaning, its
   severity, its class or its subject kind; it is never renamed, never
   removed and never reused.** Its `params` may gain members, never lose one
   or change one's type. A code the tool no longer emits stays in the
   registry, marked `"retired": "<minor>"`. A new minor adds codes. A consumer
   that meets a code it does not know acts on its `severity`.
5. **A code's `class` says what kind of failure it reports**, for D12:
   - `usage` — the command line is wrong;
   - `refused` — the package cannot be opened or trusted at all: not a zip or
     source directory, an unsafe entry, a digest mismatch, an unsupported
     specification major, a package resource limit;
   - `content` — the package's or the inputs' content breaks a rule;
   - `destination` — the place the outputs go cannot take them: a non-empty
     directory without `--overwrite`, a write that fails;
   - `internal` — a defect in the tool.
6. **`params` are normative.** Each is a JSON value: a number in ADR-002's
   canonical JSON number form, a string, a boolean, or an array of those.
   Parameter names are lowercase `snake_case`. Example, `input.min_length`:
   `{"limit": 10, "length": 4}`.
7. **Message and hint templates** are plain text with `{name}` placeholders
   naming a parameter or one of the built-ins `{field}` (D9 item 3),
   `{input}`, `{output}` and `{package}`; `{{` and `}}` write literal braces.
   They are **not Liquid**: a finding's wording must never depend on the
   template engine it may be reporting on. The templates are the reference
   wording; the specification states that a conforming implementation may
   word messages and hints differently, and that the code, severity, subject,
   location, schema, field and params are what conformance compares (D10
   item 5, *Conformance*).

### D4 — Subjects and origins

**Every finding names the kind of thing it is about, and every located finding
names where the value came from.**

1. `subject.kind` is one of: `invocation` (the command line, and the inputs
   object as a whole when a finding names no one input *(A3)*), `package` (the
   package as a whole, or one of its entries as a file), `manifest`, `schema`
   (an input's or an output's schema as a document), `input` (a supplied
   value), `template`, `asset`, `reference`, `output`, `example`, `report`.
   A kind is added only by a new specification minor.
2. `subject.name` is the input name for `input` and `schema`, the output's id
   for `output` (ADR-005 names how an output is identified), the template's
   entry path for `template`, the example's name for `example`, and the entry
   path for `asset` and `reference`. It is absent for `invocation`,
   `package`, `manifest` and `report`.
3. `subject.title` is the input's declared title (ADR-003 D1), present for
   kinds `input` and `schema`; outputs declare no title (ADR-005 D1), so an
   `output` subject has none.
4. `location.origin` says which source the located value or file came from:

   | Origin | `file` is | Notes |
   |---|---|---|
   | `data_dir` | the path relative to the data directory, `/`-separated (`orders.csv`, `articles/intro.md`) | the data directory's own path is never included |
   | `inputs_json` | absent (its text is the argument itself, or standard input) | `argument` is the 1-based ordinal of that `--inputs-json` among all `--inputs-json` flags |
   | `set` | absent | `argument` is the 1-based ordinal of that path set among all `--set` and `--set-json` flags, in command-line order |
   | `default` | absent | the value came from a schema `default` (R5 item 3); no line or column |
   | `sample` | absent | the value came from the sample generator (ADR-007) |
   | `example` | the package entry, e.g. `examples/basic/inputs/vendor.yaml` | inputs under `test` |
   | `package` | the entry path, e.g. `templates/index.html.liquid`, `metais.yaml` | the package's own content |
   | `output` | the output path relative to the output root | rendered outputs, before they are written |

5. **The origin of a value is the source that won the precedence merge**
   (ADR-003), recorded per value. A field set by `--set` over a data file is
   reported with origin `set`, not with the file's line.

### D5 — Locations

**A location is a JSON Pointer into the data model, plus the file, line and
column of the bytes the value came from, measured in one unit everywhere.**

1. **`pointer`** is an RFC 6901 pointer in its JSON-string form (not the URI
   fragment form), into the value the schema validates: the parsed input after
   CSV coercion, collection assembly and default application (ADR-003). The
   root is `""`. For a CSV input the pointer is `/<row>/<column>`, the row
   being the 0-based index of the data row (the header is not a row), the
   column being the header name, escaped per RFC 6901. For a collection it is
   `/<item>/...`, the item index in the collection's order (ADR-003). A
   Markdown item's body is `/<item>/body`.
2. **A pointer may name a value that is absent.** For `input.required` and
   `input.dependent_required` the pointer names the missing member
   (`/vendor/name`), which a supplier reads as "this is where the value
   goes"; the finding's `params.property` repeats the member name. No other
   code points at an absent value.
3. **`line` and `column` are 1-based. A column counts Unicode code points**
   from the start of the line, a tab counting as one. Lines end at LF, CRLF or
   a lone CR, each counting as one line break. The byte order mark ADR-003
   D8 item 1 removes from the start of an input file is not counted. These are SARIF's
   `unicodeCodePoints` columns; they differ from GNU's display-width columns
   deliberately, because display width depends on the terminal and the locale
   and a finding must not (R10 item 3).
4. **Conversion.** A parser that reports byte offsets or byte columns (JSON
   offsets, CSV `FieldPos`) is converted through a per-file line index built
   from the raw bytes; a parser that already counts characters (go-yaml's
   `Column`) is used as it is, after the conformance vectors prove the two
   agree on non-ASCII lines.
5. **Where there is no line, there is no line.** Values from `--set`, from a
   `default` and from the sample generator have no `line` or `column`. A
   plain-text input, or a whole Markdown input used as text, is a single
   value: its findings carry `file` and no `line`. A value from
   `--inputs-json` carries the line and column within that argument's
   text.
6. **Which span a code's line and column mark is registered with the code**
   (D3 item 3), from these:
   - `value` (the default): the first character of the value — the opening
     quote of a JSON string, the first character of a YAML scalar or of a
     flow or block collection's first entry, the first character of a CSV
     field (its opening quote if quoted);
   - `key`: the first character of the member's name — used by
     `additional_property`, `unevaluated_property` and `property_name`;
   - `parent`: the value of the object that lacks a member — used by
     `required` and `dependent_required`;
   - for Markdown frontmatter, lines count from the first line of the file,
     not of the frontmatter; a body's position is its first line.
7. **No end positions in v1.** A finding gives where a value starts, not where
   it ends. go-yaml's `Node` carries no end position, and a supplier needs the
   start to find the value; a later minor may add `end_line` and
   `end_column` as optional members.
8. **No path in a location is made absolute or resolved.** The tool never
   prints the working directory, a home directory, a resolved symbolic link
   or the data directory's own path; `file` is always relative to the root of
   its origin (item 4 of D4). This is one of the rules SC7 proves.
9. **`related`** is an array of objects, each a location (same members) plus
   `note`, a short fixed phrase from the registry (`"first occurrence"`,
   `"collides with"`). `input.unique_items` gives the earlier duplicate;
   ADR-005's case-collision finding gives the other path; ADR-002's duplicate
   member finding gives the first definition.

### D6 — Mapping JSON Schema failures to findings

**A schema failure becomes one finding per failed assertion at its instance
location, named by its keyword; applicators are transparent except where a
choice between branches has to be explained.**

1. **The mapping is the same for every document validated against a JSON
   Schema**: a supplied input (family `input`), an output with an output
   schema (family `output`), and the manifest against the manifest schema
   (family `manifest`, ADR-002). Only the family prefix differs. The rules
   below write `input.`; read `output.` and `manifest.` alike.
2. **The adapter builds a library-neutral failure tree** (D13 item 3) from the
   validator's error tree: per failure, the keyword, the evaluation path
   (`keyword_location`), the schema entry and pointer, the instance pointer,
   the keyword's structured data, and children. The rules operate on that tree,
   so ADR-010 may change the library without changing a finding.
3. **One finding per failed assertion keyword, at its instance location.**
   Where the library reports one failure for several members, it is split:
   `required` and `dependentRequired` give one finding per missing member;
   `additionalProperties: false` gives one per extra member; `items: false`
   after `prefixItems` gives one per extra item.
4. **The keyword codes** (`params` in order; the span per D5 item 6):

   | Keyword | Code | `params` | Span |
   |---|---|---|---|
   | `type` | `input.type` | `expected` (array of type names), `actual` (the value's JSON type: `null`, `boolean`, `object`, `array`, `number`, `string`) | value |
   | `enum` | `input.enum` | `allowed` (the first 20 allowed values), `allowed_count`, `suggestion` (D9 item 7, when one exists) | value |
   | `const` | `input.const` | `expected` | value |
   | `required` | `input.required` | `property` | parent |
   | `dependentRequired` | `input.dependent_required` | `property`, `because` (the present member that requires it) | parent |
   | `additionalProperties` (false) | `input.additional_property` | `property`, `suggestion` (a declared member, D9 item 7) | key |
   | `unevaluatedProperties` (false) | `input.unevaluated_property` | `property`, `suggestion` | key |
   | `propertyNames` | `input.property_name` | `property` | key |
   | `minProperties` / `maxProperties` | `input.min_properties` / `input.max_properties` | `limit`, `count` | value |
   | `minLength` / `maxLength` | `input.min_length` / `input.max_length` | `limit`, `length` (code points) | value |
   | `pattern` | `input.pattern` | `pattern` | value |
   | `format` | `input.format` | `format` | value |
   | `minimum`, `maximum`, `exclusiveMinimum`, `exclusiveMaximum` | `input.minimum`, `input.maximum`, `input.exclusive_minimum`, `input.exclusive_maximum` | `limit`, `value` (D9 item 5) | value |
   | `multipleOf` | `input.multiple_of` | `divisor`, `value` (D9 item 5) | value |
   | `minItems` / `maxItems` | `input.min_items` / `input.max_items` | `limit`, `count` | value |
   | `uniqueItems` | `input.unique_items` | `index`, `duplicate_of`; `related` gives the earlier item | value (the later item) |
   | `contains` | `input.contains` | — | value |
   | `minContains` / `maxContains` | `input.min_contains` / `input.max_contains` | `limit`, `count` | value |
   | `items: false` after `prefixItems` | `input.additional_item` | `index`, `prefix_count` | value (the item) |
   | `unevaluatedItems` (false) | `input.unevaluated_item` | `index` | value (the item) |
   | a `false` subschema elsewhere (e.g. `properties: {x: false}`) | `input.forbidden` | `property` when reached through `properties` | value |
   | `not` | `input.not` | — | value |
   | `anyOf` / `oneOf` with no branch matching | `input.any_of` / `input.one_of` | `branch_count` | value |
   | `oneOf` with more than one branch matching | `input.one_of.ambiguous` | `branches` (the two indexes) | value |
   | `contentEncoding` / `contentMediaType` / `contentSchema` | `input.content_encoding` / `input.content_media_type` / `input.content_schema` | `encoding` / `media_type` / — | value |

   A keyword ADR-003 does not assert produces no finding, and its code is not
   emitted. A `false` schema is named by the applicator that reached it:
   under `additionalProperties`, `unevaluatedProperties`, `items` or
   `unevaluatedItems` it takes that keyword's code; anywhere else it is
   `input.forbidden`. A `contentSchema` failure is one finding with no
   children, because a pointer into decoded content would not address
   anything in the input.
5. **Transparent applicators.** `$ref`, `$dynamicRef`, `allOf`, `properties`,
   `patternProperties`, `additionalProperties` and `unevaluatedProperties`
   with a schema value, `items`, `prefixItems`, `contains`'s subschema and
   `dependentSchemas` produce no finding of their own; their failed
   descendants are reported as if the keyword were not there. The evaluation
   path is kept in `schema.keyword_location`.
6. **`anyOf` and `oneOf` with no branch matching** are reported by this rule,
   chosen so that a tagged union reports the one branch the supplier meant:
   1. A failure in a branch is **discriminating** when it is a `type` failure
      at the applicator's own instance location, or a `const`, `enum` or
      `false`-schema failure at the applicator's instance location or at a
      direct member of it.
   2. A branch with a discriminating failure is **rejected**.
   3. **If exactly one branch is not rejected**, its findings are reported as
      ordinary findings, each with a `via` entry `{"keyword": "oneOf" |
      "anyOf", "schema_pointer": …, "branch": i, "title": …}`, `title` being
      the branch's title when it has one (D6 item 9's lookup applied to the
      branch).
   4. **Otherwise** one `input.any_of` or `input.one_of` finding is reported,
      with `alternatives`: one object per branch in index order —
      `{"branch": i, "title": …, "findings": [...], "omitted": n}` — whose
      `findings` are that branch's findings mapped by these same rules
      (recursively), sorted by D8, at most 16 per branch, `omitted` counting
      the rest.
   5. `oneOf` with two branches matching is `input.one_of.ambiguous`, with no
      alternatives.
7. **`if`/`then`/`else` are transparent with a reason.** The `if` subschema
   produces no finding. A failure under `then` or `else` is reported as an
   ordinary finding with a `via` entry `{"keyword": "then" | "else",
   "schema_pointer": …, "condition": {"schema_pointer": <the if>, "title":
   <the if subschema's title, when it has one>}}`, so a stage-dependent
   requirement reads "required (when: Final report)" (PRD §4.2, SC2). `via`
   is an array, outermost first, when conditions and branches nest.
8. **`schema`**: `entry` is the package entry the keyword was written in (an
   input schema file, an output schema file, or the manifest schema's
   published name for `manifest.` codes); `pointer` is the keyword's location
   **within that entry**, resolvable by RFC 6901 against the entry as
   authored; `keyword` is the keyword's name; `keyword_location` is the
   evaluation path from the input's root schema through every `$ref`, in
   JSON Schema's own `keywordLocation` form. Validation compiles the schemas
   from their original entries, which every package carries, so `entry` and
   `pointer` always point at text an author wrote; a schema resource with its
   own `$id` inside an entry is mapped back to the entry and the pointer from
   the entry's root.
9. **`field`** is found by walking the evaluation path from the failed
   keyword back toward the root: the first schema object whose instance
   location equals the finding's instance location and that has a `title`
   supplies `field.title`, and likewise, independently, `description`.
   For an absent member (D5 item 2) the lookup is the `properties/<name>`
   subschema of the schema object holding `required`, following its `$ref`
   chain when it has neither. A schema object at a *different* instance
   location never supplies a field's title: a parent's title describes the
   parent. `field` is absent when neither is found. For `manifest.` codes the
   manifest schema's titles and descriptions apply (ADR-002 promises every
   property has both).

10. **What "every finding" means where a validator answers only yes or
    no.** Three contexts never contribute a finding from inside them: the
    subschema of `not` (the finding is `input.not` itself, item 4), the `if`
    subschema (item 7), and the branches of an `anyOf` or `oneOf` left
    untried or unreported once a branch has matched (a valid `anyOf`, or a
    `oneOf` that is valid or `one_of.ambiguous`, item 6). A validator may
    therefore evaluate those for a boolean only, as the library studied does
    (ADR-010 D5 item 6), and the report is still every finding (R6 item 1,
    SC8). Everywhere else — `allOf`, `properties` and the other transparent
    applicators of item 5, the `then` or `else` a condition selects, and
    every branch of an `anyOf` or `oneOf` with no matching branch — every
    failure is collected (ADR-003 D3 item 7).

### D7 — Every finding in one pass

**A failure stops only the work that depends on it; everything independent of
it is still checked and reported.**

1. The stopping rules, by phase:

   | Phase | A failure here stops | Still reported |
   |---|---|---|
   | Command line | the command (nothing else is meaningful) | every usage finding the argument parser can detect; at least the first |
   | Opening the package (container, entries, digests, specification version, package limits) | every later phase | every refusal detectable in one scan of the central directory and the digest file (ADR-001): every unsafe entry name, every mismatched digest |
   | The manifest (consumer commands) | every later phase | every manifest finding, in ADR-002's gated steps |
   | Parsing one input | that input's schema validation, except D7 item 4 | every other input, parsed and validated |
   | Validating inputs | rendering | every finding of every input |
   | Rendering one output | that output | every other output, rendered to staging, and its output schema validation |
   | Output schema validation | writing | every finding of every output |
   | The destination | writing | everything above, already reported |

2. **An unparseable file stops that file only.** A JSON or YAML syntax error
   is one finding with its line and column, and that file's input is not
   validated against its schema. The parse finding's hint says that the
   input's schema was not checked and that validating again after the fix may
   report more.
3. **An authoring command checks what does not depend on a failure.** `check`
   with an unparseable manifest still parses every template and compiles
   every schema it can find under the layout ADR-001 fixes; what needs the
   manifest (the cross-reference, output paths) is skipped. ADR-007 owns the
   algorithm; this is the rule it follows.
4. **An array input with unparseable items is still validated, in part.**
   When some items of a collection (a file that does not parse) or of a CSV
   input (a record ADR-003 can reject alone, such as a wrong field count)
   fail to parse, the remaining items are validated against the input's full
   schema as an array with the failed items removed, and every finding is
   reported at the item's **original** index and file. Findings whose
   instance location is the array itself, and findings whose evaluation path
   passes through `prefixItems` or `unevaluatedItems` applied to the array,
   are suppressed, because removing items changes what they measure. One
   `input.validation.partial` finding (info) says so, with `params`
   `{"unparsed": n, "suppressed": m}`. This is what lets SC8 plant a parse
   error and a value error in the same collection and see both.
5. **Render validates first** (R6 item 4): no output is rendered while any
   input has an error, and nothing is written while any finding of class
   `content` is an error. A destination refusal is detected before
   validation where it can be (an existing non-empty directory) and reported
   with the input findings, so a caller fixes both in one round; the exit
   status follows D12 item 3.

### D8 — Ordering, de-duplication and the cap

**Findings are reported in one total order that depends only on the findings,
with exact duplicates removed and the count bounded without losing the
totals.**

1. **The sort key**, compared member by member, all comparisons of strings by
   UTF-8 bytes:
   1. `subject.kind`, in the order: `invocation`, `package`, `manifest`,
      `schema`, `input`, `template`, `asset`, `reference`, `output`,
      `example`, `report` — the order in which a reader has to fix things;
   2. `subject.name` (absent first);
   3. `location.origin`, in the order: `package`, `example`, `data_dir`,
      `inputs_json`, `set`, `default`, `sample`, `output`;
   4. `location.file` (absent first), then `location.argument`;
   5. `location.line`, then `location.column` (a finding with no line sorts
      before one with a line in the same file);
   6. `location.pointer`, compared token by token, a token that is an array
      index of the instance comparing numerically (`/9` before `/10`), a
      prefix before its extensions;
   7. `code`;
   8. `schema.entry`, then `schema.pointer`;
   9. the RFC 8785 canonical form of `params`.
2. **Severity is not in the key.** Errors and warnings about one field sit
   together; the summary (D10 item 3) gives the counts by severity.
3. **Two findings are duplicates** when their `code`, `subject`,
   `location.origin`, `location.file`, `location.argument`,
   `location.pointer` and `params` are equal. One is kept: the one whose
   `schema.keyword_location` sorts first. Duplicates arise when one keyword is
   reached by two evaluation paths (a shared `$ref` under two `allOf`
   branches), or one member is required twice; the supplier needs to hear it
   once. `via` is not part of the key.
4. Findings inside `alternatives` are de-duplicated and sorted within their
   branch, and never against top-level findings.
5. **The cap.** A report lists at most ADR-009's `findings` limit (default
   100,000, which a caller lowers with `--limit findings=<n>`). The tool
   keeps the first that many in sort order while it runs (a bounded heap),
   so memory is bounded by the cap and not by the input.
6. **Totals are exact.** Every finding is counted, after de-duplication, in
   the summary, whether listed or not. When findings are omitted, one
   `report.truncated` finding (info, subject `report`, `params`
   `{"listed": n, "omitted": m}`) is appended after the sorted findings and
   does not count toward the cap. De-duplication is exact even when the
   duplicate falls outside the cap: the tool keeps a set of the SHA-256
   digests of the canonical duplicate keys, bounded by ADR-009's input limits.
7. **The order is a specification rule**, proved by vectors. The validator's
   own order is never relied on; the library studied emits
   `unevaluatedProperties` failures in Go map order.

### D9 — Wording, and what a finding may echo

**A message says what is wrong in terms of the field, a hint says what to do
next, and neither ever repeats a string value from the inputs.**

1. **A message is one plain-text sentence**, at most 240 code points, no line
   break, no Markdown, no terminal formatting. It names the field and the rule
   and states the gap: "Remarks must be at least 10 characters long; it has
   4." It never repeats the location, which has its own member.
2. **A hint is one or two plain-text sentences**, at most 240 code points,
   imperative, naming the next step concretely: a value range ("Write between
   10 and 2000 characters."), the edit ("Add `name` to the vendor object."),
   or the command that shows more (``Run `metais schema <package> vendor` to
   see every field.``). The package is always written `<package>` in a hint,
   never the path the caller gave, so a hint is the same on every machine.
   Every code has a hint, so every refusal names the next step (R17 item 3).
3. **`{field}`** in a template is the field's title when D6 item 9 finds one,
   otherwise the pointer in backticks (`` `/5/remarks` ``), or "the input
   `<name>`" at the root.
4. **Never a string value from the inputs.** No message, hint, param or
   related note contains a string value, or a part of one, taken from a
   supplied input, an example's inputs or a rendered output: a string is
   described by its type and length ("a string of 4 characters"), and found by
   its location. An input may legitimately hold a secret the caller chose to
   supply (R14 item 4), and findings go to logs, CI output and models that the
   input never should; the location already lets the supplier find the value
   in data it holds. A library's message text is never used, because the
   library echoes values.
5. **Numbers, booleans and `null` may be echoed** (`params.value` for the
   numeric keywords), **except** where the schema at the instance location
   carries `writeOnly: true`, the 2020-12 annotation whose own example is a
   password field: then no value, no length and no suggestion appears.
6. **Package content may be echoed**: allowed values, limits, patterns,
   formats, titles, descriptions, member names the schema declares, and the
   names of members the supplier wrote (an object key is structure, and
   naming it is the only way to say which member is extra). The host
   contributes nothing: no environment value, no absolute path, no user or
   host name, no clock (R14 item 4, D5 item 8).
7. **Suggestions come only from package content.** Where a code registers a
   `suggestion` param, it names the package-defined value closest to the
   supplied one under one rule: equal under Unicode simple case folding after
   trimming leading and trailing white space; otherwise the smallest
   Levenshtein distance over code points, at most `max(1, ⌊n/3⌋)` for a
   candidate of `n` code points; ties broken by UTF-8 byte order. Candidates
   are the schema's allowed strings (for `enum`, at most 1000 candidates of
   at most 256 code points each, else no suggestion) or its declared member
   names. The same rule serves ADR-002's unknown manifest member and
   ADR-003's unknown input name. `check`'s cross-reference suggestions
   follow ADR-007 D3's own rule instead: the nearest name within edit
   distance 2. *(A3)*
8. **Operating-system errors are mapped, never quoted.** A failure to read or
   write a file is reported with `params.reason` from a closed set —
   `not_found`, `permission_denied`, `is_directory`, `not_regular_file`,
   `io_error` — because an OS error's text differs by platform and can carry
   a path the caller did not give.
9. **Text from the package or the inputs is neutralised when printed.** In the
   human report every C0 control character (line breaks and tabs included),
   DEL, every C1 control and every bidirectional control (U+202A–U+202E, U+2066–U+2069) in a title,
   description, member name or file name is written as a visible escape
   as `\u` followed by four lowercase hex digits (`\u001b`, `\u202e`), and
   an invalid UTF-8 byte in a file name as U+FFFD. The JSON report writes
   the same characters as the same `\u` escapes. A hostile description
   cannot move the terminal's cursor or reorder what a reviewer sees.
   ADR-009 D15 item 3 applies the same escape to package and input text
   that other human output echoes (`ls`, `outline`, `grep`), keeping a tab
   there, since those lines are content rather than a finding's one-line
   fields.

### D10 — The JSON encoding

**Findings are encoded as a `findings` array and a `summary` object, members in
a fixed order, the same bytes for the same findings on every platform.**

1. **Member order** within a finding is D1's table order; within `location`:
   `origin`, `file`, `argument`, `line`, `column`, `pointer`; within `schema`:
   `entry`, `pointer`, `keyword`, `keyword_location`; within `params`: the
   registry's order. Absent members are omitted. Numbers use ADR-002's
   canonical number form; strings escape D9 item 9's characters, `"` and `\`,
   and nothing else.
2. An example, `metais validate --json` on a CSV row:

   ```json
   {
     "code": "input.min_length",
     "severity": "error",
     "message": "Remarks must be at least 10 characters long; it has 4.",
     "hint": "Write between 10 and 2000 characters explaining how the product meets the criterion.",
     "subject": {"kind": "input", "name": "wcag_criteria", "title": "WCAG 2.2 criteria"},
     "location": {"origin": "data_dir", "file": "wcag_criteria.csv", "line": 7, "column": 41, "pointer": "/5/remarks"},
     "schema": {"entry": "inputs/wcag_criteria.schema.json", "pointer": "/items/properties/remarks/minLength", "keyword": "minLength", "keyword_location": "/items/properties/remarks/minLength"},
     "field": {"title": "Remarks", "description": "How the product meets, partly meets or fails the criterion, with the evidence."},
     "params": {"limit": 10, "length": 4}
   }
   ```

3. **`summary`**: `{"errors": n, "warnings": n, "info": n, "listed": n,
   "omitted": n, "by_code": {"<code>": n, …}}`, the counts exact after
   de-duplication (D8 item 6), `by_code` with keys in byte order; top-level
   findings only, never those inside `alternatives`.
4. **Every command's JSON envelope carries `findings` and `summary`**, both
   always present, and the exit status (ADR-008 owns the envelope, its member
   names and its version; this ADR owns these two members' content). The
   finding shape is versioned with the envelope: within the envelope's major
   version a finding only gains optional members, and a consumer ignores a
   member it does not know.
5. **The specification states the finding object normatively**, with its JSON
   Schema published beside the manifest schema, at
   `spec/schemas/1.0/finding.schema.json` (ADR-011 D7 item 2), so a
   harness can rely on any conforming tool. Conformance compares `code`,
   `severity`, `subject`, `location`, `schema` (`entry`, `pointer`,
   `keyword`), `field`, `params`, `via`, `alternatives` and `related`, and the
   order of the list; it does not compare `message`, `hint` or
   `keyword_location`.

### D11 — The human rendering

**The human report leads each finding with a line an editor can jump to, then
indented lines for the field, the rule and the hint.**

1. Each finding:

   ```
   wcag_criteria.csv:7:41: error: Remarks must be at least 10 characters long; it has 4. [input.min_length]
     at /wcag_criteria/5/remarks (input wcag_criteria: WCAG 2.2 criteria)
     field: Remarks — How the product meets, partly meets or fails the criterion, with the evidence.
     rule: minLength at /items/properties/remarks in inputs/wcag_criteria.schema.json
     when: Final report
     hint: Write between 10 and 2000 characters explaining how the product meets the criterion.
   ```

   The first line is `<where>: <severity>: <message> [<code>]`. `<where>` is
   `file:line:column` (GNU's shape, D5's columns), `file:line`, `file`,
   `--set #2` or `--inputs-json #1`, `input <name>`, `package` or `metais`
   (usage), in that order of preference. The `at` line appears when there is
   a pointer; for a finding about an input it prints the inputs pointer —
   `/`, the input name escaped per RFC 6901, then `location.pointer` — which
   is exactly the path `--set` and `--set-json` take (ADR-003 D17 item 2),
   so a supplier pastes it into the fix; `field` when there is a title or
   description, the description cut at its first line break or 160 code points
   with `…`; `rule` when there is a `schema`; `when` (or `option`) for each
   `via` entry; `hint` always.
2. **Alternatives** are printed under their finding, each branch as
   `option <i+1>: <title>` followed by its findings indented two more spaces.
3. **The last line is the summary**: `2 errors, 1 warning, 0 info` and, when
   truncated, `(100000 listed, 4120 more not shown; use --json for the totals
   by code)`. A command whose product is the report (`validate`, `check`,
   `test`) prints the summary even with no findings.
4. **Colour** only on the severity word, and only as ADR-008's colour policy
   allows. The human format is the reference tool's, not the specification's:
   it may be improved in any release, and machines read the JSON.

### D12 — Exit codes

**Every command exits with one status from one fixed table, decided by the
classes of its error findings.**

1. The table:

   | Status | Meaning |
   |---|---|
   | `0` | Success: no finding failed the command (warnings and info may exist). |
   | `1` | Findings: at least one error of class `content` — invalid inputs, a failed check, a failed test, an output refused by its schema, an output path refused — or a warning under `--fail-on warning`. |
   | `2` | Internal error: a defect in the tool. Also what the Go runtime exits with on a crash, so status 2 always means "metais failed, not your data". |
   | `3` | Usage: the command line is wrong. |
   | `4` | Package refused: an error of class `refused`, or a `content` error about the package itself under a consumer command (item 2). |
   | `5` | Destination refused: an error of class `destination`. |

2. **Who the package's own defects fail.** An error of class `content` whose
   subject is `package`, `manifest`, `schema`, `template`, `asset` or
   `reference` exits `4` under a consumer command (`inspect`, `ls`, `read`,
   `outline`, `grep`, `schema`, `extract`, `validate`, `render`), because the
   caller there received a package and cannot fix it; and `1` under an
   authoring command (`init`, `check`, `test`, `pack`, `unpack`, `sample`,
   and `render --sample`), because the caller there is the author fixing
   it. Class `refused` exits `4` under every command. `version`, `help` and
   `limits` open no package. ADR-008 D13 states each command's role.
3. **When several classes occur, the status is the first that applies** in the
   order `2`, `3`, `4`, `1`, `5`: a destination refusal reported beside input
   errors exits `1`, because fixing the destination alone would not help.
4. **Status 2 is never used for usage.** The convention of `2` for usage
   errors (the validator's own `jv` uses it) collides with the Go runtime's
   crash status, and a harness must be able to tell "my command line is
   wrong" from "the tool crashed". The tool recovers a panic, reports
   `internal.error` (`params`: `tool_version`, `phase`; no panic value and
   no stack trace, in the report or on any stream, ADR-009 D14 item 5), and
   exits 2; an unrecoverable runtime failure exits 2 by itself.
5. **Signals** are not the tool's to number: a process killed by a signal
   exits as the shell reports it (128 + the signal), and a write to a closed
   standard output ends the process with SIGPIPE as Go programs do.
6. **The table is normative** for any implementation with a command line, so
   that a harness's handling of status is portable between conforming tools.

### D13 — The implementation shape, and how it is proved

**Findings are a small internal package every command uses, fed by a
library-neutral failure tree and by parsers that keep positions.**

1. **`internal/finding`** (package naming is ADR-010's) holds the model:

   ```go
   type Severity uint8 // SeverityError, SeverityWarning, SeverityInfo
   type Class uint8    // ClassUsage, ClassRefused, ClassContent, ClassDestination, ClassInternal
   type Code string

   type Entry struct { // one registry entry, loaded from the registry table
       Code     Code
       Severity Severity
       Class    Class
       Subject  SubjectKind
       Params   []ParamSpec
       Span     SpanKind // Value, Key, Parent
       Message, Hint string // templates
       Since    string
   }

   // New is the only constructor; it renders Message and Hint from the
   // registry templates and panics in tests on an unregistered code or a
   // parameter the entry does not declare.
   func New(code Code, subj Subject, loc *Location, p Params) Finding

   type Report struct{ /* bounded heap, duplicate-key digests, exact counts */ }
   func NewReport(maxFindings int) *Report
   func (r *Report) Add(f Finding)
   func (r *Report) Findings() []Finding // sorted, de-duplicated, capped, plus report.truncated
   func (r *Report) Summary() Summary
   func (r *Report) Exit(role Role, failOn Severity) int
   ```

2. **The position interface ADR-003's parsers provide** (one per parsed file
   or argument):

   ```go
   type Pos struct{ Line, Column int } // 1-based; Column in code points; zero = unknown

   type Positions interface {
       Value(ptr string) (Pos, bool) // the first character of the value at ptr
       Key(ptr string) (Pos, bool)   // the first character of the member name, for an object member
   }

   type Parsed struct {
       Value     any       // the data model the schema validates
       Raw       []byte    // the original bytes (R4 item 4)
       Origin    Origin    // per D4; per-value origins after the merge via OriginOf
       File      string    // relative to the origin's root
       Positions Positions // nil for --set, default and sample values
   }
   ```

   The JSON parser records a byte offset per value and per key while decoding;
   the CSV parser records `FieldPos` per field and the record's start line; the
   YAML parser records each node's `Line` and `Column`; frontmatter positions
   are offset by the frontmatter's first line. Positions are stored per pointer
   in a map built during the one parse, and a byte-to-code-point line index
   per file converts offsets (D5 item 4).
3. **The failure tree** (`internal/schemafail`, say) is what D6's rules
   consume: `Failure{Keyword, KeywordLocation, Entry, SchemaPointer,
   InstancePointer []string, Data (the keyword's structured fields), Children
   []Failure}`. ADR-010's adapter builds it from the chosen library's tree (for
   the library studied: from `SchemaURL`, `InstanceLocation`, the `ErrorKind`'s
   fields and `KeywordPath()`, and `Causes`, with `kind.Reference` nodes giving
   the `$ref` steps of the evaluation path). The mapping has unit tests on
   hand-built trees, independent of any library.
4. **The registry has one source.** `spec/findings/registry.json` is the
   normative text; the tool's Go table is checked against it by a test that
   fails on any difference, and a second test fails when a code literal in the
   source is not in the registry, or a registered, unretired code has no
   conformance vector that produces it.
5. **Pinned false first**: the echo test (*Conformance*, "no echo") is written
   red against a mapping that uses the library's messages, which echo
   `pattern` and `format` values; the ordering test is written red against a
   report that keeps the validator's order and is run with `-count` high
   enough to see the map-order variation; the SC8 test is written red against
   a validator that stops at the first error.

## Interfaces with sibling ADRs

**ADR-001 (container).** Relies on: every container, entry-name, layout,
digest and `.compiled/` failure reported as one of the `package.*` codes
ADR-001 lists (`package.container.*`, `package.name.*`, `package.layout.*`,
`package.compiled.*`, `package.digest.*`), of class `refused` except the
source-directory warning `package.layout.not_packed`, with
`location.origin: package` and `file` the entry name (absent for a
whole-archive failure); one verification pass that reports every unsafe
entry and every digest mismatch together (D7 item 1; ADR-001 D8 item 2).
Promises: the model, ordering and exit status 4 for all of them.

**ADR-002 (manifest).** Relies on: canonical JSON numbers (RFC 8785) for
`params` and the summary; every manifest-schema property carrying `title`
and `description` (ADR-002 D10 item 4); manifest positions from its YAML and
JSON profiles through D13 item 2's interface; the manifest schema's
published entry name for `schema.entry`. Promises (to what ADR-002 asked
for): a stable code per refusal, which ADR-002 registers in the `manifest`
family and, for the version gate, as `package.version.*`; locations carrying
both line and column and a JSON Pointer; the unknown-member finding's
`suggestion` computed by D9 item 7; the keyword mapping of D6 under the
`manifest.` prefix; an unknown specification major as class `refused` with
`params.implemented` listing the versions. ADR-002 wants some findings to
quote manifest source text (`metais: 1.0`): manifest text is package content
and may be quoted (D9 item 6).

**ADR-003 (data).** Relies on: parsers that return D13 item 2's `Parsed` with
positions for JSON, YAML, CSV and Markdown frontmatter and bodies; the
per-value origin after the precedence merge (D4 item 5); the collection item
order that indexes pointers; CSV data-row indexing from 0 with the header
excluded; which keywords and formats are asserted (D6 item 4 emits only
those); the input's title; schemas compiled from their original entries, the
combined `.compiled/schema.json` being for consumers (D6 item 8); which CSV
record failures are recoverable alone (D7 item 4); a CSV value that does not
coerce left as a string for `input.type` to report, with the field's
position (ADR-003 D14 item 3); the boolean-only contexts of D6 item 10,
which ADR-003 D3 item 7 states the same way. ADR-003 registers the `json`,
`yaml`, `csv`, `text`, `markdown` and `collection` codes and the supply
codes in `input` (its Interfaces list them), each parse code's hint saying
the schema was not checked. Promises: D6's mapping, D7's partial validation,
and the inputs pointer on the human `at` line (D11 item 1).

**ADR-004 (templates).** Relies on: template parse and render errors as the
`template.*` codes of ADR-004 D16, with `location.origin: package`, `file`
the template entry and line and column within it; a render error stopping
only its output (D7 item 1); no template error message carrying a value from
the inputs (D9 item 4) — an engine's error text is mapped to codes, never
quoted; a limit's breach as ADR-009's `limit.*` code.

**ADR-005 (outputs).** Relies on: output schema validation reported with D6's
mapping under `output.`, `subject.kind: output`, `location.origin: output`,
`file` the output path; output path refusals (traversal, Windows-invalid
name, case collision with `related`) as class `content` (exit 1, since a
templated path can come from the data); the non-empty directory, a leftover
staging name and write failures as class `destination`; the destination
check made before validation where it can be (D7 item 5); the declaration
`id` as `subject.name`, with no `subject.title`. A dry run reports findings
like a render and writes nothing.

**ADR-007 (authoring).** Relies on: `check`, `test`, `pack`, `unpack`,
`sample` and `render --sample` reporting through this model; `check`
following D7 item 3, naming any step it could not run in
`report.step_skipped`; `test`'s byte mismatch as an `example.*` code with
`subject.kind: example`, `location.file` the expected entry and
`line`/`column` of the first differing line; the sample generator's
unsatisfiable field as `schema.sample_unsatisfiable` with a hint suggesting
an `examples` value; `pack` refusing on errors and not on warnings (D2 item
6). The cross-reference's unknown field is an error, its unused input a
warning (R13 item 2).

**ADR-008 (command line).** Relies on: an envelope carrying `findings`,
`summary` and the exit status on every command (D10 item 4); the role of each
command, consumer or authoring (ADR-008 D13); the `--fail-on error|warning`
flag on every command that reports findings (ADR-008 D4); no flag that
prints a stack trace; the human report on standard output for `validate`,
`check` and `test`, and on standard error for every other command
(ADR-008 D11 item 3); a JSON report even for a usage error when `--json` was
recognised (ADR-008 D10 item 7). ADR-008 registers the `usage.*` codes and
`package.not_found` / `package.not_readable` (class `refused`). Promises:
D12's table, which ADR-008 D13 maps every outcome onto and its help text
prints.

**ADR-009 (security).** Relies on: the `findings` limit (default 100,000,
lowered with `--limit findings=<n>`); each limit breach as ADR-009 D19's
`limit.package` (class `refused`) when the package's own content breaches
it, and `limit.exceeded` (class `content`) when inputs, a render's work or
its outputs do, with `limit.invalid` (class `usage`) for a bad `--limit`; a
regular-expression budget whose breach in a schema `pattern` or in `grep` is
a finding, not a hang; SC7's sentinel test covering every finding and
message of every vector and hostile package; ADR-009 D15 item 3 using D9
item 9's escape form for other human output. Promises: D5 item 8 and D9
items 4, 8 and 9 as the finding side of host isolation.

**ADR-010 (implementation).** Relies on: a JSON Schema library whose error
tree carries, per failure, the instance location, the schema location, the
keyword and its structured data, and nested causes, and which collects every
error outside D6 item 10's boolean-only contexts (the library studied does,
ADR-010 D5 item 6); an adapter to D13 item 3's tree; parsers able to report
positions (D13 item 2); the registry drift test. Promises: the mapping never
depends on library message text.

**ADR-011 (specification and conformance).** Relies on: `12-findings.md` for
the findings chapter, `spec/findings/registry.json` for the registry, and
`spec/schemas/1.0/finding.schema.json` and
`spec/schemas/1.0/finding-registry.schema.json` for their schemas (ADR-011
D1 and D7 item 2); a vector format whose expected result is a findings list
compared on D10 item 5's members and in order (ADR-011 D10 item 2); a vector
per registered code (ADR-011 `TestFindingCodesCovered`); a trace from each
registry entry's `statements` to its normative text.

## Conformance

Each of these is observable and needs vectors (ADR-011 owns their format):

- **Every finding in one pass (SC8).** An input set with errors planted in
  several inputs, several fields of one input, several rows of a CSV input and
  several files of a collection, plus a parse error in one collection item:
  every planted finding is listed, and `input.validation.partial` appears.
- **Required members.** `code`, `severity`, `message`, `hint`, `subject` and
  `params` in every finding of every vector; `location.pointer` in every
  finding about a value.
- **Keyword mapping.** One vector per keyword code in D6 item 4, including the
  split of `required` into one finding per member, the pointer to the absent
  member with the parent's line, and the key span for an extra member.
- **Applicators.** `anyOf`/`oneOf` collapsing to the one non-rejected branch
  (a tagged union by `const` on a member), the `alternatives` form when every
  branch is rejected, `one_of.ambiguous`, `if`/`then` with `via` and the
  condition's title, `$ref` transparency with `schema.entry` and `pointer`
  resolvable into the authored file.
- **Field lookup.** A title beside a `$ref`, a title only on the `$ref`'s
  target, a failing keyword inside `allOf` under a titled property, and a
  parent's title not used for a child.
- **Positions.** Line and column for JSON, YAML, CSV (including a quoted field
  spanning lines) and frontmatter; a line with multi-byte characters and a
  tab before the value (code-point columns); CRLF and lone-CR files; `--set`
  with no line; `--inputs-json` with a line within the argument.
- **Pointer escaping.** A member named `a/b~c` gives `/a~1b~0c`.
- **Ordering and de-duplication.** Findings presented to the validator in
  scrambled order come out in D8's order; `/9` before `/10`; a keyword reached
  twice through `allOf` is reported once.
- **The cap.** `findings` lowered to 3 (`--limit findings=3`) on an input with 10 errors: 3
  listed, `report.truncated` with `omitted: 7`, the summary's exact totals.
- **No echo.** Every string value in every input of a vector set to a
  sentinel; no message, hint, param or note contains it; a `writeOnly` number
  is not echoed; a `suggestion` names an allowed value only.
- **Neutralisation.** A schema description holding ESC and U+202E appears as
  `\u001b` and `\u202e` in the JSON report.
- **Exit codes.** One vector per row of D12's table but rows 2 and 3, which
  `internal/cli` tests prove because no vector step can exit with them
  *(A3)*; the role rule for a manifest defect under `validate` (4) and under
  `check` (1); and the precedence of item 3.

## Alternatives rejected

### JSON Schema's own basic or detailed output as the report

It is a standard, and the library produces it. But it covers only schema
validation, not parsing, packages, templates or usage; it has no file, line,
stable code, title, description or hint, which R6 item 2 requires; and its
error wording is undefined by the specification itself. The finding keeps its
two pointers (`pointer`, `schema.keyword_location`) in JSON Schema's own
terms, so nothing is lost.

### SARIF as the report format

SARIF is the right prior art and much of the model follows it: a stable
hierarchical rule identifier, a fixed level set, 1-based lines and columns,
code-point columns, a summary-first message. As the report itself it is a log
format for analysis runs, with runs, tools, artifacts and rules a supplier
does not need, and it has no place for an instance pointer or a field's
title. The mapping is direct (code → `ruleId`, severity → `level` with `info`
→ `note`, location → `physicalLocation.region` with `columnKind:
unicodeCodePoints`), so a later minor can add SARIF as an output without
changing the model.

### Only the first error, or the library's tree as it is

The first error alone fails R6 item 1 and SC8. The raw tree reports a node per
applicator, puts `required` at the parent with a list, and orders some
failures by Go map iteration; a supplier reading it has to understand JSON
Schema's evaluation to find the field.

### A relevance heuristic for `anyOf` and `oneOf`

`best_match` and its relatives pick an error by depth and keyword weight, and
their own documentation says the choice may change between versions. A second
implementation cannot reproduce a heuristic that is allowed to change.
D6 item 6's rule is a few lines, deterministic, and right for the case that
matters — a union tagged by a `const` or a `type`.

### Opaque numeric codes (`E1042`)

SARIF recommends opaque identifiers, and numbers sort neatly. But a dotted
name is read correctly by a person and a model without a lookup
(`csv.header.missing`), its family says who owns it, and stability comes from
the registry's rule, not from opacity.

### Echoing input values, truncated or filtered by a secret detector

An echoed value is convenient, and a detector looks safe. But a detector is a
guess: a token, an account number or a personal name passes any pattern, and
a finding is exactly the text that lands in logs. Since every value has a
location, the supplier — who holds the data — loses nothing by not seeing it
repeated.

### Severity first in the order

Errors first reads well at the top of a long list, but separates an error and
a warning about the same field and makes the order depend on a property that
is not a location. The summary gives the counts; the JSON consumer filters.

### Byte or UTF-16 columns, or GNU display-width columns

Byte columns depend on the encoding of what precedes the value; UTF-16
columns are an artefact of JavaScript and Windows strings; display width
depends on the terminal and the locale, which R10 item 3 excludes. Code points
are what a text editor's column counter shows for most text and what SARIF
offers as `unicodeCodePoints`.

### Exit 2 for usage errors

It is the common convention, and the library's `jv` follows it. But any Go
program that crashes exits 2, so with that convention a harness cannot tell a
wrong command line from a crash and may retry a bug as if it were its own
mistake.

### A severity set by the caller or the package

Letting a package downgrade an error, or a caller turn off a code, would make
the same inputs valid under one invocation and invalid under another. The
only caller control is whether warnings fail the command (D2 item 5), which
changes the exit status and nothing in the report.

## Consequences

- A supplier — a person, a program or a model — sees every problem at once,
  each with a place it can open and a step it can take, and a program
  branches on codes and statuses rather than on prose.
- The registry becomes part of the specification's surface: every new
  refusal anywhere in the tool is a registry entry, a vector, and a normative
  statement it traces to. That is overhead per code, and it is the overhead
  that makes a second implementation's findings comparable.
- Messages never show a supplier's string values, so a message about a bad
  enum value reads "is not one of the allowed values" plus a suggestion,
  rather than quoting the typo. The location makes up for it.
- Positions must be kept by every parser through the precedence merge, which
  rules out a parse-to-`any` library that discards them; ADR-010 has to
  choose or wrap parsers accordingly.
- Bounded reports on huge inputs list the first 1000 findings in order with
  exact totals, so a 100 000-row CSV with one systematic mistake reports its
  scale without exhausting memory.

## What this does not decide

- The codes of the families other ADRs own (D3 item 2): their members, their
  params and their wording are registered by those ADRs.
- The JSON envelope, its version member, and the stream each report goes to
  (ADR-008).
- The value of the `findings` limit and of every other limit (ADR-009).
- Which JSON Schema keywords and formats are asserted (ADR-003), and which
  library validates them (ADR-010).
- The vector file format and runner (ADR-011).
- A SARIF output, end positions and localisation of messages: each can be
  added by a later minor without changing a code.

## Proposed rulings

1. **Three severities, `info` among them**, `info` never affecting the exit
   status. Lean: yes — adding a severity later would break consumers that
   switch on the set, and partial validation and truncation need a
   non-problem severity now. Alternative: `error` and `warning` only, with
   truncation and partial validation as warnings.
2. **Severity and class fixed per code**, no override by package or caller.
   Lean: yes, for one meaning per code. Alternative: per-package severity
   overrides.
3. **`--fail-on warning`** makes warnings exit 1 without changing the report.
   Lean: yes — authors gate CI on unused inputs. Alternative: no such flag.
4. **Dotted family codes with a JSON registry in `spec/`**, a drift test and a
   vector per code. Lean: yes. Alternative: opaque numeric codes; or a
   Markdown-only registry.
5. **Exit table 0 success, 1 findings, 2 internal, 3 usage, 4 package
   refused, 5 destination refused**, normative for any command-line
   implementation, with the precedence 2, 3, 4, 1, 5. Lean: yes, because Go
   owns status 2. Alternative: 2 for usage, internal elsewhere.
6. **A package's own content error exits 4 under a consumer command and 1
   under an authoring command.** Lean: yes — the caller who can fix it is the
   author. Alternative: one status for every content error.
7. **Output path refusals are class `content` (exit 1)**, not refused, since a
   templated path can come from the data. Lean: yes. Alternative: exit 4.
8. **No string value from the inputs is ever echoed**; numbers, booleans and
   `null` are, except under `writeOnly`; object keys and package content are.
   Lean: yes. Alternative: echo strings up to a length, or behind a flag.
9. **The `anyOf`/`oneOf` rule of D6 item 6** (discriminating `type` at the
   node, `const`/`enum`/`false` at the node or a direct member; collapse when
   exactly one branch survives). Lean: yes, confidence medium — it handles
   tagged unions and nullable types and falls back to showing every branch.
   Alternative: always show every branch.
10. **One finding per missing member, pointing at the absent member** with the
    parent's line. Lean: yes. Alternative: JSON Schema's single finding at the
    parent with a list.
11. **Partial validation of array inputs with unparseable items** (D7 item
    4). Lean: yes — it is what SC8 needs for collections and CSV.
    Alternative: skip the whole input's schema validation.
12. **Order by location, not severity**, with subject names in byte order.
    Lean: yes. Alternative: severity first; manifest declaration order for
    names.
13. **Columns in Unicode code points, tabs as one column; no end positions in
    v1.** Lean: yes. Alternative: byte columns; GNU display width; ranges now.
14. **A finding cap — ADR-009's `findings` limit, default 100,000 — with
    exact totals and `report.truncated`.** Lean: yes; the number is
    ADR-009's. Alternative: no cap.
15. **Messages and hints non-normative; codes, params, locations, fields
    normative.** Lean: yes — wording improves without a minor version.
    Alternative: normative wording.
16. **Hints write the package as `<package>`.** Lean: yes, for determinism and
    for SC7. Alternative: echo the path the caller gave.
17. **One near-match rule for every `suggestion`** (case-fold-trim, then
    Levenshtein within `max(1, ⌊n/3⌋)`). Lean: yes. Alternative: per-ADR
    rules.
18. **Control and bidirectional characters neutralised** in both reports.
    Lean: yes. Alternative: raw text.
19. **No stack trace on any stream, and no debug flag** (D12 item 4). Lean:
    yes. Why: a trace is host-derived text a sentinel test cannot vouch
    for, and a defect is reproduced from the command line and the package.
    Alternative: a trace to standard error behind a flag.
20. **The human `at` line prints the inputs pointer, the path `--set`
    takes** (D11 item 1). Lean: yes. Alternative: the pointer within the
    input plus the input's name.

## References

- JSON Schema 2020-12, Core, §12 Output Formatting (formats; `keywordLocation`,
  `absoluteKeywordLocation`, `instanceLocation`; error wording undefined;
  detailed-format condensation):
  https://json-schema.org/draft/2020-12/json-schema-core#section-12
- JSON Schema 2020-12, Validation, §9.4 `readOnly` and `writeOnly` (the
  password-field example):
  https://json-schema.org/draft/2020-12/json-schema-validation#section-9.4
- RFC 6901, JavaScript Object Notation (JSON) Pointer (syntax, `~0`/`~1` and
  decoding order, array indices and `-`, URI fragment form):
  https://www.rfc-editor.org/rfc/rfc6901
- SARIF 2.1.0 Errata 01, OASIS (§3.27.5 `ruleId`, §3.27.10 `level`, §3.30.5–
  §3.30.8 region lines and columns, §3.14.27 `columnKind`, §3.11.3 plain-text
  messages):
  https://docs.oasis-open.org/sarif/sarif/v2.1.0/errata01/os/sarif-v2.1.0-errata01-os-complete.html
- `github.com/santhosh-tekuri/jsonschema/v6`, v6.0.3: `ValidationError` and
  `ErrorKind` (validator.go), error collection and `if`/`not`/`oneOf`
  boolean contexts (validator.go `condValidate`), `unevaluatedProperties` map
  iteration (validator.go `unevalValidate`), structured kinds and their
  messages (kind/kind.go), basic and detailed outputs (output.go), `jv` exit
  codes (README): https://github.com/santhosh-tekuri/jsonschema
- `python-jsonschema`, `best_match` and `by_relevance`
  (jsonschema/exceptions.py):
  https://github.com/python-jsonschema/jsonschema/blob/main/jsonschema/exceptions.py
- Go `runtime` package documentation, GOTRACEBACK (exit code 2 on an
  unrecovered panic): https://pkg.go.dev/runtime
- Go `os/signal` package documentation, SIGPIPE on standard output:
  https://pkg.go.dev/os/signal
- Go `encoding/csv`, `Reader.FieldPos` (1-based, byte columns) and
  `ParseError`: https://pkg.go.dev/encoding/csv
- Go `encoding/json`, `Decoder.InputOffset` and `Decoder.Token`:
  https://pkg.go.dev/encoding/json
- YAML organisation's `go-yaml` v3, `Node.Line`/`Node.Column` (yaml.go) and
  per-character column advance (scannerc.go `skip`):
  https://github.com/yaml/go-yaml ; the original repository, archived:
  https://github.com/go-yaml/yaml
- GNU Coding Standards §4.4, Formatting Error Messages:
  https://www.gnu.org/prep/standards/html_node/Errors.html
- Unicode Standard Annex #9, Unicode Bidirectional Algorithm (explicit
  embedding, override and isolate controls): https://www.unicode.org/reports/tr9/
- CVE-2021-42574 (bidirectional control characters reorder displayed text):
  https://nvd.nist.gov/vuln/detail/CVE-2021-42574
- RFC 8785, JSON Canonicalization Scheme (the canonical form of `params` in
  the sort key; ADR-002's choice): https://www.rfc-editor.org/rfc/rfc8785

## Amendments

### A1 — The `manifest` family registers only the keyword codes the manifest schema can produce

**Status:** Accepted 2026-10-19 by Ed Funnekotter, in design session
`01a15348-4ab5-7743-9802-86072e41b5bf` (PLAN-2026-10-metais-v1, W3-A1).
**Amends:** D6 item 1 (the three families) and the registry
`spec/findings/registry.json`.

1. **A keyword code is registered in the `manifest` family only when the
   specification's manifest schema uses its keyword.** The 1.0 manifest
   schema uses none of `additionalItems`, `exclusiveMaximum`,
   `exclusiveMinimum`, `format`, `maxContains`, `maxProperties`, `maximum`,
   `minContains`, `minProperties`, `minimum`, `multipleOf`,
   `propertyNames`, `unevaluatedItems` and `unevaluatedProperties`, so
   their `manifest.*` codes are not registered: `manifest.additional_item`,
   `manifest.exclusive_maximum`, `manifest.exclusive_minimum`,
   `manifest.format`, `manifest.max_contains`, `manifest.max_properties`,
   `manifest.maximum`, `manifest.min_contains`, `manifest.min_properties`,
   `manifest.minimum`, `manifest.multiple_of`, `manifest.property_name`,
   `manifest.unevaluated_item` and `manifest.unevaluated_property`.
2. **D6's mapping is unchanged.** It names a manifest failure exactly as it
   names an input's or an output's, with the family prefix. A keyword the
   manifest schema does not use has no failure to name.
3. **The `input` and `output` families keep every keyword code**, because
   a package's own schemas may use any keyword ADR-003 admits.
4. **A tripwire holds item 1.** A test walks the embedded manifest schema
   and fails when a keyword it uses maps to a `manifest.*` code the
   registry lacks. A minor that extends the manifest schema with such a
   keyword therefore registers its code, with `since` set to that minor,
   and `TestFindingCodesCovered` then requires a vector that produces it.
5. **D3 item 4 is not broken.** Its rule binds a registry that has been
   released: a released code is never removed. No specification has been
   released, so these codes leave a draft. The registry that 1.0 publishes
   lists only codes the 1.0 tool can emit. A3 item 1 reads item 4 where
   each keyword stands, which takes `manifest.any_of` and
   `manifest.one_of.ambiguous` out of the 1.0 registry too.

### A2 — The registry file is generated from the tool's table

**Status:** Accepted on merge. It records Ed Funnekotter's ruling B4 of
2026-10-19 (PLAN-2026-10-metais-v1 Wave 5b, W5b-3): adding a finding code
edits one hand-written place, plus regeneration.
**Amends:** D13 item 4 (the registry's one source), and how D3 item 3's
file is produced.

1. **The one hand-written form of the registry is the tool's table** in
   `internal/finding`, one file per family (`registry_<family>.go`), with
   the JSON Schema keyword codes expanded from one table per family.
2. **`spec/findings/registry.json` is generated from that table** by
   `go generate ./internal/finding`, and committed. It stays part of the
   specification, as D3 item 3 says, with the form and schema D3 item 3
   gives it: what it states is unchanged, and only how it is produced
   changes. A second implementation reads the file, never the table.
3. **The test D13 item 4 names now pins the file to the table byte for
   byte.** A drift test regenerates the file and fails on any difference
   from the committed file or from the copy the binary embeds, so a hand
   edit of the file fails, as a hand edit of the catalog does. The
   member-for-member test of the table against the file stays.
4. **ADR-011 D3 item 9 stands as written.** The catalog in
   `12-findings.md` is generated from `spec/findings/registry.json`, which
   is the one source of the catalog. The chain is table, then registry
   file, then catalog, and each step is regenerated and pinned by its own
   drift test.
5. **The tool and the specification still cannot disagree unseen.** A
   change to the table that changes what a code means changes the
   generated registry file in the same change. That is a change to the
   specification, and it is reviewed as one.

### A3 — A1's tripwire reads each keyword where it stands; the train's reporting readings

**Status:** Accepted 2026-10-22 by Ed Funnekotter, at the ratification of
PLAN-2026-10-metais-v1 (T-A4). It writes into the decision text readings
the train took and its logs record: item 1 from W7-1 and W7-NB2, items 2
and 3 from Wave 3 (plan §9), item 4 from plan §8.
**Amends:** A1 (items 1 and 4), D4 item 1, D9 item 7 and *Conformance*'s
exit codes, marked *(A3)* in place.

1. **A1 item 4's tripwire applies D6's mapping where each keyword
   stands.** A keyword whose failure the mapping never reports maps to no
   code: the walk does not enter the subschema of `not`, the `if`
   subschema or the subschema of `propertyNames` (D6 items 7 and 10). A
   `oneOf` maps to `manifest.one_of`, and to `manifest.one_of.ambiguous`
   only when two of its branches could both match. Two branches are
   exclusive only when one allows only an object (`type` is `object`, or
   a list of it alone) and requires a member the other, a closed object,
   does not allow; every other pair counts as able to match together,
   the stricter reading. In the 1.0 manifest schema the one `anyOf` is
   beneath the name's `not`, and both `oneOf`s are over exclusive
   branches, so `manifest.any_of` and `manifest.one_of.ambiguous` are not
   registered either, beside the codes A1 item 1 names. A minor that
   moves the `anyOf` or gives a `oneOf` branches that can overlap makes
   `TestManifestKeywordCodes` require the code back, and
   `TestFindingCodesCovered` then requires its vector.
2. **Subjects: the inputs object as a whole is the invocation.** D4 has
   no subject for a failure at the root of the inputs object that names
   no one input, so `input.constraints_failed` (an input constraint other
   than `required` or `dependentRequired`) and `input.json_arg_invalid`
   (an `--inputs-json` finding outside every member) are about the
   invocation.
3. **Suggestions: `check`'s cross-reference follows ADR-007 D3.** Its
   edit distance 2 is the specific rule for a template's unknown names,
   and D9 item 7's `max(1, ⌊n/3⌋)` stays the rule for every other
   suggestion.
4. **Exit statuses 2 and 3, and codes of class `usage` and `internal`,
   are proved by `internal/cli` tests, not vectors.** The conformance
   adapter fails a step that exits 2 or 3, and the command line has no
   chapter, so ADR-008 D16 item 2's test for each refusal code proves
   them (ADR-011 A9 item 4).

## Implementation log

This log is this section together with the entry files in
`docs/adr/logs/ADR-006/`. Each work item writes its entry as a file of its
own there, named as AGENTS.md's "Writing documentation" says. The entries
below are the ones written in this file before that rule, and stay where
they were written.

- **2026-10-19, PLAN-2026-10-metais-v1 W2-11.** D6 and D7 item 4 landed as
  `schemafail.Map`, in `internal/schemafail` beside the tree it reads. The
  keyword fields an adapter writes in a failure's `Data` are listed in
  `internal/schemafail/data.go`; a false schema the instance reached is a
  failure of its own, keyword `false`, named by the applicator its pointer
  shows. The keyword codes are registered in each of D6 item 1's three
  families, with `input.validation.partial`; the content vocabulary has no
  code, since ADR-003 D3 item 4 never asserts it. `contains` is reported,
  and its subschema's failures beneath it, as D6 items 4 and 5 both say.
  D13 item 5's echo test is `TestNoEcho`, with its control beside it;
  `TestChildOrder` permutes every node's children.
- **2026-10-19, PLAN-2026-10-metais-v1 W4-1.** D9 items 1 and 2 hold every
  message and hint the finding renders: a value that would carry the text
  past 240 code points is cut to its share of what the template's own
  text leaves, `{field}`'s pointer keeping its end behind `…` and any
  other value its start before it, and the finding's members keep the
  whole of each. The digests `internal/jsonx` and `internal/inputs` pin
  for their hostile reports were taken again, the messages the only bytes
  that changed. The report lists at most the command's `findings` limit,
  `--limit findings=` included (D8 item 5). `fault.FirstFinding` is a
  `Report` built by `NewReportFaults`, which keeps the first error and
  drops every later one; `internal/guarantees`' `TestOnePassDetector` is
  SC8's planted-errors test over the vector
  `sup/every-finding-one-pass`, with the fault as its control. Chapter 12
  gains D6's mapping, `schema` and applicators, D6 item 9's field, D5's
  positions, D8's cap and totals, and D9's echo and neutralisation rules,
  each with its vector; a vector compares a finding's members, so the
  bytes of D9 item 9's escapes are proved by `internal/cli`'s
  `TestValidateEscapesPackageText`. D12's exit statuses are the command
  line's and are not a vector's to state: the cli adapter checks every
  step's status against its findings, which proves the consumer role's
  rule for a manifest defect under `validate`
  (`fnd/refused-package-validates-nothing`).
