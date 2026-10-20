# PLAN-2026-10: METAIS test economy — each property proved once, at the size that can fail it

**Author:** drafted 2026-10-20 by the `metais-test-economy` design Work
Order, under Ed Funnekotter's direction of 2026-10-20
**Date:** 2026-10-20
**Feature:** `metais-test-economy`. Every Work Order and pull request of this
train inherits it verbatim: `metais-test-economy [W1-2]: <action>` for a
Work Order, and `<type>(metais-test-economy): [W1-2] <change>` for a pull
request and its squash subject, by `kottera/kottera`'s
`docs/dev/work-order-and-pr-naming.md`, which AGENTS.md adopts.
**Status:** Proposed; awaiting Ed Funnekotter's acceptance. It is run by a
plan orchestrator filed separately once it merges.
**For ADR:** [ADR-012](../adr/ADR-012-test-economy.md), and the amendments
it makes: ADR-005 A2, ADR-009 A7, ADR-010 A6 and ADR-011 A7.
**Repository:** `kottera/metais`, branch `main`. The train writes this
repository only.
**Depends on:** the gate-selection fix filed in the same cohort, for W1-2's
and W1-3's placement (ADR-012 D10 item 4). Nothing else; the first wave can
start as soon as this plan merges.

## 1. Purpose

On 2026-10-20 the scoped gate timed out four times in a row on
`internal/conformance` and `internal/guarantees`, and `internal/cli` took
583 s. The audit Ed asked for found that nearly all of the time goes to about
fifteen tests, most of which prove nothing a cheaper test would not.
ADR-012 decides how each property is proved instead. This plan does it.

**Done means:**

- Every decision of ADR-012 is in the tree, each by the item §7 to §9 name.
- Each package meets its target in §4, measured as §4 says, at the
  verification of the wave that last touches it.
- No guarantee was dropped silently. Every test that was shrunk, merged,
  moved or deleted names, in its pull request, the property it proved and
  what still proves it, and §10's ledger is complete.
- The gate is green on `main`.

## 2. Execution model

Waves run one at a time; the items inside a wave run in parallel, and each
item is one pull request. Unlike PLAN-2026-10-metais-v1, this plan carries its
items, because they are small and were derived with the design. Each wave's
orchestrator may split an item, never merge two, and never widen one.

**Each item carries these rules whole:**

- **It states the property it keeps, and what proves it after the change.**
  The ledger of §10 is the item's checklist. An item that finds a property
  the ledger does not name adds it, and says what proves it.
- **Red first, where a property is guarded.** The reworked test is shown
  failing against the defect it exists to catch before it merges: a scratch
  commit that injects the defect, run, and its failing output quoted in the
  pull request. The scratch commit never merges. A detector's control
  (ADR-010 D13) is the permanent form of the same proof; a red-first run is
  the one-time form, for tests with no fault to inject.
- **It measures its saving, alone** (ADR-012 D12). It quotes the package's
  CPU-s and wall time under the gate's settings before and after, on one
  host, named, and each reworked test's own CPU-s from a run of that test
  alone, less `TestMain`. A parallel test is counted from its first `cont`.
- **No test's expectation is widened to pass**, no timeout raised, and no
  `-parallel` changed. The specification's text and vectors do not change;
  `spec/conformance/README.md`'s prose changes only in W2-3, by ADR-012 D2
  item 8.
- **It follows AGENTS.md**, including the conventions ADR-012 D13 adds,
  `merge-setup.sh` before the first rebase and `regenerate.sh` after every
  rebase.

### Harness policy — Claude throughout

| Role | Work Order type |
|---|---|
| Plan orchestration | `plan-orchestrator-claude` |
| Wave derivation and supervision | `wave-orchestrator-derive` |
| Implementation and corrective work | `code-implementation-direct` |
| Whole-wave verification | `wave-verification` |

Every Work Order gets `repo: kottera/metais` and `branch: main`.

## 3. Overview and sequencing

| Wave | Objective | Depends on |
|---|---|---|
| 0 | ADR-012, its amendments and this plan, merged to `main` | — |
| 1 | **Everything outside the conformance runner's core**: the growth tests and calibration's full counts; the release builder; the guarantee package's builds, listings and gate-script tests; the command line's walks; the render goroutine's race test; the driver protocol's static round trip | 0 |
| 2 | **The conformance runner**: the release build and the small tests; the representatives; the record and the one pass, with the process budget | 1 (W2-1 needs W1-5) |
| 3 | **The detectors that the one pass makes redundant**, and the train's measurement | 2 |

Wave 2's items touch one seam, the runner in `internal/conformance`
(`runner_test.go`, `adapter_test.go`, `sandbox_test.go`, `main_test.go`), so
they run in sequence: W2-1, then W2-2, then W2-3, each with `depends_on`.
W1-6 also touches `internal/conformance`, but only `driver_test.go`, so it
runs in Wave 1. Wave 3 needs W2-3's pass, because it deletes what that pass
replaces.

## 4. Targets, and how each is measured

**How.** ADR-012 D12, on a named host:

```bash
go test -race -count=1 -run '^$' ./internal/<pkg>          # warm the cache
/usr/bin/time -f '%e %U %S' \
  go test -race -count=1 -p 1 -parallel 4 -json ./internal/<pkg> > run.json
```

- CPU-s is user plus system time from `/usr/bin/time`, children included.
- `-parallel 4` is the gate's share on a 12-core host (ADR-010 A5 item 7).
- Nothing else runs on the host. A host that cannot be had idle measures
  baseline and result back to back, alternating three times, and reports
  the medians.
- `internal/conformance`'s process count is its pass's log line (ADR-012
  D9). The package's total processes and builds are logged by `TestMain`.

**The baseline** was measured on linux/amd64, 12 cores, go1.26.2, at
`7c3b362`, alone. The audit's figures, on a 12-core Apple M at `647c0cf`,
are beside it. The targets are for the linux host; a Mac verification
compares ratios with its own baseline.

| Package | Baseline, linux (wall / CPU-s) | Baseline, Apple M | Target CPU-s, linux | Other target |
|---|---|---|---|---|
| `internal/conformance` | 287 s / 1,492 | 275 s / 959 | **≤ 300** | the pass ≤ 1,600 processes for the current 780 steps, equal to its plan; the package ≤ 2,500 processes and ≤ 11 builds of `metais` |
| `internal/cli` | 180 s / 379 | 98 s / 194 | **≤ 230** | — |
| `internal/guarantees` | 138 s / 228 | 88 s / 126 | **≤ 120** | ≤ 2 builds of `metais` |
| `internal/tools/release` | 96 s / 325 | — | **≤ 170** | — |
| `internal/hostfs` | 36 s / 36 | — | **≤ 8** | — |
| `internal/author` | 8 s / 8 | — | **≤ 5** | — |
| `internal/testenv` | — | — | **≤ baseline + 20** | receives the cgo toolchain test |
| every other package | — | — | **no worse than baseline** | — |

Each target is the baseline less the saving the items below estimate, with
about a quarter left as margin. An item that misses its share says why, and
by how much. A target missed at a wave's verification is a finding for Ed,
not a reason to cut further.

## 5. Safety discipline (always on)

- **The gate is AGENTS.md's**, with the two tiers of ADR-010 A5. W1-3 (it
  moves a test into `internal/testenv`) and W2-3 (it adds a step to
  `regenerate.sh` and a line to `.gitattributes`) touch shared seams. Each
  runs the wave tier before it merges.
- **Every headline guarantee keeps its detector and its fault control**
  (ADR-010 D13), even where its main loop shrinks. Each of ADR-009 D16 item
  2's canaries keeps its probe and must stay red.
- **Do not widen a refusal to make a proof pass**, as PLAN-2026-10-metais-v1
  §4 says, whole.
- **No new dependency.**
- **Never write to another repository.**

## 6. What this train cannot prove in a sandbox, and the out-of-band acts

- **Platform agreement of the record.** W2-3 makes a record on its host. The
  first gate on the other operating system is the first proof that records
  agree across platforms. Wave 2's verification runs the pass on the
  platform W2-3 did not run on: darwin if W2-3 ran on linux, and the other
  way round. A difference is an SC5 defect, and is fixed in the tool, never
  by a record per platform (ADR-012, "What this does not decide").
- **The release act grows**, by ADR-012 D10 item 3. Ed runs it at the next
  release, as PLAN-2026-10-metais-v1 §6's matrix act:
  - `-conformance.live` with the full fan-out in both path modes;
  - the driver protocol over the whole suite;
  - the calibration suite with the full counts;
  - the release builder's two runs.

  `docs/dev/release.md` and `docs/dev/calibration.md` name each, by W1-1,
  W1-2, W1-6 and W2-3.

## 7. Wave 1 — everything outside the conformance runner's core

### W1-1 — bounded handles, proved by growth; the full counts move to calibration

ADR-012 D5.

- **Property kept:** a write holds a fixed set of handles however many
  files it plans, and none per planned file (ADR-005 A1 item 7). The same
  holds for `Unpack`.
- **Change:**
  - Add a test-only `openHandles()`. Rewrite
    `TestExistingWritesItsLimitUnderTheOpenFileLimit` and
    `TestShapesWriteTheirLimitUnderTheOpenFileLimit` as growth tests along
    three axes: 200 directories of one file each, one directory of 200
    files, and one chain at depth 32 with a file at each level.
  - `TestUnpackWritesTheEntriesLimitUnderTheOpenFileLimit` keeps the
    open-file limit of 128 as its detector, at the same three axes.
  - Add calibration cases: render into a new and into an existing
    directory at `outputs`'s default, and unpack at `entries`'s default,
    each through the release binary under an open-file stub of 128.
  - Update `docs/dev/calibration.md`.
  - Log against ADR-010 and ADR-005, answering W5b-2's and W5b-NB1's
    reasons for the full counts.
- **What proves it after:** the growth tests, with a writer holding a
  handle per entry as their control, at every change; the full counts in
  the calibration suite, at release.
- **Red first:** a scratch commit leaks one handle per directory in
  `Writer.mkdir`. Today's tests pass against it, which the item shows. The
  new tests fail. Then the same with one handle per file, and one per depth
  level.
- **Saving, measured on the baseline host:** `TestShapes…` 20.0 s and
  `TestExisting…` 13.5 s of `internal/hostfs`'s 36 CPU-s; `TestUnpack…`
  3.7 s of `internal/author`'s 8. About 30 CPU-s per change that reaches
  either package.

### W1-2 — the release builder, once

ADR-012 D6.

- **Properties kept:**
  - the release's two builds are byte-identical;
  - its archives are deterministic;
  - it refuses an unreleased revision, a revision with no tag, and a tool
    that implements nothing;
  - it builds by the release line.
- **Change:**
  - `TestReleaseBuildsTwiceAndWritesDist` runs once. A new in-process test
    runs `archive()` twice on one input and compares the bytes.
  - `released()` becomes a table test over `fakeRepo`. One end-to-end case
    (a revision with no tag) stays, and so does "implements nothing" (or
    its JSON reading is split out and tabled).
  - `TestHarnessBuildsByTheReleaseLine` moves to `internal/conformance`.
  - `docs/dev/release.md` says that the release's two runs and their
    `SHA256SUMS` comparison are the release act's.
- **What proves it after:** `copy0` against `copy1` within the one run, for
  the builds; the in-process archive test, for archiving; the table and the
  end-to-end case, for the refusals.
- **Red first:** a scratch commit puts the wall clock into a tar header in
  `archive()`. Today the second run catches it, and the new in-process test
  must too. A scratch commit makes `released()` accept a revision with no
  tag, and the table must fail.
- **Saving:** `TestReleaseBuildsTwiceAndWritesDist` 52 s of work, half of
  its builds; `TestReleaseRefusesAnUnreleasedRevision` 18 s. Of the
  package's 325 CPU-s, the item's own measurement settles the CPU share.
  With the gate-selection fix, the package stops running on changes that
  do not reach it.

### W1-3 — the guarantee package: one build, one listing, and the gate scripts' tests

ADR-012 D7 items 3 to 6.

- **Properties kept:** the gate selector's reach and working-tree rules;
  the go command takes no host Go environment and keeps the host's C
  toolchain; every listing-based tripwire.
- **Change:**
  - Delete `TestGateSelectWorkingTree`, a strict subset of
    `TestGateSelectWorkingTreeHostGit`.
  - Build `metais` once per test binary for the four tests that only run
    it.
  - Cache `go list` per target under a once each, and parse
    `moduleFiles` once.
  - Move `TestGoCommandKeepsHostCToolchain` to `internal/testenv`. That is
    a shared seam, so the wave tier runs before merge.
- **What proves it after:** `TestGateSelectWorkingTreeHostGit` (same
  assertions, plus the hostile git configuration); the moved test, in the
  package whose code it tests; the tripwires, unchanged, over the cached
  listing.
- **Red first:** not applicable to the consolidations, which change no
  assertion. The item shows each tripwire's control still red over the
  cached listing.
- **Saving:** `TestGoCommandKeepsHostCToolchain` 17.8 s of work leaves the
  package. `TestGateSelectWorkingTree` takes about 3 s. The builds and
  listings are measured by the item.

### W1-4 — the command line's walks become representatives and a static check

ADR-012 D8.

- **Properties kept:**
  - every declared refusal of every reading command is triggered (ADR-008
    D16 item 2);
  - the reading commands refuse as `inspect` does (ADR-011 D16 item 3,
    in-process);
  - each table is required by exactly its editions;
  - presentation never reaches JSON;
  - pathological Markdown is bounded through outline and the frontmatter
    reader;
  - SC3.
- **Change:** as ADR-012 D8 items 1 to 6. The clean-package check is new:
  a reading command must not report `internal.error` over a package that
  opens.
- **What proves it after:** the per-family packages through every reading
  command, and the static check of `packageRefusals` against the vectors'
  expected findings; `TestFindingCodesCovered`, unchanged; the binary's
  every-command reach, in the wave tier.
- **Red first:**
  - In a scratch copy of the vectors, a code is removed from every
    expected-findings file, and the static check must fail.
  - A scratch commit makes `outline` panic on a clean package. Today's walk
    passes, because the panic is recovered as `internal.error` and
    skipped. The new check fails.
  - A scratch commit couples two ACR tables, and the ten cases must catch
    it.
- **Saving:** `TestReadingRefusalsTriggered` 48 s and
  `TestInspectRefusalsTriggered` 22 s of work; `TestHostileMarkdown` 19 s,
  most of it; `TestACREditionDecidesTables` 6 s, three quarters of it;
  `TestPresentationNeverReachesJSON` 6 s, three quarters of it.

### W1-5 — the render's one goroutine, under the race detector

ADR-012 D4 items 3 and 4.

- **Property kept:** the tool has no data race. It has one goroutine
  (`internal/cli/render_write.go:125`).
- **Change:**
  - A focused in-process test drives `content()`'s producer through every
    path: a whole container, an error mid-write, and the reader closing
    early.
  - ADR-010 D2 item 3 is resolved by Ed's answer to Q4: the amendment
    stands, or the goroutine goes.
- **What proves it after:** the focused test under `-race`, at every change
  to `internal/cli`. This must land before W2-1 stops race-building the
  tool for the suite.
- **Red first:** a scratch commit adds an unsynchronised write shared by
  the producer and the commit. The focused test must fail under `-race`.
- **Saving:** none here. It enables W2-1's.

### W1-6 — the driver protocol, by its round trip and its fixtures

ADR-012 D11.

- **Property kept:** the published driver protocol cannot rot unrun
  (ADR-011 D11 item 7).
- **Change:**
  - A static round trip over every step of the suite: each request
    against `driver-request.schema.json`, and each step's expected findings
    as a response against `driver-response.schema.json` and back through
    the adapter.
  - `/fixtures` gains a stream render if it has none.
  - Delete `/suite`.
  - `docs/dev/release.md` adds the whole-suite driver run to the release
    act.
- **What proves it after:** the round trip, at every change to the suite or
  the adapter; `/fixtures`, which drives every operation through the
  reference driver; the release act's whole-suite run.
- **Red first:**
  - A scratch commit renames a request member in the adapter, and the round
    trip must fail.
  - A scratch commit drops `stream_file` handling from the reference
    driver, and `/fixtures` must fail.
- **Saving:** `TestDriverProtocol`, 265 CPU-s alone on the baseline host,
  less what `/fixtures` costs: about 3,200 processes.

## 8. Wave 2 — the conformance runner

### W2-1 — the release build, the whole sandbox, and the small tests

ADR-012 D4 items 1 and 5, D2 item 1's check 4, and the audit's finding 1f.

- **Properties kept:**
  - the suite runs against the binary a user gets;
  - nothing is written outside a declared tree, now over the whole sandbox;
  - each fixture behaviour the small tests prove.
- **Change:**
  - `TestMain` builds the release build once, and every test shares it. No
    race-built `metais` remains.
  - Check 4 snapshots the whole sandbox.
  - `TestCalibrationLowered` runs in parallel.
  - `TestRecordAndCompare` makes one record run.
  - `TestUnwritablePreState` runs `cli` only, with its control in A only.
  - `TestSelfProofAccruingRowsIgnoreWall` becomes a unit test of `rowWall`
    plus one end-to-end row (cpu, `cli`).
  - The pass gets its own process counter, logged.
  - The amendments to ADR-011 D11 item 2 and D13 item 3 take effect here.
    The race multipliers stay for a race-built tool someone else names.
- **What proves it after:** the same tests, cheaper; W1-5's focused test,
  for races.
- **Red first:** a fake implementation of the self-proof (ADR-011 D18)
  writes into the sandbox root, beside `work/`. Today's check 4 passes it,
  and the new one must not. That fault joins the self-proof permanently.
- **Saving:** at about 0.11 CPU-s per race-built process against 0.04 per
  release-built one on the baseline host, `TestConformance`'s 5,881
  processes cost about 400 CPU-s less before W2-3 removes most of them.
  `TestCalibrationLowered` is about 23 CPU-s and 31 s of serial wall time;
  the item measures the rest.

### W2-2 — the opening fan-out, by representative

ADR-012 D3.

- **Property kept:** every package-opening command refuses every refusal
  of a security `open` step with the same findings, the mapped exit code,
  and nothing written (ADR-011 D16 item 3).
- **Change:**
  - The runner chooses a representative per pair of code and package form,
    which runs every opening command. Every other security `open` step runs
    `inspect`.
  - Redefine `TestHostileVectorsOpenEverywhere`.
  - Under `-conformance.live`, every step keeps the full fan-out. If W2-3
    has not landed, a flag with the same meaning stands in for it.
- **What proves it after:** the representatives at every change; the
  redefined tripwire and its control; the full fan-out under
  `-conformance.live` at release.
- **Red first:** a new self-proof fault. A fake whose `ls` accepts a
  package with a bad digest must be caught. That needs the digest code's
  representative to run `ls`, and a selection that omits it fails the
  tripwire.
- **Saving:** 2,310 processes per environment become about 815. The item
  measures the CPU.

### W2-3 — one pass against a committed record, and the process budget

ADR-012 D2 and D9.

- **Properties kept:**
  - SC7's four checks, over every vector and every case;
  - the vectors' expectations;
  - ADR-009 D16 item 2's canaries;
  - the published method.
- **Change:**
  - `TestConformance` against the reference build becomes ADR-012 D2 item
    1's pass.
  - Add the per-vector record under `internal/conformance/testdata/record/`,
    `-conformance.regenerate`, its step in `regenerate.sh`, its line in
    `.gitattributes`, and `TestRecordsCurrent`.
  - `TestHostIsolation` runs only under `-conformance.reach`.
  - Add `-conformance.live`.
  - The canaries `pid`, `tz` and `clock` must be caught by both the record
    comparison and live A/B.
  - D9's tripwire.
  - `spec/conformance/README.md`'s "Running the suite", as ADR-012 D2 item
    8 says, correcting along the way its claim that the reach runs on
    every run.
  - The `verification-gate` skill's ladder for a differing record, and
    ADR-012 D12's method for publishing a cost.
  - `merge-check.sh` after the `.gitattributes` change. The wave tier runs
    before merge.
- **What proves it after:** the pass, at every change; the canaries,
  against both methods; `TestRecordsCurrent`; D9's tripwire; the release
  act's live run.
- **Red first:**
  - The `pid`, `tz` and `clock` canaries, run against the committed record
    with live B off, must each fail on the record comparison.
  - A record edited by one byte must fail the pass.
  - A plan that adds a third environment must fail D9's tripwire.
  - A fake that echoes a relative data path as an absolute one must fail
    check 2 at regeneration.
- **Saving:** `TestHostIsolation`'s 438 CPU-s and `TestConformance`'s
  remaining live B go. The package's process count falls from about 21,000
  to under 2,500. The item measures the CPU against §4's target.

## 9. Wave 3 — the detectors the pass makes redundant, and the measurement

### W3-1 — determinism and confinement keep what the pass does not check

ADR-012 D7 items 1 and 2, with ADR-005 A2 and ADR-010 A6's D13 item 4.

- **Properties kept:**
  - byte-identical outputs within one build (R10 items 2 and 3);
  - pack determinism under changed file times and modes;
  - no write outside the output directory, in-process (R12 item 1, SC6).
- **Change:**
  - `TestDeterminismDetector`'s main loop becomes its control's probe,
    clean and under `MapOrder`.
  - `TestPackDeterminism` keeps `disturb` and drops the `GOMAXPROCS` loop.
  - `TestConfinementDetector`'s main loop becomes the hostile traversal
    members, with `SkipConfine` as its control.
- **What proves it after:**
  - the pass's comparison of every vector against its record, from
    another process, environment and day (W2-3);
  - check 4 over the whole sandbox (W2-1);
  - the detectors' probes and controls;
  - `TestRenderUnderAClosedUmask`, for umask.
- **Red first:** each control stays red, and that is shown. A scratch
  commit makes one output depend on the process id. The pass's record
  comparison must catch it, not the shrunk detector.
- **Saving:** `TestDeterminismDetector` 31 s and `TestConfinementDetector`
  21 s of work, most of both; `TestPackDeterminism` 6 s, part of it.

### W3-2 — the train's measurement

The wave's verification measures every package of §4 as §4 says, on an idle
host. It reports each against its baseline and its target, and the
conformance package's process and build counts. It runs the pass on the
second platform if Wave 2's verification did not.

## 10. The ledger: every test this plan changes, and what still proves its property

| Test | Item | Change | Property | Still proved by |
|---|---|---|---|---|
| `TestConformance` | W2-3 | becomes the one pass | vectors' expectations; check 3 | the pass, against the record |
| `TestHostIsolation` | W2-3 | runs only the reach | SC7 checks 1 to 4 | the pass; the reach in the wave tier |
| `TestDriverProtocol/suite` | W1-6 | deleted | the protocol does not rot | the static round trip; `/fixtures`; the release act |
| security `open` fan-out | W2-2 | representatives | every command refuses alike | representatives; `-conformance.live` at release |
| `TestHostileVectorsOpenEverywhere` | W2-2 | redefined | the fan-out covers every refusal | itself, over representatives; its control |
| `TestCalibrationLowered` | W2-1 | release build, parallel | each case reaches its limit | itself |
| `TestRecordAndCompare` | W2-1 | one record run | record and compare plumbing | itself; `TestCompareRecordsEveryField` |
| `TestUnwritablePreState` | W2-1 | `cli` only, control short | the pre-state is applied | itself; `TestUnwritableSandboxIsLockedAndUnlocked` |
| `TestSelfProofAccruingRowsIgnoreWall` | W2-1 | unit test and one row | a starved process is CPU, not wall | the unit test; the row |
| `TestExistingWritesItsLimitUnderTheOpenFileLimit` | W1-1 | growth test | handles bounded | growth along three axes; calibration |
| `TestShapesWriteTheirLimitUnderTheOpenFileLimit` | W1-1 | growth test | handles bounded | growth along three axes; calibration |
| `TestUnpackWritesTheEntriesLimitUnderTheOpenFileLimit` | W1-1 | three axes, small | `Unpack`'s handles bounded | itself; calibration |
| `TestReleaseBuildsTwiceAndWritesDist` | W1-2 | one run | builds identical; archives deterministic | `copy0`/`copy1`; the in-process archive test; the release act |
| `TestReleaseRefusesAnUnreleasedRevision` | W1-2 | table | refusals | the table; one end-to-end case |
| `TestHarnessBuildsByTheReleaseLine` | W1-2 | moved to `internal/conformance` | the harness builds by the release line | itself |
| `TestGateSelectWorkingTree` | W1-3 | deleted | the selector's working-tree rule | `TestGateSelectWorkingTreeHostGit` |
| `TestGoCommandKeepsHostCToolchain` | W1-3 | moved to `internal/testenv` | `GoEnv` keeps the C toolchain | itself |
| guarantee builds and listings | W1-3 | shared | none changed | the same tests |
| `TestReadingRefusalsTriggered` | W1-4 | per family, static, clean | every refusal triggered | per-family packages; the static check; the reach |
| `TestInspectRefusalsTriggered` | W1-4 | walk deleted | as above, for `inspect` | `TestReadingRefusalsTriggered`; its own line cases |
| `TestACREditionDecidesTables` | W1-4 | 10 cases | tables by edition | itself |
| `TestPresentationNeverReachesJSON` | W1-4 | about 16 cases | no presentation in JSON | itself; its control |
| `TestHostileMarkdown` | W1-4 | frontmatter shape and two outline cases | bounded hostile Markdown | itself; `internal/markdown` `TestPathological` |
| `TestSC3StatusWidget` | W1-4 | reads the example | SC3 | itself; `TestExamplePackages` |
| `TestDeterminismDetector` | W3-1 | probe and control | byte-identical outputs | the pass's record comparison; itself |
| `TestPackDeterminism` | W3-1 | `disturb` kept, `GOMAXPROCS` loop dropped | pack determinism | itself; the pass over pack steps |
| `TestConfinementDetector` | W3-1 | hostile traversal members and control | confinement | check 4 over the whole sandbox; itself |

Not changed, though the audit named them: `TestUnmetRequirementIsSkippedAndReported`,
`TestBStartsTwoSecondsAfterA`, the Liquid import tests, the `cmd/metais`
builds, and the runner's machinery for live A/B (ADR-012's ledger says why).

## 11. Questions for Ed before this plan runs

Each is also in the design's pull request. The plan runs on the leans
unless he rules otherwise.

- **Q1. A committed record per vector (ADR-012 D2).** *Lean: yes.* The
  alternative is one live A/B pass: about twice the processes, and a
  two-second wait on every case.
- **Q2. A representative per finding code and package form (D3).** *Lean:
  per code, about 700 processes.* Per family would be about 210.
- **Q3. Full counts of limits at release only (D5).** *Lean: yes.* A change
  cannot shrink a host's limits; growth proves the property.
- **Q4. The render's one goroutine (D4 item 4).** *Lean: amend ADR-010 D2
  item 3 to name it, with W1-5's test as its proof.* It streams a
  container's zip through a pipe into the commit (`render_write.go:122-136`).
  Removing it means either buffering the zip in memory or turning the
  commit's write path from a reader into a writer callback, a change to
  `internal/hostfs`'s write API.
- **Q5. AGENTS.md's new conventions (D13) land with this design**, before
  the rework. *Lean: yes*, so Wave 1's own tests follow them.
