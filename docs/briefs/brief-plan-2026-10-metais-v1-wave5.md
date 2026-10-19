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

The wave orchestrator wrote this closeout (Work Order
`01a1546e-07bc-7284-83a6-6d47767b3b0a`) after the wave drained. It reports
what landed and what carries forward. **It is not a correctness verdict.**

**What I checked, and how:**
- I read every child's report, the verifier's full report, and `main`'s
  log from `6c300b3` to `06ecb75`.
- In the merged tree at `06ecb75`, I read:
  - `unwired.txt`: no `W5` line remains;
  - `internal/fault`: bits 5 to 8 are as assigned at breakdown;
  - `TestNoDeclarationStops` in `internal/cli/fanout_test.go`;
  - the canary entries in ADR-009's implementation log;
  - `internal/cli/render.go`: no `panic` names an unimplemented
    declaration; the panics left are invariant guards.
- I ran no test, no gate and no mutation. Where something is said below to
  be proved, the proof is the producer's or the verifier's, and is
  attributed to them.

**Wave range:** `6c300b3..06ecb75` on `main`.
- **Breakdown:** thirteen implementation items. The plan orchestrator
  approved them with confirmations (a) to (d) and one condition on (b).
  They were released as one set, with four sanctioned edges to W5-0.
- **W5-I1:** an integration item that Ed filed mid-wave (below).
- **Verification:** one round, which found two blocking defects. Each got
  one fix child, W5-V1 and W5-V2. Both merged and are not re-verified.
- **Non-blocking findings** went to three children, with reasons for those
  not filed (question 4). **The wave completed without waiting for them,
  deliberately.**
- **Not an item:** `7186c72` (#93), the agent skills.
- **People consulted:** the breakdown review went to the plan orchestrator
  (one escalation). Ed's supervising session steered the wave by guidance
  (W5-I1, below), and W5-12's question was answered from outside this
  session (below).

### 1. Did each item land, and does it match the ask?

| Item | Work Order | PR | Merge | Landed |
|---|---|---|---|---|
| W5-0 `hostfs` for existing destinations | `01a15478-efa5-73b1-9bc7-fa6ad13c50d6` | #91 | `1875810` | `hostfs.Existing`: per-path checks giving create/replace or a refusal, a held `<out>/.metais-staging`, commit by unit with `CommitError`, and the ownership rule. `RemoveFile`. Faults `ReadExisting` (bit 5) and `RemoveForeign` (bit 6). `TestExistingDestinationDetector`, which every writer joined |
| W5-1 filters and `date` | `01a15479-926f-7101-ab1a-5e143701ef64` | #97 | `f1d8aec` | `json`, `yaml`, `csv`, `markdown_table`, `slug` and `date`. MTS-FLT-004 to 014, and the charge table MTS-TPL-063. The `tz` and `clock` canaries. Golden Liquid's `date` cases restored (3 pass, 7 `dialect` lines). The hold-out removed: the delegated counts are now "1845 pass, 549 departures fail as listed, 0 fail otherwise, 0 departures pass" |
| W5-2 escaping | `01a1547a-2462-7cdc-b4f1-50aac2129a78` | #90 | `5b7207f` | `escape: html`, `safe` (`template.safe_position`) and `json_script`. MTS-TPL-059 to 062 and MTS-FLT-001 to 003. Shared filters in `internal/template/filters` |
| W5-3 `embed` and `provenance` | `01a1547a-c341-7995-b59f-2666ecc28304` | #89 | `4cfd2ff` | Both tags; `input.raw_unavailable` through `embed`; the output kind (MTS-OUT-012); `template.xref.unknown_input` (MTS-AUT-030); the `pid` canary. MTS-TAG-001 to 011 |
| W5-4 fan-out and paths | `01a1547b-6096-75e3-9545-015a5a0b9837` | #87 | `f9ca1c2` | `for_each`, `when` and Liquid paths; `item` and `fan_out` with D7 item 10's table (MTS-TPL-056 to 058); MTS-OUT-003 to 011 and MTS-PTH-008/009. The data-derived hostile path cases, with no `requires`. Created `TestNoDeclarationStops` |
| W5-5 assets and the mirror | `01a1547b-fb14-7320-bafe-28cf2c54dda2` | #86 | `2d2695a` | Asset copies streamed at write time, the mirror, the `asset` tag, `template.xref.missing_asset`, and the `home` canary. MTS-OUT-013 to 020, MTS-TAG-012 to 017, MTS-RND-007 (`--fail-on warning`) and MTS-AUT-032 |
| W5-6 containers and profiles | `01a1547c-9f6e-7ee0-9946-5e5c435a8f71` | #88 | `fa1c6a5` | Zip containers written while streamed (`zipfmt.WriteParts`); `ooxml`, `odf` and `epub`; `check`'s profile checks (MTS-AUT-031); MTS-TRN-001 to 012; five `output.container.*` codes |
| W5-7 transforms | `01a1547d-3a46-7841-891f-534c77880a80` | #92 | `c872fc9` | `newline: crlf` and `validate`. All 32 `output.*` codes are produced, and `output.parse` is registered. `check` compiles validate schemas (MTS-AUT-033). The lowest-minor note. MTS-TRN-013 to 015 |
| W5-8 existing destinations | `01a1547d-e45d-7b13-b2fd-1792ae55b31d` | #96 | `2fb3946` | `render` into an existing directory and `--overwrite`, under A1. MTS-RND-013 to 022, which state the X1 answer; RND-019 is marked not observable. `output.destination.link`, `.case_variant`, `.not_regular_file` and `.commit_failed`. The link and FIFO hostile cases |
| W5-9 dry run, archive, `--only` | `01a1547e-9024-714f-814d-39ce721387d4` | #98 | `f53945e` | `--dry-run`, `--out -` and `--only`; `usage.archive_to_terminal` and `usage.unknown_output`; fault `StdoutNotice` (bit 7) with the purity test shown red. MTS-RND-008 to 012 |
| W5-10 `sample` and `render --sample` | `01a1547f-32ba-765c-889f-ed911d1dd8f7` | #95 | `cdce4d9` | Both commands through one generate step. MTS-SMP-001 to 009, with repeatable vectors followed by `validate`. `schema.sample_unsatisfiable` is produced. A Wave 3 generator defect was fixed red-first |
| W5-11 `test` | `01a1547f-d8d3-720b-9b74-7362f4113d07` | #99 | `124ffc4` | `metais test`, the fixed identity, byte comparison with difference reporting, and `--update` confined to `expected/`. The fault is `UpdateBeyondExpected` (bit 8). `hostfs.RemoveEmptyDir`, under ruling R-W5-11 |
| W5-12 `extract --overwrite`, `unpack --dry-run` | `01a15480-6f28-7b17-9ec7-f50307de125b` | #94 | `8278575` | `extract` into existing directories, through W5-8's codes. `unpack --dry-run` as the tool's contract (ruling (b)) |
| W5-I1 integration gate | `01a154da-5a5f-72ee-a22c-ec66a7bec93d` | #100 | `30929d5` | The full gate on `main` at `8278575`, go1.26.9 darwin/arm64: all eight steps passed, with nothing to fix. Recorded in plan §11 |
| W5-V1 blocking fix | `01a1556a-8486-73bd-bf22-c22f3b1094a3` | #101 | `204843e` | A code declared more than once on one command now gives each cause its own next step: render's `usage.flag_excluded` and `usage.bad_value`, and sample's `usage.bad_value`. `TestRefusalNextPerCause` was red first |
| W5-V2 blocking fix (3 members) | `01a1556a-ed3d-70fa-8e32-5c94cda958d0` | #102 | `06ecb75` | A: `sample --to` stages empty collection directories through `hostfs.Existing` (planned directory, status `Present`). B: every per-path refusal is reported, through `pathRefusal`. C: "Wrote 1 file". All red first |

**The gate:**
- **Items stopped running the full gate mid-wave**, by Ed's ruling
  (guidance #1). Each ran directed checks instead: its packages and their
  importers under `-race`, the spec tripwires, conformance, the help goldens
  and the specdata drift.
- **Where the full gate ran:**
  - W5-I1 ran it on `main` at `8278575`: green;
  - the verifier ran it on a fresh clone at `30929d5`: green;
  - W5-V1 and W5-V2 each ran it on their own rebased head: green.
- **Earlier full runs:** W5-0, W5-3, W5-4 and W5-6 each ran the full gate on
  their own heads before the ruling.
- **W5-12** ran its directed checks on go1.26.2, because its dispatch set
  `GOTOOLCHAIN=local`. Its merge is inside W5-I1's go1.26.9 run.
- **Linux:** `go test -race ./internal/hostfs/...` was run by W5-8 only
  (linux/amd64). Every other item ran on darwin.

**Divergences from the ask:**
- **W5-3.4 asked for a digest token**, and ADR-011 D10 item 3 provides only
  the tool's two tokens. The producer followed the ADR. The criterion was
  my error.
- **W5-8.3 named `output.link`.** The producer used
  `output.destination.link`, from ADR-009's code table and ADR-005's
  interfaces. ADR-009 D8 item 3's text disagrees with its own table
  (question 3).
- **W5-5.1's "an empty directory produces nothing" is unreachable:** a
  package cannot hold an empty directory.
- **W5-6:**
  - the 65,534-entry acceptance case is not a vector, because of its cost;
  - "a container that lists itself" can only be written with a YAML alias,
    which the manifest refuses;
  - no real `.docx`, `.odt` or `.epub` was opened by its format's checker,
    as ADR-005's conformance section asks.
- **W5-9.4:** no real pseudo-terminal was available, so the terminal
  refusal is proved through the `isTerminal` seam only.
- **W5-10:** a draft `MTS-SMP-010` (`--data` with `--sample`) was withdrawn,
  because the runner treats usage exit 3 as adapter drift. The rule lives in
  help and `docs/cli/sample.md`. `sample --overwrite` lives there too, under
  ruling (b).
- **W5-11.4:** a container output is compared byte for byte until Wave 6
  (ruling (c)). If a render writes no file, `--update` empties `expected/`
  and keeps it, because R-W5-11 forbids removing `expected/` itself.

### 2. What children flagged, degraded, declined or discovered

None of this was verified by me.

- **Stricter rules that won, each named in its PR:**
  - **W5-9:** `archive/tar` imports `os/user`, which ADR-010 D11 item 10
    bans from the binary. ADR-005 D17 assumes `archive/tar`, so
    `internal/tarw` now encodes the archive itself, and is proved
    byte-identical to `archive/tar` in tests.
  - **W5-3:** `template.tag_context` for tags in a path, at open (ADR-004
    D16 item 5), rather than `provenance_style` (MTS-TAG-011).
- **Readings taken, each in its ADR's implementation log:**
  - W5-4:
    - only a wrong `for_each` type gets `output.expression.for_each_type`;
    - a path or `outputs` breach caused by data is `limit.exceeded.output`;
    - refusals in help are sorted stably.
  - W5-5: a render step in the vector format takes `fail_on` (ADR-011's
    log); a malformed `asset` tag is `template.syntax`.
  - W5-6: a profile's `mimetype` entry has `source: profile`; a finding
    inside an entry names `<container>/<entry>`.
  - W5-7: `output.parse` carries `form` and `rule`; regex work is one
    shared meter.
  - W5-8: the per-file checks run with or without `--overwrite`; a
    `CommitError` with nothing committed is `write_failed`.
  - W5-10: `schema.sample_invalid`, `usage.flag_excluded`,
    `usage.flag_requires`, and `usage.not_a_source_directory`, which W4-5's
    log named but never registered.
  - W5-V2: an empty planned directory is listed in `files` with a trailing
    `/`.
- **Flagged by W5-1 and not changed:**
  - the engine charges every string a standard filter builds against
    `assign-bytes`, which ADR-009 bounds only for `assign` and `capture`;
  - ADR-004 D10 item 6 applies its letter table before NFKD, so
    `"Ǿrsted"` slugs to `"rsted"`;
  - findings with equal code and parameters are reported once per template.
- **The liquid-spec run.** W5-1 made a real run, which exited 0 in both
  locales. It was not clean: the gem was cloned from GitHub after a
  rubygems 403, and two `/tmp` paths were redirected. The clean run is the
  carried act (question 5).
- **Conformance timing.**
  - TestConformance sits at ADR-011 D11 item 9's 60 s, and over it: 47 to
    56 s early in the wave, then 62.9, 64.2 and 71.5 s on loaded hosts;
    W5-I1 measured 56.5, 65.6 and 84.1 s, and the verifier 60.07 s.
  - W5-I1 diagnosed the mechanism. The run is sleep-bound: runCase's
    2-second A/B offset holds the case's `t.Parallel` slot, so the time
    grows with every case added, whatever the host's speed.
  - Nothing enforces the budget yet. Carried as W5-NB3.
- **The tz canary** embeds `time/tzdata` in the canary build only, because
  the producer's sandbox hid `/usr/share/zoneinfo`.

### 3. What carries to the next wave

- **For the next brief:**
  - **W5-NB3** (TestConformance's budget) should land before Wave 6 adds
    the hostile corpus in full, which adds many cases to a sleep-bound run.
    If no cure fits ADR-011 as written, it parks with what the ADR would
    have to say.
  - **Wave 6** compares container outputs by entries in `test`, which
    replaces ruling (c)'s byte comparison.
  - **Wave 6's release builder:** `internal/tarw`, not `archive/tar`, writes
    the archive (above).
  - **The open format check:** ADR-005's conformance section asks for a real
    `.docx`, `.odt` and `.epub`, each opened by its checker. Wave 6's matrix
    act, or a Wave 7 example, is the natural place.
- **For the plan orchestrator, or Ed:**
  - **ADR-009 D8 item 3 writes `output.link`;** its code table and ADR-005
    write `output.destination.link`, which shipped. Correcting an accepted
    ADR's decision text is a ratification act, so I relay it rather than
    file a child.
  - **The per-item gate.** Ed's mid-wave ruling, directed checks per item
    plus one integration gate, worked: W5-I1 found no red. It also left two
    composition defects for the verifier to find. Decide whether it stands
    for Wave 6.
  - **For Ed to ratify:**
    - rulings (a) to (d), R-W5-11, and ruling R-W5-12 below;
    - `hostfs.RemoveEmptyDir`, and W5-V2's planned directory with its
      `Present` status, both of which extend W5-0's surface;
    - the readings in question 2.
- **What I would do differently:**
  - **Find every caller of a shared code before filing.** I missed that
    `extract`'s A1 refusals use W5-8's codes, which cost W5-12 a wait. I also
    missed that two items would each declare `usage.flag_excluded` on render,
    which was the blocking W5-V1 defect.
  - **Check each criterion against ADR-011 D8's operation arguments and
    D10's tokens before filing.** W5-3.4 was unmeetable as written.
  - **Thirteen parallel items over one `render.go`** worked, but with heavy
    rebasing and two composition defects. A shared refusal table is a
    meeting point that needs a named owner, like the help goldens.

### 4. What the verification round found, and where every finding went

**Verifier:** `01a15550-4112-7a4c-8e4e-b3ffa7cf66c4` (`wave-verification`),
on a fresh clone of `main` at `30929d5`, go1.26.9 darwin/arm64, with
`gate.sh`. Verdict: **blocking**, two blocking findings. The gate was green.

- **Every id I sent came back**, all 14. Twelve were checked at depth
  **mutated**, with 30 mutations in total. **W5-10 and W5-I1 were only
  inspected**: their diffs were read and their guards were not mutated, so
  this round says less about them.
- **Coverage audit:** all 14 merges are ancestors of `main`, and the only
  other commit in the range is #93. Completeness of the item list is
  trusted, not verified.
- **Every meeting in the breakdown has its vector in the tree**, checked by
  path and content. The three W5-11 had not confirmed are present:
  `aut/test-fixed-identity`, `aut/test-crlf-validate` and
  `aut/test-sampled-example`.
- **Also confirmed:** `unwired.txt` holds no W5 entry; `internal.error` is
  reachable only from a panic; fault bits 5 to 8 are distinct; and the four
  canaries are in ADR-009's log and in `TestHostIsolationCanaries`.
- **The mutations that went red, as they should:**
  - hostfs's staging-name match and its device check;
  - `%Z` at offset zero, and `slug_empty`;
  - `json_script`'s U+2028, and safety carried through a later filter;
  - provenance's `--` rule, and `embed`'s BOM strip;
  - `fan_out.last` and `previous`, and the `outputs` limit;
  - the mirror's `_` exclusion and its explicit-output exclusion;
  - container `first` order, the profile's `mimetype` placement, and the
    entry-count refusal;
  - CRLF's idempotence;
  - W5-8's hidden-entry emptiness and its staging-leftover refusal;
  - the tar mtime, JSON on stdout, and the `StdoutNotice` seam;
  - `test --update`'s dry run and its failed-example guard;
  - `unpack --dry-run`'s refusals, and `extract`'s per-file refusals.
- **Mutations that stayed green, explained by the verifier:**
  - two hostfs link re-checks: other guards keep the behaviour safe (finding
    e);
  - the 65,534 acceptance boundary (finding f);
  - an equivalent mutant in `test --update`.
- **Mutations refused:** two rows that did not compile were re-run.

**The blocking findings and their fix children.** Neither fix is
re-verified: one round, hard stop.

| Finding | Source quoted | Fix child |
|---|---|---|
| (A) Render declared `usage.flag_excluded` twice (W5-10 and W5-8), so `--overwrite` with `--out -` printed W5-10's `--sample` next step, against help (cross-item) | ADR-008 D14 item 4: "Each refusal code is declared with its command (D15), with the template of its next step." | W5-V1 `01a1556a-8486-73bd-bf22-c22f3b1094a3`, #101 `204843e`. It is a new item scoped to the clause; it also covers render's and sample's duplicated `usage.bad_value` |
| (B) `sample --to` created empty collection directories with `MkdirAll` after the A1 commit, outside the staging shape | W5-10.2: "`--to` writes through W5-0's shape (ADR-008 D9 `sample` item 3, ADR-005 A1)." | W5-V2 `01a1556a-ed3d-70fa-8e32-5c94cda958d0`, #102 `06ecb75`, member A |

**Non-blocking findings.** Each is listed with where it went. **The wave
completed without waiting for any child below, deliberately.**

| Finding | Item | Where it went |
|---|---|---|
| (c) `sample --to` reported only the first per-file refusal, as `write_failed`, where render and extract report each (also the verifier's cross-item finding) | W5-10 | W5-V2, member B: same file as (B), merged |
| (d) `sample` printed "Wrote 1 files" | W5-10 | W5-V2, member C: same file, merged |
| (e) Two write-time link re-checks in hostfs's commit are not pinned | W5-0 | child W5-NB1 `01a155c1-d0b9-7d5d-894b-8a813295774c`, member A |
| (f) The 65,534-entry acceptance boundary is not pinned | W5-6 | W5-NB1, member B |
| (g) The staging-leftover refusal says files were written when nothing was | W5-8 | child W5-NB2 `01a155c1-fe11-7cbd-b33c-2288473545a2` |
| (h) TestConformance sits at the 60 s budget (cross-item) | all | child W5-NB3 `01a155c2-3d8a-7af0-8fea-3d7e111536e1` |
| (i) Vectors write the package digest literally, not as a token | W5-3 | not filed: the ADR is right, and my criterion was wrong; nothing in the repository needs to change |
| (j) The link code is `output.destination.link` where W5-8.3 wrote `output.link` | W5-8 | not filed as a child: ADR-009 is inconsistent with its own table; the reading is logged in ADR-005 and ADR-009; relayed for ratification (question 3) |
| (k) W5-I1's squash commit `30929d5` carries a `Co-authored-by` trailer, which AGENTS.md forbids | W5-I1 | not filed: squashed history on `main` is not rewritten. Fix children were told to check their squash message, and W5-V1 and W5-V2 did |

- **Why (c) and (d) rode with the blocking fix.** They touch
  `internal/cli/sample.go`, as (B) does. Two children editing one file
  would have been a conflict scheduled on purpose.
- **Why (e) and (f) share W5-NB1.** Both are guards a mutation showed
  blind. They sit in different packages and are severable.
- **Cohorts:** W5-NB2 and W5-NB3 were filed into the blocking-fixes cohort
  by my error; they are non-blocking and not waited on.

**Where the verifier found nothing:** W5-1, W5-2, W5-4, W5-5, W5-7, W5-9,
W5-11 and W5-12 have no finding, and every mutation on them went red or
was explained. On W5-10 the verifier inspected only.

### 5. Out-of-band acts

| Act | Type | Work Order | Placement |
|---|---|---|---|
| Re-run the pinned liquid-spec selection, as plan §6's liquid-spec row requires after an engine change and changes to chapters 7 and 8 | `manual-qa-pass` | `01a155c2-5c85-7762-a3d8-2541e178e29c` | **Carried, not waited on**, as the brief places it. `depends_on` W5-1 and W5-2. `build` is `cdce4d9`, the latest commit touching chapters 7 or 8 or the engine at the wave's close. Its `focus` carries the README's commands, both locales, the versions to record, and the expected counts line |

It is open when this closeout merges.

### Rulings made during this wave

- **Breakdown, confirmed by the plan orchestrator:**
  - **(a)** An existing empty directory is an existing destination: render,
    extract and `sample --to` write into it without `--overwrite`, and
    unpack keeps its in-place write.
  - **(b)** `unpack --dry-run`, `test --update` and `sample --overwrite` are
    proved by `internal/cli` tests that check the host. A normative
    statement about them must meet `TestCoverage`, or the rule lives in
    `docs/cli` and help. `--update`'s confinement keeps a red-first proof on
    bit 8.
  - **(c)** `test` compares a container byte for byte until Wave 6.
  - **(d)** The SC7 harness's full reach is Wave 6's.
- **Ed, guidance #1 (2026-10-19).** Items run directed checks instead of
  the full gate. W5-I1 runs the full gate once on `main`.
- **R-W5-12.** W5-12 found that `extract`'s refusals need W5-8's unmerged
  codes. I decided it should wait for W5-8 and reuse its `pathRefusal`, and
  held the answer until W5-8 merged. The question was answered first from
  outside this session, the same way; W5-12 reports it waited for W5-8.
- **R-W5-11 (the wave orchestrator).** `hostfs.RemoveEmptyDir` was allowed
  in W5-11's PR, following ADR-007 D5 item 5 ("makes `expected/` exactly
  the rendered tree"), R1-006 and `package.layout.empty_directory`.
  Conditions:
  - `--update` is its only caller, and only beneath
    `examples/<name>/expected/`, for directories its own removals emptied;
  - the bit-8 proof covers directory removal;
  - it is logged in ADR-009 and ADR-010;
  - the windows branch is compiled and vetted only.

  W5-11 reports all four met.
