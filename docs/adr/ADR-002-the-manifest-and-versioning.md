# ADR-002: The manifest and versioning — one closed manifest in a strict YAML or JSON profile, canonicalized by RFC 8785, naming a `major.minor` specification version that only grows into what earlier minors refuse

**Status:** Proposed
**Date:** 2026-10-18
**Deciders:** Ed Funnekotter
**Designs:** PRD-001 R1 item 4 (the canonical manifest), R3 item 3 (the
supplier guide), R13 item 1 (the manifest in YAML or JSON), R14 item 3 (an
out-of-spec manifest is refused), R15 items 1 to 4 (versions), R16 item 1 (the
manifest's JSON Schema)
**Depends on:** ADR-001 (the layout that reserves the manifest's entries and
`.compiled/`; the digest file that covers `.compiled/manifest.json`);
ADR-006 (the finding model every refusal here is reported in); ADR-009 (the
size and depth limits applied before a manifest is parsed); ADR-011 (where
the manifest schema is published and how each specification addition is
marked)

## Context

### What the PRD requires

- **The manifest is the package's description**: identity, versions, the
  guide for suppliers, its inputs and its outputs (PRD-001 §5). Its `inputs`
  section is ADR-003's and its `outputs` section is ADR-005's; everything
  else in it is this ADR's.
- **An author writes it in YAML or JSON** (R13 item 1). **`pack` writes it as
  canonical JSON to `.compiled/manifest.json`** (R1 item 4, ruling), so a
  consumer of a packed package needs only a JSON parser, and the bytes are
  stable enough to digest.
- **The guide for suppliers is part of the manifest** (R3 item 3, §5).
- **The specification is versioned, `1.0` for v1; the package names the
  version it targets** (R15 item 1). **The package carries its own semantic
  version** (R15 item 2). **The tool states the versions it implements,
  accepts any of them, and refuses an unknown major** (R15 item 3). **Within
  a major the specification only adds: a package valid under `1.0` stays
  valid, and renders the same, under every later `1.x`** (R15 item 4).
- **A package outside the specification is refused, not approximated**
  (R14 item 3), and the tool adds nothing a package cannot rely on in
  another conforming implementation (AGENTS.md).
- **The manifest's JSON Schema is published beside the specification**
  (R16 item 1).

### Facts that constrain the design (verified)

- **YAML 1.2.2's Core Schema resolves plain scalars by regular expression**
  (§10.3.2): `null`, `Null`, `NULL`, `~` and the empty scalar are null;
  `true`/`True`/`TRUE`/`false`/`False`/`FALSE` are booleans; `[-+]?[0-9]+`
  is a decimal integer (so `012` is the integer 12), `0o[0-7]+` is octal,
  `0x[0-9a-fA-F]+` is hexadecimal; `[-+]?(\.[0-9]+|[0-9]+(\.[0-9]*)?)([eE][-+]?[0-9]+)?`
  is a float (so `1.10` is the number 1.1 and `1.` is a float); `.inf` and
  `.nan` in three spellings each are floats; anything else is a string.
  YAML 1.1's `yes`/`no`/`on`/`off` booleans — the source of the "Norway
  problem" — are not in 1.2.
- **YAML 1.2.2's JSON Schema** (§10.2.2) resolves only `null`, `true`,
  `false` and JSON-shaped numbers, and says a plain scalar matching none of
  them "should" be an error — every string would have to be quoted.
- **YAML 1.2.2 says mapping keys are unique** (§3.2.1.1) and lists a
  non-unique key as a loading failure (§3.3). **It allows a byte order mark
  at the start of any document** (§5.2). **It limits an implicit key to 1024
  Unicode characters** (§7.4.2, §8.2.2), so not every JSON text is YAML.
- **The merge key `<<` is a YAML 1.1 type** (yaml.org/type/merge.html, a
  working draft of 2005-01-18), not part of YAML 1.2.
- **The Go YAML library most programs use does not implement 1.2
  resolution.** `go.yaml.in/yaml` (the YAML organization's maintained fork of
  the archived `go-yaml/yaml`) states that its v3 decodes `0777` as an octal
  per YAML 1.1 and accepts 1.1 booleans when decoding into a typed bool; v3
  is frozen and work continues in v4. Its node tree exposes each node's
  kind (including alias nodes), style (including an explicit-tag flag and
  plain versus quoted), anchor, value text, line and column. A METAIS
  profile can therefore be enforced over the node tree, but not by trusting
  the library's own resolution.
- **RFC 8785 (JSON Canonicalization Scheme, Informational)** requires I-JSON
  input (RFC 7493: unique names, valid Unicode, IEEE 754 double numbers);
  emits no whitespace; escapes only `"`, `\` and U+0000–U+001F (as `\b \t \n
  \f \r` or lowercase `\u00hh`), writing every other character as UTF-8;
  serializes numbers as ECMAScript's `Number.prototype.toString` does
  (`4.50` → `4.5`, `1E30` → `1e+30`, `-0` → `0`); refuses NaN, infinities and
  lone surrogates; sorts object members recursively by their names' UTF-16
  code units; preserves array order. Its output is UTF-8 with no trailing
  newline.
- **Go's `encoding/json` is not JCS.** It formats floats "as if by ES6", but
  escapes `<`, `>`, `&`, U+2028 and U+2029 by default and sorts map keys by
  byte order, which differs from UTF-16 order for characters above U+FFFF.
  `github.com/gowebpki/jcs` is an Apache-2.0 Go implementation of RFC 8785,
  not archived, last pushed 2026-09-21. The choice of library is ADR-010's.
- **Semantic Versioning 2.0.0** forbids leading zeros in numeric
  identifiers, says `v1.2.3` is not a semantic version, says build metadata
  is ignored for precedence, sets no length limit ("use good judgment"), and
  publishes a regular expression for the grammar.
- **JSON Schema 2020-12**: `$id` must resolve to an absolute URI without a
  non-empty fragment, "is an identifier and not necessarily a network
  locator", and implementations should not assume a network operation
  (draft-bhutton-json-schema-01 §8.2.1, §8.2.3). Unknown keywords are
  treated as annotations (§4.3.1), so a validator ignores them.
- **SPDX license expressions** (SPDX 2.3.1 Annex D) are a small grammar:
  identifiers of `ALPHA / DIGIT / "-" / "."`, a `+` suffix, `WITH`, `AND`,
  `OR` (operators case-sensitive), parentheses, and user references
  `LicenseRef-<idstring>`. The list of identifiers changes over time.
- **Windows reserves the device names** `CON`, `PRN`, `AUX`, `NUL`,
  `COM1`–`COM9` and `LPT1`–`LPT9` (and superscript-digit variants), and
  treats one "followed immediately by an extension" as the device:
  `NUL.tar.gz` is `NUL`.
- **The `.org` registry answered "Domain not found" for `metais.org` on
  2026-10-18; `metais.io` is registered to another party.** No domain is yet
  recorded for METAIS.

## Decision

### D1 — The manifest file

1. **A source directory holds exactly one manifest, at its root: `metais.yaml`
   or `metais.json`.** `metais.yaml` is parsed under the YAML profile (D2),
   `metais.json` under the JSON profile (D3); both produce the same data
   model. `init` writes `metais.yaml`; that is the golden path, and
   `metais.json` is the escape hatch for an author whose tooling emits JSON.
2. **Both present is refused**, naming both. **Neither present is refused**,
   naming `metais.yaml` and `metais init`.
3. **No other spelling is read.** A root entry `metais.yml`, or any entry
   whose name equals `metais.yaml` or `metais.json` ignoring letter case but
   not exactly, is refused with a finding naming the exact name. The tool
   compares names byte for byte, so the result does not depend on the
   file system's case sensitivity.
4. **A packed package carries the source manifest verbatim**, byte for byte
   as the author wrote it, beside `.compiled/manifest.json` (D9). Comments
   and formatting therefore survive `unpack`.

### D2 — The YAML profile

The manifest's YAML is YAML 1.2.2 syntax, restricted so that every accepted
document means one thing to every conforming implementation and maps onto
I-JSON (D4's data model). The tool parses to a node tree and applies these
rules itself; it never uses a library's own type resolution.

1. **Encoding.** UTF-8, strictly decoded; an invalid sequence is refused. A
   byte order mark is refused ("remove the byte order mark"). The stream may
   not contain a character YAML forbids.
2. **One document.** An optional `---` start marker and `...` end marker are
   allowed; a second document is refused. **Directives (`%YAML`, `%TAG`) are
   refused.**
3. **The root node is a mapping.**
4. **Anchors and aliases are refused**, both `&name` and `*name`. Without
   them a document's data model is never larger than its text, which removes
   the alias-expansion bomb outright; ADR-009 still bounds size and depth.
5. **Tags are refused**: every explicit tag, including `!!str`, `!!int`,
   local tags and the non-specific `!`. Quoting does what `!!str` would.
6. **The merge key is refused**: a plain-scalar mapping key `<<` is a
   finding, whatever its value. (Quoted `"<<"` is an ordinary string key, and
   D4 refuses it where it is not a defined key.)
7. **Mapping keys are scalars, taken as their text.** A key is the string its
   scalar's content denotes, plain or quoted, with no type resolution: `on:`
   is the key `"on"`, `012:` is the key `"012"`, `true:` is the key `"true"`.
   A key that is a mapping or a sequence (the `? ` complex-key form, or a flow
   collection used as a key) is refused, as is the empty key.
8. **Duplicate keys are refused**, comparing the strings rule 7 produces, so
   `a:` and `"a":` in one mapping collide. Every duplicate is reported, with
   the line of each occurrence.
9. **Scalar values resolve by this table and no other.** Quoted scalars
   (single or double) and block scalars (`|`, `>`, with any chomping or
   indentation indicator) are always strings. A plain scalar is matched in
   order:

   | Plain scalar | Resolves to |
   |---|---|
   | empty (`key:` with nothing after it) | null |
   | `null` | null |
   | `true`, `false` | boolean |
   | `-?(0\|[1-9][0-9]*)(\.[0-9]+)?([eE][-+]?[0-9]+)?` (the RFC 8259 number grammar) | number (D4 item 3) |
   | `~`, `Null`, `NULL`, `True`, `TRUE`, `False`, `FALSE` | **refused**: "write `null`/`true`/`false`, or quote it if it is text" |
   | any other YAML 1.2.2 Core Schema int or float form: a leading `+`, a leading zero (`012`), `0o…`, `0x…`, `.5`, `1.`, `.inf`/`.Inf`/`.INF` with either sign, `.nan`/`.NaN`/`.NAN` | **refused**: "quote it if it is text, or write the number as JSON does" |
   | anything else | string |

   So `no`, `NO`, `yes`, `on`, `2026-10-18`, `12:30`, `1_000` and `1.0.0`
   are strings; `1.0` is the number 1; `012` and `0x1F` are refused rather
   than silently becoming 12 and 31.
10. **Comments are allowed and carry no meaning.**
11. **A syntax error is one finding**, with line and column, and parsing
    stops. Every violation of items 1 to 9 in a syntactically valid document
    is reported in the same pass (ADR-006), each with line and column, and
    the manifest is refused before D7 or D10 run.

### D3 — The JSON profile

`metais.json` is RFC 8259 JSON restricted to I-JSON (RFC 7493):

1. UTF-8, strictly decoded; a byte order mark is refused (as in D2 item 1,
   so the two profiles agree).
2. Exactly one value, an object; whitespace only around it. No comments, no
   trailing commas, no `NaN`/`Infinity`.
3. **Duplicate member names are refused**, comparing unescaped names;
   every duplicate is reported with its line.
4. A string containing an escaped lone surrogate (`\ud800` without its pair)
   is refused.
5. Numbers follow D4 item 3.
6. Findings carry line and column, as in D2 item 11.

### D4 — The data model and the top-level structure

1. **The manifest's data model is an I-JSON value**: objects with unique
   string names, arrays, strings of Unicode scalar values, numbers, `true`,
   `false`, `null`. Both profiles produce it; D9 serializes it; the manifest
   schema (D10) validates it. Every rule below is stated against the data
   model, so it reads the same for YAML and JSON sources.
2. **The root is an object with exactly these members**, in no required
   order. **A member not in this table is refused** (D4 item 4).

   | Member | Required | Type | Owned by |
   |---|---|---|---|
   | `metais` | yes | string, the specification version (D7) | this ADR |
   | `name` | yes | string, the package's name (D5) | this ADR |
   | `version` | yes | string, the package's version (D6) | this ADR |
   | `title` | yes | string | this ADR |
   | `description` | yes | string | this ADR |
   | `guide` | yes | string, CommonMark | this ADR |
   | `authors` | no | array of author objects | this ADR |
   | `license` | no | string, an SPDX license expression | this ADR |
   | `homepage` | no | string, an absolute `https` or `http` URI | this ADR |
   | `inputs` | yes | the data specification | ADR-003 |
   | `outputs` | yes | the output declarations | ADR-005 |

3. **Numbers.** A number is converted to the nearest IEEE 754 double
   (round half to even). A number that overflows to an infinity is refused.
   A number whose value is an integer and whose magnitude exceeds
   2⁵³ − 1 is refused, because it cannot survive the conversion exactly
   (RFC 7493 §2.2). `-0` is the number zero.
4. **Closed everywhere.** Every object this ADR defines (the root and each
   author) refuses a member it does not define, and ADR-003 and ADR-005 are
   asked to do the same for theirs (see *Interfaces*). There is no extension
   prefix (`x-` or any other) in `1.0`. The finding for an unknown member
   names it, its JSON Pointer and line, and — where a defined member is close
   to it — the member it probably meant (wording per ADR-006).
5. **`null` is never a value of a member this ADR defines.** An optional
   member is omitted, not set to `null`; `homepage:` with nothing after it is
   `null` under D2 and is refused with "remove the key to leave it unset".
6. **Text rules used below.** A *line* is a string with no character of
   Unicode general category Cc (no line break, no tab, no other control) and
   neither beginning nor ending with U+0020. *Prose* is a string whose only Cc
   characters are U+000A and U+0009, with no leading or trailing U+0020,
   U+000A or U+0009. Lengths count Unicode scalar values.
7. **`title`**: a line of 1 to 200 characters. What the package is, as a
   person names it ("Accessibility Conformance Report").
8. **`description`**: prose of 1 to 1000 characters, displayed verbatim (not
   Markdown). What the package produces, in a sentence or a short paragraph.
9. **`guide`**: prose of 1 to 16 384 characters, in CommonMark. It is written
   for the supplier: what the package as a whole is for, how its inputs
   relate, the order in which to gather them, and where to read further. It
   is what the summary prints in full (ADR-008), so it is bounded to what a
   supplier reads before anything else; longer material belongs in each
   input's meaning document (ADR-003) or in `references/`, where `outline`,
   `grep` and `read --lines` serve it selectively. A relative link target in
   the guide is a package path resolved from the package root. In YAML the
   golden form is a literal block scalar (`guide: |`).
10. **`authors`**: an array of 1 to 64 objects, each with:
    - `name` (required): a line of 1 to 200 characters — a person or an
      organization;
    - `email` (optional): a line of 3 to 254 characters containing exactly
      one `@` with at least one character on each side and no U+0020. It is
      not validated further, and the tool never sends mail;
    - `url` (optional): as `homepage`.
11. **`license`**: an SPDX license expression of 1 to 512 characters,
    checked against SPDX 2.3.1 Annex D's grammar only: any `idstring` is
    accepted where a license or exception identifier stands, the operators
    `AND`, `OR` and `WITH` must be upper case, and the `DocumentRef-` form is
    refused (it names a document outside the package). Identifiers are
    deliberately not checked against the SPDX License List: the list grows,
    and checking it would make a package's validity depend on when the tool
    was built. `license` states the terms of the package's own content; it
    says nothing about the terms of a rendered output.
12. **`homepage`** (and an author's `url`): an RFC 3986 absolute URI of at
    most 2048 characters, scheme `https` or `http` (lower case), a non-empty
    host, and no userinfo. No command ever fetches it.
13. **`inputs` and `outputs` sit at the root** and are required. Their
    values, and whether either may be empty, are ADR-003's and ADR-005's.
    The mirror shorthand (ADR-005) is expressed inside `outputs`, not as a
    root member. **A sibling ADR that needs another root member adds it to
    this table**; no root member is defined anywhere else.

### D5 — The package's name

1. **`name` is one or more segments joined by `.`.** A segment is one or more
   runs of `[a-z0-9]` joined by single hyphens: it matches
   `[a-z0-9]+(-[a-z0-9]+)*`, so it neither begins nor ends with `-` and never
   holds `--`. A segment is at most 63 characters; the whole name 1 to 128.
2. **The leading segment is not a Windows device name**: `con`, `prn`,
   `aux`, `nul`, `com1`–`com9`, `lpt1`–`lpt9`. `con.metais` and
   `nul.reports.metais` are device names on Windows, and nothing in the format
   precludes Windows (R12 item 2).
3. **Lower case only, ASCII only**, so a name is a portable file-name stem
   with no case-only collision, and two names compare byte for byte.
4. **A package published beyond its author should begin with a domain its
   publisher controls, reversed** (`org.example.vpat-acr`); a package used
   only where it is written may be a single segment (`status-widget`). METAIS
   has no registry, so no tool can check uniqueness or ownership; the
   specification states the convention, and the digest (ADR-001) is what
   identifies content.
5. **A release is identified by `name` and `version` together; its bytes by
   the package digest.** These are the identity facts provenance may record
   (ADR-004): the "package id" there is `name`.

### D6 — The package's version

1. **`version` is a Semantic Versioning 2.0.0 version**, matching the
   grammar semver.org publishes, of at most 128 characters. Pre-release
   (`2.1.0-rc.1`) and build metadata (`2.1.0+20261018`) are allowed. A
   leading `v` is refused with "write `1.2.0`, not `v1.2.0`". In YAML,
   `version: 1.0` is the number 1 (D2 item 9) and is refused as not a
   string, with the fix stated.
2. **The tool never compares one package's version with another's.** v1 has
   no registry, no dependency between packages, and no persisted state, so
   nothing needs precedence; `version` is identity and information. Two
   versions differing only in build metadata are different strings and the
   tool treats them as different releases.
3. **What a version change promises is defined against the package's
   contract**, which is: the input sets it accepts, the output paths it
   writes for them, and the content of those outputs. The specification
   states, as guidance an author follows and no tool can enforce:
   - **major** when some input set the previous version accepted is refused,
     or an output path it wrote for an accepted input set is no longer
     written;
   - **minor** when every input set the previous version accepted is still
     accepted and writes at least the same output paths — new optional
     inputs or fields, new outputs, wider enumerations;
   - **patch** when the accepted input sets and the output paths are
     unchanged and only output content changes.
4. **No tool enforces item 3**, because enforcing it needs the previous
   version, which a stateless tool without a registry does not have. Whether
   `check` should accept an earlier package to compare against is not
   decided here (see *What this does not decide*).

### D7 — The specification version and what it promises

1. **`metais` is a string `MAJOR.MINOR`**, each part `0` or a decimal without
   a leading zero, at most four digits (`^(0|[1-9][0-9]{0,3})\.(0|[1-9][0-9]{0,3})$`).
   v1 is `"1.0"`. There is no patch part and no pre-release part: a package
   names a set of rules, and an editorial revision of the specification text
   that changes nothing a program can observe does not change that set
   (how the document labels its own revisions is ADR-011's). **A number is
   refused**, even where its value looks right: in YAML `metais: 1.0` is the
   number 1, and `1.10` would be 1.1, so the finding quotes the source text
   and says to write `metais: "1.0"`.
2. **A tool implements a set of specification versions, and within each
   major it implements every minor from `.0` to its highest.** A tool that
   implements `1.3` implements `1.0`, `1.1` and `1.2`.
3. **A tool processes a package under exactly the rules of the version the
   package declares**: that minor's manifest schema (D10), that minor's input
   forms, keywords, filters, tags, variables, transforms and findings. The
   specification marks every addition with the minor that introduced it
   (ADR-011), and the tool carries the same table.
4. **What a minor may add**: only constructs that every earlier minor of the
   same major refuses. A new manifest member, a new enumeration value, a new
   input form, a new filter, tag or variable member, a new transform or
   option, a new provenance syntax, a new warning — each is something an
   earlier minor rejects as unknown, so adding it changes no package that
   minor accepts.
5. **What a minor never does**: change what an accepted construct means;
   remove one; change a default; refuse, or render differently, any package
   and input set an earlier minor of the same major accepts; give meaning to
   a JSON Schema keyword (unknown keywords are annotations in JSON Schema
   2020-12, so a validator would silently ignore it); or begin asserting a
   `format` an earlier minor left as an annotation. Anything in this list
   waits for a new major.
6. **The promise this buys is stronger than R15 item 4's minimum: changing a
   package's `metais` from `1.m` to any later `1.k`, and nothing else,
   changes neither its validity nor any byte it renders, for any input set.**
   (It may add warnings, item 4.) An author can raise the declared minor
   without re-testing a single output.
7. **A package declaring a major the tool does not implement is refused**:
   "this package targets METAIS 2.0; this metais implements 1.0, 1.1; use a
   metais release that implements 2.0". The finding carries the declared
   version and the implemented list in its JSON form.
8. **A package declaring a minor newer than the tool's highest in that major
   is refused** the same way: "this package targets METAIS 1.3; this metais
   implements 1.0 to 1.1; use a metais release that implements 1.3". The tool
   does not try to render it on the chance that it uses nothing newer: an
   older tool cannot know what a newer minor added, so it could never tell a
   package it renders correctly from one it renders wrongly, and R14 item 3
   says refuse rather than approximate.
9. **A package that uses a construct added after the minor it declares is
   refused**, naming the construct and the minor that added it: "the filter
   `foo` was added in METAIS 1.2; this package declares 1.1 — declare
   `metais: \"1.2\"` or remove it". This keeps the declaration truthful, so a
   tool implementing only the declared minor renders the package.
10. **`check` reports the lowest minor the package needs**, from the same
    table, and warns when the declared minor is higher: "this package uses
    nothing newer than METAIS 1.0; declaring 1.0 lets every 1.x tool render
    it". `init` declares the highest version the tool implements.
11. **A missing `metais` member is refused** ("add `metais: \"1.0\"`"); a
    string not matching item 1 is refused, naming the grammar.
12. **There is no `0.x`.** The specification starts at `1.0`; a draft of a
    future minor is not nameable by a released package.

### D8 — How the tool states what it implements

1. **The tool reports its own version and the specification versions it
   implements**, on request and as JSON. The command or flag that does so is
   ADR-008's; its content is:

   ```json
   {
     "tool": { "name": "metais", "version": "1.0.0" },
     "specification": { "implements": ["1.0"] }
   }
   ```

   inside ADR-008's output envelope. `implements` lists every implemented
   version explicitly, ascending by major then minor, so a caller tests
   membership without knowing D7 item 2's rule. `tool.version` is a semantic
   version; how the tool is versioned is ADR-010's.
2. **Every refusal under D7 items 7 and 8 carries the same list**, so a
   caller learns what the tool supports from the failure itself.
3. **The package summary (ADR-008) shows the declared `metais`** and whether
   this tool implements it.
4. **The tool's version is independent of the specification's.** A tool
   release may implement a new minor without a new tool major.

### D9 — The canonical form

1. **`.compiled/manifest.json` is the RFC 8785 (JCS) serialization of the
   manifest's data model** (D4 item 1): UTF-8, no byte order mark, no
   whitespace between tokens, members sorted by UTF-16 code units at every
   depth, array order kept, strings escaped exactly as RFC 8785 §3.2.2.2
   says, numbers as ECMAScript formats them, and **no trailing newline**.
2. **It is the data model and nothing more.** Optional members the author
   omitted stay omitted; no default is materialized; no member is added.
   The canonical form therefore depends only on what the author wrote, never
   on the tool or the specification minor that packed it.
3. **The profiles make every manifest canonicalizable**: D2 and D3 guarantee
   unique names, valid Unicode and finite doubles, which are RFC 8785's
   preconditions.
4. **When the tool reads a packed package**, after ADR-001's digest
   verification, it parses the verbatim source manifest (D1 item 4) under its
   profile, canonicalizes it, and compares the result byte for byte with
   `.compiled/manifest.json`. **A difference is refused**, as a digest
   mismatch is: otherwise a hostile package could show a person reading
   `metais.yaml` one manifest and render from another. The same comparison
   proves `.compiled/manifest.json` is itself canonical.
5. **Readers of a packed package treat `.compiled/manifest.json` as the
   manifest**; a consumer that is not the reference tool needs only a JSON
   parser.
6. **This canonical form is offered to every JSON file under `.compiled/`**
   (the combined input schema, the digest file, the references index), so
   one rule makes all of them byte-stable (see *Interfaces*).

### D10 — The manifest's JSON Schema

1. **One schema per specification minor**, each a JSON Schema 2020-12
   document (`"$schema": "https://json-schema.org/draft/2020-12/schema"`)
   that accepts exactly the data models D4 accepts for that minor: every
   object it defines is closed (`additionalProperties: false`, or
   `unevaluatedProperties: false` where subschemas compose), and `metais` is
   `{"const": "1.0"}` in the `1.0` schema.
2. **Its `$id` is `https://metais.org/schemas/<MAJOR>.<MINOR>/manifest.schema.json`**
   — `https://metais.org/schemas/1.0/manifest.schema.json` for v1 — subject to
   the domain ruling (*Proposed rulings* 15). The `$id` is an identifier; it
   need not resolve for any tool to work, and no tool fetches it.
3. **The `inputs` and `outputs` sections are `$defs` of this one schema**
   (`#/$defs/inputs`, `#/$defs/outputs`), whose content ADR-003 and ADR-005
   write. The published file is one self-contained document with no external
   `$ref`.
4. **Every property carries `title` and `description`**, so a finding about
   the manifest gives the field's title and description exactly as a finding
   about an input does (R6 item 2).
5. **The tool embeds each implemented minor's schema**, byte-identical to the
   published file (a test compares them), and validates every manifest with
   the schema of the minor it declares, after D7 items 7, 8 and 11 have
   passed.
6. **Rules the schema cannot express are stated in the specification's prose
   and checked by the tool as manifest rules**, each with its own finding
   code: the profile rules (D2, D3, which act before a data model exists),
   D7 items 7 to 11, the SPDX grammar (parenthesized expressions are not a
   regular language), and D9 item 4. ADR-003, ADR-005 and ADR-007 add their
   own.
7. **Patterns in the schema avoid lookaround**, so any ECMA-262 or RE2-class
   regular-expression engine evaluates them alike; D5 item 2 is expressed as
   `"not": {"pattern": "^(con|prn|aux|nul|com[1-9]|lpt[1-9])(\\.|$)"}`.

### D11 — The order a manifest is processed in

1. Locate the manifest (D1).
2. Bound it (ADR-009's manifest size limit) before reading it.
3. Parse under its profile (D2 or D3). A syntax error ends processing with
   that finding; otherwise every profile violation is reported, and any one
   ends processing.
4. Gate on `metais` (D7 items 7, 8, 11). A refusal ends processing.
5. Validate against the declared minor's schema (D10), reporting every
   violation.
6. Check the manifest rules (D10 item 6), reporting every violation, in the
   same pass as step 5.
7. Hand the data model to ADR-003, ADR-005 and ADR-007's rules.
8. For a packed package, D9 item 4 runs after ADR-001's digest verification
   and before step 4.

Each step gates the next because a later step's findings would be about a
manifest the earlier step has already shown to be something else; within a
step, every finding is reported (ADR-006).

## Interfaces with sibling ADRs

**ADR-001 (container and layout).** Relies on: the layout reserving
`metais.yaml` and `metais.json` at the root (exactly one present, D1), and
`.compiled/manifest.json`; the source manifest stored verbatim in a packed
package (D1 item 4); the digest file covering `.compiled/manifest.json` and
the source manifest; digest verification running before D9 item 4. Promises:
the canonical form (D9) for any JSON file ADR-001 writes under `.compiled/`,
if ADR-001 adopts it; `name` (D5) as a portable file-name stem if `pack`
derives the package file name from it.

**ADR-003 (data specification).** Relies on: `inputs` being the root member
of D4's table, its value defined as `#/$defs/inputs` in D10's schema, every
object in it closed (D4 item 4), and nothing in it giving meaning to a JSON
Schema keyword in a later minor (D7 item 5); the set of asserted `format`s
fixed per major (D7 item 5). Promises: D2's YAML profile and D4's data model,
which ADR-003 may adopt for YAML inputs, frontmatter and YAML-written
schemas so that one YAML reading holds across the package; if ADR-003 needs
schema-directed coercion for YAML data (as it has for CSV), that is its
decision, and the manifest does not use it.

**ADR-004 (template environment).** Relies on: the `package` variable
exposing `metais`, `name`, `version`, `title`, `description`, `guide`,
`authors`, `license` and `homepage` with the values of the canonical manifest
(an omitted optional member is `nil`), and whatever digest member ADR-004
defines; provenance's "package id" recording `name` (D5 item 5); **an
undefined variable, an undefined member of `package`, an unknown filter and
an unknown tag each refused**, not rendered as nothing — D7 item 4 depends on
it, because a minor can add a member or a filter only if `1.0` refuses its
use.

**ADR-005 (outputs).** Relies on: `outputs` being the root member of D4's
table, its value defined as `#/$defs/outputs`, every object in it closed; the
mirror shorthand expressed inside `outputs`; transforms, transform options
and escape modes as closed enumerations a minor can extend (D7 item 4).
Promises: D5 item 2 uses the same device-name list as ADR-005's portable-path
rule; the consistency pass should check the two lists match.

**ADR-006 (findings).** Relies on: a stable code for each refusal in D1, D2,
D3, D4, D7, D9 and D10 item 6; locations carrying both line/column (from the
source) and JSON Pointer (into the data model); the unknown-member finding
able to name a close defined member. Promises: every manifest finding has a
field title and description from D10 item 4 where a field is involved.

**ADR-007 (authoring).** Relies on: `init` writing `metais.yaml` with
`metais` quoted and set to the tool's highest implemented version, a literal
block `guide`, and every required member; `check` reporting the lowest needed
minor and warning when the declared one is higher (D7 item 10); `check`
reporting a relative link in `guide` whose target is not a package entry;
`pack` storing the source manifest verbatim and writing D9's canonical form;
`unpack` restoring the verbatim source manifest.

**ADR-008 (command line).** Relies on: a command or flag reporting D8 item 1's
content in its envelope; the summary showing `title`, `description`, `guide`
in full, `name`, `version`, `metais` and whether it is implemented.

**ADR-009 (security).** Relies on: a manifest size limit and a nesting-depth
limit applied before and during parsing (D11 item 2). Promises: D2 item 4
removes alias expansion, so the YAML-bomb defence for the manifest reduces to
size and depth.

**ADR-010 (implementation).** Relies on: a YAML parser that exposes a node
tree with node kind (including alias), style (plain, quoted, block, explicit
tag), anchor, value text, line and column, and that reports duplicate keys or
lets the tool detect them — `go.yaml.in/yaml` qualifies on these properties,
but its own resolution must not be used (it reads `0777` as octal); a JSON
parser that detects duplicate names and reports positions; an RFC 8785
serializer (`github.com/gowebpki/jcs`, or METAIS's own — `encoding/json`'s
output is not JCS); the tool's own semantic version for D8.

**ADR-011 (specification and conformance).** Relies on: publishing each
minor's manifest schema at `spec/schemas/<MAJOR>.<MINOR>/manifest.schema.json`
(or wherever ADR-011 places it, with the `$id` of D10 item 2 unchanged);
marking every normative addition with the minor that introduced it (D7
item 3); a vector format that can express "this package, declared at minor
m, is refused with code X" and the canonical bytes of a manifest.

## Conformance

Vectors are needed for each of these observable rules (format per ADR-011):

- **Manifest file**: `metais.yaml` alone; `metais.json` alone; both; neither;
  `metais.yml`; `Metais.yaml`.
- **YAML profile**: a byte order mark; invalid UTF-8; a second document;
  `%YAML 1.2`; an anchor; an alias; `!!str`; `!`; a `<<` key; a complex key; an
  empty key; `a` and `"a"` duplicated; one vector per row of D2 item 9's
  table, including `no` → `"no"`, `2026-10-18` → string, `1.0` → `1`, `012`,
  `+1`, `0o17`, `0x1F`, `.5`, `1.`, `.inf`, `.nan`, `~`, `True`, `NULL`, and an
  empty value → null.
- **JSON profile**: a byte order mark; a trailing comma; a comment; a
  duplicate name; a lone surrogate escape; `NaN`.
- **Numbers**: `1e400`; `9007199254740992` (refused) and `9007199254740991`
  (accepted); `-0` canonicalized to `0`.
- **Structure**: each required member missing; an unknown root member; an
  unknown author member; `null` for an optional member.
- **Fields**: `title` and `description` at and beyond their bounds and with
  each forbidden character class; `guide` at 16 384 and 16 385 characters;
  each `name` rule (a `--`, a trailing `-`, upper case, `con`,
  `con.reports`, `console` accepted, a 64-character segment); `version` as
  `v1.0.0`, `1.0`, `01.0.0`, with pre-release and with build metadata;
  `license` valid, with lower-case `and`, with `DocumentRef-`, unbalanced
  parentheses; `homepage` with userinfo, with `ftp:`, relative.
- **Specification version**: `metais` missing; a number; `"1.00"`; `"1"`;
  `"2.0"` (unknown major, the finding listing the implemented versions);
  `"1.99"` (newer minor); and, from `1.1` onward, a construct used under a
  declaration older than its minor, and a package whose declaration is raised
  from `1.0` to the new minor rendering byte-identical outputs.
- **Canonical form**: a manifest whose `.compiled/manifest.json` bytes are
  given exactly, covering member sorting with characters above U+FFFF beside
  characters in U+E000–U+FFFF, every escape class, non-ASCII written as UTF-8,
  and number forms; a packed package whose source manifest and compiled
  manifest disagree (refused); one whose compiled manifest is valid JSON but
  not canonical (refused).
- **Supported versions**: the version report's JSON content.

## Alternatives rejected

### The YAML 1.2 Core Schema as it stands

It is the YAML default, but it silently turns `012` into 12, accepts `0o17`,
`0x1F`, `.inf`, `.nan` and `~`, and offers `True` and `NULL` beside `true`
and `null`. Each is a value an author did not mean or a spelling a second
implementation might resolve differently. D2 keeps Core's string default and
JSON's literals, and refuses the forms in between rather than guessing.

### YAML 1.2's JSON Schema

It removes every ambiguity by requiring every string to be quoted. Writing
`title: "Status widget"` and quoting every enum value is a cost on every line
for every author, to prevent hazards D2 refuses at a handful of spellings.

### Resolving YAML scalars by the manifest schema

Reading a plain scalar as a string wherever the schema wants a string would
accept `metais: 1.0`. It ties parsing to validation, makes the same text mean
different things in different places, and gives YAML and JSON sources
different rules, since JSON's `1.0` is a number whatever the schema says. The
finding that says "quote it" costs one edit, once.

### Anchors, aliases and merge keys for repeated output declarations

They save repetition in a long `outputs` list, but they are the alias bomb,
they vanish in the canonical form so a reader of `.compiled/manifest.json`
cannot see them, and `<<` is not YAML 1.2. `for_each` and the mirror shorthand
(ADR-005) are the format's answers to repetition.

### JSON accepted only as the content of `metais.yaml`

YAML 1.2 is close to a superset of JSON, but not exactly (the 1024-character
implicit-key limit), and an author whose tool emits JSON expects a `.json`
name and a JSON parser's errors. Two names, each with its own profile and one
data model, is clearer than one name with two languages behind it.

### Accepting `metais.yml`

A second spelling is a second thing to document and to look for, and a
package containing both would need a rule. The finding names the one
spelling.

### An extension prefix (`x-`) or a free-form `annotations` member

An extension member no conforming tool interprets is harmless only until one
tool interprets it, and then packages depend on that tool — exactly what
METAIS refuses. Refusing unknown members also catches typos (`ouputs`), and
is what lets a minor add a member safely (D7 item 4). If a later minor finds
a real need for author-defined metadata, it can add a member then; a member
added in `1.0` could never be taken away.

### A number for `metais`, or a three-part specification version

Accepting `metais: 1.0` as a number breaks at `1.10`, which YAML reads as
1.1. A three-part version (`1.0.0`) would always be a string in YAML, but a
package names a set of rules, and a patch-level change to the rules is not
something this format has; editorial revisions are the document's affair.

### Rendering a newer-minor package when it appears to use nothing newer

It would let an old tool render more packages, and it is sound only if every
minor forever adds solely into refused space and the old tool's notion of
"refused" exactly matches. A single slip — a newly asserted `format`, an
erratum — and an old tool renders a newer package wrongly with no finding.
D7 item 10 recovers the reach honestly: `check` tells the author the lowest
minor the package needs.

### The guide as a file, or as either a file or inline text

A file could be outlined and searched, but the PRD puts the guide in the
manifest, the minimal package would gain a file, and a member that is
sometimes a path and sometimes Markdown is ambiguous on its face. The bound
in D4 item 9 keeps the guide to what is read first, and long material has its
homes in meaning documents and references.

### A scoped name (`publisher/name`) or a required URI identity

A `/` cannot be part of a file-name stem, and requiring a domain or URI costs
the one-file widget author something for a uniqueness no tool can check
without a registry. Dotted segments allow the reverse-domain convention
without demanding it.

### Checking `license` identifiers against the SPDX License List

It would catch a misspelled identifier, but the list grows, so a package's
validity would depend on the tool's build date, and two conforming tools
built at different times would disagree.

### A pretty-printed canonical form, or Go's `encoding/json` output

Indented, sorted JSON is easier to read, but its rules (indent, key order,
number and string forms) would be METAIS's own to specify and every second
implementation's to reproduce. RFC 8785 is published, specifies numbers and
strings exactly, and has implementations in several languages. A person
reads `metais.yaml`; `.compiled/manifest.json` is for programs and digests.

### A `$schema` member in the manifest for editor completion

It would duplicate `metais` and could disagree with it. `init` can write a
YAML comment that editors read (ADR-007), which carries no meaning to the
format.

## Consequences

- An author meets a handful of YAML refusals (`012`, `~`, aliases,
  `metais: 1.0`) that other tools accept; each finding names the one-edit fix,
  and in exchange every accepted manifest means the same thing in every
  conforming implementation.
- Every JSON file under `.compiled/` can be produced and checked by an
  off-the-shelf RFC 8785 implementation in any language.
- A specification minor is cheap to add only for things `1.0` refuses; a
  change of meaning costs a major. ADR-003, ADR-004 and ADR-005 must refuse
  every unknown construct for this to hold.
- An older tool refuses a package that declares a newer minor even when it
  would have rendered it correctly; `check`'s lowest-minor warning is what
  keeps authors from declaring more than they need.
- The tool carries a per-minor table of additions from `1.1` onward, and the
  conformance suite carries a vector per addition.
- `$id`s are permanent; the domain they name must be held for the life of the
  format.

## What this does not decide

- The contents of `inputs` (ADR-003) and `outputs` (ADR-005).
- The digest file and its verification (ADR-001), the limits applied to the
  manifest (ADR-009), and the finding codes and wording (ADR-006).
- The command or flag that prints D8's report, and the summary's layout
  (ADR-008).
- How the reference tool itself is versioned and released (ADR-010), and
  whether a tool keeps implementing `1.x` once `2.0` exists (the `2.0`
  decision).
- How the specification document labels its own editorial revisions, and
  where under `spec/` the schemas are published (ADR-011).
- Whether `check` should compare a package against an earlier release of
  itself to enforce D6 item 3; v1 does not.
- Hosting anything at the `$id` URLs.

## Proposed rulings

1. **`metais.json` is accepted beside `metais.yaml`, exactly one present;
   `metais.yml` and case variants are refused** (D1). Lean: yes. Why: the
   PRD's "YAML or JSON" with a JSON parser behind a JSON name; YAML is not
   quite a JSON superset. Alternative: JSON only as `metais.yaml`'s content.
2. **The YAML profile resolves plain scalars by D2 item 9's table**: JSON's
   literals and number grammar, refusal for the other Core forms, strings
   otherwise. Lean: yes. Why: no silent wrong typing, unquoted prose stays
   cheap. Alternatives: Core as it stands; YAML's JSON Schema; schema-directed
   resolution.
3. **Anchors, aliases, tags, merge keys, directives, multiple documents, a
   byte order mark and complex keys are refused** (D2). Lean: yes. Why: one
   meaning, no alias bomb, nothing the canonical form loses.
4. **Mapping keys are their scalar text, never type-resolved** (D2 item 7).
   Lean: yes. Why: JSON keys are strings; `on:` stays `"on"`.
5. **Unknown members are refused everywhere; no `x-` prefix in 1.0** (D4
   item 4). Lean: yes. Why: typos caught, minors can add safely, no tool-only
   dialects. Alternative: an `x-` prefix ignored by tools.
6. **`metais` must be a string; a number is refused with the fix** (D7
   item 1). Lean: yes. Why: `1.10` reads as 1.1.
7. **`name` is dotted lower-case segments, reverse-domain recommended, a
   leading Windows device name refused** (D5). Lean: yes. Alternative: a
   single slug; a required domain.
8. **`guide` is required, inline CommonMark, at most 16 384 characters**
   (D4 item 9). Lean: yes. Alternatives: optional; a file; no bound.
9. **`title` (≤ 200) and `description` (≤ 1000) are required** with D4
   item 6's text rules. Lean: yes.
10. **`authors`, `license` and `homepage` are the only optional metadata;
    `license` is checked by grammar, not by the SPDX list** (D4 items 10 to
    12). Lean: yes. Alternative: more metadata members (`repository`,
    `keywords`) — a minor can add them.
11. **The canonical form is RFC 8785 with no trailing newline; integers
    beyond 2⁵³ − 1 are refused** (D4 item 3, D9). Lean: yes. Alternative: a
    METAIS-specified pretty form.
12. **A packed package's source manifest must canonicalize to
    `.compiled/manifest.json`, or the package is refused** (D9 item 4). Lean:
    yes. Why: what a person reads is what renders.
13. **A tool refuses a package declaring a minor newer than it implements,
    and a package using a construct newer than its declared minor; `check`
    reports the lowest minor needed** (D7 items 8 to 10). Lean: yes.
    Alternative: render a newer-minor package if nothing newer appears used.
14. **R15 item 4 is read strongly: raising a package's declared minor
    changes neither its validity nor its output** (D7 item 6). Lean: yes.
    Why: authors can raise it without re-testing; it forbids minors that gate
    new validation on the declaration.
15. **The manifest schema's `$id` is
    `https://metais.org/schemas/<MAJOR>.<MINOR>/manifest.schema.json`, and
    `metais.org` is registered for the format** (D10 item 2). Lean: register
    it before the specification is published. Why: a neutral domain outlives
    any one tool or organization; the `.org` registry reported it unregistered
    on 2026-10-18. Alternative: a path under a domain the project's creator
    already holds.
16. **Package-version semantics are specified against the package's contract
    (accepted input sets, output paths, content) as unenforced guidance**
    (D6 item 3). Lean: yes. Alternative: say nothing beyond "semver".
17. **Pre-release and build metadata are allowed in `version`; the tool
    never orders versions** (D6 items 1, 2). Lean: yes.

## References

- YAML Ain't Markup Language version 1.2.2 (§3.2.1.1, §3.3, §5.2, §6.8.1,
  §7.4.2, §8.2.2, §10.2.2, §10.3.2): https://yaml.org/spec/1.2.2/
- Merge Key Language-Independent Type for YAML Version 1.1, working draft
  2005-01-18: https://yaml.org/type/merge.html
- `go.yaml.in/yaml` README (project status, v3 compatibility notes, v4):
  https://github.com/yaml/go-yaml
- `go-yaml/yaml`, archived: https://github.com/go-yaml/yaml
- RFC 8785, JSON Canonicalization Scheme (JCS):
  https://www.rfc-editor.org/rfc/rfc8785
- RFC 7493, The I-JSON Message Format: https://www.rfc-editor.org/rfc/rfc7493
- RFC 8259, The JavaScript Object Notation (JSON) Data Interchange Format:
  https://www.rfc-editor.org/rfc/rfc8259
- RFC 3986, Uniform Resource Identifier (URI): Generic Syntax:
  https://www.rfc-editor.org/rfc/rfc3986
- Go `encoding/json` encoder source (float formatting, HTML and U+2028
  escaping): https://github.com/golang/go/blob/master/src/encoding/json/encode.go
- `github.com/gowebpki/jcs`, an RFC 8785 implementation in Go:
  https://github.com/gowebpki/jcs
- Semantic Versioning 2.0.0: https://semver.org/spec/v2.0.0.html
- JSON Schema 2020-12 Core (draft-bhutton-json-schema-01; `$id`, unknown
  keywords): https://json-schema.org/draft/2020-12/json-schema-core
- SPDX Specification 2.3.1, Annex D, SPDX license expressions:
  https://spdx.github.io/spdx-spec/v2.3.1/SPDX-license-expressions/
- Microsoft, Naming Files, Paths, and Namespaces (reserved device names):
  https://learn.microsoft.com/en-us/windows/win32/fileio/naming-a-file
- CommonMark Spec: https://spec.commonmark.org/
- `whois metais.org` and `whois metais.io`, run 2026-10-18 (the `.org`
  registry's "Domain not found"; `metais.io` created 2023-04-07).

## Implementation log

Nothing implemented.
