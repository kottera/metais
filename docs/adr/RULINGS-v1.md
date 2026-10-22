# METAIS v1 — the rulings register

**Date:** 2026-10-18
**For:** Ed Funnekotter, to ratify
**Covers:** ADR-001 to ADR-011 after the consistency pass
**Status:** Ratified 2026-10-18 by Ed Funnekotter: every lean, in Tier 1 and Tier 2, is accepted.

This is the one document to read to ratify the v1 ADRs. Tier 1 needs a
judgement; Tier 2 is offered as one batch; "Resolved in this pass" records
each conflict between the ADRs and which one won; "PRD follow-ups" lists
where an ADR narrows PRD-001's wording. Each ADR's own *Proposed rulings*
section carries the full argument; the reference after each item (`PRn`)
points there. Nothing here reopens the brief's *Rulings already made* or
*Ruled OUT*.

## Tier 1 — decisions that need your judgement

**R1-001 — One manifest file name** (ADR-001 D5 item 2 / PR10; ADR-002 D1 /
PR1). Is the manifest always `metais.yaml`, and how is JSON content read?
Options: (a) one name, read under the JSON profile when its first
non-whitespace byte is `{`, else the YAML profile; (b) `metais.yaml` or
`metais.json`, exactly one; (c) one name, JSON parsed as YAML. **Lean: (a).**
One golden path and never two manifests, while JSON still meets a JSON
parser (YAML is not quite a JSON superset).

**R1-002 — `x-` in two scopes** (ADR-002 D4 item 4 / PR5; ADR-003 D3 item 5
/ PR6). Manifest objects are closed with no `x-` member; inside JSON Schema
documents an unknown keyword is refused except one beginning `x-`, carried
as an author annotation. Options: as stated; `x-` allowed in both; refused
in both. **Lean: as stated.** The manifest is METAIS's grammar and must grow
only by minors; schemas are often copied from published sources that use
`x-` annotations, and refusing other unknown keywords still catches typos.

**R1-003 — Unicode version and toolchain** (ADR-010 D1 / PR1; ADR-001 D2;
ADR-005 D12). Specification 1.0 pins Unicode 15.0.0 for every rule (paths,
NFC, case folding, `slug`, case mapping), built with Go 1.26.9. Verified:
`golang.org/x/text` v0.43.0 selects its tables by the compiling toolchain
(`//go:build !go1.27` → 15.0.0), not by its module version. Options: (a)
15.0.0 / Go 1.26; (b) 17.0.0 / Go 1.27 now. **Lean: (a).** Moving later
needs a 17.0.0 minor or METAIS's own 15.0.0 "assigned" table.

**R1-004 — METAIS writes its own Liquid engine** (ADR-010 D4 / PR2; ADR-007
D3 item 1 / PR1). `osteele/liquid` fails object order, `blank`/`empty`, core
tags, host isolation and AST access, all inside the library. Options: (a)
own engine, whose syntax tree `check`'s cross-reference also walks; (b)
`osteele/liquid` plus a second parser for analysis and a narrower dialect.
**Lean: (a)**, strongly. It is the largest build item of the plan.

**R1-005 — The sample-data commands** (ADR-008 D1, D8, D9 / PR1; ADR-007 D9
/ PR19, PR21; ADR-011 D8). Preview is `render --sample` (all of render's
flags); keeping data is `sample --to <dir>` or `sample --example <name>`;
`--set`, `--set-json`, `--inputs-json` are fixed points the generator fills
around; `--data` is refused with `--sample`. Option: one `sample` command
that prints JSON and also renders. **Lean: as stated.**

**R1-006 — `test --update` may delete** (ADR-007 D5 item 5 / PR10; ADR-008
D9 / PR21). It makes `examples/<name>/expected/` exactly the render,
removing stale files there and nowhere else; "never deletes" governs
rendering into a caller's directory. Option: write only, report stale
files. **Lean: delete inside `expected/` only.**

**R1-007 — `NO_COLOR` is the only environment variable read** (ADR-008 D12
/ PR11; ADR-009 D14 / PR11). Terminal detection is an `isatty` call;
`TERM` is not read. Option: also honour `TERM=dumb`. **Lean: `NO_COLOR`
only** — the stricter reading of the key security rule.

**R1-008 — The `$id` domain** (ADR-002 D10 item 2 / PR15; ADR-011 D7).
Schemas are identified as `https://metais.org/schemas/…`; `.org` reported
`metais.org` unregistered on 2026-10-18. Registering is an outward act.
Options: register `metais.org` before the specification is published; use a
path under a domain the project's creator holds; a URN. **Lean: register
`metais.org`**, since `$id`s are permanent.

**R1-009 — Media type and IANA** (ADR-001 D9 / PR9). The type is
`application/vnd.metais+zip`. Options: register with IANA now; register
when the specification is public; no media type. **Lean: name it now,
register when the specification is public** (registration needs a citable
specification; the repository is private).

**R1-010 — Byte identity per tool build** (ADR-001 D4 item 6 / PR11). Go's
deflate output is outside its compatibility promise and changed in Go 1.27,
so packages and zip containers are byte-identical within one build; across
builds and implementations, identity is the package digest and the
entry-by-entry comparison. Options: as stated; vendor a pinned encoder;
stored-only packages. **Lean: as stated.** Needs the PRD follow-ups below.

**R1-011 — No path the caller gave is ever printed** (ADR-009 D15 / PR6;
ADR-008 D11 item 5, D14 item 2 / PR27; ADR-006 D9 item 2 / PR16). Findings,
JSON results and `next` command lines name the package `<package>` and
directories by role, so a harness substitutes its own arguments. Option:
echo caller paths exactly as given, never made absolute. **Lean: never
print** — SC7 forbids a sentinel planted in a path reaching a message.

**R1-012 — Existing output files are never read** (ADR-009 D8 item 4;
ADR-005 D15 / PR10). There is no `unchanged` status: every existing file a
render writes is replaced, so its timestamp moves. Option: compare bytes and
leave identical files alone (kinder to build watchers, but an equality
oracle over a file the supplier cannot read). **Lean: never read.**

**R1-013 — Whole-package verification at every open** (ADR-001 D7, D8 /
PR6, PR7; ADR-009 D5 items 8 and 9 / PR10). Every command hashes every entry
and re-derives the three `.compiled/` files before printing anything; no
skip flag. Option: verify only the entries a command reads (cheaper `read`
and `ls`). **Lean: whole package** — a reported digest is always one the
tool established.

**R1-014 — A second end-record signature is refused** (ADR-009 D5 item 2 /
PR16; ADR-001 D8 step 2). No other `PK\x05\x06` may lie in a package's final
65,557 bytes. Cost: a stored nested zip near the end of a small package is
refused (`pack` reports it). Options: (a) as stated; (b) refuse only a
second signature that parses as a self-consistent end record. **Lean: (a)
for v1** — (b) is a relaxation of a security rule and is yours to choose.

**R1-015 — Limits are never raised, and their defaults are exact** (ADR-009
D3, D4 / PR1, PR3; ADR-006 D8 item 5 / PR14). Includes `findings` at
100,000 listed (all counted), `path-length` 512 bytes, `package-size` 256
MiB. Options: allow raising data limits to a ceiling; ratify defaults only
after calibration. **Lean: never raised; adopt the defaults now and amend
from calibration.**

**R1-016 — Unasserted formats and unknown keywords are refused** (ADR-003
D3 item 5, D4 / PR5). Every `format` used is asserted from a closed list;
`idn-*`, `iri*`, `regex` and unknown formats are refused. Published schemas
using them need editing. Options: annotation-only `format`; assert known and
ignore the rest. **Lean: refuse.**

**R1-017 — Strict templates** (ADR-004 D6, D4 item 4 / PR6, PR9, PR10). A
declared-but-absent member is `nil`; an undeclared one is an error; a
property of `nil` is an error (`{% if a and a.b %}` guards); object output
is refused; unknown filters are refused even in dead branches. Option: `nil`
propagation and lax names. **Lean: strict** — it is what lets a minor add a
filter safely and makes typos findings.

**R1-018 — Version compatibility** (ADR-002 D7 / PR13, PR14). A tool
refuses a package declaring a newer minor than it implements, and one using
a construct newer than its declared minor; raising a declared minor never
changes validity or output. Option: render a newer-minor package that
appears to use nothing new. **Lean: refuse; strong reading of R15 item 4.**

**R1-019 — The exit-code table is normative** (ADR-006 D12 / PR5, PR6). 0
success, 1 content, 2 internal (Go's crash status), 3 usage, 4 package
refused, 5 destination; a package's own defect exits 4 under consumer
commands and 1 under authoring commands. Option: 2 for usage, as most tools.
**Lean: as stated.**

**R1-020 — The command line is not part of the specification** (ADR-008 PR7;
ADR-011 D8, D11 / PR6, PR15). Vectors are written against abstract
operations; the reference tool is driven by a `cli` adapter, other
implementations by a documented driver protocol. Option: specify the CLI and
write vectors as argv. **Lean: as stated.**

**R1-021 — Findings never echo input strings** (ADR-006 D9 item 4 / PR8).
Strings are described by type and length and found by location; numbers,
booleans and `null` may be echoed except under `writeOnly`. Option: echo
strings up to a length or behind a flag. **Lean: never** — findings land in
logs and models the input never should.

**R1-022 — Licence of the specification and vectors** (ADR-011 PR16).
Options: Apache-2.0, like the repository; CC BY 4.0 for the text. **Lean:
Apache-2.0.** Outward-facing once the specification is published.

**R1-023 — One strict portable-path rule** (ADR-005 D12 / PR6; ADR-001 D2 /
PR4; ADR-009 D8 item 6 / PR9). For entry names and output paths alike:
categories L, M, N, P, S plus space (no format, bidirectional or private-use
characters), Windows device names (with `COM0`/`LPT0`) and reserved
characters refused on every platform, `.git` refused, NFC refused not
normalized, collisions by full case folding over paths and directory
prefixes. `aux.c` and `Con.md` are refused everywhere. **Lean: as stated.**

**R1-024 — Empty CSV cells** (ADR-003 D11 item 5 / PR11). An unquoted empty
field is absent (so its default applies and a template sees `nil`); `""` is
the empty string. Option: every empty field is the empty string, which
Liquid treats as true. **Lean: as stated** — R5 item 4 needs it for
spreadsheet exports.

## Tier 2 — technical rulings to accept as a batch

Lean is "yes" (adopt as written) unless stated.

### ADR-001 — the container and its layout

| ID | Ruling | Lean |
|---|---|---|
| R2-001 | PR1 — the reading profile equals the writing profile | yes |
| R2-002 | PR2 — no zip64 in major 1; ceilings 65,534 entries and 0xFFFFFFFE bytes | yes |
| R2-003 | PR3 — no directory entries; an empty source directory is a finding | yes |
| R2-004 | PR5 — package digest = SHA-256 of the `sha256sum` list of every entry but `digests.json`; a source directory has its package's digest | yes |
| R2-005 | PR8 — closed top level; in a source directory dot-names and `*.metais` ignored, other names warned and not packed | yes |
| R2-006 | PR12 — fixed header values (0x0314, 0x81A40000, 1980-01-01, version 10/20, `auto` at level 9) | yes |

### ADR-002 — the manifest and versioning

| ID | Ruling | Lean |
|---|---|---|
| R2-007 | PR2 — plain scalars resolve by D2 item 9's table (JSON literals and numbers, other Core forms refused) | yes |
| R2-008 | PR3 — anchors, aliases, tags, merge keys, directives, multiple documents, BOM, complex keys refused | yes |
| R2-009 | PR4 — mapping keys are their scalar text | yes |
| R2-010 | PR6 — `metais` must be a string | yes |
| R2-011 | PR7 — `name` is dotted lower-case segments, reverse-domain recommended, no leading device name | yes |
| R2-012 | PR8 — `guide` required, inline CommonMark, ≤ 16,384 characters | yes |
| R2-013 | PR9 — `title` ≤ 200 and `description` ≤ 1000 required | yes |
| R2-014 | PR10 — `authors`, `license` (SPDX grammar only), `homepage` the only optional metadata | yes |
| R2-015 | PR11 — canonical form RFC 8785, no trailing newline; integers beyond 2⁵³−1 refused | yes |
| R2-016 | PR12 — a source manifest must canonicalize to `.compiled/manifest.json` | yes |
| R2-017 | PR16 — package-version semantics as unenforced guidance | yes |
| R2-018 | PR17 — pre-release and build metadata allowed; versions never ordered | yes |
| R2-019 | D4 table — `input_constraints` is a root member (for ADR-003 D6) | yes |

### ADR-003 — the data specification

| ID | Ruling | Lean |
|---|---|---|
| R2-020 | PR1 — `inputs` an ordered list; `title`, `schema` required; `forms` default `[json, yaml]` | yes |
| R2-021 | PR2 — input names `^[a-z][a-z0-9_]{0,63}$`, Liquid words reserved | yes |
| R2-022 | PR3 — slot title and description live in the manifest | yes |
| R2-023 | PR4 — `input_constraints`, a cross-input schema | yes |
| R2-024 | PR7 — "inlined" means bundled into one self-contained document | yes |
| R2-025 | PR8 — `metais:///<path>` retrieval URIs; an `$id` file reached only by its `$id` | yes |
| R2-026 | PR9 — one number type; big integers written as integers refused | yes |
| R2-027 | PR10 — line breaks normalized in CSV, text and Markdown; BOM removed; text's final LF removed | yes |
| R2-028 | PR12 — schema-directed coercion; failing text stays a string | yes |
| R2-029 | PR13 — Markdown form is an object; a prose `.md` for a text input is text | yes |
| R2-030 | PR14 — collections flat, NFC names in byte order, unique stems, never one file | yes |
| R2-031 | PR15 — a data-directory entry naming no input is an error; links refused | yes |
| R2-032 | PR16 — `--set` pointer + coerced text, `--set-json`, repeatable `--inputs-json` | yes |
| R2-033 | PR17 — precedence: data dir, every `--inputs-json`, every path set | yes |
| R2-034 | PR18 — defaults through `properties`/`items`/`prefixItems`/`$ref` only | yes |
| R2-035 | PR19 — raw bytes only for untouched files; absent raw refuses the render | yes |
| R2-036 | PR20 — input digest = SHA-256 of canonical JSON after defaults | yes |
| R2-037 | PR21 — constraints skipped when an input failed to parse, with an `info` finding | yes |
| R2-038 | PR22 — "every error" excludes the boolean-only contexts (`not`, `if`, untried branches) | yes |

### ADR-004 — the template environment

| ID | Ruling | Lean |
|---|---|---|
| R2-039 | PR1 — gem 5.14.0, `strict2`, strict variables and filters, `render!` | yes |
| R2-040 | PR2 — exclude `self` and theme-platform filters | yes |
| R2-041 | PR3 — include `liquid`, `echo`, `#`, `doc`, `ifchanged`, counters, `cycle`, `tablerow` | yes |
| R2-042 | PR4 — pin the blank-body rule | yes |
| R2-043 | PR5 — `reversed` only after the collection; offset, limit, reversed order | yes |
| R2-044 | PR7 — integral data numbers are integers; Ruby float text; exact decimal arithmetic | yes |
| R2-045 | PR8 — RFC 8785 member order for every object | yes |
| R2-046 | PR11 — globals `now`, `output`, `fan_out` with `previous`/`next` | yes |
| R2-047 | PR12 — globals visible in partials | yes |
| R2-048 | PR13 — partial names are literal full paths under `templates/` | yes |
| R2-049 | PR14 — template-scoped bindings cannot take a global's name | yes |
| R2-050 | PR15 — autoescape scope; `safe` last only | yes |
| R2-051 | PR16 — simple case mapping, Unicode 15.0.0 | yes |
| R2-052 | PR17 — `date`: RFC 3339 or Unix seconds, own offset, `%Z` as `UTC` or offset | yes |
| R2-053 | PR18–PR23 — `json` (RFC 8785, `indent:`), `yaml`, `csv`, `markdown_table`, `slug`, `json_script` as specified | yes |
| R2-054 | PR24 — no `sha256` filter in 1.0 | no filter |
| R2-055 | PR25 — `embed` by input or collection item; BOM dropped; never escaped | yes |
| R2-056 | PR26 — `asset` modes `path`, `contents`, `base64` | yes |
| R2-057 | PR27 — provenance: one line, closed fields, style by kind, `style:` | yes |
| R2-058 | PR28 — provenance input digest is ADR-003's | yes |
| R2-059 | PR29 — `test` renders with a fixed identity; `render` and vectors record the real one | yes |
| R2-060 | PR30 — every output attempted; each reports its first render error | yes |
| R2-061 | PR31 — template bytes kept as written; U+0000 refused | yes |
| R2-062 | PR32 — `sources` exposes item file names, indexed by position | yes |
| R2-063 | PR33 — one table says where each global is bound | yes |

### ADR-005 — outputs, transforms and rendering

| ID | Ruling | Lean |
|---|---|---|
| R2-064 | PR1 — container entries are inline declarations | yes |
| R2-065 | PR2 — optional `profile` (`ooxml`, `odf`, `epub`), the tool writing `mimetype` | yes |
| R2-066 | PR3 — `for_each` a value expression, `when` a condition, per item | yes |
| R2-067 | PR4 — output kind from a closed extension table, overridable | yes |
| R2-068 | PR5 — mirror excludes `_` names and explicitly named templates | yes |
| R2-069 | PR7 — 255 bytes per segment; whole path within ADR-009's limits | yes |
| R2-070 | PR8 — new directory staged beside and renamed; existing one committed per file; no fsync | yes |
| R2-071 | PR9 — `executable: true` on template and asset outputs | yes |
| R2-072 | PR11 — pax archive with fixed headers, compared by entries | yes |
| R2-073 | PR12 — `--only <rendered path>` after a full render | yes |
| R2-074 | PR13 — container nesting to depth 3 | yes |
| R2-075 | PR14 — `newline: crlf` only | yes |
| R2-076 | PR15 — output schema validation for `json`/`yaml` outputs only | yes |
| R2-077 | PR16 — fixed staging and temporary names, created exclusively | yes |
| R2-078 | PR17 — output `id` uses the input-name rule | yes |

### ADR-006 — validation and findings

| ID | Ruling | Lean |
|---|---|---|
| R2-079 | PR1 — three severities, `info` never affecting exit | yes |
| R2-080 | PR2 — severity and class fixed per code | yes |
| R2-081 | PR3 — `--fail-on warning` | yes |
| R2-082 | PR4 — dotted family codes, JSON registry, drift test, a vector per code | yes |
| R2-083 | PR7 — output path refusals are class `content` | yes |
| R2-084 | PR9 — the `anyOf`/`oneOf` collapse rule | yes (confidence medium) |
| R2-085 | PR10 — one finding per missing member, at the absent member | yes |
| R2-086 | PR11 — partial validation of arrays with unparseable items | yes |
| R2-087 | PR12 — order by location, not severity | yes |
| R2-088 | PR13 — code-point columns; no end positions in v1 | yes |
| R2-089 | PR15 — messages and hints non-normative | yes |
| R2-090 | PR17 — one near-match rule for every suggestion | yes |
| R2-091 | PR18 — control and bidirectional characters neutralised | yes |
| R2-092 | PR19 — no stack trace on any stream, no debug flag | yes |
| R2-093 | PR20 — the human `at` line prints the inputs pointer `--set` takes | yes |

### ADR-007 — authoring

| ID | Ruling | Lean |
|---|---|---|
| R2-094 | PR2 — "declared" excludes what an absent `additionalProperties` allows | yes |
| R2-095 | PR3 — union semantics across applicators; no control flow | yes |
| R2-096 | PR4 — dynamic references warn and mark inputs used | yes |
| R2-097 | PR5 — unpassed partial parameter and unused template are warnings | yes |
| R2-098 | PR6 — OOXML split runs are errors, by declared namespace | yes |
| R2-099 | PR7 — invalid `default`/`examples` errors; undeclared `required`, CRLF templates warnings | yes |
| R2-100 | PR8 — optional `example.yaml` with `title`, `description`, `now` | yes |
| R2-101 | PR9 — examples positive only | yes |
| R2-102 | PR11 — containers compared by entries in `test` | yes |
| R2-103 | PR12 — `pack` tests after a clean check, no skip flag, replaces its own file, no `--overwrite` | yes |
| R2-104 | PR13 — quiet list; dotfiles inside layout trees packed | yes |
| R2-105 | PR14 — `unpack --to` required, empty target only | yes |
| R2-106 | PR15 — `init --name` from the directory operand, never the working directory | yes |
| R2-107 | PR16 — generator non-normative, deterministic per tool version, SplitMix64 | yes |
| R2-108 | PR17 — value preference `const`, `examples`, `default`, `enum` | yes |
| R2-109 | PR18 — optional modes also choose `null` | yes |
| R2-110 | PR20 — saved data in the first declared form; `--example` refuses an existing directory | yes |
| R2-111 | PR22 — a source `.compiled/` refused; OS litter packed and warned | yes |

### ADR-008 — the command line

| ID | Ruling | Lean |
|---|---|---|
| R2-112 | PR2 — supply flags `--set`, `--set-json`, `--inputs-json` | yes |
| R2-113 | PR3 — `--only <rendered path>` | yes |
| R2-114 | PR4 — envelope `format`/`ok`/`findings`/`result`/`next` | yes |
| R2-115 | PR5 — compact JSON, HTML escaping off | yes |
| R2-116 | PR6 — with `--json`, one document per run, on stderr only when stdout carries bytes | yes |
| R2-117 | PR8 — no package from standard input in v1 | yes |
| R2-118 | PR9 — flags may follow operands | yes |
| R2-119 | PR10 — a repeated single-valued flag is a usage error | yes |
| R2-120 | PR12 — `grep` exits 0 on no match | yes |
| R2-121 | PR13 — globs never enter dot-segments unless named | yes |
| R2-122 | PR14 — text = valid UTF-8 with no NUL | yes |
| R2-123 | PR15 — `pack` default `./<name>-<version>.metais`, replacing its own file | yes |
| R2-124 | PR16 — `init` name from the operand | yes |
| R2-125 | PR17 — sample defaults `--optional all`, `--seed 0` (and `--items 3`) | yes |
| R2-126 | PR18 — no tar, package or binary entry to a terminal | yes |
| R2-127 | PR19 — every writing command takes `--dry-run` | yes |
| R2-128 | PR20 — `validate --resolved` | yes |
| R2-129 | PR22 — `--limit` repeatable; `limits` command | yes |
| R2-130 | PR23 — `grep` defaults and bounds | yes |
| R2-131 | PR24 — `extract` refuses a glob that matches nothing | yes |
| R2-132 | PR25 — `render --out` required, even with `--dry-run` | yes |
| R2-133 | PR26 — no abbreviations, no aliases | yes |
| R2-134 | PR28 — `--fail-on` on every command that reports findings | yes |
| R2-135 | PR29 — references index: `path`, `size`, `text`, `lines`, Markdown `title` | yes |
| R2-136 | PR30 — `unpack` has no `--overwrite` | yes |

### ADR-009 — the security model

| ID | Ruling | Lean |
|---|---|---|
| R2-137 | PR2 — no CPU or wall-clock limit; deterministic counters | yes |
| R2-138 | PR4 — every symbolic link inside data or source directories refused | yes |
| R2-139 | PR5 — hard links not detected (documented residual) | yes |
| R2-140 | PR7 — YAML anchors, aliases, merge keys and duplicate keys refused everywhere | yes |
| R2-141 | PR8 — the METAIS pattern profile (ECMA-262 translated to RE2) | yes |
| R2-142 | PR12 — canary builds stay in the tree | yes |
| R2-143 | PR13 — syntactic tripwire with an exact allow-list | yes |
| R2-144 | PR14 — `findings` and `grep-matches` truncate with a marker | yes |
| R2-145 | PR15 — human output neutralises controls; `read` and stream output raw | yes |
| R2-146 | PR17 — zip permission bits never applied | yes |
| R2-147 | PR18 — `entry-ratio` only above 1 MiB | yes |
| R2-148 | PR19 — `limit.package` (refused) versus `limit.exceeded` (content) | yes |
| R2-149 | PR20 — `container-depth` limit, default 3 | yes |
| R2-150 | PR21 — hostile corpus committed as bytes, large cases at lowered limits | yes |

### ADR-010 — the implementation

| ID | Ruling | Lean |
|---|---|---|
| R2-151 | PR3 — Golden Liquid and JSON-Schema-Test-Suite vendored as test data | yes |
| R2-152 | PR4 — `santhosh-tekuri/jsonschema/v6` v6.0.3, closed loader, own formats | yes |
| R2-153 | PR5 — `go.yaml.in/yaml/v3` v3.0.5 node trees | yes |
| R2-154 | PR6 — own strict JSON reader; `gowebpki/jcs` v1.0.2 | yes |
| R2-155 | PR7 — own CSV reader | yes |
| R2-156 | PR8 — goldmark v2.1.6 for outline; own frontmatter split. The library choice is superseded by [ADR-010 Amendment A3](ADR-010-the-implementation.md#amendments) (accepted 2026-10-22): `outline` is METAIS's own CommonMark reading; the frontmatter split stands | v2 |
| R2-157 | PR9 — RE2 the only regex engine | yes |
| R2-158 | PR10 — own argument parser | yes |
| R2-159 | PR11 — own strict zip reader | yes |
| R2-160 | PR12 — version string rules | yes |
| R2-161 | PR13 — release artefacts built twice and compared | yes |
| R2-162 | PR14 — faults by options injection, controls in every `go test` | yes |
| R2-163 | PR15 — single-threaded render | yes |
| R2-164 | PR16 — spec files embedded via `internal/specdata` copies | yes |
| R2-165 | PR17 — `internal/hostfs` and `internal/hostenv.Isolate` as the single host seams | yes |
| R2-166 | PR18 — the conformance runner is a black box over the built binary | yes |

### ADR-011 — the specification document and conformance

| ID | Ruling | Lean |
|---|---|---|
| R2-167 | PR1 — `MTS-<AREA>-<NNN>` identifiers, lower-case anchors | yes |
| R2-168 | PR2 — keywords only inside statements; five forms only | yes |
| R2-169 | PR3 — one living text per major, `spec-<M>.<m>` tags, `since` marks | yes |
| R2-170 | PR4 — released vectors are the tie-breaker; only security errata tighten them | yes |
| R2-171 | PR5 — `statements.json` generated and committed | yes |
| R2-172 | PR7 — vector manifests are JSON | yes |
| R2-173 | PR8 — sentinel environment A and plain environment B must agree | yes |
| R2-174 | PR9 — a step with no declared tree writes nothing anywhere | yes |
| R2-175 | PR10 — release matrix on every target, pairwise-identical records | yes |
| R2-176 | PR11 — core Liquid proved by a pinned liquid-spec run at releases | yes |
| R2-177 | PR12 — spec examples are vectors included by reference | yes |
| R2-178 | PR13 — hostile packages: committed bytes, one defect each, refused by every opening command | yes |
| R2-179 | PR14 — runner bounds 10 s CPU, 1024 MiB, 120 s wall | yes |
| R2-180 | PR17 — suite paths checkout-safe on Windows, `-text` | yes |
| R2-181 | PR18 — archives and containers compared by entries | yes |
| R2-182 | PR19 — liquid-spec pinned at `84bf25e0`, `strict2`, feature gates as listed | yes |

## Resolved in this pass

Each line: what disagreed → which ADR won, and why. "Owner" means the
brief's ownership table; "stricter" means the security rule that refuses
more.

1. Manifest file name: ADR-002 accepted `metais.json` too → ADR-001 (lean:
   one golden path); ADR-002 keeps its JSON profile, chosen by the first
   byte (R1-001).
2. Inline JSON flag `--json` (ADR-003) vs `--inputs-json` (ADR-008) → ADR-008
   (owns names); one family `--set`, `--set-json`, `--inputs-json`, all
   repeatable as ADR-003's semantics need.
3. `x-` prefix: ADR-002 refuses, ADR-003 allows → both kept, scopes stated
   (manifest objects vs JSON Schema documents); ADR-002 D7 item 5's claim that
   unknown keywords are annotations corrected to match ADR-003 (R1-002).
4. Exit codes: ADR-008 assumed 1/2 → ADR-006's table (usage 3); ADR-008 D13
   and ADR-011's adapter rewritten.
5. Unicode 15.0.0 (ADR-001) vs 17.0.0 (ADR-005) → 15.0.0 with Go 1.26
   (ADR-010 owns the toolchain; x/text follows the toolchain) (R1-003).
6. zip64: ADR-005 and ADR-009 left it open → ADR-001 (owner): refused;
   ADR-005 refuses a container over 65,534 entries.
7. Single raw output `--file` (ADR-005) vs `--only` (ADR-008) → ADR-008;
   selection by rendered path only, ids not accepted.
8. Go ≥ 1.27 (ADR-005) and ≥ 1.24 (ADR-009) → ADR-010's Go 1.26, which has
   everything ADR-005 needs.
9. Sample commands (ADR-007 `sample --out/--save-as`, prints JSON) vs ADR-008
   (`render --sample`, `sample --to/--example`) → ADR-008's names, ADR-007's
   fixed-point semantics; ADR-011's `sample` operation aligned (R1-005).
10. `test --update` deletes (ADR-007) vs never deletes (ADR-008) → lean:
    deletes inside `expected/` only (R1-006).
11. `TERM` read (ADR-008, ADR-010) vs `NO_COLOR` only (ADR-009) → stricter
    (R1-007).
12. Object member order: code points (ADR-003) vs UTF-16 (ADR-004) → RFC 8785
    UTF-16 order for members; names and paths stay in UTF-8 byte order.
13. `asset` tag syntax → ADR-004 (owner): `{% asset "<path>" %}` or `as:`.
14. Bindings in `when`/`for_each`/paths → one table in ADR-004 D7 item 10;
    ADR-005 points to it; `output` never bound while a path is computed.
15. Validator error collection → "every finding" defined to exclude the
    boolean-only contexts, which produce no finding (ADR-006 D6 item 10,
    ADR-003 D3 item 7, ADR-010 D5 item 6).
16. Vector root `spec/vectors/` (ADR-010) → ADR-011's
    `spec/conformance/vectors/`.
17. Finding-code families (ADR-001's `container.*` etc., ADR-007's `xref.*`,
    `source.*`, `test.*`, ADR-008's `cli.*`, ADR-009's `entry.*`, hyphenated
    ADR-004 codes) → ADR-006 (owner): one family table, every list respelled
    (`package.container.*`, `template.xref.*`, `example.*`, `usage.*`, …).
18. `.compiled/inputs.schema.json` (ADR-003) → ADR-001's `schema.json`; `$id`
    `metais:///.compiled/schema.json`.
19. Staging inside the output directory (ADR-009) vs beside it (ADR-005) →
    ADR-005 (owner); `os.Root` confinement restated: every package- or
    input-derived path is written through a root on the staging or output
    directory, only the caller's own `--out` path through its ancestor.
20. `metais limits` missing from ADR-008 → added; limit names unified
    (ADR-004's `max_render_depth`, ADR-006's `max_findings` → ADR-009's
    `partial-depth`, `template-nesting`, `findings`).
21. Fixed test identity vs real tool version in vectors → `test` uses the
    fixed identity (ADR-004 owns); `render` vectors record the real digest
    (content-defined) and use ADR-011's tokens for the tool; ADR-010's
    `version.Override` limited to in-process tests.
22. osteele/liquid → METAIS's own engine (ADR-010); ADR-004 and ADR-009 no
    longer speak of forking or fallback; ADR-007 walks the engine's AST
    instead of a second front end (R1-004).
23. Random staging and temporary names (ADR-005) vs no random source
    (ADR-009, ADR-010 tripwires) → stricter: fixed names from a SHA-256 of a
    name, created exclusively.
24. ADR-005's `unchanged` status read existing files vs ADR-009 D8 item 4 →
    stricter: never read (R1-012).
25. ADR-008 echoed caller paths in output and `next` vs ADR-009 D15 and
    ADR-006's `<package>` → stricter: never printed (R1-011).
26. ADR-005 deflated every non-stored entry vs ADR-001's `auto` (empty entries
    stored is a fixed field) → ADR-001 (owner of writing rules); `test` and
    vectors compare a method only where `stored: true` fixes it.
27. Tar stream compared byte for byte (ADR-011) vs by entries (ADR-005) →
    ADR-005 (owner of the tar profile).
28. `and`/`or`: ADR-004 "left to right" vs ADR-010 "right to left" → verified
    against Shopify 5.14.0 source: grouped from the right, left operand
    evaluated first and short-circuited; ADR-004 (owner) restated, ADR-010
    aligned.
29. Several syntax errors per template (ADR-010) vs one (ADR-004) → ADR-004
    (owner of template errors).
30. In-process conformance runner and `-digests` flag (ADR-010) vs black-box
    runner and `-conformance.record` (ADR-011) → ADR-011 (owner of the runner).
31. Three overlapping SC7 harnesses (ADR-009 subprocess + build-tag canaries,
    ADR-010 in-process `LeakEnv`/`LeakClock` with build tags rejected,
    ADR-011 sentinel runner) → ADR-009 (owner of the isolation proof) built
    on ADR-011's runner and sentinels; ADR-010's injected faults keep the
    other guarantees; build tags only for the subprocess canaries.
32. Per-entry verification for `read`/`grep` (ADR-009 D5 item 8) and
    partial-read budgets (ADR-010 D15) vs whole-package verification
    (ADR-001) → ADR-001 (owner) (R1-013).
33. ADR-003 D1 item 7 materialized input defaults in the canonical manifest
    vs ADR-002 D9 item 2 → ADR-002 (owner of the canonical form).
34. `inputs` "may be absent" (ADR-003) vs required (ADR-002) → ADR-002 (owns
    the top level): required, may be `[]`; `input_constraints` added to
    ADR-002's root table, which said siblings add members there.
35. Source `.compiled/` a warning (ADR-007) vs refused (ADR-001); OS litter
    excluded silently (ADR-007) vs every file packed (ADR-001) → ADR-001
    (owner of the layout); `check` warns about litter; top-level `*.metais`
    added to ADR-001's silent set (needed by `pack`'s default).
36. Example `inputs/` "required, possibly empty" (ADR-007) vs no empty
    directories (ADR-001) → ADR-001; `inputs/` optional.
37. `init --name` from the working directory's name (ADR-007) vs from the
    operand (ADR-008) → stricter (ADR-009 forbids the cwd read); emptiness
    rule stays ADR-007's.
38. `pack --overwrite` and `unpack --overwrite`, `unpack [<dir>]` (ADR-007 vs
    ADR-008) → flags ADR-008's (`--to` required), behaviour ADR-007's (pack
    replaces its own file; unpack needs an empty target).
39. ADR-007's `--set incident.stage=final` → ADR-003's pointer syntax.
40. Entry-name and output-path predicates differed (ADR-001 category
    whitelist and simple folding; ADR-005 blacklist and full folding; no
    `.git`; no directory-prefix collisions) → one predicate in ADR-005 D12,
    the stricter union (R1-023).
41. Path length 1024 (ADR-005) vs `path-length` 512 (ADR-009) → ADR-009
    (owner of limits; limits cannot be raised); `container-depth` added to
    ADR-009 for ADR-005.
42. ADR-009 allowed directory entries and setuid bits → ADR-001's profile
    refuses both.
43. `metais:/inputs/…` (ADR-009) vs `metais:///…` (ADR-003) → ADR-003; output
    validation schemas' `$ref` scope (unspecified) defined as the `validate`
    file's top-level directory.
44. `"now"` in `date` means `--now` (ADR-009) vs always refused (ADR-004) →
    ADR-004 (owner of filters).
45. ADR-006 sent a stack trace to stderr behind a debug flag ADR-008 lacked →
    stricter: no trace anywhere; `--fail-on` added to ADR-008.
46. Neutralisation: tab escaped (ADR-006) vs kept (ADR-009), `\u001b` vs
    `\u{XXXX}`; ADR-006's text held a literal U+202E → one escape form; tab
    escaped in findings, kept in listings; the literal character replaced.
47. One `limit.exceeded` code with two classes vs ADR-006's fixed class per
    code → `limit.package` (refused) and `limit.exceeded` (content).
48. `.compiled/references.json` defined by nobody (ADR-001 and ADR-008 each
    deferred to the other) → defined in ADR-008 D17, as ADR-001 assigned it.
49. ADR-010 parsed ECMA patterns directly with `regexp/syntax` → ADR-009's
    translate-then-parse profile.
50. ADR-010's parser "long flags only" vs ADR-008's short aliases → ADR-008.
51. `internal/fsout` and `cmd/metais.isolate` (ADR-010, ADR-009) → one
    `internal/hostfs` and `internal/hostenv.Isolate`; `internal/dates` added
    for ADR-009's allow-list.
52. ADR-007's analysis budget "ADR-009 may lower" with no such limit →
    fixed constants of the analysis, never refusing.
53. ADR-007's generator listed formats ADR-003 refuses → only asserted
    formats.
54. `inspect` digest `null` on a source directory (ADR-008) vs computable
    (ADR-001) → ADR-001; shown.
55. ADR-002's device-name list lacked `com0`/`lpt0` → aligned with ADR-005.
56. ADR-003 assumed YAML `~` is null and core tags allowed → ADR-002's
    profile (both refused).
57. Hostile cases generated at test time (ADR-009) vs committed bytes
    (ADR-011) → ADR-011 (owner of the form); FIFO setups and `requires:
    fifos` added to ADR-011; device cases become unit tests.
58. ADR-011's liquid-spec pin waited on "the commit ADR-004 pins", which
    ADR-004 never pinned → ADR-011 pins `84bf25e0` and the feature gates.
59. One code `layout.unknown_top_level` used as an error and a warning →
    two codes (`package.layout.unknown_top_level`, `package.layout.not_packed`).
60. Output ids with hyphens vs input-name rule → one identifier rule.
61. ADR-004's "declared" set vs ADR-007's → ADR-007's is a stated superset
    adding only open-map names, so `check` never rejects what render accepts.
62. ADR-006's registry and schema paths (`spec/schemas/finding.schema.json`)
    vs ADR-011's per-minor layout → `spec/schemas/1.0/…`; registry at
    `spec/findings/registry.json`, catalog generated from it.
63. ADR-009's corpus "setuid bits accepted" and "a directory entry" →
    refused per ADR-001.

## PRD follow-ups

Applied to PRD-001 on 2026-10-18, as worded below.

1. **§2 Goals, third bullet** — "The same package and inputs give
   byte-identical outputs on every platform from one build of the tool; across
   builds and implementations they are identical except for the compressed
   bytes of zip containers, which are identical entry by entry." (ADR-001 D4
   item 6.)
2. **R9 item 5** — "The tool's own containers are byte-identical across
   platforms and runs of one build of the tool." (ADR-001 D4 item 6.)
3. **R10 item 2** — "The same package and the same inputs give byte-identical
   outputs on every supported platform and in every run of one build of the
   tool; across builds, zip containers are identical by entries." (ADR-001 D4
   item 6.)
4. **R13 item 4** — "…then writes a package whose package digest depends only
   on the source directory, and whose bytes depend only on the source
   directory and the build of the tool." (ADR-001 D4 item 6, D6.)
5. **SC5** — "…and rendering the same vectors on two of them with one build
   of the tool gives byte-identical output." (ADR-011 D15.)
6. **R5 item 1** — "On the command line, a single value is set by an inputs
   JSON Pointer (`--set` for text the schema coerces, `--set-json` for a JSON
   value), and a whole input or several inputs as one JSON object
   (`--inputs-json`)." (ADR-003 D17, ADR-008 D7.)
7. **R13 item 1** — "…with the manifest, `metais.yaml`, written in YAML or
   JSON." (ADR-001 D5 item 2, ADR-002 D1.)
8. **R12 item 3** — "Rendering never deletes a file. (Updating an example's
   expected outputs, R13 item 3, may remove stale files inside that example's
   own `expected/` tree.)" (ADR-007 D5 item 5.)
9. **R6 item 1** — "Validation reports every finding in one pass, never only
   the first; at most the `findings` limit are listed, and every finding is
   counted." (ADR-006 D8, ADR-009 D3.)
10. **R4 item 2** — "…a comma unless the manifest declares a semicolon, a pipe
    or a tab…" (ADR-003 D11 item 2.)
11. **R3 item 1** — "…formats from the closed list the specification asserts
    (a schema using any other `format` is refused)…" (ADR-003 D4.)
12. **R7 item 1** — "…Liquid's core tags and filters as the Shopify reference
    implementation (5.14.0, `strict2`) defines them, without `include` and
    `self`, with the deterministic differences the specification names
    (object output refused, simple case mapping, `date` inputs), plus…"
    (ADR-004 D1.)
13. **R11 item 2** — "…and the tool's version (recorded as `test` when
    examples are tested)…" (ADR-004 D15.)
14. **R12 item 2** — "…a path whose name Windows cannot hold, a name holding
    a character outside letters, marks, numbers, punctuation, symbols and
    the space (Unicode 15.0.0), a `.git` component, or two outputs whose paths
    differ only in letter case, is refused on every platform." (ADR-005 D12.)
15. **R12 item 5** — "…or, when the caller names exactly one output by its
    rendered path, as that file's bytes alone." (ADR-005 D18, ADR-008 D8.)
16. **R2 item 4** — "…and what to read next, as command lines that name the
    package and directories by placeholders rather than the caller's paths."
    (ADR-008 D14, ADR-009 D15.)
17. **§24 Dependencies** — "YAML 1.2 (a strict profile the specification
    defines)." (ADR-002 D2.)
