# ADR-010: The implementation — one Go module of internal packages on the Go 1.26 toolchain, METAIS's own Liquid engine, a short list of verified parsers behind seams, and every headline guarantee proved by a test that also runs against a deliberately broken tool

**Status:** Accepted 2026-10-18 by Ed Funnekotter; its proposed rulings are ratified in [RULINGS-v1](RULINGS-v1.md).
[Amendment A1](#amendments) (`golang.org/x/sys` and `golang.org/x/term` on
unix only) is proposed by PLAN-2026-10-metais-v1 Wave 1 and awaits Ed
Funnekotter's ratification; the passages it changes are marked *(A1,
proposed)*.
**Date:** 2026-10-18
**Deciders:** Ed Funnekotter
**Designs:** PRD-001 R18 item 1 (one static binary, the supported targets,
nothing precluding Windows), and the implementation side of R16 item 3 (the
vectors are part of the test suite), R10 items 1–2 (purity and byte identity),
R14 item 4 (host isolation) and §24's last dependency ("the Go libraries the
implementation ADR names, each only if it passes the conformance vectors;
where none does, METAIS implements its own").
**Depends on:** ADR-001 (the zip profile, the writer, the deflate level, the
Unicode tables it pins), ADR-002 (the YAML profile, RFC 8785 canonical form,
the tool's semantic version), ADR-003 (the parsing rules each input form
needs, the schema dialect, the `metais:///` retrieval URIs), ADR-004 (the
pinned Liquid dialect the engine implements), ADR-005 (`os.Root`, no-replace
rename, the tar profile, the determinism proofs), ADR-006 (`internal/finding`,
the failure tree, positions), ADR-007 (static analysis over the template AST,
the sample generator), ADR-008 (the argument parser's behaviour, the single
reader of the environment, `outline`), ADR-009 (every limit and the hostile
corpus), ADR-011 (the vector format and layout).

## Context

### What the PRD requires of the implementation

- **One static binary**, darwin and linux on amd64 and arm64, nothing
  precluding Windows (R18 item 1).
- **No public Go API** (§3): every package is `internal/`.
- **Dependencies only if they pass the vectors**; otherwise METAIS writes its
  own (§24). For Liquid specifically the ratified ruling is: use
  `github.com/osteele/liquid` if it passes the conformance vectors, otherwise
  METAIS implements its own engine.
- **Pure, deterministic, isolated rendering** (R10, R14 item 4): no code from
  a package, no network, nothing of the host reaching a template, the same
  bytes on every platform and in every run.
- **The vectors are part of `go test`** (R16 item 3), and the headline
  guarantees are pinned red first (AGENTS.md; SC7 says so for host
  isolation).
- **Dependencies are few and justified, each named in an ADR** (AGENTS.md),
  and code from another template engine, scaffolding tool or schema validator
  is never copied — a library is used through its module or the code is
  written here.

### The toolchain (verified 2026-10-18)

- go.dev/dl lists **go1.27.2 and go1.26.9** as the current releases. "Each
  major Go release is supported until there are two newer major releases", so
  Go 1.26 stays supported until Go 1.28 ships.
- **Go 1.27 changed `compress/flate`'s output**: "The exact encoded output
  from `Writer` may be different from Go 1.26 … the outputs from those
  packages [`archive/zip`, …] may also have changed." Go 1.27 also makes
  `encoding/json/v2` and `encoding/json/jsontext` available without an
  experiment flag; in go1.26.9 both are still behind `goexperiment.jsonv2`.
- **Unicode tables follow the toolchain.** `unicode.Version` is `15.0.0` under
  go1.26.8 and `17.0.0` under go1.27.2. `golang.org/x/text` v0.43.0 ships
  both `tables15.0.0.go` and `tables17.0.0.go` for `unicode/norm` and
  `cases`, the latter behind `//go:build go1.27`: `norm.Version` reports
  `15.0.0` when built with go1.26 and `17.0.0` with go1.27.
- **Go 1.24** made `go build` stamp the main module's version from the VCS tag
  (with `+dirty` for uncommitted changes; `-buildvcs=false` omits it) and
  added `os.Root`; Go 1.25 added `Root.Rename` and `Root.MkdirAll`.
- **`GODEBUG` in the environment wins** over `go.mod` `godebug` lines and
  `//go:debug` directives ("When a GODEBUG setting is not listed in the
  environment variable, its value is derived from …"). `zipinsecurepath` and
  `tarinsecurepath` change `archive/zip` and `archive/tar` reading.
- **The race detector "requires cgo to be enabled, and on non-Darwin systems
  requires an installed C compiler."** A static release build is
  `CGO_ENABLED=0`; `go test -race` cannot be.
- **`//go:embed` patterns "may not contain ‘.’ or ‘..’ … path elements"**, so
  a package under `internal/` cannot embed a file under `spec/`.

### Liquid: the two Go engines, measured

The ruling makes `osteele/liquid` the first candidate. It and the only other
Go engine claiming Shopify parity were probed on 2026-10-18 with throwaway
programs outside this repository, against **Golden Liquid** (jg-rp/golden-liquid,
MIT, commit `65c2f76`, 1,098 cases, "All tests pass with Shopify/liquid Ruby";
its `Gemfile.lock` resolves Shopify Liquid 5.12.0). The probe gave each case's
`templates` as an in-memory store, decoded data with integers as integers,
and counted a case as passing when the output matched or, for an `invalid`
case, when an error was returned. The figures are the probe's, not a
certified score; the categories below are unambiguous whatever the harness.

**`github.com/osteele/liquid` v1.9.2** (MIT; released 2026-08-16; active:
v1.9.0–v1.9.2 in August 2026 added the `render` tag and "harden rendering";
`go 1.25.0`; depends on `osteele/tuesday` and `gopkg.in/yaml.v2`):

- **448 of 1,098 cases fail.** Tags absent: `echo`, `liquid`, `increment`,
  `decrement`, `ifchanged`, `doc` (each an "undefined tag" error), and the
  inline comment `{%- # … -%}`, which is **rendered literally with no error**.
  Filters absent: `base64_encode`, `base64_decode`, `base64_url_safe_encode`,
  `base64_url_safe_decode`, `find`, `find_index`, `has`, `reject`, `squish` —
  all members of Shopify's `standardfilters.rb` at v5.14.0.
- **`blank` and `empty` are not implemented**: they evaluate as undefined
  variables, so `'' == empty` and `[] == empty` are false, `nil == empty` is
  true, and `blank == empty` is true.
- **`for` over an object is non-deterministic**: 20 renders of
  `{% for kv in m %}{{ kv[0] }},{% endfor %}` over a 20-key map gave 14
  distinct orders (Go map iteration). Binding objects as `yaml.MapSlice`
  restores order but breaks `sort: "key"` (no sort) and `json` (emits
  `Key`/`Value` structs). R10 item 3 cannot be met either way from outside the
  library.
- **Loop modifiers differ from Shopify by design** (its
  `docs/loop-semantics.md`: `reversed limit:2` gives `54` where Shopify gives
  `21`; "Changing it requires a deliberate compatibility decision"); `offset:
  continue`, `forloop.parentloop` and `forloop.name` fail.
- **Numbers render differently**: `{{ 1.0 }}` → `1`, `{{ 3.0 | times: 2 }}` →
  `6` (Shopify `1.0`, `6.0`).
- **Host reads inside the library**: `values.ParseDate` returns `time.Now()`
  for `"now"` and parses every layout with `time.ParseInLocation(…,
  time.Local)` (the probe printed `EST -0500` from the host zone); the default
  `TemplateStore` is a `FileTemplateStore` whose empty root is the working
  directory (`{% render "x" %}` gave `openat x: no such file or directory`).
- **No resource limits** (SECURITY.md: "no built-in CPU, memory, iteration, or
  output limits"; `FRender` checks a context only when the template writes),
  and no depth limit on `render` recursion.
- **Panics escape `Render`** on valid templates: `{{ 'Liquid' | slice: -2, -1
  }}` (slice bounds out of range) and two `sort_natural` cases.
- **Expressions are compiled closures** (`expressions.expression{evaluator
  func(Context) values.Value}`); the AST exposes tag names and argument text
  but no expression tree, so ADR-007's static cross-reference would need a
  second parser of the expression grammar.
- **Non-Shopify filters are always registered** (`json`, `inspect`, `type`,
  `handleize`, `pluralize`, …) and there is no way to unregister a filter.

**`github.com/Notifuse/liquidgo`** (MIT; a structural port of the Ruby code,
"Full feature parity with Ruby Liquid 5.10.0"; last commit 2026-01-31; its
`v5.x` tags are not valid for its unsuffixed module path, so `go get` resolves
only the pseudo-version `v0.0.0-20260131093335-c0b2d83983a7`):

- **245 of 1,098 cases fail** (ranges in `join`, `for` modifiers, `reject`,
  `where`, `squish`, `liquid`, …). `blank`/`empty` work and `for` over a map is
  stable.
- **Host reads inside the library**: `"now"` and `"today"` become
  `time.Now()`; Unix timestamps become `time.Unix(ts, 0)` in the local zone;
  `LocalFileSystem` and `i18n` read files with `os.ReadFile`.
- `{{ 1.0 }}` → `1`.

**Shopify Liquid** (MIT) at tag **v5.14.0** (2026-09-17, `ef311c08`) has the
tags `assign break capture case comment continue cycle decrement doc echo for
if ifchanged include increment inline_comment raw render tablerow unless` plus
the `liquid` tag in `block_body.rb`; `Block::MAX_DEPTH = 100`; and
`ResourceLimits` with `render_length_limit`, `render_score_limit` and
`assign_score_limit`.

### JSON Schema 2020-12

- **`github.com/santhosh-tekuri/jsonschema/v6` v6.0.3** (Apache-2.0; released
  2026-08-06; last commit 2026-10-18; `go 1.21`; production dependency
  `golang.org/x/text`, `dlclark/regexp2` for its tests only). README: passes
  the JSON-Schema-Test-Suite for 2020-12 excluding optional tests; its own
  suite runner skips `ecmascript-regex.json`, `zeroTerminatedFloats.json`,
  `idn-email.json`, `idn-hostname.json`. Built-in formats: `regex`, `ipv4`,
  `ipv6`, `json-pointer`, `relative-json-pointer`, `uuid`, `duration`,
  `period`, `hostname`, `email`, `date`, `time`, `date-time`, `uri`, `iri`,
  `uri-reference`, `iri-reference`, `uri-template`, `semver`; no IDN formats.
  `Compiler.RegisterFormat` overrides any format but `regex`;
  `Compiler.UseRegexpEngine` replaces the regex engine (default Go `regexp`);
  `Compiler.UseLoader` replaces the loader; `Compiler.AddResource` registers a
  document under a URL.
- **Its default loader is `FileLoader`, which `os.Open`s `file://` URLs**;
  meta-schemas are embedded (`//go:embed metaschemas`) and never fetched.
- **Probe**: with a loader serving only `metais:///…` entries, a relative
  `$ref` resolved to `metais:///inputs/common.json`; `$ref`s to
  `file:///etc/passwd` and `https://example.com/x.json` were refused by the
  loader; `(?=a)b` as a `pattern` failed meta-schema validation ("invalid or
  unsupported Perl syntax"); an instance with five errors reported all five
  with `keywordLocation` and `instanceLocation`, and `{}` reported `missing
  properties 'n', 'd'` once. **The order of sibling errors varied between
  runs** (it follows map iteration).
- **`github.com/kaptinlin/jsonschema` v0.9.9** (MIT): pre-1.0; `go 1.27.0` in
  its `go.mod`; `http` and `https` loaders "ship pre-registered with a 10s
  timeout"; depends on `goccy/go-yaml`, `google/uuid`, `kaptinlin/go-i18n` and
  more.
- The **JSON-Schema-Test-Suite** is MIT.

### YAML

- **`go.yaml.in/yaml`** is the YAML organization's continuation of the
  archived `go-yaml/yaml`. Its README: "Versions v1, v2, and v3 will remain as
  frozen legacy … security-fixes only"; work happens in v4, at
  **v4.0.0-rc.6** (2026-06-13) with commits through 2026-10-17 and no final
  release. **v3.0.5** is dated 2025-11-03. Licence: Apache-2.0, with the files
  ported from libyaml under MIT.
- **Probe of v3.0.5** (v4.0.0-rc.6 behaved the same): the node tree carries
  kind (alias nodes included), tag, value text, style, line and column;
  decoding into `any` resolves `0777` to 511 and `2026-01-02` to a
  `time.Time` (YAML 1.1 behaviour), leaves `yes` a string, refuses a
  duplicate key ("mapping key "a" already defined at line 1") but the node
  tree does not; **`Unmarshal` silently ignores a second document**; it
  refuses an alias bomb ("document contains excessive aliasing") and nesting
  beyond 10,000 ("exceeded max depth of 10000").
- **`github.com/goccy/go-yaml` v1.19.2** (MIT, 247 open issues, last push
  2026-04-11): also resolves `0777` to 511, and **accepted the same alias
  bomb without error**.

### CSV

- **Go's `encoding/csv`** "converts all \r\n sequences in its input to plain
  \n, including in multiline field values"; "Blank lines are ignored"; it does
  not strip a byte order mark (the probe's first header was `"﻿a"`);
  `FieldPos` gives line and byte column; a field-count mismatch is a non-fatal
  `ParseError`. **It does not tell a quoted empty field `""` from an unquoted
  empty field**, which ADR-003 D11 item 5 needs (unquoted empty is absent,
  quoted empty is the empty string).

### Markdown

- **`github.com/yuin/goldmark/v2` v2.1.6** (MIT; 2026-09-27; `v2.0.0`
  2026-08-27; no dependencies; `go 1.25`): "compliant with CommonMark
  0.31.2"; "v2 all nodes hold the start position"; maintenance covers "up to
  one major version prior to the latest". v1 continues at v1.8.6.
- **Probe**: headings found for ATX, setext, inside a block quote and inside a
  list item; a `# not` inside a fenced block and an indented `# …` were not
  headings; an empty `#` heading had a position.
- Frontmatter is not CommonMark (ADR-003 D13 defines it).

### Regular expressions

- **Go's `regexp`** "is guaranteed to run in time linear in the size of the
  input"; `regexp/syntax` exposes the parse tree.
- **`github.com/dlclark/regexp2`** "doesn't have constant time guarantees";
  its defence is `MatchTimeout` (wall-clock) and a backtracking-stack cap.

### Archives

- **`archive/zip`**: `CreateHeader` sets the data-descriptor flag on every
  file entry and adds an extended-timestamp extra field when `Modified` is
  set; `CreateRaw` writes a header exactly as given, with sizes in the local
  header when bit 3 is clear and without `CreateHeader`'s UTF-8 flag
  detection. The reader takes names and sizes from the central directory and
  reads a local header only to skip it (ADR-001 Context); `NewReader`'s
  insecure-path behaviour follows `GODEBUG`.
- **`compress/flate.NewReader`**: "If r does not also implement
  io.ByteReader, the decompressor may read more data than necessary"; "Any
  trailing data after the final block is ignored."
- **`archive/tar`** writes ustar where every field fits, otherwise a pax `x`
  header whose records are sorted (`slices.Sorted(maps.Keys(paxHdrs))`) and
  whose name is `PaxHeaders.0/<file>` — no process id.

### JSON

- **`encoding/json`** replaces invalid UTF-8 and unpaired surrogate escapes
  with U+FFFD, accepts duplicate names (last wins), caps nesting at 10,000,
  and escapes `<`, `>`, `&` unless `SetEscapeHTML(false)`. Its `Decoder.Token`
  with `UseNumber` exposes each number's literal text.
- **`github.com/gowebpki/jcs` v1.0.2** (Apache-2.0, 2026-09-21): an RFC 8785
  transform over JSON text, including ECMAScript number formatting.

### Command-line parsing

- Go's `flag` stops at the first operand. **`spf13/pflag`** (BSD-3-Clause)
  intersperses flags but combines boolean shorthands (`-abc`) and lets a
  repeated single-valued flag silently win. ADR-008 requires interspersed
  flags, no grouping, no abbreviation and a refusal on a repeated
  single-valued flag.

## Decision

### D1 — Toolchain and module

**One module, `github.com/kottera/metais`, on the Go 1.26 toolchain, with
everything under `internal/`.**

1. **`go.mod`** begins:

   ```
   module github.com/kottera/metais

   go 1.26.0

   toolchain go1.26.9
   ```

   The `go` line is the language floor; the `toolchain` line is the exact
   toolchain every build and every test uses (`GOTOOLCHAIN` default `auto`
   fetches it). A toolchain patch bump is an ordinary change; a minor bump is
   a decision recorded in this ADR's log, because it moves two things a
   package can observe (D1 item 3).
2. **No `godebug` lines and no `//go:debug` directives.** No observable
   behaviour may depend on a `GODEBUG` setting (D11 item 4 removes the ones
   that would), so there is nothing to pin.
3. **The toolchain pins two observable tables**, and a tripwire test fails
   when either moves (`internal/unitext/version_test.go`):
   - **Unicode**: `unicode.Version` and `norm.Version` must equal the
     version the specification pins; on Go 1.26 both are `15.0.0`.
   - **Deflate**: a golden-bytes canary packs a fixed source tree
     (`internal/zipfmt/testdata/canary/`) and compares the package's file
     SHA-256 with a committed value, so a toolchain whose encoder changes is
     noticed as a change of the tool's output (ADR-001 D4 item 6), not
     discovered in a user's diff.
4. **The specification pins Unicode 15.0.0** (ADR-001 D2, ADR-005 D12), the
   version Go 1.26's `unicode` and the `golang.org/x/text` tables it selects
   carry. `x/text` chooses its tables by the compiling toolchain, not by its
   module version: v0.43.0 ships `tables15.0.0.go` under `//go:build
   !go1.27` and `tables17.0.0.go` under `//go:build go1.27`, for both
   `unicode/norm` and `cases`. **Moving to Go 1.27** therefore moves every
   table to 17.0.0 at once, and a `1.0` package must still be judged by
   15.0.0's set of assigned characters (ADR-002 D7 item 3). That move is one
   change, taken only together with either a specification minor that pins
   17.0.0 or METAIS's own copy of the 15.0.0 "assigned" table for packages
   declaring an earlier minor (normalization and case folding of assigned
   characters do not change, by Unicode's stability policy): the `go` and
   `toolchain` lines, the canary digest, the Unicode tripwire's expected
   version, and that table. Until then the tripwire holds the toolchain at
   Go 1.26. (*Proposed rulings* 1.)
5. **Every Go file starts with the AGENTS.md licence header**; a test in
   `internal/guarantees` checks it.

### D2 — Layout

**`cmd/metais` is a shell around `internal/cli`; each subsystem is one
internal package owned by one ADR; the Liquid engine knows nothing of
METAIS.**

```
cmd/metais/main.go            d := hostenv.Isolate(); os.Exit(cli.Main(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, d))
internal/
  cli/                        command table, dispatch, help, JSON envelope      ADR-008
  cli/args/                   the argument parser (D10)                         ADR-008
  hostenv/                    Isolate (NO_COLOR, Clearenv, UTC); IsTerminal     ADR-008 D12, ADR-009 D14
  dates/                      RFC 3339 `--now` parsing, the date filter's maths ADR-004 D11
  version/                    tool version, implemented specification versions ADR-002 D8
  finding/                    finding model, registry, report                   ADR-006 D13
  schemafail/                 library-neutral failure tree                      ADR-006 D13
  limits/                     every limit, its default, lowering                ADR-009
  unitext/                    NFC, simple case folding, categories, tripwires   ADR-001 D2, ADR-005 D12
  zipfmt/                     strict zip reader, deterministic zip writer       ADR-001 D1, D4, D8
  pkgtree/                    a package as a confined read-only tree            ADR-001 D5, D8
  digest/                     the digest file and package digest                ADR-001 D6
  jsonx/                      strict JSON reader with positions; JCS            ADR-002 D9, ADR-003 D9
  yamlx/                      the YAML profile over yaml.v3 nodes               ADR-002 D2, ADR-003 D10
  manifest/                   the manifest model and its checks                 ADR-002
  schema/                     JSON Schema compile/validate, formats, regex hook ADR-003 D3–D7
  regexprofile/               the package regular-expression profile           ADR-009
  inputs/                     forms, collections, supply, precedence, defaults ADR-003
  inputs/csv/                 the RFC 4180 reader (D8)                          ADR-003 D11
  markdown/                   frontmatter split, heading outline                ADR-003 D13, ADR-008
  liquid/                     the engine: lexer, parser, renderer, core filters ADR-004 (dialect)
  liquid/ast/                 the AST, walkable                                 ADR-004, ADR-007
  liquid/value/               the value model                                   ADR-003 D8, ADR-004
  template/                   METAIS variables, filters, tags, partials, escape ADR-004
  render/                     plan, transforms, staging, dry run, stdout        ADR-005
  hostfs/                     every host file access: os.Root reads and writes,  ADR-009 D7–D8,
                              no-replace rename, staging names                  ADR-005 D14–D15
  tarw/                       the standard-output archive                       ADR-005 D17
  browse/                     ls, read, outline, grep, schema, extract          ADR-008
  author/                     init, check, test, pack, unpack                   ADR-007
  xref/                       template-to-schema cross-reference                ADR-007
  sample/                     the sample-data generator                         ADR-007
  specdata/                   embedded copies of the spec files the binary needs D2 item 4
  fault/                      test-only fault switches (D13)
  conformance/                the vector runner                                 ADR-011
  guarantees/                 tripwires and pinned-false proofs (tests only)    D13
  testenv/                    hermetic test helpers (HOME, temp dirs)
  tools/release/              the release builder, run with `go run`            D12
spec/                         the specification                                 ADR-011
```

1. **Package names are these.** ADR-006's `internal/finding` and
   `internal/schemafail` are adopted as named; ADR-008's and ADR-009's single
   reader of the environment is `internal/hostenv.Isolate`, and ADR-009's
   single package for host files is `internal/hostfs`.
2. **Layering is enforced** by `internal/guarantees/layers_test.go`, which
   runs `go list -deps -json ./...` and checks a table committed beside it.
   The rules that carry weight:
   - `internal/liquid/...` imports no other `internal/` package and nothing
     outside the standard library. It is a Liquid engine, testable against
     Golden Liquid with no METAIS concept in scope.
   - `internal/browse` reaches `internal/hostfs`'s write functions only
     through `extract`'s single entry point, so listing, reading and
     searching cannot write (R2 item 1).
   - Only `internal/hostenv` imports `golang.org/x/term`; only
     `internal/hostfs` imports `golang.org/x/sys`. Each does so only in
     files built for darwin and linux: on windows the binary links neither
     module, and both packages use the standard library's `syscall` there
     (D3 item 3). *(A1, proposed)*
   - Only `cmd/metais` refers to `os.Args`, `os.Stdin`, `os.Stdout`,
     `os.Stderr` and `os.Exit`; everything else receives them, so the whole
     tool runs in-process under test (D13).
3. **The render is single-threaded.** No goroutine is started on any path
   that reads a package, validates or renders, so no output can depend on
   scheduling. Parallelism is a later optimization, taken only with a
   benchmark that needs it.
4. **Spec files the binary embeds** — the manifest schemas (ADR-002 D10), the
   findings registry and finding schema (ADR-006 D13) — live canonically under
   `spec/` and are copied byte for byte into `internal/specdata/` by `go
   generate ./internal/specdata`, because `//go:embed` cannot reach `../`.
   `TestSpecDataMatchesSpec` fails on any difference. This keeps `spec/` free
   of Go files, so it is not an importable package.

### D3 — Dependency policy

**A library may parse; METAIS decides what is accepted and emits every byte
the specification defines.**

1. **A parsing library is admitted only behind a METAIS check.** The library
   turns bytes into a structure with positions; METAIS's own code applies the
   specification's accept/refuse rules to that structure. No library's own
   type resolution, defaulting or leniency reaches the data model.
2. **Every byte the specification defines is produced by METAIS code, or by
   an implementation of a published standard whose output that standard fixes
   completely** (RFC 8785 is the one case). Serializing filters (`json`,
   `yaml`, `csv`, `markdown_table`), provenance comments, findings, the
   digest file and the tar headers are METAIS code. The deflate stream is the
   exception ADR-001 D4 item 3 allows: the specification fixes its validity,
   not its bytes.
3. **The admitted modules**, each at an exact version in `go.mod`:

   | Module | Version | Licence | Used for | Fallback when it fails |
   |---|---|---|---|---|
   | `github.com/santhosh-tekuri/jsonschema/v6` | v6.0.3 | Apache-2.0 | JSON Schema 2020-12 validation (D5) | METAIS's own validator for the asserted keywords and formats |
   | `go.yaml.in/yaml/v3` | v3.0.5 | Apache-2.0 and MIT | YAML node trees (D6) | `go.yaml.in/yaml/v4` once final; then METAIS's own parser of the profile |
   | `github.com/yuin/goldmark/v2` | v2.1.6 | MIT | heading outline (D9) | `github.com/yuin/goldmark` v1.8.6 |
   | `github.com/gowebpki/jcs` | v1.0.2 | Apache-2.0 | RFC 8785 serialization (D7) | METAIS's own RFC 8785 writer |
   | `golang.org/x/text` | v0.43.0 | BSD-3-Clause | NFC check, case folding (D11) | none needed; already required by the JSON Schema library |
   | `golang.org/x/sys` | v0.49.0 | BSD-3-Clause | no-replace rename on darwin and linux (D11); not linked on windows | none |
   | `golang.org/x/term` | v0.47.0 | BSD-3-Clause | `IsTerminal` on darwin and linux (ADR-008); not linked on windows | none |

   Every licence here is compatible with Apache-2.0 distribution of a binary,
   given the notices of D12 item 6.

   **`golang.org/x/sys` and `golang.org/x/term` are linked on unix targets
   only.** `golang.org/x/sys/windows` imports `net`, which D11 item 10 and
   ADR-009 D16 item 3 keep out of the binary's closure on every target, and
   `golang.org/x/term` imports `golang.org/x/sys/windows` on windows. On
   windows `internal/hostenv` asks `syscall.GetConsoleMode` whether a stream
   is a terminal (D11 item 8), and `internal/hostfs` takes the no-replace
   rename from `syscall.MoveFile` (D11 item 3). *(A1, proposed)*
4. **Rejected and not in `go.mod`**: `osteele/liquid`, `Notifuse/liquidgo`
   (D4); `kaptinlin/jsonschema` (Alternatives); `goccy/go-yaml`;
   `dlclark/regexp2`; `spf13/pflag`; `encoding/csv` for reading (D8);
   `adrg/frontmatter`.
5. **A dependency update is a change like any other**: it runs the whole
   suite, including every vector and the canary, and names the module and
   version in its commit. `govulncheck ./...` is run before each release
   (it needs the network; it is a developer step, never a test).
6. **Tests may read third-party test data** vendored under a `testdata/`
   directory with its licence and the commit it was taken from (D14 item 4);
   they add no module dependency.

### D4 — Liquid: METAIS implements its own engine

**`osteele/liquid` fails requirements the conformance vectors must prove and
that no caller of the library can repair, so `internal/liquid` is METAIS's own
engine of the dialect ADR-004 pins.**

1. **Why the ruling's condition is already settled.** The vectors do not
   exist yet, but the failures in Context are of rules the PRD fixes, not of
   corner cases: R10 item 3 (object iteration order), R7 item 1 (Shopify's
   core tags and filters, including `echo`, `liquid`, `increment`, the
   inline comment and `blank`/`empty`), R14 items 2 and 4 (no limits; the
   clock, the local zone and the working directory read inside the library),
   and ADR-007's static analysis (no expression tree). Each fix is inside the
   library; METAIS cannot fork it (AGENTS.md forbids copying another engine's
   code), and its loop-modifier divergence is a deliberate upstream choice.
   `Notifuse/liquidgo` fails a quarter of Golden Liquid, reads the clock and
   the local zone, has no taggable module version, and has had no commit since
   2026-01-31. The engine is therefore METAIS's.
2. **Scope — what the engine is.** Written from the Shopify documentation,
   the specification ADR-004 writes, and Golden Liquid's cases — never from
   another engine's source:
   - **Lexer**: text, `{{ … }}`, `{% … %}`, the `-` whitespace-control
     markers, `raw`, `comment`, `doc`, the inline comment `#`, and the
     line-oriented body of `{% liquid %}`. Every token keeps its byte span,
     line and column (ADR-007 needs spans for the OOXML split-run finding).
   - **Parser**: a strict parser (Shopify's strict mode; no lax recovery)
     producing `internal/liquid/ast` — block, text, output, every tag as its
     own node type, and a full expression tree: literals (string, integer,
     float, `true`, `false`, `nil`, `blank`, `empty`), ranges `(a..b)`,
     variable paths with `.name`, `[expr]` and the `size`/`first`/`last`
     properties, comparisons `== != <> < > <= >= contains`, `and`/`or`
     grouped from the right with the left operand evaluated first and
     short-circuited, as ADR-004 D2 states, and filter chains with
     positional and keyword arguments. A template's parse stops at its first
     syntax error, which is its one `template.syntax` finding (ADR-004 D16
     item 4), with line and column. Nesting deeper than ADR-009's
     `template-nesting` (default 100, Shopify's) is an error.
   - **Tags**: `assign`, `capture`, `case`/`when`/`else`, `comment`, `#`,
     `doc`, `cycle`, `increment`, `decrement`, `echo`, `for`/`else` with
     `limit`, `offset` (including `continue`), `reversed` and ranges, `break`,
     `continue`, `if`/`elsif`/`else`, `ifchanged`, `liquid`, `raw`, `render`
     (`with`, `for`, `as`, named parameters, isolated scope, literal template
     name), `tablerow` (`cols`, `limit`, `offset`), `unless`/`elsif`/`else`.
     **`include` is not a tag**: it parses to a dedicated error that names
     `render` (ADR-004 words it). The exact set is ADR-004's; the engine's
     `Core()` dialect holds exactly the set ADR-004 pins, and a test compares
     the two lists.
   - **Filters**: every filter of Shopify's `standardfilters.rb` at the
     version ADR-004 pins — at v5.14.0 `abs append at_least at_most
     base64_decode base64_encode base64_url_safe_decode
     base64_url_safe_encode capitalize ceil compact concat date default
     divided_by downcase escape escape_once find find_index first floor has
     join last lstrip map minus modulo newline_to_br plus prepend reject
     remove remove_first remove_last replace replace_first replace_last
     reverse round rstrip size slice sort sort_natural split squish strip
     strip_html strip_newlines sum times truncate truncatewords uniq upcase
     url_decode url_encode where` — with ADR-004's semantics where it pins
     one (stable sorting, the `date` filter's format directives over a value
     with an explicit offset, number rendering). No other filter exists in
     `Core()`. METAIS's own filters and tags are registered by
     `internal/template` through the engine's extension API, which is
     internal Go code, never a package's.
   - **Value model** (`internal/liquid/value`): nil, boolean, integer (int64;
     overflow is an error, never a wrap), float (binary64), string,
     bytes (an `embed`ded input's raw bytes, rendered unchanged), array,
     **ordered object** (keys in a defined order, ADR-004's), lazy range, and
     the `forloop`/`tablerowloop` drops. **No Go map is ever iterated by the
     engine**; objects iterate their key slice.
   - **Renderer**: one goroutine, a scope stack, the `break`/`continue`
     interrupts, the partial loader, an escaper hook for ADR-004's per-output
     `escape: html`, and **the meter** (item 4).
3. **Interface** (sketch; names may be refined, the shape may not):

   ```go
   package liquid

   type Dialect struct{ /* frozen tag and filter tables */ }
   func Core() *Dialect                                    // ADR-004's core set, include excluded
   func (d *Dialect) Extend(tags []TagDef, filters []FilterDef) *Dialect

   type Source struct{ Name string; Text []byte }          // Name is the package path, for errors
   func Parse(d *Dialect, s Source, lim ParseLimits) (*Template, []*Error)
   func ParseValue(d *Dialect, s Source) (*ast.Expr, *Error)     // ADR-005 D3 for_each, path parts
   func ParseCondition(d *Dialect, s Source) (*ast.Cond, *Error) // ADR-005 D3 when

   type Template struct{ Name string; Root *ast.Block }   // immutable; safe to render repeatedly

   type Partials interface{ Partial(name string) (*Template, *Error) } // render tag; package-confined
   type Env struct {
       Globals  *value.Object   // ADR-004 D7's globals, bound where its item 10 says — nothing else
       Partials Partials
       Escape   func(dst []byte, s string) []byte // nil = no escaping
       Meter    *Meter
   }
   func Render(w io.Writer, t *Template, env *Env) *Error
   func Eval(e *ast.Expr, env *Env) (value.Value, *Error)
   func EvalCondition(c *ast.Cond, env *Env) (bool, *Error)

   type Error struct {
       Source     string
       Line, Col  int     // 1-based; Col in code points (ADR-006 D13 item 2)
       Code       string  // engine code; internal/template maps it to an ADR-006 code
       Args       map[string]string
   }
   ```

   `ast.Walk(node, func(ast.Node) bool)` visits every node, with each
   variable path as `ast.Path{Root string; Segments []ast.Segment}` and each
   binding construct (`assign`, `capture`, `for`, `tablerow`, `render … as`)
   marked, so ADR-007's cross-reference resolves locals to input paths
   without a second parser.
4. **The meter counts work; it never reads a clock.** Limits are
   deterministic counters checked on every step: steps (one per node
   executed, loop iteration, filter application and partial entry), output
   bytes, bytes held by strings built in `assign`/`capture`/filters, range
   length before materialization, `render` depth, and total partials
   rendered. Exceeding one is an `*Error` with its own code, located at the
   node. The defaults and names are ADR-009's (mirrored in
   `internal/limits`); because the counts are deterministic, the same package
   and inputs exceed the same limit at the same node on every machine, which
   a wall-clock timeout could not promise (R10 item 2).
5. **The engine has no host access.** `internal/liquid` imports no `os`,
   `time` (except for formatting a `time.Time` value already given it, with
   its offset, never `time.Local`), `net`, `syscall` or `math/rand`;
   `layers_test.go` checks it. Its only inputs are the source, the globals and
   the partial loader.
6. **No panic escapes.** `Render`, `Eval` and `Parse` recover a panic into an
   internal-error `*Error` (ADR-006's internal class), so a defect shows as a
   finding with a code and never as a Go stack trace. Each recovered panic is
   a bug: the fuzz targets (D14 item 6) treat it as a failure.
7. **Proving the engine**:
   - **Golden Liquid** (D14 item 4) runs in `go test ./internal/liquid`; every
     case passes or is listed in `testdata/golden-liquid/exclusions.txt` with
     one reason per line from a closed set: `include` (excluded tag),
     `lax` (a lax-mode case), `error-string` (Shopify renders an error
     message inline; METAIS refuses), `dialect` (ADR-004 pins a different
     behaviour — the line names the ADR-004 item), `version` (Shopify changed
     after the corpus's 5.12.0). The exclusion list is reviewed like code;
     `dialect` lines are the only kind that may grow without an ADR-004
     change.
   - **Dialect vectors** (ADR-011) are the normative proof; Golden Liquid is
     the breadth check.
   - **Unit tests** per tag and filter, written from ADR-004's text.
8. **Complexity**: large. It is the biggest single work item in the plan:
   lexer and parser, the value model with Ruby-compatible comparison and
   arithmetic, the renderer and meter, and the filter catalog, each with
   tests. It is also the piece that most needs to be right, which is the
   argument for owning it.

### D5 — JSON Schema: `santhosh-tekuri/jsonschema/v6`, configured closed

**The validator compiles schemas from the package only, asserts METAIS's
formats with METAIS's definitions, and uses the package regex profile.**

1. **One constructor**, `schema.NewCompiler(tree pkgtree.Tree, lim
   limits.Limits)`, is the only call site of `jsonschema.NewCompiler` in the
   module (a tripwire checks). It:
   - calls `DefaultDraft(jsonschema.Draft2020)`;
   - calls `UseLoader` with a loader that refuses every URL (D5 item 2), so
     `FileLoader` is never reached;
   - registers every schema file of the package with `AddResource`, under its
     retrieval URI `metais:///<package path>` and, where its root declares
     `$id`, its canonical URI (ADR-003 D5 items 1–2);
   - calls `AssertFormat()` and, for every format ADR-003 D4 asserts,
     `RegisterFormat` with METAIS's own validator implementing the
     specification's grammar (the library's built-ins are never used, so a
     library update cannot change what a format accepts);
   - calls `UseRegexpEngine` with `regexprofile.Compile` (D10).
2. **No reference leaves the package.** The loader returns a refusal naming
   the URL for anything not added in item 1, and the finding is ADR-003's
   reference finding. Meta-schemas come from the library's embedded copies.
3. **What METAIS checks before the library sees a schema** (in
   `internal/schema`): unknown keywords (ADR-003 D3 item 5 — the library
   treats them as annotations), formats outside ADR-003 D4's list, the
   `$schema` value, and the nesting and size limits of ADR-009. The library
   then compiles; a compile error becomes a finding through the same adapter.
4. **Instances are METAIS's data model**, converted for the library as
   `map[string]any`, `[]any`, `string`, `bool`, `nil` and `json.Number` (the
   literal text from the parser), never `float64`, so a number's value is not
   re-rounded.
5. **The adapter** turns `*jsonschema.ValidationError` into ADR-006's
   `schemafail.Failure` tree from `SchemaURL`, `InstanceLocation`,
   `ErrorKind` (with `KeywordPath()` and the kind's structured fields) and
   `Causes`. **It never reads the library's message text**, so a library
   update cannot change a finding's wording, and **it never relies on the
   library's order**: ADR-006's report sorts (the probe showed sibling order
   varies between runs).
6. **Error collection matches what a finding needs.** The library does not
   collect errors in contexts that need only a yes or no (`not`'s subschema,
   `if`, and the branches of an `anyOf` or `oneOf` once one has matched).
   Those are exactly the contexts from which ADR-006 D6 item 10 never
   reports a finding, and ADR-003 D3 item 7 asks for full collection only
   elsewhere: `allOf`, `properties` and the other applicators collect
   fully, and an `anyOf`/`oneOf` with no matching branch reports every
   branch's causes. A vector for each composition keyword proves what is
   reported; if one shows a finding ADR-006 requires that the library does
   not give, D3's fallback is taken for that keyword.
7. **The JSON-Schema-Test-Suite** (D14 item 4) runs over `internal/schema`'s
   configured compiler for the 2020-12 required tests, minus a committed list
   of the cases METAIS's own rules change (unknown keywords refused, formats
   asserted, `$ref` outside the package refused), each with its ADR-003 item.

### D6 — YAML: `go.yaml.in/yaml/v3` node trees under METAIS's profile

**The library tokenizes and builds nodes; `internal/yamlx` enforces ADR-002
D2's profile and resolves every scalar itself.**

1. **Entry point**: `yamlx.Parse(name string, src []byte, lim) (*yamlx.Doc,
   []finding.Finding)`. It runs a `yaml.Decoder` into `yaml.Node`, then calls
   `Decode` once more and requires `io.EOF`, so **a second document is
   refused** (the library's `Unmarshal` would drop it silently).
2. **Over the node tree** it refuses alias nodes and anchors, explicit tags
   outside the profile, merge keys, non-string keys and duplicate keys (the
   node tree does not check them), and it **resolves plain scalars by YAML
   1.2.2 Core Schema rules as ADR-002 D2 narrows them**, reading
   `Node.Value` and `Node.Style`; the library's `Tag` is never trusted for a
   type (`0777` and `2026-01-02` would otherwise be 511 and a `time.Time`).
3. **Positions**: every value and key keeps `Node.Line` and `Node.Column`
   (the column converted to code points, ADR-006 D13 item 2).
4. **Bounds**: ADR-009's size limit applies before parsing; its depth limit
   must be below the library's own 10,000, and `yamlx` checks it on the node
   tree. Aliases are refused, so the library's alias-expansion limit is a
   second line, not the defence.
5. **One profile implementation** serves the manifest (ADR-002), YAML inputs
   (ADR-003 D10) and frontmatter (ADR-003 D13).
6. **v3 is frozen upstream.** The node-tree API it exposes is the one v4
   keeps; the move to v4 is taken once a final v4.0.0 exists and passes the
   YAML vectors, as one change confined to `internal/yamlx`.

### D7 — JSON: METAIS's own strict reader; RFC 8785 through `gowebpki/jcs`

1. **`jsonx.Parse(name, src, lim) (jsonx.Doc, []finding.Finding)`** is a
   strict RFC 8259 scanner written here, because every requirement ADR-003
   D8–D9 places on it is one `encoding/json` hides: invalid UTF-8 located by
   line and byte, an escaped unpaired surrogate or noncharacter refused, every
   duplicate member name reported, each number's literal text kept, a byte
   offset for every value and key (ADR-006 D13 item 2), and a depth limit set
   by ADR-009 rather than the library's 10,000. It produces the same
   `value` tree and position table as `yamlx`, so the manifest and inputs
   behave identically from either form.
2. **Canonical JSON** (ADR-002 D9): `jsonx.Canonical(v) []byte` serializes the
   data model to JSON text with METAIS's encoder and passes it through
   `jcs.Transform`. RFC 8785's output is fully determined by the standard, so
   using a library here keeps D3 item 2. The RFC's own examples are unit
   tests.
3. **Machine output of the command line** (ADR-008's envelope) uses
   `encoding/json` with `SetEscapeHTML(false)` over structs whose field order
   is the declared member order; no `map` is marshalled in machine output
   (a tripwire forbids `map[string]` in the envelope types).

### D8 — CSV: METAIS's own RFC 4180 reader

**`internal/inputs/csv` implements ADR-003 D11 directly; `encoding/csv` is
not used to read inputs.**

1. **Why not `encoding/csv`**: it cannot distinguish `""` from an empty
   unquoted field (ADR-003 D11 item 5), it does not remove a byte order mark,
   and after an unterminated quote it consumes the rest of the file. The
   reader ADR-003 specifies is small and fully defined by that ADR.
2. **Input** is the file after ADR-003 D8's UTF-8 check, byte-order-mark
   removal and line-break normalization. **Output**: the header, then records
   whose fields are `{Text string; Quoted bool; Line, Col int}`, and every
   structural finding the reader can reach (an unterminated quote stops the
   reader; every other finding lets it continue at the next record).
3. **Delimiter** is one of ADR-003 D11 item 2's set, as a byte.
4. **Writing CSV** (the `csv` filter) is METAIS code too (D3 item 2), with
   the quoting rule ADR-004 states.
5. **Tests**: every bullet of ADR-003 D11 item 1 as a table test; a fuzz
   target asserting that the reader never panics and that, for any record set
   the `csv` filter writes, reading it back yields the same fields and
   quotedness.

### D9 — Markdown: `goldmark/v2` for the outline; frontmatter split by METAIS

1. **`markdown.SplitFrontmatter(src) (front []byte, frontLine int, body
   []byte, finding)`** implements ADR-003 D13 item 2 in METAIS code: the first
   line exactly `---`, the next line exactly `---` closes it, after line-break
   normalization. A library is not used: the rule is a dozen lines of
   specification, and the common frontmatter libraries add TOML and JSON forms
   the specification does not have.
2. **`markdown.Outline(src) []Heading`** (ADR-008's `outline`) parses with
   `goldmark/v2`'s CommonMark parser and walks `*ast.Heading` nodes, giving
   level, text (inline content flattened to its literal characters) and the
   line computed from `Node.Pos()`. The frontmatter is split off first and
   line numbers are offset by its length, so a heading's line is its line in
   the file.
3. **No Markdown is rendered.** The tool converts no Markdown to HTML; R4
   item 3's `body` is a string (ADR-003 D13 item 3).
4. **Tests**: the CommonMark heading examples relevant to ATX and setext
   headings, fences and containers, plus the probe's cases; a fuzz target
   over `Outline`.

### D10 — Regular expressions and the argument parser

1. **Go's `regexp` (RE2) is the only regex engine** — for `grep` and for
   JSON Schema `pattern`/`patternProperties`. Its linear-time guarantee is the
   ReDoS defence ADR-009 relies on; a backtracking engine bounded by a
   wall-clock timeout would make a validation's outcome depend on the
   machine's speed.
2. **`regexprofile.Compile(pattern) (jsonschema.Regexp, error)`** reads the
   pattern as ECMA-262 `u`-mode syntax with METAIS's own tokenizer, refuses
   every construct ADR-009 D10 item 6 refuses (lookaround, backreferences,
   repeat bounds above 1000, RE2-only syntax such as `(?i)`, `\A`, `\z`,
   `\Q…\E` and `(?P<…>)`), each named in `schema.pattern_unsupported`,
   translates the rest to RE2 by ADR-009's table (`\uXXXX` and `\u{…}` to
   `\x{…}`, `.`, `\s`, `\S`, `[]` and `[^]` to their ECMA-262 classes),
   then parses the result with `regexp/syntax`, checks `regex-program`, and
   compiles. The translated tree is what `sample` walks to generate a
   matching string (ADR-007 D9 item 18). `grep` patterns are RE2 as written
   (ADR-008 D5) and skip the translation.
3. **`internal/cli/args`** is METAIS's own parser, because the behaviour
   ADR-008 specifies is not the standard `flag` package's (it stops at the
   first operand) nor `pflag`'s (shorthand grouping, a repeated flag silently
   wins). It supports long flags (`--name value`, `--name=value`), ADR-008
   D3 item 2's closed list of short aliases (`-h`, `-r`, `-C`, `-i`; `-C 3`
   and `-C3`) with no grouping, flags interspersed with operands, `--` as
   the end of options, `-` as an operand, no abbreviation, a refusal on a
   repeated single-valued flag, repeatable flags declared as such, and one
   declaration per flag from which both the parser and the help printer
   read.

### D11 — Files, time, randomness, text, and the environment

1. **Reading a package**: a packed package is read through `internal/zipfmt`'s
   own strict reader of ADR-001 D1's profile — end of central directory,
   central directory, and every local header compared field by field with its
   central entry — using a `ReadAt` over the file and `compress/flate` over a
   byte-counting `io.ByteReader`, so the decoder never reads ahead and "ends
   exactly at the compressed size" is checkable (ADR-001 Interfaces). CRC-32
   is `hash/crc32`. **`archive/zip` is not used for reading**, which also
   removes `zipinsecurepath` from the tool's behaviour. A source directory is
   read through `os.Root`, refusing every non-regular file by `Lstat` before
   it is opened (ADR-001, ADR-008). Both are one interface,
   `pkgtree.Tree{List, Open, Stat}`, and nothing else reads a package.
2. **Writing a zip** (ADR-001 D4): `zipfmt.Write(w io.Writer, entries
   []Entry)` deflates each entry into memory with `compress/flate` at
   ADR-001's level, computes CRC-32 and sizes, and writes every header field
   from ADR-001 D1's table through `zip.Writer.CreateRaw` (or directly; the
   bytes are the same and the test compares them). `Modified` is never set,
   so no extended-timestamp field is written; bit 3 is never set, so no data
   descriptor is written.
3. **Every host file access** goes only through `internal/hostfs`: reading
   a package file, a source directory and a data directory under `os.Root`
   (ADR-009 D7), and writing outputs, built on `os.Root` for the staging or
   output directory, `Root.MkdirAll` and `Root.Rename`, and the no-replace
   rename ADR-005 D14 names — `renameat2(RENAME_NOREPLACE)` on linux and
   `renamex_np(RENAME_EXCL)` on darwin, from `golang.org/x/sys/unix`, and
   `MoveFileW` on windows, from the standard library's `syscall.MoveFile` —
   each in a build-tagged file. `MoveFileW` takes no flags and fails on any
   existing target, file or directory, which is the refusal ADR-005 D14
   asks of `MoveFileExW` without `MOVEFILE_REPLACE_EXISTING`. *(A1,
   proposed)* It names staging directories and temporary files as
   ADR-005 D14 fixes them, from a SHA-256 of a name, never from a random
   source or the process. **No code outside `internal/hostfs` opens,
   creates, renames or removes a file**, and `hostfs` removes only entries
   it created for the current render (ADR-005 D14 item 5) and the stale
   expected files `test --update` replaces (ADR-007 D5 item 5); a tripwire
   checks both.
4. **`GODEBUG` cannot change behaviour**: with `archive/zip` reading gone,
   the remaining archive code is `archive/tar` writing (no insecure-path check
   applies to writing) and `zipfmt` writing. A host-variation run (D13) sets
   `GODEBUG=zipinsecurepath=0,tarinsecurepath=0`, ADR-011 D12's environment
   A sets the opposite, and both expect identical results (ADR-009 D14
   item 2).
5. **Time**: the tool never calls `time.Now`, never uses `time.Local` and never
   loads a time-zone database. The only instant a render sees is `--now`,
   parsed in `internal/dates` with `time.Parse(time.RFC3339, …)` into a fixed
   offset. Formatting is METAIS's own `strftime` subset (ADR-004 D11), over
   the value's own offset.
6. **Randomness**: none. `math/rand`, `math/rand/v2` and `crypto/rand` are
   forbidden outside tests, with no exception. The sample generator's seeded
   generator is METAIS code implementing the SplitMix64 algorithm ADR-007 D9
   specifies, because the standard library does not promise its generators'
   outputs across releases.
7. **Text**: `golang.org/x/text/unicode/norm` for the NFC check and NFD,
   `golang.org/x/text/cases` (`Fold`, full case folding) for the canonical
   caseless match ADR-001 D2 item 4 and ADR-005 D12 item 6 use, and
   `unicode` for general categories — all Unicode 15.0.0 under Go 1.26 and
   pinned by the toolchain (D1 items 3 and 4). No code depends on the
   locale; `strings.ToUpper` and friends are Unicode-defined and locale-free
   in Go, and the host-variation run sets `LC_ALL=tr_TR.UTF-8` to prove it.
8. **The environment**: `internal/hostenv.Isolate` is the only caller of
   `os.Getenv`, `os.LookupEnv`, `os.Environ` and `os.Clearenv` in the module.
   It reads `NO_COLOR` only (ADR-008 D12), clears the environment and sets
   `time.Local` to UTC (ADR-009 D14 item 1); `cmd/metais` calls it before
   anything else. `internal/hostenv` is also the only caller of
   `term.IsTerminal` on darwin and linux and of `syscall.GetConsoleMode` on
   windows, which is the call `golang.org/x/term` makes there. *(A1,
   proposed)*
9. **Forbidden in non-test code**: ADR-009 D16 item 3's tripwire,
   `TestNoHostFacts` in `internal/guarantees/`, over the `go/ast` of every
   non-test file, with ADR-009 D12's symbol lists plus these: `os.Getwd`,
   `os.Hostname`, `os.UserHomeDir`, `os.UserConfigDir`, `os.UserCacheDir`,
   `os.TempDir`, `os.Getuid`/`Geteuid`/`Getgid`/`Getpid`/`Getppid`,
   `os.Executable`, `os.ExpandEnv`, `os.Expand`, `os/user`, `time.Now`,
   `time.Local`, `time.LoadLocation`, `time.Since`, `time.Until`,
   `runtime.GOOS` (behaviour must not vary by platform; build-tagged files
   are the one sanctioned variation, and only in `hostfs` and `hostenv`),
   `unsafe`, `reflect` outside `internal/schema`'s adapter, and the `os`
   calls listed in item 8 outside `hostenv`. The allow-list is ADR-009 D16
   item 3's, each entry with its reason: `internal/hostenv.Isolate`,
   `internal/hostfs`, `cmd/metais` and `internal/dates`.
10. **Forbidden in the binary's dependency closure**, checked by the same
    tripwire over `go list -deps ./cmd/metais`: `net` (the socket package;
    `net/url` and `net/netip` are pure parsers and allowed — the JSON Schema
    library and goldmark import them), `net/http` and below, `net/rpc`,
    `net/smtp`, `crypto/tls`, `os/exec`, `os/signal`, `plugin`, `os/user`,
    `log/syslog`, `runtime/cgo`, `syscall/js`. The probe found none of them
    in the closure of a program importing every admitted module at the
    versions of D3's table, built for darwin and linux. Built for windows,
    that closure holds `net`, through `golang.org/x/sys/windows`, which is
    why D3 item 3 links neither `golang.org/x/sys` nor `golang.org/x/term`
    there. *(A1, proposed)*

### D12 — Build, release and the tool's version

1. **Static build**: `CGO_ENABLED=0 go build -trimpath -buildvcs=true
   -ldflags "-s -w -X github.com/kottera/metais/internal/version.release=<semver>"
   ./cmd/metais`. `-trimpath` keeps file paths of the build machine out of the
   binary (and so out of any panic text a finding could carry); `-s -w` drop
   the symbol table and DWARF, keeping the function table panics need.
2. **Targets shipped**: `darwin/amd64`, `darwin/arm64`, `linux/amd64`,
   `linux/arm64`, with `GOAMD64=v1` and `GOARM64=v8.0` set explicitly.
   **Compiled but not shipped**: `windows/amd64` and `windows/arm64` — `go
   build` and `go vet` for both are part of the gate (D15), so nothing that
   precludes Windows can land unnoticed.
3. **The version string** (ADR-002 D8's `tool.version`, a semantic version
   without a leading `v`):
   - a release build sets `internal/version.release` by `-X`; it is the tag
     without its `v`;
   - otherwise `version.String()` takes `runtime/debug.ReadBuildInfo().Main.Version`
     when it is a semantic version and has no `+dirty` suffix (so `go install
     github.com/kottera/metais/cmd/metais@v1.0.0` reports `1.0.0`);
   - otherwise `0.0.0-devel`.
   The provenance tag records this string (ADR-004), and `test` records
   `test` instead (ADR-004 D15). The conformance runner drives the built
   binary as a black box and substitutes the version it reports for ADR-011
   D10 item 3's `@@TOOL_VERSION@@` token, so no vector depends on it.
   In-process unit tests may fix it through `version.Override`, which only
   test files may call (a tripwire checks); no flag or environment variable
   sets it.
4. **Reproducible**: the release builder builds each target twice, in two
   fresh directories with separate `GOCACHE`s, and refuses to publish unless
   the two binaries' SHA-256 are equal. It also refuses a dirty working tree,
   a tree whose `HEAD` is not the tag, and a binary whose `go version -m`
   does not show `CGO_ENABLED=0`, `-trimpath=true` and the expected
   `GOOS`/`GOARCH`.
5. **Release artefacts**, written by `go run ./internal/tools/release
   -version <semver>` into `dist/`:
   - `metais_<version>_<os>_<arch>.tar.gz` per shipped target, holding
     `metais`, `LICENSE`, `NOTICE` and `THIRD-PARTY-NOTICES.txt`, written
     deterministically by `archive/tar` and `compress/gzip` (entries sorted,
     mode 0755 for the binary and 0644 otherwise, uid and gid 0, empty owner
     names, ADR-001's fixed instant, gzip header with no name and zero
     mtime);
   - `SHA256SUMS`, one line per archive.
   Publishing (a GitHub release or otherwise) is outside this ADR; the
   repository is private.
6. **`THIRD-PARTY-NOTICES.txt`** is committed at the repository root and holds
   each linked module's licence text. `internal/guarantees/notices_test.go`
   fails when a module in the binary's closure (`go list -deps -f
   '{{.Module}}' ./cmd/metais`) has no entry or an entry names a module no
   longer linked.
7. **Versioning of the tool** is independent of the specification's
   (ADR-002 D8 item 4): semantic versioning over the command line's
   documented behaviour and its JSON output's `/1` shape (ADR-008).

### D13 — Proving the headline guarantees: a detector, and a fault it must catch

**Each headline guarantee has a detector test and a deliberately broken tool
the detector must fail on, and both run in every `go test`.**

1. **Why both run every time.** "Pin the property false first" proves once
   that a test can fail. A detector that later goes blind — a sentinel string
   no longer planted, a directory snapshot taken in the wrong place — passes
   silently. Running the detector against a broken tool in the same test run
   makes the red permanent: a blind detector fails the build.
2. **Faults are injected by dependency injection, never by flags.**
   `internal/fault` defines `type Set uint64` and named faults. The pipeline's
   options struct carries a `fault.Set`; a seam consults it with `if
   opts.Faults.Has(fault.X)`. `internal/cli` builds options with a zero
   `fault.Set` and no command, flag or environment variable sets one; a
   tripwire asserts that no non-test file outside `internal/fault` and the
   seams names a fault constant, and that `internal/cli` never assigns the
   field. Host isolation (R14 item 4, SC7) is proved instead by ADR-009
   D16's subprocess harness and its build-tagged canaries, because a leak
   through the process environment, the working directory or the clock is
   only realistic in a separate process, which no injected option reaches.
   The seams are few and named here:

   | Guarantee | Fault (`fault.…`) | Seam |
   |---|---|---|
   | confinement (R12 item 1, SC6) | `SkipConfine`: the output path check returns success | `internal/hostfs` |
   | digests verified (R1 item 5) | `SkipDigest`: verification returns success | `internal/pkgtree` |
   | read in place (R2 item 1) | `UnpackToTemp`: browse extracts the entry to a temporary file and reads it back | `internal/browse` |
   | determinism (R10 items 2–3) | `MapOrder`: object iteration follows a Go map | `internal/liquid/value` |
   | every finding at once (R6 item 1, SC8) | `FirstFinding`: the report keeps only the first error | `internal/finding` |

3. **The harness runs the tool in-process.** `internal/cli.Run(ctx,
   cli.Invocation{Args, Stdin, Stdout, Stderr, Options})` is what `Main`
   calls, so a test drives every command with real files, real environment
   (`t.Setenv`), a real working directory (`t.Chdir`) and real `HOME`, and
   can add a fault. Tests that set the environment or working directory do
   not use `t.Parallel`.
4. **The detectors** (in `internal/guarantees/`):
   - **Host isolation** (SC7): ADR-009 D16's `TestHostIsolation` and
     `TestHostIsolationCanaries`, over the built binary, with its canaries as
     the controls.
   - **Confinement** (SC6): every render runs in `<tmp>/parent/out`; the tree
     under `<tmp>` is snapshotted (paths, types, sizes, SHA-256) before and
     after, and the only differences allowed are under `out`. Control:
     `SkipConfine` over the hostile corpus's traversal members must be
     detected.
   - **Digests**: every digest-mismatch member of the hostile corpus is
     refused. Control: `SkipDigest` must let at least one through.
   - **Read in place**: browse commands run with `TMPDIR`, the working
     directory and `HOME` pointing at snapshotted directories; nothing may
     appear. Control: `UnpackToTemp` must be detected.
   - **Determinism**: every vector renders three times in one process with
     `GOMAXPROCS` 1 and `runtime.NumCPU()`, and once in a fresh process of the
     built binary, under `TZ=Pacific/Chatham`, `LC_ALL=tr_TR.UTF-8`, `umask
     077` and `GODEBUG=zipinsecurepath=0,tarinsecurepath=0`; every output's
     SHA-256 matches the baseline (ADR-005 D19). Control: `MapOrder` over a
     vector iterating a 20-member object must be detected (the probe saw 14
     orders in 20 runs, so three renders detect it with overwhelming
     probability; the control uses 20).
   - **One pass**: the planted-errors inputs of SC8 report every planted
     error. Control: `FirstFinding` must be detected.
   - **No network, no execution**: the import tripwire of D11 item 10.
     Control: the same check run over `internal/guarantees/testdata/netleak/`,
     a main package importing `net`, must flag it.
   - **Host APIs**: the `go/ast` tripwire of D11 item 9. Control: a fixture
     file in `testdata/` calling `os.Getenv` must be flagged.
5. **The subprocess runs**: the conformance runner (ADR-011 D11) and the
   host-isolation harness drive the built binary, so a difference between
   `Main` and `Run` cannot hide behind the in-process harness.

### D14 — Test organisation

1. **Unit tests** live beside the code (`*_test.go` in each package), are
   table-driven, use `internal/testenv` for temporary directories and a
   temporary `HOME`, and never read the developer's configuration.
2. **The conformance runner** is `internal/conformance`, as ADR-011 D11
   designs it: a black box that builds `./cmd/metais` once and runs it as a
   subprocess per step in a sentinel sandbox, never linking the
   implementation, with `TestConformance` walking
   `spec/conformance/vectors/` (ADR-011 D9 item 1) and running each vector
   as a subtest named by its path. A vector passes or the test fails; there
   is no exclusion list, and the only skips are ADR-011 D9 item 6's
   `requires` probes, each reported. The hostile corpus is vectors (ADR-011
   D16), so the same runner proves SC6.
3. **Platform runs** (SC5): the runner writes ADR-011 D15's cross-platform
   record (`go test ./internal/conformance -args
   -conformance.record=<file>`) and compares records with
   `-conformance.compare`; the procedure is ADR-011's.
4. **Third-party corpora**, vendored as test data with their licence and a
   `SOURCE` file naming the repository and commit:
   - `internal/liquid/testdata/golden-liquid/` — `golden_liquid.json` from
     jg-rp/golden-liquid at `65c2f76` (MIT), with `exclusions.txt` (D4 item 7);
   - `internal/schema/testdata/json-schema-test-suite/` — the `tests/draft2020-12/`
     tree of json-schema-org/JSON-Schema-Test-Suite at the commit taken when
     the work item lands (MIT), with its exclusion list (D5 item 7).
   Updating either is a deliberate change that re-reads the exclusions.
5. **Golden files**: the zip canary (D1 item 3), the tar layout (ADR-005 D17:
   the exact bytes `archive/tar` writes for a ustar-fitting name and for a
   pax-requiring one), and the RFC 8785 examples.
6. **Fuzz targets** (Go native fuzzing; their seed corpora under
   `testdata/fuzz/` run in every `go test`): `FuzzZipRead`, `FuzzYAMLProfile`,
   `FuzzJSONParse`, `FuzzCSVRead`, `FuzzFrontmatter`, `FuzzOutline`,
   `FuzzLiquidParse`, `FuzzLiquidRender` (with the meter at small limits),
   `FuzzRegexProfile`, `FuzzPortablePath`. Each asserts no panic and no
   recovered internal error; the parsers also assert that positions are
   within the input. Longer fuzzing (`go test -fuzz`) is a developer step.
7. **Bounded-work tests count, they do not time** (AGENTS.md): a test that
   proves a limit holds asserts the meter's counters or the bytes read, never
   wall time; where a duration must be asserted it is the process's CPU time
   (`syscall.Getrusage` in the test only).
8. **Race detection**: the gate's `go test -race ./...` runs with cgo enabled
   (the race detector requires it); the static build is checked separately
   (D15). The tool has no concurrency of its own (D2 item 3), so `-race`
   chiefly guards the test harness.

### D15 — Benchmarks and the gate

1. **Benchmarks** (`Benchmark…` with `b.ReportAllocs()`) exist for: Liquid
   parse and render over fixtures in `internal/liquid/testdata/bench/` (a
   10,000-row table through `markdown_table`, a 500-page site over a
   collection with `render` partials, deeply nested conditionals); JSON, YAML
   and CSV parsing of a 10 MiB input; schema validation of a 10,000-item
   collection; zip reading of a package with 10,000 entries; `pack` of the
   §4.1 example. Results are compared by hand with `benchstat` against a
   baseline committed beside each benchmark when its work item lands.
2. **Initial budgets**, on an Apple M-series or comparable 2024 x86-64 core,
   CPU time, each recorded in the implementation log when first measured and
   revised there by evidence: rendering the 10,000-row table ≤ 200 ms;
   `inspect` of the 10,000-entry package, whole-package verification
   included, ≤ 500 ms; validating the 10,000-item collection ≤ 500 ms. Two
   budgets are deterministic and are tests, not benchmarks: **opening a
   packed package reads every byte of the file once and decodes every entry
   once** (ADR-001 D8; a counting `ReaderAt` asserts the bytes read), and
   **`read --lines a-b` decompresses its entry at most once more after
   verification**.
3. **The gate** (AGENTS.md's list, extended by this ADR; the AGENTS.md text is
   updated by the work item that creates `go.mod`):

   ```bash
   go build ./...
   go vet ./...
   go test -race ./...
   go mod tidy -diff
   gofmt -l .
   CGO_ENABLED=0 go build -trimpath ./cmd/metais
   for t in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64; do
     GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go vet ./... && \
     GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go build ./...
   done
   go generate ./internal/specdata && git diff --exit-code internal/specdata
   ```

## Interfaces with sibling ADRs

**ADR-001 — the container and its layout.** Relied on: D1's every-field-fixed
profile with no data descriptors, no zip64, no extra fields and local headers
equal to central entries (that is what makes an own reader small);
whole-package verification at every open (ADR-001 D8); D4's
`flate.BestCompression`; identity across builds by package digest.
Promised: the strict reader and the `CreateRaw`-based writer of D11 items
1–2; the byte-counting `io.ByteReader` under the decoder; the NFC, full
case-folding and category code in `internal/unitext` with the
Unicode-version tripwire (D1 items 3 and 4); the deflate canary. The
specification's Unicode version is 15.0.0, which Go 1.26 carries (D1 item
4).

**ADR-002 — the manifest and versioning.** Relied on: D2's profile (single
document, no anchors, aliases or tags, METAIS's own scalar resolution), the
choice of profile by a `metais.yaml`'s first byte (ADR-002 D1), D9's RFC
8785 canonical form, D8's `tool.version`. Promised: `yamlx` over
`go.yaml.in/yaml/v3` nodes with duplicate-key detection and positions (D6);
`jsonx` with duplicate names and positions (D7); `gowebpki/jcs` for RFC
8785; the version rules of D12 item 3; the manifest schemas embedded
through `internal/specdata` with the byte-identity test ADR-002 D10 item 5
asks for.

**ADR-003 — the data specification.** Relied on: `metais:///` retrieval URIs
and `$id` canonical URIs (D5); the asserted-format list (D4); the regex
profile reference to ADR-009 (D3 item 6); the CSV rules (D11), frontmatter
rule (D13) and number rules (D8 item 5); error collection outside the
boolean-only contexts (ADR-003 D3 item 7). Promised: the closed compiler of
D5, METAIS's own format validators, unknown-keyword detection before
compilation, the own CSV reader of D8 with BOM removal and quoted/unquoted
distinction, the strict JSON reader of D7 with literal number text.

**ADR-004 — the template environment.** Relied on: the pinned Shopify
Liquid 5.14.0 dialect under `strict2` (ADR-004 D1), whose tag and filter
lists this ADR verified; its object iteration order (RFC 8785), number
rendering, stable sorting, `date` directives over a value's own offset,
`now` from `--now` only, strict variables and filters, `and`/`or` grouping
(ADR-004 D2), one syntax finding per template (ADR-004 D16 item 4), and the
`include` refusal's wording. Promised: an engine whose `Core()` is exactly
that dialect, METAIS filters and tags registered from `internal/template`,
an escaper hook per output, engine error codes that `internal/template`
maps one-to-one onto ADR-004 D16's `template.*` codes, and the walkable AST
ADR-007 analyses.

**ADR-005 — outputs, transforms and rendering.** Relied on: D3's
`ParseCondition`/`ParseValue` contract, D14's `os.Root`, fixed staging
names and no-replace rename, D17's tar profile, D19's proofs. Promised:
`liquid.ParseValue`, `liquid.ParseCondition`, `liquid.Eval` and
`liquid.EvalCondition` (D4 item 3); `golang.org/x/sys/unix` for
`renameat2` and `renamex_np`, and the standard library's `syscall.MoveFile`
(`MoveFileW`) on windows, inside `internal/hostfs` *(A1, proposed)*; the tar golden
test proving D17's claim about `archive/tar`; D19's host-variation and
repetition runs as D13's determinism detector; Go 1.26, which has
everything ADR-005 needs (`os.Root` with `Rename` and `MkdirAll`, Unicode
15.0.0 tables).

**ADR-006 — validation and findings.** Relied on: `internal/finding` and
`internal/schemafail` as specified in D13; the boolean-only contexts of
ADR-006 D6 item 10. Promised: those package names; an adapter from
`jsonschema.ValidationError` that never reads message text and never
relies on order (D5 item 5); positions from every parser in the
`Positions` shape, columns in code points; panics recovered as
internal-class findings (D4 item 6), with no stack trace on any stream.

**ADR-007 — authoring.** Relied on: the cross-reference walking the engine's
AST (ADR-007 D3 item 1); the sample generator's algorithm and PRNG
(ADR-007 D9). Promised: a walkable AST with expression trees, binding
constructs marked, and byte spans, lines and columns for every token (D4
items 2–3); `regexprofile`'s translated `regexp/syntax` tree, reusable for
pattern generation (D10 item 2); no standard-library PRNG (D11 item 6); the
packages `internal/author`, `internal/xref` and `internal/sample`.

**ADR-008 — the command line.** Relied on: D12's environment rule (`NO_COLOR`
only), the parser behaviour of D3 (long flags, the closed short-alias list,
no grouping, no abbreviation). Promised: `internal/cli/args` (D10 item 3),
`internal/hostenv` as the single reader of `NO_COLOR` and of the terminal
state, goldmark v2 for `outline` and the references index (CommonMark
0.31.2), Go `regexp` for `grep`, `encoding/json` with HTML escaping off and
declared member order, the version string of D12 item 3.

**ADR-009 — the security model.** Relied on: every limit with its default,
including `render-work`, `loop-iterations`, `value-size`, `array-size`,
`assign-bytes`, `partial-depth` and `template-nesting`, and parser depths
(`value-depth`) below the libraries' 10,000; the package regex profile and
its translation table (ADR-009 D10 item 6); the host-isolation harness,
canaries and tripwire (ADR-009 D16); the hostile corpus. Promised:
`internal/limits` mirroring its table; the meter of D4 item 4 counting
rather than timing; `internal/hostenv.Isolate` and `internal/hostfs` as the
only packages touching the environment and the host filesystem; D13's
detectors with permanent fault controls for the guarantees proved
in-process; D11's tripwire lists, merged into ADR-009's `TestNoHostFacts`;
the `GODEBUG` result of D11 item 4.

**ADR-011 — the specification and conformance.** Relied on: the vector root
`spec/conformance/vectors/` and format (ADR-011 D9), the black-box runner
(ADR-011 D11), the cross-platform record (ADR-011 D15), the hostile corpus
among the vectors (ADR-011 D16), and ADR-011 D10 item 3's tokens for the
tool's name and version in provenance. Promised: the runner in
`internal/conformance` in `go test`, with no exclusion list; the embedded
schema copies under `internal/specdata` with their equality test; the
liquid-spec JSON-RPC server as a test-only program under `internal/`; the
reference driver for `TestDriverProtocol`; the programs that produce the
committed hostile packages, with their reproduction tests; and spec files
kept free of Go code (D2 item 4).

## Conformance

This ADR adds no normative statement of the format; its rules are about the
reference tool. The vectors that exercise implementation choices, and so must
exist for this ADR's decisions to be checked (ADR-011 owns their format):

- **Liquid**: every tag and filter of ADR-004's dialect, including the ones
  `osteele/liquid` lacks (`echo`, `liquid`, `#`, `increment`, `decrement`,
  `ifchanged`, `doc`, `base64_*`, `find`, `find_index`, `has`, `reject`,
  `squish`), `blank` and `empty` comparisons, `for` over an object (order),
  loop modifiers in combination, `offset: continue`, number rendering
  (`1.0`, `6.0`), `"now"` not reading the clock, and a meter limit reached at
  a stated node.
- **YAML**: a second document refused; `0777`, `2026-01-02`, `yes` resolved
  by the profile, not the library.
- **CSV**: `""` versus empty; a byte order mark; CRLF inside a quoted field.
- **JSON**: duplicate names, an unpaired surrogate escape, invalid UTF-8 at a
  stated line.
- **Schema**: a `$ref` to `file:` and to `https:` refused; a pattern outside
  the profile refused; several errors reported in a stable order.
- **Determinism across builds**: none — byte identity across builds is not
  promised (ADR-001 D4 item 6); the canary is a test, not a vector.

## Alternatives rejected

### Use `osteele/liquid` and work around its gaps

Every gap that matters is inside the library: map iteration in `for`, absent
`blank`/`empty`, absent core tags, the clock and local zone in `ParseDate`, no
limits, no expression tree. Wrapping cannot reach them, re-registering filters
cannot remove non-dialect ones, and forking is copying another engine's code.
Upstream contributions are welcome work but cannot be a dependency of v1's
correctness, and the loop-modifier difference is a deliberate upstream choice.

### Use `Notifuse/liquidgo`

The closest in behaviour, and still a quarter of Golden Liquid failing; it
reads the clock and the local zone, has no usable module version, and has been
idle since January 2026. Adopting it would mean owning a fork of a port.

### Keep a Liquid library and add a second expression parser for analysis

Two parsers of one grammar disagree at the edges, and the cross-reference
check would then approve templates the renderer reads differently.

### `kaptinlin/jsonschema`

Pre-1.0, requires Go 1.27 in its `go.mod`, registers network loaders by
default, and brings several dependencies. `santhosh-tekuri/jsonschema/v6`
has the structured error tree ADR-006 is designed around.

### `goccy/go-yaml`

It accepted the alias bomb the probe gave it, and offers nothing METAIS's
profile needs that the `go.yaml.in` node tree lacks.

### `go.yaml.in/yaml/v4` now

It has no final release after six release candidates; the node API METAIS uses
is the same, so the move is cheap once it is final (D6 item 6).

### `encoding/csv` with a pre-pass

A pre-pass that tells `""` from an empty field and strips the byte order mark
is most of a CSV reader already; two readers of one file can disagree.

### `dlclark/regexp2` for ECMA-262 fidelity

A backtracking engine bounded by `MatchTimeout` makes whether a value is
valid depend on how fast the machine is; RE2's linear time with a profile that
refuses non-shared constructs keeps validation deterministic.

### `spf13/pflag` or `cobra`

`pflag` groups shorthands and lets a repeated flag win silently, both refused
by ADR-008; `cobra` adds a framework, completion generators and, through
`viper` if used, configuration from the environment.

### Vendoring a pinned deflate encoder so pack bytes survive toolchain upgrades

It would copy a compressor's code into the repository, narrow but not remove
the caveat, and buy nothing the package digest does not already give
(ADR-001 D4 item 6).

### Fault injection by build tag for the in-process guarantees

A `metaisfault` build tag compiles a different program from the one shipped;
the negative controls would run only when someone remembers the tag. Options
injection keeps one program and runs the controls in every `go test`. The
one place build tags are used is ADR-009 D16's host-isolation canaries,
because a leak through the process's environment must be shown in a
separate process, which an injected option cannot reach; their test builds
every canary in every `go test`, so they cannot be forgotten either.

### Faults or test switches reachable from the command line or the environment

Any switch a caller can reach is a switch a package's caller can be tricked
into, and an environment switch contradicts R14 item 4's spirit.

## Consequences

### Positive

- The template engine, the part most likely to surprise, is METAIS's: its
  dialect, iteration order, limits and errors are exactly what the
  specification says, and ADR-007's analysis uses the parser the renderer
  uses.
- The tool cannot read the host or the network by construction, and the
  tests that say so fail whenever their own detection breaks.
- Every library sits behind a METAIS check, so an update can change no
  accepted input and no emitted byte without a vector noticing.
- The binary is static, reproducible and built the same way on every target.

### Negative

- Writing a Liquid engine is the largest work item of the implementation
  plan and delays the first render until it exists.
- METAIS owns more parsers than a typical tool (Liquid, JSON, CSV,
  frontmatter, argument parsing), each needing fuzzing and maintenance.
- Go 1.26 leaves support when Go 1.28 ships; the move to 1.27 changes the
  deflate canary and, if the specification follows, the Unicode version.
- The in-process harness is only as good as the discipline that keeps
  `os.Std*`, `os.Args` and `os.Exit` in `cmd/metais`; the tripwire enforces
  it.

### Neutral

- Golden Liquid and the JSON-Schema-Test-Suite live in the repository as test
  data, with their licences.
- `archive/zip` is used only for `CreateRaw`, and may be replaced by direct
  header writing without changing a byte.

## What this does not decide

- The Liquid dialect, its filters' semantics, number rendering, iteration
  order and the METAIS filter and tag catalog (ADR-004).
- The zip profile and its constants (ADR-001); the tar profile and the
  staging protocol (ADR-005).
- Limit names and defaults; the regex profile's exact content; the threat
  model (ADR-009).
- The vector format, layout and platform-run procedure (ADR-011).
- The finding codes for engine errors (ADR-006 registers them; ADR-004 words
  them).
- Continuous integration: there is none yet, and the gate of D15 is run by
  hand until there is.
- Where releases are published.

## Proposed rulings

1. **Toolchain: `go 1.26.0`, `toolchain go1.26.9`, and specification 1.0
   pins Unicode 15.0.0, the version Go 1.26 and the `x/text` tables it
   selects carry** (D1; ADR-001 D2, ADR-005 D12). Lean: yes. Why: one
   Unicode version for every rule, the one the toolchain provides, and Go
   1.26 is supported until Go 1.28. Alternative: Go 1.27 now, which gives
   Unicode 17.0.0, `encoding/json/v2`, and a support window to Go 1.29, at
   the price of a different deflate encoder from the start. Moving later
   needs a 17.0.0 minor or METAIS's own 15.0.0 "assigned" table (D1 item
   4).
2. **METAIS implements its own Liquid engine** (D4). Lean: yes, strongly.
   Why: `osteele/liquid` fails iteration order, `blank`/`empty`, core tags,
   host isolation and AST access, none repairable from outside. Alternative:
   adopt `osteele/liquid` and accept a narrower dialect plus a second parser
   for analysis.
3. **Golden Liquid and the JSON-Schema-Test-Suite are vendored as test data**
   with their MIT licences and source commits (D14 item 4). Lean: yes. Why:
   breadth checks a test runner can use offline; they are test cases, not
   engine or validator code. Alternative: no third-party corpora; vectors
   only.
4. **JSON Schema: `santhosh-tekuri/jsonschema/v6` v6.0.3**, closed loader,
   METAIS's own format validators (D5). Lean: yes. Alternative: METAIS's own
   validator now.
5. **YAML: `go.yaml.in/yaml/v3` v3.0.5 node trees**, moving to v4 when final
   (D6). Lean: yes. Alternative: v4 release candidate now.
6. **Own strict JSON reader; RFC 8785 through `gowebpki/jcs` v1.0.2** (D7).
   Lean: yes. Alternative: own RFC 8785 writer too; or Go 1.27's `jsontext`.
7. **Own CSV reader; `encoding/csv` not used for input** (D8). Lean: yes.
   Why: ADR-003 D11 item 5 is not expressible over `encoding/csv`.
8. **goldmark v2 (v2.1.6) for the outline; own frontmatter split** (D9).
   Lean: v2. Why: start positions on every node, the maintained line.
   Alternative: goldmark v1.8.6, more mature, positions from segments.
9. **RE2 is the only regex engine, behind a profile walk** (D10). Lean: yes.
   Alternative: `regexp2` in ECMAScript mode with a timeout.
10. **Own argument parser** (D10 item 3). Lean: yes. Alternative: `pflag`
    with wrappers refusing grouping and repeats.
11. **Own strict zip reader; `archive/zip` only for writing** (D11). Lean:
    yes; ADR-001 already relies on it.
12. **Version string: `-X` in releases, build info otherwise,
    `0.0.0-devel` last** (D12 item 3). Lean: yes. Alternative: build info
    only.
13. **Release artefacts: per-target `tar.gz` with `SHA256SUMS`, built twice
    and compared** (D12). Lean: yes. Alternative: bare binaries.
14. **Faults by options injection, with permanent negative controls in every
    `go test`** (D13). Lean: yes. Alternative: red-first only at the time of
    writing, recorded in the commit.
15. **The render is single-threaded** (D2 item 3). Lean: yes, until a
    benchmark demands otherwise.
16. **Spec files embedded through `internal/specdata` copies with a drift
    test** (D2 item 4). Lean: yes. Alternative: a Go file in `spec/` making it
    an importable package.
17. **`internal/hostfs` is the one package for host files and
    `internal/hostenv.Isolate` the one reader of the environment** (D2, D11;
    ADR-009 D7, D14). Lean: yes. Alternative: separate read and write
    packages (`fsout` for writes only).
18. **The conformance runner is a black box over the built binary** (D14
    item 2; ADR-011 D11). Lean: yes. Why: only a subprocess's environment,
    working directory and home can be planted and scanned. Alternative: an
    in-process runner over `cli.Run`, faster but unable to prove SC7.

## References

- PRD-001: `docs/prd/PRD-001-metais-v1.md`; AGENTS.md.
- Go downloads (current releases go1.27.2, go1.26.9): https://go.dev/dl/?mode=json
- Go release policy: https://go.dev/doc/devel/release
- Go 1.27 release notes (flate output changed; `encoding/json/v2`): https://go.dev/doc/go1.27
- Go 1.24 release notes (VCS version stamping; `os.Root`): https://go.dev/doc/go1.24
- Go 1.25 release notes (`Root.Rename`, `Root.MkdirAll`): https://go.dev/doc/go1.25
- GODEBUG precedence, `zipinsecurepath`, `tarinsecurepath`: https://go.dev/doc/godebug
- Race detector and cgo: https://go.dev/doc/articles/race_detector
- `embed` pattern restrictions: https://pkg.go.dev/embed
- `os.Root`: https://pkg.go.dev/os#Root
- `regexp` linear-time guarantee: https://pkg.go.dev/regexp
- `encoding/csv`: https://pkg.go.dev/encoding/csv
- `archive/zip` (`CreateRaw`, `CreateHeader`, `NewReader`): https://pkg.go.dev/archive/zip ; writer source: https://go.googlesource.com/go/+/refs/tags/go1.26.8/src/archive/zip/writer.go
- `archive/tar` writer (sorted pax records, `PaxHeaders.0`): https://go.googlesource.com/go/+/refs/tags/go1.26.8/src/archive/tar/writer.go
- `compress/flate.NewReader`: https://pkg.go.dev/compress/flate#NewReader
- `encoding/json` nesting limit: https://go.googlesource.com/go/+/refs/tags/go1.26.8/src/encoding/json/scanner.go
- `golang.org/x/text` (v0.43.0; `unicode/norm` tables 15.0.0 and 17.0.0 under `go1.27`): https://pkg.go.dev/golang.org/x/text/unicode/norm
- osteele/liquid (v1.9.2, README, SECURITY.md, loop semantics): https://github.com/osteele/liquid ; https://github.com/osteele/liquid/blob/main/SECURITY.md ; https://github.com/osteele/liquid/blob/main/docs/loop-semantics.md ; `values/parsedate.go`: https://github.com/osteele/liquid/blob/v1.9.2/values/parsedate.go
- Notifuse/liquidgo: https://github.com/Notifuse/liquidgo
- Shopify Liquid v5.14.0 (tags, `standardfilters.rb`, `block.rb` `MAX_DEPTH`, `resource_limits.rb`): https://github.com/Shopify/liquid/tree/v5.14.0/lib/liquid
- Golden Liquid (commit 65c2f76, Gemfile.lock at liquid 5.12.0): https://github.com/jg-rp/golden-liquid
- santhosh-tekuri/jsonschema v6.0.3 (README, `loader.go`, `format.go`, `suite_test.go`): https://github.com/santhosh-tekuri/jsonschema
- Bowtie JSON Schema implementation reports: https://bowtie.report
- JSON-Schema-Test-Suite: https://github.com/json-schema-org/JSON-Schema-Test-Suite
- kaptinlin/jsonschema v0.9.9: https://github.com/kaptinlin/jsonschema
- go.yaml.in/yaml (README version intentions; v3.0.5; v4.0.0-rc.6): https://github.com/yaml/go-yaml
- goccy/go-yaml v1.19.2: https://github.com/goccy/go-yaml
- yuin/goldmark v2.1.6 and v1.8.6: https://github.com/yuin/goldmark
- dlclark/regexp2: https://github.com/dlclark/regexp2
- gowebpki/jcs v1.0.2: https://github.com/gowebpki/jcs
- spf13/pflag (interspersed flags, combined shorthands): https://github.com/spf13/pflag
- RFC 8785, JSON Canonicalization Scheme: https://www.rfc-editor.org/rfc/rfc8785
- Probes (2026-10-18, throwaway Go test programs outside this repository):
  Golden Liquid against `osteele/liquid` v1.9.2 and `Notifuse/liquidgo`
  `c0b2d83`; map-order, `blank`/`empty`, number rendering and `date "now"`
  probes of both; `santhosh-tekuri/jsonschema/v6` v6.0.3 loader, error and
  pattern probes; `go.yaml.in/yaml` v3.0.5 and v4.0.0-rc.6, and
  `goccy/go-yaml` v1.19.2 resolution, duplicate, multi-document, alias and
  depth probes; `encoding/csv` BOM, CRLF, blank-line and error-recovery
  probe; goldmark v2.1.6 heading-position probe; `go list -deps` of the
  admitted modules for forbidden packages; `unicode.Version` and
  `norm.Version` under go1.26.8 and go1.27.2; `go list -deps` of a program
  importing every admitted module (`golang.org/x/sys` v0.49.0 and
  `golang.org/x/term` v0.47.0 being the latest tags resolved that day).

## Amendments

### A1 — `golang.org/x/sys` and `golang.org/x/term` are linked on unix targets only

**Status:** Proposed 2026-10-18 by PLAN-2026-10-metais-v1 Wave 1 (W1-F2,
from W1-4's finding); awaiting ratification by Ed Funnekotter.
**Amends:** D2 item 2 (the layering bullet), D3 item 3 (the
`golang.org/x/sys` and `golang.org/x/term` rows and the paragraph after the
table), D11 item 3 (the windows no-replace rename), D11 item 8 (the caller
of the terminal check), D11 item 10 (what the probe found), and the
promise to ADR-005 under *Interfaces with sibling ADRs*. Cross-referenced
from ADR-005 D14 item 1.4, ADR-008's *Interfaces* (ADR-010) and ADR-009
D16 item 3.

1. **On windows the binary links neither `golang.org/x/term` nor
   `golang.org/x/sys`.** `golang.org/x/sys/windows` imports `net`, and
   `golang.org/x/term` imports `golang.org/x/sys/windows` on windows. ADR-009
   D16 item 3's import rule and D11 item 10 keep `net` out of the binary's
   closure on every target, and D15's gate builds windows. Where the
   accepted text admitted both modules for every target and the security
   rule refused `net` on every target, the stricter rule wins (RULINGS-v1's
   convention; PLAN-2026-10-metais-v1 §4). On darwin and linux both modules
   are used as D3 item 3's table says, at its versions.
2. **`internal/hostenv`'s windows terminal check is
   `syscall.GetConsoleMode`**, the standard library's binding of the call
   `golang.org/x/term` itself makes on windows. Behaviour is unchanged:
   a stream is a terminal when it is a console.
3. **The windows no-replace rename is `MoveFileW`, through the standard
   library's `syscall.MoveFile`.** `MoveFileW` takes no flags; Microsoft's
   documentation states that its new name "must not already exist", so it
   fails on an existing file or directory, which is the refusal ADR-005 D14
   item 1.4 requires. It differs from `MoveFileExW` without flags in one
   respect: it may move a *file* to another volume by copying it, and a
   directory never. ADR-005 D14 never renames across volumes, because the
   staging directory sits beside the output directory in its parent and
   each temporary file sits in its target's own directory, so the
   difference is never reached. `MoveFileW` takes paths, as `MoveFileExW`
   does, so the choice changes nothing about how `internal/hostfs` confines
   the rename. Calling `MoveFileExW` without `golang.org/x/sys` would need a
   procedure loaded by name through `syscall.NewLazyDLL` and `unsafe`
   pointers, which ADR-009 D16 item 3's directive rule allows only in an
   allow-listed function; `syscall.MoveFile` needs neither.
4. **The layering rule is per target.** Only `internal/hostenv` imports
   `golang.org/x/term` and only `internal/hostfs` imports `golang.org/x/sys`,
   each in files built for darwin and linux only. Any import of
   `golang.org/x/sys/windows` by the binary is refused by the import rule,
   which `TestNoHostFacts` runs for every target of the gate.

Both modules stay in `go.mod` at D3's versions, since the darwin and linux
builds need them.

## Implementation log

- **2026-10-18, PLAN-2026-10-metais-v1 W1-4.** D2 item 2's layering rules
  landed as `internal/guarantees/layers_test.go` over `layers.json`; D11
  items 9 and 10 as ADR-009's `TestNoHostFacts`; D13 item 2's fault rule
  and D12 item 3's version rule beside them; and D12 item 6 as
  `notices_test.go` over `THIRD-PARTY-NOTICES.txt`. The import rule found
  that `golang.org/x/sys/windows` imports `net`, so D3's `golang.org/x/term`
  is linked on unix only: on windows `internal/hostenv.IsTerminal` calls
  `syscall.GetConsoleMode`, which is what `x/term` calls there. D11 item 3's
  windows no-replace rename cannot come from `golang.org/x/sys` for the same
  reason; the standard library's `syscall.MoveFile` (`MoveFileW`, which
  never replaces an existing file) is in reach.
