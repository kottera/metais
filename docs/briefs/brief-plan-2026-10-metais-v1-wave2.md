# Brief — PLAN-2026-10-metais-v1, Wave 2: the leaf libraries and the proof harness

**Plan:** [PLAN-2026-10-metais-v1](../plans/PLAN-2026-10-metais-v1.md), §8
**Feature:** `metais-v1`. Work Orders are titled `metais-v1 [W2-<item>]: <action>`;
pull requests and their squash subjects are `<type>(metais-v1): [W2-<item>] <change>`
(AGENTS.md, Git and pull requests).
**Repository:** `kottera/metais`, branch `main`. Every Work Order of this wave
gets `repo: kottera/metais` and `branch: main`.
**Gated on:** Wave 1, complete. Its verdict is under plan §7, and its
closeout is in
[brief-plan-2026-10-metais-v1-wave1.md](brief-plan-2026-10-metais-v1-wave1.md).
The gate has cleared.

## What this wave is for

When Wave 2 is done, every package that depends only on the seams exists
and is proved by its own tests, fuzz targets and third-party corpora:

- zip, Unicode and the portable-path predicate, with the packed
  `pkgtree.Tree`;
- host files, with the source-directory `pkgtree.Tree`;
- JSON, YAML, CSV and Markdown;
- the JSON Schema compiler with the pattern profile;
- failure mapping, bundling and the static walk;
- the tar writer and dates;
- the Liquid front end;
- the manifest library.

The conformance runner exists and is proved against deliberately broken
implementations. The document tripwires guard `spec/`. The command line
parses, helps, and answers `version`, `limits` and `help`. No package can be
opened yet, and the suite has no vector: with none, it reports that it ran
none, and passes.

## What to read first

- **The plan:** §2 (the execution model and the rules carried below), §4
  (safety discipline), §5 (what a sandbox cannot prove), and §8 (this wave:
  objective, carries, does-not-touch, scope outline, acceptance, settled
  decisions). §8 is the authoritative scope. This brief binds it and does
  not restate its outline. Read §7's rulings and verdict too.
- **Wave 1's closeout**, all of it. Questions 2 and 3 say what this wave
  inherits.
- **The ADR decisions §8 carries:**
  - ADR-001 D1 to D4;
  - ADR-002 D2 to D10;
  - ADR-003 D7 to D14;
  - ADR-004 D1 to D3 and D16 item 4;
  - ADR-005 D12 and D17;
  - ADR-006 D6, and D7 item 4;
  - ADR-008 D1, D3, D4, D10 to D13, D15, and D16 items 1, 3 and 7;
  - ADR-009 D5 to D10;
  - ADR-010 D3 to D11, D13 item 3, and D14 items 4 and 6;
  - ADR-011 D3, D7, D9 to D15 and D18.
- **ADR-010 Amendment A1** (proposed, in ADR-010's *Amendments*), with the
  passages it marks *(A1, proposed)*.
- **RULINGS-v1**: the rulings §8 lists under "Decisions settled before
  filing".
- **AGENTS.md**, whole. Its Development section is the gate.

The repository now has a module and the Wave 1 packages under `internal/`.
Read the code you build on: `internal/finding`, `internal/limits`,
`internal/liquid/value`, `internal/pkgtree` and `pkgtreetest`,
`internal/specdata`, `internal/guarantees` (`layers.json` and
`TestNoHostFacts`' rules and allow-lists), `internal/cli` and
`internal/testenv`.

## What Wave 1 learned that changes this one

- **Windows links neither `golang.org/x/sys` nor `golang.org/x/term`.**
  `golang.org/x/sys/windows` imports `net`, and ADR-009 D16 item 3 refuses
  `net` on every target. `TestNoHostFacts` refuses it, and
  `internal/hostenv` already uses `syscall.GetConsoleMode` on windows.
  ADR-010 Amendment A1, proposed and awaiting Ed's ratification, writes this
  into the ADR. The windows no-replace rename in `internal/hostfs` is
  therefore a standard-library call, `syscall.MoveFile` (`MoveFileW`, which
  never replaces), as A1 proposes. This holds whether or not A1 is ratified
  first, because the stricter security rule wins (plan §4). The linux and
  darwin branches use `golang.org/x/sys/unix` as ADR-010 D11 item 3 says.
- **The data model already exists in `internal/liquid/value`** (Wave 1
  ruling (a)). It is a sealed six-kind `Value`: numbers keep their literal
  text, and objects hold unique names in RFC 8785's canonical order, which
  is the order ADR-004 D4 item 5 iterates. §8's "`internal/liquid/value`
  with ordered objects" is that canonical order, already there. The Liquid
  front end extends the package where it needs to and does not re-create
  it. It does not add insertion order: ADR-004 rejects preserving the
  supplier's member order.
- **Not every Wave 1 gate ran on the pinned toolchain.** Some dispatches
  ran go1.26.2 under `GOTOOLCHAIN=local`. The deflate canary and the
  Unicode tripwire land in this wave and depend on the toolchain, so the
  pinned toolchain is a ruling below.
- **Meeting points left to "whichever merges second" were missed twice.**
  So the wave orchestrator names each meeting point between this wave's
  items in the criteria of the item expected to merge second, or of both.
  This wave has many: `pkgtreetest.Check` over both new trees; the registry,
  `unwired.txt` and `TestFindingCodesCovered`; the runner and the document
  tripwires reading `vector.json`; the CLI's `limits` output and
  `internal/limits`' units and classes; `jsonx` and `yamlx` producing the
  same `value` tree; the manifest library and the schema compiler; the
  layer table and every new package. Because there are no vectors yet, a
  meeting point here is proved by a test.
- **The specdata copier cannot make the first copy of a new spec file.**
  Every spec file newly embedded in this wave is added to
  `internal/specdata`'s `specFiles` and copied by hand once. After that,
  the drift check maintains it. This applies to `manifest.schema.json`,
  the conformance schemas, and anything else the binary embeds.
- **Wave 1 left the empty shell exiting 2 on every invocation.** The
  command-line core replaces this with real dispatch and ADR-008's exit
  mapping.
- **W1-F1 may still be running** while this wave starts. It pins four
  small test gaps in `cmd/metais`, `internal/finding`'s `Exit`,
  `layers.json` and `internal/liquid/value`. An item that collides with it
  rebases.

## Rulings already made — do not reopen

From plan §8, "Decisions settled before filing", ratified with RULINGS-v1:

- R1-001 and R2-007 to R2-019: the manifest.
- R1-002 and R1-016: keywords and formats.
- R1-004: METAIS's own Liquid engine.
- R1-010: byte identity per build.
- R1-014: a second end record is refused.
- R1-020 and R2-166: the command line is not the specification, and the
  runner is a black box.
- R1-021 and R2-091: no echo, and neutralisation.
- R1-023: one portable-path rule.
- R2-001 to R2-003 and R2-006: the zip profile.
- R2-024 to R2-028, R2-038 and R2-084: the schema and the mapping.
- R2-039 to R2-043, R2-048, R2-049 and R2-061: the dialect's grammar.
- R2-052 (`date`), R2-072 (pax) and R2-077 (fixed names).
- R2-114 to R2-116, R2-118, R2-119, R2-129 and R2-133: the command line.
- R2-138, R2-140, R2-141, R2-146, R2-151 to R2-159, and R2-172 to R2-182.
- **`date` lives in `internal/dates`** and is registered by
  `internal/template` through `Extend`. `Core()` holds the rest of ADR-004's
  standard set. This narrows ADR-010 D4 item 2's list by one filter. Wave 3
  wires the registration; this wave builds `internal/dates`.
- **Codes of class `usage` and `internal` are proved by `internal/cli`
  tests, not by vectors.** `TestFindingCodesCovered` exempts them, and
  their registry entries name no statement. Wave 1's `limit.invalid` is
  class `usage`.
- **The unwired list.** It lives at `internal/finding/testdata/unwired.txt`,
  and each entry names the wave expected to wire its code. Its three rules
  are those of plan §8. Wave 1's `report.truncated` is class `content`, has
  no vector, and belongs on it.
- **`TestHostileVectorsOpenEverywhere` reads the opening set from the
  command declarations.**
- **The runner's own uses of the JSON Schema validator are in test files.**

Wave 1's rulings stand, recorded under plan §7: the data model's home, and
the reading of the dependency-scan allow-list.

Rulings made by the plan orchestrator for this wave:

- **The JSON envelope carries `summary`, always present, immediately after
  `findings`.** ADR-006 D10 item 4 requires every command's envelope to
  carry `findings` and `summary`. ADR-008 D10 item 1's member list omits
  `summary`, and ADR-008 owns member names and order. The omission is
  ADR-008's. `summary` carries the exact totals and `by_code` that keep a
  capped report honest (ADR-006 D8), and dropping it would lose them. So
  the envelope is `format`, `ok`, `findings`, `summary`, `result`, `next`.
  `docs/cli/schemas/metais.envelope.1.json` states it that way. The item
  that builds the envelope records the reconciliation in ADR-008's
  implementation log, and it goes to Ed for ratification at the train's
  close.
- **No `depends_on` edge between this wave's items.** §8 says each bullet
  is a leaf. Wave 1's one sanctioned edge, onto the module, has no
  counterpart here. Some files many items touch: `go.mod`, `go.sum`,
  `THIRD-PARTY-NOTICES.txt`, `layers.json`, `registry.json`, `unwired.txt`
  and specdata's `specFiles`. Collisions in those are resolved by rebasing
  at merge, not by edges. An item that finds it needs a sibling's code has
  found §2's slicing defect, and it parks.
- **The gate runs on go1.26.9.** Every item's gate run, and the
  verification round's, is under `GOTOOLCHAIN=auto` (or an explicit
  `go1.26.9`), and the PR records the `go version` it ran. A run on another
  toolchain is not the gate. A dispatch that cannot get go1.26.9 parks and
  says so. It does not substitute another toolchain.
- **Third-party corpora are vendored only from their real sources**, per
  ADR-010 D3 item 6 and D14 item 4: under `testdata/`, with the licence and
  the commit they were taken from. These are Golden Liquid, the
  JSON-Schema-Test-Suite and cmark's pathological cases. A dispatch that
  cannot fetch a corpus parks. It does not hand-write a stand-in and call
  it the corpus (AGENTS.md, "This is a product").

## Rules carried whole from the plan (§2 and §4)

- **The specification is written with the code, never after.** An item
  that makes a behaviour reachable from `metais` adds, in the same pull
  request, the `spec/` statements that state it, the vectors under
  `spec/conformance/vectors/` that cite them, and any finding codes it
  emits in `spec/findings/registry.json`. `TestCoverage` (ADR-011 D14)
  lands in this wave and makes a statement without a vector a red gate.
  The commands this wave makes reachable (`version`, `limits`, `help`) are
  the command line, which has no chapter (ADR-011 D1 item 2). They are
  proved by `internal/cli` tests and their committed `docs/cli/schemas/`.
- **A library is built ahead of its wiring, and its statements land with
  its wiring.** Every library of this wave is proved by its unit tests,
  fuzz targets and third-party corpora, and writes no statement of
  chapters 2 to 14.
- **A finding code is registered by the item whose code emits it.** A
  library not yet wired registers its codes in the unwired list, naming the
  wave expected to wire each one.
- **Where two items of one wave meet, the one that merges second adds the
  test for where they meet**, after rebasing onto what landed first. Name
  the meeting in the item's criteria (above). The verification round checks
  that every meeting has one.
- **An item that finds it needs a sibling's unmerged work has found a
  defect in the plan's slicing.** It does not reach into the sibling's
  scope; it parks and asks.
- **Pin the property false first.** This wave's guards each have a test
  that is red against a tool that breaks them:
  - the runner's self-proof, one fault per check (ADR-011 D18 item 1);
  - the echo test of ADR-006 D13 item 5, red against a mapping that used
    the library's messages;
  - the deflate canary and the Unicode-version tripwire, each red on a
    changed value;
  - the "every byte read once" test;
  - the sentinel environment and the A/B differential of the runner.

  Both detector and control stay in every `go test`.
- **Do not widen a refusal to make a proof pass.** That means no
  verification step skipped (R1-013), no limit raised (R1-015), no format
  or keyword let through (R1-016), and no relaxing of the portable-path
  predicate (R1-023). No Golden Liquid exclusion except a `dialect` line
  naming its ADR-004 item (ADR-010 D4 item 7). No JSON-Schema-Test-Suite
  exclusion without its ADR-003 item. No runner skip except a `requires`
  probe (ADR-011 D9 item 6). A proof that cannot pass without one of these
  is a finding for Ed, through the wave orchestrator.
- **The stricter security rule wins** where two ADRs, or an ADR and the
  specification text, disagree on what is accepted or refused. The item
  names the disagreement in its pull request body, and the closeout records
  it.
- **No new dependency.** Only ADR-010 D3 item 3's modules, at their exact
  versions, and on windows not `x/sys` or `x/term` (A1). Third-party test
  corpora are vendored only as ADR-010 D3 item 6 and D14 item 4 allow.
  Never copy code from another template engine, scaffolding tool or schema
  validator.
- **Bounded work is measured in CPU time**, by counters or `getrusage`,
  never by wall time alone.
- **Every test is isolated** with a temporary `HOME` and temporary
  directories, through `internal/testenv`.
- **Never write to another repository.** METAIS names, imports and requires
  nothing of tera.

## Harness policy — Claude throughout

| Role | Work Order type |
|---|---|
| Wave derivation and supervision | `wave-orchestrator-derive` |
| Implementation and corrective work | `code-implementation-direct` |
| Whole-wave verification | `wave-verification` |

These are `plugin.dev-pipeline.v1`'s types; name one by its full key wherever
its leaf is ambiguous. Never substitute another harness's type to get past a
capacity wait: park on `awaiting-capacity` and let the seat come back.

## Scope boundary

In scope: everything in plan §8's scope outline and nothing else.

These belong to later waves and must not be absorbed here (plan §8 "Does
not touch", §9):

- the package loader, `inspect`, and every command that opens a package
  (Wave 3 onward);
- Liquid rendering, the renderer, the standard filter catalog, and the
  `date` filter's registration through `Extend` (Wave 3). This wave builds
  the front end and the interfaces Wave 3 implements;
- every statement and vector of chapters 2 to 14. The container, manifest
  and version chapters arrive with the loader in Wave 3;
- the inputs layer, the sample generator, the cross-reference and `init`
  (Wave 3);
- every fault constant beyond those the runner's self-proof needs. The
  detectors' faults (`SkipDigest`, `SkipConfine`, `MapOrder` and the rest)
  arrive with their detectors. `MapOrder`'s home is awaiting design session
  W1-F3; do not settle it here;
- the pseudo-terminal proof of ADR-008 D16 item 6 (§6's matrix act), and
  `internal/tools/release` (Wave 6).

## What must not break, and how it is checked

- **The gate**, AGENTS.md's Development list, run whole on go1.26.9 by
  every item before merge, and by the verification round on the merged
  tree.
- **Plan §8's acceptance**, each criterion proved by a test in the tree:
  - the runner's self-proof fails on each faulty implementation of ADR-011
    D18 item 1 and passes on the clean one;
  - `TestDriverProtocol` passes through `exec`;
  - the cross-platform record is written and compared on the host;
  - every named fuzz target has its seed corpus and runs in `go test`;
  - Golden Liquid's parse cases pass or carry a closed-set exclusion;
  - the JSON-Schema-Test-Suite passes minus committed exclusions, each
    naming its ADR-003 item;
  - the deflate canary and the Unicode-version tripwire are each red on a
    changed value;
  - the echo test was red against a message-reading mapping and is green
    now;
  - `metais version --json` validates against `metais.version/1`, and
    `metais limits --json` lists ADR-009 D3's table exactly;
  - ADR-008 D16 item 7's argument-syntax table tests pass.
- **Wave 1's guards stay green:** `TestNoHostFacts` (with every new
  package placed in `layers.json` and no entry beyond what ADR-009 D16
  item 3 and R2-143 prescribe), the layering table, the licence header,
  the notices check (every new module named in `THIRD-PARTY-NOTICES.txt`),
  and the spec-data drift check.
- **The conformance suite runs within ADR-011 D11 item 9's 60 seconds**
  under `go test -race`.
- **What a sandbox cannot prove** (plan §5): the darwin, linux and windows
  branches of `internal/hostfs`'s no-replace rename are compiled and
  vetted for every target, but run only on the host's own. The
  cross-platform record is compared on one host, so SC5 itself is §6's
  matrix act. Each pull request says where this applies, and claims
  nothing it did not run.

## Out-of-band acts of this wave

None. Plan §6 names no act whose **When** is Wave 2. The acts carried from
Wave 1, W1-A2 (register `metais.org`) and W1-F3 (the MapOrder design
session), are the plan orchestrator's to track, not this wave's.

## Closeout

The wave orchestrator wrote this closeout (Work Order
`01a15174-89f0-7923-8ef2-78de92cbcb7a`) after the wave drained. It reports
what landed and what carries forward. **It is not a correctness verdict.**

What I checked, and how:
- I read each child's report and the merge commits on `main`.
- I read the two fix diffs (#39, #40) and the parts of ADR-010 that A3 changes.
- I mutated nothing. Where a guarantee is said below to be proved, the proof
  is the verification round's, and it is attributed to it, not to me.

**Wave range:** `59d007f..1fd6d8a` on `main`.
- 16 implementation items were approved and released as one set, with no
  edges between them, and all 16 merged.
- One verification round found two blocking defects. Each got one fix child,
  and both merged. They are not re-verified in this wave.
- Non-blocking findings went to one child and one carried act (question 4).
  **The wave completed without waiting for them, deliberately.**
- A person was consulted twice: at the breakdown review, and on W2-9's
  parser.

### 1. Did each item land, and does it match the ask?

| Item | Work Order | PR | Merge | Landed |
|---|---|---|---|---|
| W2-1 command-line core | `01a1517b-80ee-7897-8fe3-a25f0010ae18` | #22 | `6c17811` | `internal/cli` and `cli/args`; the declaration table; the envelope `format, ok, findings, summary, result, next`; `version`, `limits`, `help` and four topics; five schemas under `docs/cli/schemas/`; eight `usage`/`internal` codes. Wave 1's exit-2 shell is gone |
| W2-2 conformance runner | `01a1517c-1b11-732c-a7a4-93e343650cec` | #27 | `de36579` | `internal/conformance`, all test code: the A/B sandbox, sentinel scan, snapshot, rusage bounds, the `cli` and `exec` adapters, the record and compare, and a self-proof of 33 faults. Also `spec/schemas/conformance/1/` and `spec/conformance/README.md`, and three suite tripwires. It reports "ran no vector" and passes |
| W2-3 document tripwires | `01a1517c-9a88-734b-a99c-afe3fee39c74` | #20 | `dc58ff5` | Twelve ADR-011 D14 tests in `internal/guarantees`; `spec/statements.json` and the `12-findings.md` catalog, generated by `go generate ./internal/guarantees`; `unwired.txt` (`<code> W<n>`) |
| W2-4 zip, Unicode, paths | `01a1517d-215b-7f1f-abf9-ea4a684cef76` | #24 | `eb13a98` | `internal/unitext` (NFC, full folding, the one portable-path predicate, the Unicode tripwire); `internal/zipfmt` (the writer with its canary, the strict reader of D8 steps 1–5, the packed tree); `TestSuiteTreePortable` |
| W2-5 host files | `01a1517d-9bcb-70a7-8efc-1e30b35fe78c` | #21 | `c432963` | `internal/hostfs` and `OpenSource`, with ADR-001 D2's name rule taken as `SourceOptions.Names` |
| W2-6 JSON | `01a1517e-3dd8-7a6b-86d7-27b79584c0cd` | #25 | `fbb5cde` | `jsonx.Parse`, `Canonical` through jcs, the `json` writer and `FuzzJSONParse` |
| W2-7 YAML | `01a1517e-afcb-7737-a6d3-4978e8c8bd5e` | #32 | `9297003` | `internal/yamlx` and the `yaml` writer, plus one dependency-scan entry (ruling R1) |
| W2-8 CSV | `01a1517f-1ba3-72a3-b42f-dfc3ba8d9175` | #26 | `11519d7` | `internal/inputs/csv`: the reader, the writer and the round-trip fuzz target |
| W2-9 Markdown | `01a1517f-810c-7bbb-9029-924299a63b60` | #33 | `1695783` | METAIS's own two-pass CommonMark 0.31.2 heading reading under Ed's ruling R3; goldmark removed; ADR-010 **A3**, proposed |
| W2-10 schema compiler | `01a15180-2eb6-780a-a208-223f66c3338a` | #34 | `3595657` | `schema.NewCompiler`, closed; `regexprofile`; the adapter; the JSON-Schema-Test-Suite; six dependency-scan entries (ruling R2) |
| W2-11 failure mapping | `01a15180-bc51-7c0d-bfc2-e4a07f6572b4` | #28 | `1786f09` | `schemafail.Map` with 97 content codes; the echo test, red first |
| W2-12 bundling, walk, coercion | `01a15181-4172-7ef8-bae5-c211a42249d4` | #31 | `83f25b1` | `bundle.go`, `walk.go` and `coerce.go` in `internal/schema` |
| W2-13 tar writer | `01a15181-982f-7b56-a5e3-93b68d961261` | #30 | `fdd8d84` | `internal/tarw` with the ustar and pax golden bytes |
| W2-14 dates | `01a15182-1537-790c-9790-003a305c1690` | #36 | `4069b5a` | `internal/dates`: `ParseNow`, `FromValue` and the closed `strftime` set, tested under four `TZ` values in child processes |
| W2-15 Liquid front end | `01a15182-8756-7687-b15b-fa6c4873526e` | #38 | `c6333d4` | The lexer, strict2 parser, AST, dialect interfaces and Golden Liquid (1,069 of 1,098 pass; 29 `dialect` exclusions) |
| W2-16 manifest library | `01a15182-fadf-70fd-a500-a5ba9312ca28` | #37 | `924b50d` | `internal/manifest` (`Gate`, `Check`, `Build`) and `manifest.schema.json`, published and embedded |
| W2-2F fix | `01a15267-fb78-7714-8e83-bb0742e512aa` | #39 | `70f4366` | The self-proof's accruing rows (`cpu`, `rss`) run with no wall clock that can fire before their own check |
| W2-5F fix | `01a15268-0f71-7670-a196-013f2c84b2df` | #40 | `1fd6d8a` | `RemoveCreated` proves ownership with a handle held from creation, not with a reusable inode number |

**Divergences from the ask:**

- **W2-6.6 and W2-7.7 are not met as written.** Neither jcs nor yaml.v3 is
  named in `THIRD-PARTY-NOTICES.txt`, and the same holds for jsonschema/v6
  and x/text. ADR-010 D12 item 6 holds that file exactly to `./cmd/metais`'s
  closure, which none of them is in yet, so naming them would turn
  `TestThirdPartyNotices` red. The criterion was mine, and it was wrong. See
  ruling R4.
- **W2-9.5 (name goldmark/v2 in the notices) is superseded.** Ed's ruling R3
  removed goldmark.
- **The amendment is A3, not A2.** W1-F3's MapOrder amendment had already
  taken A2. The plan's reference was corrected in #33.
- **W2-11 read its meeting test differently from my wording.** My wording
  was "no pattern or format value from the schema appears". ADR-006 D9
  item 6 and D6 item 4 report patterns and formats as package content, so
  the echo test proves instead that no *input* value appears. W2-11 followed
  the ADR, which is right.
- **W2-15 deferred 97 Golden Liquid cases, without excluding them.** These
  are invalid cases that ADR-004 refuses at render, not at parse. They are
  listed in `internal/liquid/testdata/golden-liquid/render-time.txt`, each
  naming its ADR-004 item, for Wave 3's renderer. The 17 `include` cases
  carry `dialect` lines citing ADR-004 D1 item 5.
- **W2-6 widened `TestNoHostFacts`' dependency scan** to the closure of
  `./...`. That is stricter than ADR-009 D16 item 3's closure of
  `./cmd/metais`. It is why R1 and R2 came up in this wave rather than when
  Wave 3 wires these libraries.
- **W2-2F's squash message carries a `Co-authored-by` trailer** added by the
  merge. AGENTS.md forbids it. It is history now, and nothing reads it.

### 2. What children flagged, degraded, declined or discovered

None of this was verified by me or by the verifier unless it says so.

- **Not run in a sandbox (plan §5):**
  - every item ran its tests on one host, mostly darwin/arm64; W2-13 and
    W2-16 ran on linux/amd64;
  - the hostfs linux and windows rename branches, and W2-5F's windows pin
    branch, were compiled and vetted, never run. The windows pin branch rests
    on three NTFS assumptions, stated in #40;
  - W2-4's case-collision test skips on APFS;
  - SC5 is the matrix act.
- **W2-5F's cost (flagged for Ed in #40).** A render now holds one open
  handle per entry it has created and not yet committed. ADR-009's outputs
  limit (100,000) bounds that number, but the host's open-file limit can
  refuse first: darwin's default soft limit is far below 100,000. A umask
  without the owner's read bit stops the read-only pin from opening. This
  needs a decision before Wave 4's render uses `hostfs`, and act
  `01a15284-df6e-7668-b799-62f4c380a84f` carries it (question 5).
- **W2-2F's reading.** "No bound raised" was read as covering ADR-011 D13's
  bounds and the fixture's bounds. It was not read as covering the
  self-proof's own 5 s test wall on the rows where that wall raced the check
  under proof. #39 says so.
- **ADR disagreements, each resolved to the stricter rule:**
  - W2-4: ADR-010 D2's layout says "simple case folding", and everything
    else says full folding; full folding was implemented;
  - W2-5: empty directories are refused for every command;
  - W2-7: raw NEL, LS and PS are refused;
  - W2-13: sizes over 8 GiB−1 are refused;
  - W2-14: years are bounded to 0000–9999;
  - W2-16: URI and description rules.
- **ADR-level questions raised by producers, still open:**
  - W2-6: `limit.exceeded` has one subject (`input`), but ADR-009 D19 also
    uses it for render work and outputs;
  - W2-6: ADR-003 D8 item 5 refuses large integers by spelling, so `11e17`
    is accepted while its canonical text is refused;
  - W2-8: `json.duplicate_name` carries a `name` parameter, while ADR-006 D9
    item 4 forbids input strings as finding parameters;
  - W2-6: the finding model costs O(pointer length) per finding, so many
    duplicates under a long name make building findings quadratic.
- **Writer and reader gaps (W2-7):**
  - the `yaml` writer does not escape U+FFFE and U+FFFF;
  - it writes an integral number beyond 2^53−1 and below 1e21 as an integer
    literal that its own reader refuses;
  - syntax errors carry a line but no column.
- **W2-10:** `schema-work` is not counted, because the library has no hook
  per subschema. This is recorded in ADR-010's log.
- **Corpora:** all vendored from their real sources.
  - Golden Liquid was fetched as git blobs and checked against the commit's
    tree hashes.
  - CommonMark's spec is at `9103e34` (0.31.2).
  - cmark's pathological cases.
  - The JSON-Schema-Test-Suite.
  - RFC 8785's examples are the exception: they were transcribed, because the
    sandbox could not reach the RFC, and cross-checked against jcs.

### 3. What carries to the next wave

- **Wave 3's wiring must name every newly linked module in
  `THIRD-PARTY-NOTICES.txt`**: jcs, yaml.v3, jsonschema/v6 and x/text.
  `TestThirdPartyNotices` goes red until it does (ruling R4).
- **Wave 3's loader composes what this wave left as parameters:**
  - W2-4's predicate is passed as `hostfs.SourceOptions.Names`;
  - ADR-001 D8 steps 6 and 7;
  - ADR-009 D6 item 3;
  - whether a manifest path names an entry, and the Liquid parse of `path`,
    `when` and `for_each`;
  - the `template.*`, `template.date_*` and `input.*` findings (W2-14, W2-15).
- **`render-time.txt`'s 97 cases** are Wave 3's renderer's to pass. The arity
  lines go stale when the catalog's `Check` refuses literal arity at parse,
  and the test forces their removal.
- **The ADR questions under question 2** need rulings before Wave 3 or 4
  emits those codes for templates or outputs.
- **The hostfs handle cost** needs a decision before Wave 4 (act below).
- **`unwired.txt`** lists every refused, content and destination code this
  wave registered. Wave 3 onward shrinks it.
- **For Ed to ratify at the train's close:**
  - ADR-010 A3;
  - rulings R1, R2 and R4 below;
  - ADR-008's envelope reconciliation (W2-1, recorded in ADR-008's log).
- **What I would do differently:**
  - **Check acceptance criteria against the guards' own rules before
    filing.** W2-6.6 and W2-7.7 asked for something ADR-010 D12 item 6
    forbids.
  - **Copy the brief's wording, don't paraphrase it.** My "add no entry"
    was stricter than the brief's "no entry beyond what ADR-009 D16 item 3
    and R2-143 prescribe", and it cost two parks.
  - **Name in the brief a host that can run the linux branches.** Two
    producers found the hostfs defect only because they happened to land on
    linux.

### 4. What the verification round found, and where every finding went

**Verifier:** `01a15227-c42e-7841-8dea-309693524ba1` (`wave-verification`).
Verdict: **blocking**.
- It audited all 16 merge commits as ancestors of `main` against
  `base_ref`, and placed the four commits in the range that are not items
  (#19, #23, #29, #35).
- Every id I sent came back, each at depth **mutated**.
- It confirmed that the three meeting tests no producer reported adding are
  in the tree: W2-6/W2-7 `TestSameTreeAsJSON`, W2-10/W2-11
  `TestNoEchoEndToEnd`, and W2-10/W2-12 `TestCombinedSchema`. The other named
  meetings have tests too.
- Its gate run on `1695783` (go1.26.9 darwin/arm64) was red only on
  `TestSelfProof/cpu/{cli,exec}`.

**What it mutated, per item (42 rows); each went red unless noted:**
- W2-1: swapped the envelope's `findings` and `summary`.
- W2-2:
  - lowered the self-proof wall to 1.2 s, which reproduced the gate red;
  - set `zipDepth` 3→2, red;
  - set the A/B start offset 2 s→0, which stayed **green** (finding (a)).
- W2-3: disabled the check that an exempt-class code names no statement.
- W2-4:
  - three mutations of the end-record scan stayed **green** (finding (b));
  - dropped superscript `¹` from the reserved names, red;
  - removed the trailing-space trim, red;
  - disabled the tail cache, red: a byte was read twice.
- W2-5: disabled `RemoveCreated`'s identity check, red on darwin. This
  confirms the test pins that guard.
- W2-6: reported only the first duplicate name.
- W2-7: three profile mutations (the merge-key shape, `0o` octal, signed
  `.inf`).
- W2-8: two mutations of the reader's stop rules; one was an equivalent
  mutant.
- W2-9: the multi-line setext line, and link-reference definitions.
- W2-10: a built-in format left unreplaced.
- W2-11: disabled D6 item 6.3's collapse.
- W2-12: `required` in reverse order.
- W2-13: PAX format, a stray PAX record, GNU format.
- W2-14: the width of `%j`.
- W2-15: `include` accepted.
- W2-16: the version gate's early return.

**Blocking findings, and the fix child for each:**

| Finding | Quoted criterion | Fix |
|---|---|---|
| The self-proof's CPU row is load-dependent. Under `-race` the fake must burn 1.5 s of CPU inside a 5 s wall, so on a contended host the wall kill fires first and the CPU fault is reported as `[wall]` | W2-2.1 | W2-2F `01a15267-fb78-7714-8e83-bb0742e512aa`, #39 `70f4366`. `TestSelfProofAccruingRowsIgnoreWall` was red on `main` and is green after |
| `RemoveCreated` judged ownership by `os.SameFile` alone. ext4 reuses a freed inode, so another party's file was removed: W2-5's own test failed on linux/amd64 | W2-5.5 | W2-5F `01a15268-0f71-7670-a196-013f2c84b2df`, #40 `1fd6d8a`. `TestRemoveCreatedLeavesANewcomerWithTheRecordedIdentity` builds the collision on any OS, red on `main` (darwin) and green after. **Not run on linux** |
| The merged-tree gate is red (cross-item, quoting the brief's "The gate") | brief | The same two causes as above. No third child |

**Not re-verified.** Neither fix child is re-verified in this wave. Both
report the gate green on go1.26.9 darwin/arm64. I ran AGENTS.md's gate once
over `1fd6d8a` before merging this closeout, on go1.26.9 darwin/arm64 with
cgo for `-race`: green on every step. A linux run of `TestRemoveCreated*` belongs in
the next wave's brief.

**Non-blocking findings.** Each row is one finding. Findings (a) and (b) are
grouped in one child: each is a missing test in a different package, one
gate run verifies both, and the child is told they are severable.

| Finding | From | Where it went |
|---|---|---|
| (a) The A/B differential's two-second start offset is not pinned by any self-proof fault: with it at 0, `TestSelfProof` stays green, so ADR-011 D12 item 4's "the process id and the clock differ too" is unproved | W2-2 | W2-NB1 `01a15284-bed3-7226-9f3f-495ea5e82436` |
| (b) The 65,557-byte end-record search window and signatures inside the end record are unpinned: three boundary mutations stay green | W2-4 | W2-NB1 `01a15284-bed3-7226-9f3f-495ea5e82436` |
| (c) W2-6.6 and W2-7.7's notices criteria conflict with ADR-010 D12 item 6 | W2-6, W2-7 | **Ruled here, no child** (R4). The guard already forces Wave 3 to add the entries, so a child would have nothing to change |

### 5. Out-of-band acts

The brief named none for this wave. W1-A2 and W1-F3 are the plan
orchestrator's.

| Act | Type | Work Order | Placement | State |
|---|---|---|---|---|
| Decide whether `hostfs` holding one open handle per uncommitted entry (W2-5F) is acceptable against hosts' open-file limits, or which mechanism replaces it | `design-session` | `01a15284-df6e-7668-b799-62f4c380a84f` | Carried, not waited on; filed at the wave's close. Needed before Wave 4's render | Open |

### Rulings made during this wave

Given at the breakdown review:
- No `depends_on` edges.
- CSV and Markdown are separate items, and so are the failure mapping and
  bundling.
- `TestSuiteTreePortable` belongs with the predicate (W2-4), under R1-023.
- The source tree takes ADR-001 D2's name rule as a parameter (W2-5), with
  the composed test added by whichever merged second.
- **Package homes.** ADR-010 D2's layout owns package names. No two items
  create the same new package. W2-10 and W2-12 each own their own files in
  `internal/schema`.

Made during the wave:

- **R1 (W2-7).** One exact dependency-scan entry: `go.yaml.in/yaml/v3`
  `resolve.go` `time.Parse`.
  - **Grounds:** it follows from ADR-009 D16 item 3, which prescribes
    per-hit entries "giving the import path, file, symbol and the reason it
    cannot reach a template, an output or a message", and from Wave 1
    ruling (b). The brief allows "no entry beyond what ADR-009 D16 item 3
    and R2-143 prescribe".
  - **Corrected reason:** `time.Parse` does consult Local for a numeric
    offset, but the result only picks `Node.Tag`, which `yamlx` never reads.
  - **Governs:** W2-7. **For Ed to ratify.**
- **R2 (W2-10).** Six exact dependency-scan entries, on the same grounds as
  R1:
  - jsonschema/v6 `format.go` `time.Parse`;
  - `loader.go` `os.Open` and `runtime.GOOS`;
  - `util.go` `filepath.Abs` and `runtime.GOOS`;
  - x/text/message `message.go` `os.Stdout`.

  Each reason that a test can reach is backed by one: `TestBuiltInFormatsNeverRun`,
  `TestNoLoadReachesFileLoader` and `TestNoRelativeLocation`. **Governs:**
  W2-10. **For Ed to ratify.**
- **R3 (Ed, W2-9).** ADR-009 D10 item 4 stands whole, and METAIS's own
  linear CommonMark reading replaces goldmark. Two ratified decisions
  disagreed (R2-156 and ADR-010 D9 assumed goldmark met D10 item 4), and
  plan §4's stricter-rule convention settles it. It is recorded in plan §8,
  written as ADR-010 A3 (proposed), and ratified with A1.
- **R4 (this closeout).** W2-6.6 and W2-7.7's notices criteria yield to
  ADR-010 D12 item 6.
  - **Grounds:** an item's criterion cannot override an ADR. The notices
    file lists exactly what `./cmd/metais` links, and `TestThirdPartyNotices`
    makes the wiring wave add each entry.
  - **For Ed to ratify.**
