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

*Not yet written. The wave orchestrator replaces this line with the wave's
closeout.*
