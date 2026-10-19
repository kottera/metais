# Brief — PLAN-2026-10-metais-v1, Wave 5: the render's capabilities, and the rest of authoring

**Plan:** [PLAN-2026-10-metais-v1](../plans/PLAN-2026-10-metais-v1.md), §11
**Feature:** `metais-v1`. Work Orders are titled `metais-v1 [W5-<item>]: <action>`;
pull requests and their squash subjects are `<type>(metais-v1): [W5-<item>] <change>`
(AGENTS.md, Git and pull requests).
**Repository:** `kottera/metais`, branch `main`. Every Work Order of this wave
gets `repo: kottera/metais` and `branch: main`.
**Gated on:**

- **Wave 4, complete.** Its verdict is under plan §10, and its closeout is
  in [brief-plan-2026-10-metais-v1-wave4.md](brief-plan-2026-10-metais-v1-wave4.md).
- **The §6 liquid-spec act, whose Before is Wave 5.** It has ended. The
  first run (W4-A1) found that the adapter never loaded the liquid gem.
  W4-A1F fixed it (#83). The re-run (W4-A1R) passed on the committed tree,
  `a88b034`, in both the C and a UTF-8 locale: "1801 pass, 490 departures
  fail as listed, 103 held out for applying date, 0 fail otherwise, 0
  departures pass".

The gate has cleared.

## What this wave is for

When Wave 5 is done, everything `render` can do, it does, each capability
with its statements and vectors:

- the METAIS filters and `date`;
- escaping;
- `embed`, `provenance` and `asset`;
- fan-out, conditions and path templates;
- assets and the mirror;
- zip containers and their profiles;
- CRLF and output validation;
- existing destinations, under ADR-005 A1;
- dry run, the standard-output archive and `--only`.

An author previews a package with sample data, keeps it as an example, and
tests the examples. The `home`, `tz`, `clock` and `pid` canaries are
caught. No declaration a valid package can make ends in `internal.error`
any more.

## What to read first

- **The plan:** §2 (the execution model and the rules carried below), §4
  (safety discipline), §5, §6, and §11 (this wave: objective, carries,
  does-not-touch, scope outline, acceptance, settled decisions). §11 is the
  authoritative scope, and Ed's design sessions have edited it: its
  existing-destinations bullet now names ADR-005 A1. This brief binds it
  and does not restate its outline. Read the rulings and verdicts under §7
  to §10 too.
- **Wave 4's closeout**, all of it, especially questions 2 and 3.
- **The ADR decisions §11 carries:**
  - ADR-004 D7 items 4 to 6 and 9, and D9 to D15;
  - ADR-005 D1 to D11, D14 item 2, and D15 to D18;
  - ADR-007 D5 (without containers), D6 and D9;
  - ADR-008 D8 items 4 to 9, D9 (`sample`, `test`) and D16 item 5;
  - ADR-009 D8 item 4 and D17 item 8.
- **The amendments that bind this wave:**
  - **ADR-005 A1, Ed's ruling (#74).** An existing destination is prepared
    beneath one held `<out>/.metais-staging`, created exclusively, and
    committed unit by unit. The one ownership rule: only what lies within a
    held, exclusively created directory is removed, and beneath it only
    names the render created that still hold their kind. It adds two
    refusals under existing codes: another device, and a first segment
    equal to the staging name. D14 item 1 is unchanged.
  - **ADR-010 A2**, Ed's ruling: the MapOrder seam.
  - ADR-010 A1 and A3, and ADR-011 A1 to A4. Ed ruled ADR-011 A4 (#78);
    the others are proposed.
- **RULINGS-v1**: the rulings §11 lists.
- **AGENTS.md**, whole.

Read the code you extend:

- `internal/template` and `render`, which end with `internal.error` for
  every declaration this wave implements;
- `hostfs`'s write shapes from W4-0 (#67), and ruling R-W4-0;
- `internal/dates`, and the Golden Liquid `date` selector;
- `internal/sample` and its form writers;
- `internal/author`, for `check`, `unpack` and `init`;
- `internal/browse`, for `extract`;
- `internal/tarw`;
- `zipfmt`'s writer;
- the conformance runner, with its `requires` probes and setups (ADR-011
  A2's `unwritable-dirs` and A4's `absent` and `unreadable`);
- `unwired.txt`, whose W5 entries are this wave's to empty.

## What Wave 4 learned that changes this one

- **The `internal.error` stops are the scope's checklist.** Wave 4's render
  ends with `internal.error`, by design, for these declarations, and this
  wave removes every one:
  - assets, mirrors and containers;
  - `when`, `for_each`, `escape: html` and `newline: crlf`;
  - output `validate`;
  - Liquid paths;
  - `--out -`.

  When the wave is done, no declaration a valid 1.0 package can make
  reaches `internal.error`. A test pins it: one fixture per formerly
  stopped declaration renders without `internal.error`.
- **Chapter 7's remainder is this wave's:**
  - the `fan_out` and `item` members;
  - ADR-004 D7 item 10's table of contexts;
  - D9's escaping;
  - render work's per-filter charge table.
- **`check`'s remainder is this wave's:** compile the output validation
  schemas, and check container profiles. The lowest-minor computation stays
  trivial while 1.0 is the only minor; say so in chapter 13 rather than
  leaving it unstated.
- **`--fail-on warning` on render becomes observable** once render emits a
  warning. The item that adds the first render warning adds its vector.
- **Carried from Wave 4:**
  - **`extract --overwrite`** (Decision 2), under ADR-005 A1, like every
    existing destination.
  - **`unpack --dry-run`** (finding X2).
  - **One answer for an existing empty destination** across `render`,
    `extract` and `unpack` (X1). Today `render` says `write_failed`/`exists`
    and `extract` says `not_empty`. Both are interim refusals. Under
    ADR-005 A1, an existing directory is an existing destination, so the
    rule follows from A1 and ADR-005 D15. Settle it once, in chapter 11,
    and make every writer obey it.
  - **`input.raw_unavailable`'s vector through `embed`.**
  - **`template.xref.unknown_input` and `template.xref.missing_asset`**,
    listed W5.
- **Registering `date` turns the Wave 3 selector red, by design.** The item
  that registers `date` restores Golden Liquid's 10 `date` cases. It also
  removes the hold-out from the delegated run's expectations, so
  `TestDelegatedRun`'s counts change. The real liquid-spec run is then due
  again (below).
- **Give the verification round a fresh clone.** Wave 4's red came from an
  ignored file that a dirty working tree hides.
- **Settle meeting owners at breakdown, and look for shared infrastructure
  before filing.** Three writers shared `hostfs` in Wave 4, and that cost a
  sanctioned edge set. In this wave, existing destinations,
  `extract --overwrite`, `sample --to` and `test --update` all write
  through `hostfs` under A1.
- **Fault bits are assigned up front**, continuing Wave 4's allocation, so
  two parallel items cannot take the same bit.

## Rulings already made — do not reopen

From plan §11, "Decisions settled before filing", ratified with RULINGS-v1:

- R1-005: the sample commands.
- R1-006: `test --update` may delete inside `expected/`.
- R1-012: existing outputs are never read.
- R2-050 and R2-052 to R2-059: escaping, filters and tags.
- R2-054: no `sha256` filter in 1.0.
- R2-064 to R2-078: outputs.
- R2-100, R2-101 and R2-110: examples.
- R2-113, R2-125 and R2-127: the command line.

Ed's design-session rulings bind this wave: ADR-005 A1 (existing
destinations) and ADR-010 A2 (MapOrder).

The plan orchestrator's earlier rulings stand, recorded under plan §7 to
§10. Among them:

- the envelope's `summary`;
- the Markdown reading;
- the package-home constraint;
- the large-integer reach;
- `PKG`;
- the `date` selector;
- `extract --overwrite` belongs here;
- ADR-011 D11 item 9's 60 seconds binds the default conformance run
  (`TestConformance`), not the package with the tests that re-run it.

Rulings made by the plan orchestrator for this wave:

- **The `hostfs` write side changes once.** Existing destinations under
  A1, `extract --overwrite`, `sample --to`, `test --update` and `unpack
  --dry-run` share it. As in Wave 4, one item owns the change to `hostfs`
  for every writer of this wave, and the writers that need it may take a
  `depends_on` edge to that item. These edges are sanctioned. No other
  intra-wave edge is. A new need for one is still §2's slicing defect, and
  it parks.
- **The canaries' red runs are recorded in ADR-009's implementation log.**
  Each entry names the commit and command that showed the canary red, and
  quotes its output. Recording the red run inside the canary's own pull
  request, as Wave 4 did, meets ADR-011 D12 item 6, provided the entry is
  that specific.

## Rules carried whole from the plan (§2 and §4)

- **The specification is written with the code, never after.** An item
  that makes a behaviour reachable from `metais` adds, in the same pull
  request:
  - the `spec/` statements that state it;
  - the vectors under `spec/conformance/vectors/` that cite them;
  - any finding codes it emits, in `spec/findings/registry.json`.

  `TestCoverage` makes a statement without a vector a red gate. Where the
  tool and the text disagree, the tool is wrong. Every vector runs in both
  of ADR-011 D12's environments.
- **A finding code is registered by the item whose code emits it.** A code
  this wave makes producible leaves `unwired.txt` in the same pull request.
- **Where two items of one wave meet, the one that merges second adds the
  vector for where they meet**, after rebasing (§11: "each bullet is
  independent"). Name each meeting, and its owner, in the criteria. The
  verification round lists every meeting and its vector (§11 acceptance).
  This wave's capabilities compose densely: an `asset` inside a container,
  `provenance` under `test`'s fixed identity, `escape: html` with `embed`,
  fan-out paths through the predicate, a dry run of a container output.
  Derive the meeting list deliberately.
- **An item that finds it needs a sibling's unmerged work has found a
  defect in the plan's slicing.** It does not reach into the sibling's
  scope. It parks and asks.
- **Pin the property false first.** Each of these is red against a tool
  that breaks it, and stays in every `go test`:
  - **the `home`, `tz`, `clock` and `pid` canaries**, each landing red with
    the channel it probes, recorded as ruled above;
  - **standard-output purity** (ADR-008 D16 item 5), red against a build
    that writes a notice to standard output;
  - **nothing read and nothing deleted at an existing destination** beyond
    A1's ownership rule, red against a writer that removes or reads what is
    not its own.

  The detectors already in place (confinement, determinism, read-in-place,
  one pass, digest) keep their faults and stay green on the tool.
- **Do not widen a refusal to make a proof pass.** None of these happens to
  get a proof green:
  - no verification step skipped (R1-013);
  - no limit raised (R1-015);
  - no format or keyword let through (R1-016);
  - no portable-path relaxation (R1-023);
  - no Golden Liquid exclusion except a `dialect` line naming its ADR-004
    item;
  - no runner skip except a `requires` probe;
  - no released vector's expectation changed.

  A proof that cannot pass without one of these is a finding for Ed,
  through the wave orchestrator.
- **The stricter security rule wins** where two ADRs, or an ADR and the
  specification text, disagree on what is accepted or refused. The item
  names it in its PR, and the closeout records it.
- **No new dependency.** Only ADR-010 D3 item 3's modules at their exact
  versions, as A1 and A3 amend it. A newly linked module is named in
  `THIRD-PARTY-NOTICES.txt` by the item that links it. Never copy code from
  another template engine, scaffolding tool or schema validator.
- **Bounded work is measured in CPU time**, never by wall time alone.
- **Every test is isolated** through `internal/testenv`.
- **A Work Order never runs `go test -fuzz`** (AGENTS.md).
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

In scope: everything in plan §11's scope outline, and Wave 4's carried items
named above.

These belong to Wave 6 or later and must not be absorbed here (plan §11
"Does not touch", §12, §13):

- `pack`, and `test`'s comparison of container outputs (Wave 6). `test`
  here compares regular files byte for byte;
- the hostile corpus in full, the SC7 harness's full reach, limit
  calibration, ADR-009 A1's schema compile limits, the benchmarks and the
  release builder (Wave 6);
- the example packages, the worked package, the specification's 1.0 text
  complete, and emptying the W6 and W7 entries of `unwired.txt`, including
  ADR-006 A1 and ADR-011 A4's work (Waves 6 and 7).

## What must not break, and how it is checked

- **The gate**, AGENTS.md's Development list, run whole on go1.26.9 with
  `GOTOOLCHAIN=auto` by every item before merge, and by the verification
  round on a fresh clone of the merged tree. Each PR records `go version`,
  GOOS/GOARCH, and `TestConformance`'s time against the 60 s bound.
- **Plan §11's acceptance**, each criterion proved in the tree:
  - every statement of chapters 8 to 11 has a vector;
  - every `pth/` vector runs on every platform;
  - the `home`, `tz`, `clock` and `pid` canaries are caught, each recorded
    red first in ADR-009's implementation log;
  - the standard-output purity test was red against a build that writes a
    notice to standard output;
  - every meeting of two capabilities this wave added has its vector, and
    the verification round lists them;
  - `sample` vectors are repeatable across two fresh sandboxes, and the
    data they write validates.
- **No declaration a valid 1.0 package can make reaches `internal.error`**
  (above).
- **`TestCoverage` is green over chapters 2 to 13 so far, and
  `unwired.txt` holds no W5 entry.**
- **`TestConformance` stays within ADR-011 D11 item 9's 60 seconds.** The
  wave adds many vectors; a run near the bound is reported, not hidden.
- **Every earlier guard stays green:**
  - every detector with its fault;
  - every canary;
  - the hostile corpus through every opening command;
  - `TestNoHostFacts`, the layering table, notices and spec-data drift;
  - the document tripwires and the runner's self-proof;
  - `TestDelegatedRun` with its counts updated only by the `date`
    registration;
  - `TestAdapterSetup`.
- **What a sandbox cannot prove** (plan §5): operating-system branches run
  only on the host's own (A1's windows commit path is compiled and vetted
  only); the pseudo-terminal proof and SC5 are §6's matrix act. Each PR
  claims nothing it did not run. An item or verifier on linux runs `go test
  -race ./internal/hostfs/...` and reports it.

## Out-of-band acts of this wave

Plan §6 names no act whose **When** is Wave 5. One is due by the §6
liquid-spec row's own notes, which say the run is repeated "at every engine
change, at every change to chapters 7 or 8, and at release". This wave
registers `date` in the dialect and writes chapter 8.

- **Re-run the pinned liquid-spec selection** after the items that register
  `date` and write chapter 8's filter statements have merged.
  - **Type:** `manual-qa-pass`. `build` is the later of those merge
    commits.
  - **Placement:** carried, not waited on, with a `depends_on` edge to
    those items. File it at the wave's close.
  - **`focus`:** run `spec/conformance/delegated`'s README exactly as
    written, on a fresh clone with no local patch, in the C locale and in a
    UTF-8 locale. Record the host, the ruby, bundler and `go version`, the
    commit proved and the liquid-spec commit. Report liquid-spec's exit
    status and `TestDelegatedRun`'s verdict and counts line. With `date`
    registered, the 103 hold-outs should now run. Name any departure that
    passed, and any case that failed otherwise.

  W4-A1R's report is the model of what to ask for.

## Closeout

*Not yet written. The wave orchestrator replaces this line with the wave's
closeout.*
