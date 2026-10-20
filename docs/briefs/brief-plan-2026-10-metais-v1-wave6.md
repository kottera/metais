# Brief — PLAN-2026-10-metais-v1, Wave 6: pack, and hardening

**Plan:** [PLAN-2026-10-metais-v1](../plans/PLAN-2026-10-metais-v1.md), §12
**Feature:** `metais-v1`. Work Orders are titled `metais-v1 [W6-<item>]: <action>`;
pull requests and their squash subjects are `<type>(metais-v1): [W6-<item>] <change>`
(AGENTS.md, Git and pull requests).
**Repository:** `kottera/metais`, branch `main`. Every Work Order of this wave
gets `repo: kottera/metais` and `branch: main`.
**Gated on:** Wave 5 and Wave 5b, both complete. Their verdicts are under
plan §11 and §11b, and their closeouts are in
[brief-plan-2026-10-metais-v1-wave5.md](brief-plan-2026-10-metais-v1-wave5.md)
and
[brief-plan-2026-10-metais-v1-wave5b.md](brief-plan-2026-10-metais-v1-wave5b.md).
The gate has cleared.

## What this wave is for

When Wave 6 is done:

- **`pack` closes the authoring path:** check, then test, then the
  deterministic write, round-tripping through `unpack`, and pinned from
  `init` through `check`, `test` and `pack`.
- **`test` compares container outputs by entries.**
- **The hostile corpus is complete,** and SC6 holds on the host: every
  `security` vector refuses through every opening command with exact
  findings, and nothing changes anywhere.
- **The SC7 harness reaches everything ADR-009 D16 item 1 lists,** and SC7
  holds on the host: every canary is caught and the release build is clean.
- **Limits and budgets are measured:**
  - the limit-calibration suite exists;
  - ADR-009 Amendment A1's schema compile bounds are in force;
  - the benchmarks have committed baselines.

  The calibration and the benchmarks are run on real hosts by §6's acts.
- **A release can be built:** the release builder builds twice and
  compares, and refuses what it must.

## What to read first

- **The plan:** §2, including the two breakdown rules Ed added (name the
  shared seams; allot statement-number blocks, where gaps are permitted,
  R-W5b-B2), §4, §5, §6 and §12. §12 is the authoritative scope, and Ed's
  design sessions have edited it: "Schema compile bounds (ADR-009
  Amendment A1)" is now one of its bullets. This brief binds §12 and does
  not restate its outline. Read the verdicts under §7 to §11b.
- **Wave 5's and Wave 5b's closeouts.**
- **The ADR decisions §12 carries:**
  - ADR-001 D4, through `pack`;
  - ADR-007 D1 item 7, D5 item 3 (containers) and D7;
  - ADR-008 D9 (`pack`);
  - ADR-009 D3 item 3, D16 and D17 in full, and D18;
  - ADR-010 D12, D14, and D15 items 1 and 2;
  - ADR-011 D15 (the matrix procedure) and D16 in full.
- **The amendments that bind this wave:**
  - ADR-009 A1 (schema compile bounds), Ed's ruling;
  - ADR-010 A5 (the two-tier gate), Ed's ruling;
  - ADR-005 A1 (existing destinations), Ed's ruling, because `pack`'s
    atomic replace writes through `hostfs`;
  - ADR-006 A1 and ADR-011 A4, Ed's rulings, which Wave 7 implements.
    Leave their codes to Wave 7.
- **AGENTS.md**, whole, and the `verification-gate` and `root-cause`
  skills, which every item follows.

## What earlier waves learned that changes this one

- **The gate is two-tier** (ADR-010 A5):
  - each item runs `gate.sh`'s scoped tier, in the background, and resumes
    it with `gate.sh --resume` after a fix or a rebase;
  - an item that touches a shared seam (`go.mod`, `go.sum`,
    `internal/fault`, `internal/testenv`, or the gate's own scripts) also
    runs `gate.sh wave` before merging;
  - sessions run `merge-setup.sh` before their first rebase and
    `regenerate.sh` after every rebase;
  - the verification round runs `gate.sh wave` on a fresh clone;
  - every run is on go1.26.9 with `GOTOOLCHAIN=auto`, and the PR records
    `go version` and GOOS/GOARCH.

  A red is root-caused by the `root-cause` skill. "It passed alone" closes
  nothing.
- **The scoped tier's selector has been tested only by its producer.**
  W5b-1's `--resume` and selector were inspected, not mutated. Because the
  scoped tier now decides what each item tests, this wave's verification
  round mutates the selector too: a change it fails to reach is a
  blocking finding.
- **Conventions from Wave 5b:**
  - ADR implementation-log entries are files under
    `docs/adr/logs/ADR-<NNN>/<address>.md`, never appended to the ADR;
  - adding a finding code edits one hand-written place and regenerates;
  - help groups generated refusals by family, so a new refusal changes one
    family line;
  - `internal/cli/render.go` is split by phase.
- **`TestConformance` holds no slot through its A/B wait** (#112). This
  wave adds the full hostile corpus. Keep the default run within ADR-011
  D11 item 9's 60 seconds, and report its time in every PR that adds
  cases.
- **Write the squash message explicitly, under the account's own identity.**
  36 commits on `main` carry `Co-authored-by` trailers, which AGENTS.md
  forbids. GitHub adds one to the default squash body whenever a commit's
  local author identity differs from the account that merges: a
  machine-local address such as `user@host.local` is enough. So:
  - commit under the GitHub account's noreply address;
  - merge with `gh pr merge --squash --subject ... --body-file ...`;
  - check the merged commit's message before you report.
- **Name every caller of a shared code before filing,** and settle meeting
  owners at breakdown.
- **What the plan's §5 leaves to §6's matrix act**, which this wave
  rehearses:
  - the operating-system branches;
  - the pseudo-terminal proof;
  - W5b-7's merge drivers, which have run on linux only.

## Rulings already made — do not reopen

From plan §12, "Decisions settled before filing", ratified with RULINGS-v1:

- **R1-015:** a calibration result that breaks a ceiling is escalated as an
  ADR-009 amendment, never applied silently.
- **R2-102, R2-103, R2-104 and R2-123:** `pack`, and containers in `test`.
- **R2-137, R2-139, R2-142 and R2-147 to R2-150:** limits, links, canaries
  and the corpus.
- **R2-161:** the release is built twice.
- **R2-175:** the matrix.
- **R2-178:** one defect per hostile package.

Ed's rulings bind this wave:

- ADR-009 A1 (two new package limits, refused as `limit.package`, with
  defaults set at calibration);
- ADR-010 A5 (the gate);
- ADR-005 A1 (existing destinations);
- the plan's §2 breakdown rules.

The plan orchestrator's earlier rulings stand, recorded under plan §7 to
§11b.

Rulings and constraints from the plan orchestrator for this wave:

- **The release builder waits on Ed.** `internal/tools/release` must run
  `go build`, read git and its environment, and write files. ADR-009 D16
  item 3's symbol rule refuses all of that in every non-test file of the
  module, and names no allow-list entry for it. Plan §4 forbids adding one
  to get a tripwire green.
  - Design session `01a1566e-2786-7173-84ff-c1485fcf8cdf` asks Ed how the
    builder is allowed.
  - The item that builds it does not add an allow-list entry or change the
    tripwire's scope on its own. It is filed but waits for Ed's ruling,
    which I relay to the wave orchestrator when the session ends.
  - The rest of the wave does not wait on it.
- **A calibration or benchmark result that breaks a ceiling** is escalated
  through the wave orchestrator, as R1-015 says. No default is changed in
  this wave. The real numbers come from §6's acts on real hosts, because a
  contended sandbox gives numbers but not evidence (§5).
- **Codes no vector can produce.** Ed ruled them (ADR-006 A1, ADR-011 A4),
  and Wave 7 carries the work. Codes this wave registers must each be
  producible by a vector in this wave, or be listed for the wave that wires
  them. A code no vector can produce on any platform is a finding for Ed,
  not a permanent line.

## Rules carried whole from the plan (§2 and §4)

- **The specification is written with the code, never after.** An item
  that makes a behaviour reachable from `metais` adds, in the same pull
  request:
  - the `spec/` statements that state it, numbered from its allotted block;
  - the vectors that cite them;
  - any finding codes it emits.

  `TestCoverage` makes a statement without a vector a red gate. Where the
  tool and the text disagree, the tool is wrong. Every vector runs in both
  of ADR-011 D12's environments.
- **Where two items of one wave meet, the one that merges second adds the
  vector for where they meet.** Name each meeting and its owner in the
  criteria. The verification round checks every meeting. This wave's most
  important meeting: every hostile case refuses identically through every
  opening command, `pack` included.
- **Shared seams are named at breakdown** (§2). Items that touch one run in
  sequence or are split by phase. This wave's likely seams, as I see them:
  - the conformance corpus tree;
  - `internal/guarantees` (SC7 and the tripwire);
  - `internal/limits`;
  - `internal/author` (`pack` and `test`);
  - the `hostfs` write side.

  The derivation decides.
- **An item that finds it needs a sibling's unmerged work** that no
  sanctioned edge covers has found a slicing defect, and it parks.
- **Pin the property false first.** Each of these is red against a tool
  that breaks it, and stays in every `go test`:
  - **refusal of every hostile package:** each hostile case lands red
    against a tool that accepts it, and the digest detector keeps
    `SkipDigest`;
  - **no host data reaches a template, an output or a finding (SC7):**
    every canary is caught, and the release build is clean;
  - **no case exits through a panic or a runtime fatal error** (ADR-009
    D18);
  - **byte identity of `pack`'s output within one build.**

  The detectors already in place keep their faults.
- **Do not widen a refusal to make a proof pass.** None of these happens to
  get a proof green:
  - no verification step skipped (R1-013);
  - no limit raised (R1-015);
  - no format or keyword let through (R1-016);
  - no portable-path relaxation (R1-023);
  - no Golden Liquid exclusion except a `dialect` line naming its ADR-004
    item;
  - no runner skip except a `requires` probe;
  - no released vector's expectation changed;
  - no tripwire allow-list entry beyond what ADR-009 D16 item 3 and R2-143
    prescribe.

  A proof that cannot pass without one of these is a finding for Ed,
  through the wave orchestrator.
- **The stricter security rule wins** where two ADRs, or an ADR and the
  specification text, disagree on what is accepted or refused. Name it in
  the PR, and record it in the closeout.
- **No new dependency.** Only ADR-010 D3 item 3's modules, as amended.
  Never copy code from another tool.
- **Bounded work is measured in CPU time**, never by wall time alone. That
  holds for the calibration suite and the benchmarks above all.
- **Every test is isolated** through `internal/testenv`.
- **A Work Order never runs `go test -fuzz`.**
- **Never write to another repository.**

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

In scope: everything in plan §12's scope outline, ADR-009 A1 included.

These belong to Wave 7 or later and must not be absorbed here (plan §12
"Does not touch", §13):

- the example packages of PRD §4.1 to §4.5, and the worked package.
  §12's example-package harness runs over `examples/` and "reports when
  there is none". Wave 7 fills it;
- the specification's 1.0 text complete, the README, documentation and the
  ADR implementation logs' close-out;
- emptying `unwired.txt`'s W7 entries, including ADR-006 A1 and ADR-011
  A4's work;
- the render-pipeline extension points (after the train, §14);
- the release itself, the matrix's release run, and every live round (§6).

## What must not break, and how it is checked

- **The gate:** the scoped tier per item, the wave tier for a shared seam,
  and the wave tier at verification on a fresh clone.
- **Plan §12's acceptance**, each criterion proved in the tree or by the
  act named:
  - SC6 holds on the host;
  - SC7 holds on the host, with every canary caught and the release build
    clean;
  - the packed scaffold and a fixture survive the round trip with equal
    package digests and, within this build, equal bytes;
  - the release builder refuses each of its refusals, and produces
    matching `SHA256SUMS` on two runs. This rests on Ed's design-session
    ruling;
  - the matrix and calibration acts of §6 are filed.
- **`TestConformance` stays within 60 seconds** with the full corpus.
- **Every earlier guard stays green:**
  - every detector with its fault, and every canary;
  - `TestNoHostFacts` and the layering table;
  - the document tripwires;
  - `TestDelegatedRun` and `TestAdapterSetup`;
  - `TestNoDeclarationStops`;
  - the merge drivers and regeneration.
- **What a sandbox cannot prove** (plan §5): absolute CPU budgets and limit
  calibration, cross-platform byte identity, the other operating systems'
  branches, and the pseudo-terminal proof. The suites are written here and
  run by §6's acts. No PR claims a measurement a sandbox cannot give.

## Out-of-band acts of this wave

From plan §6, carried whole. The wave orchestrator places each. Both have
**Before: the v1.0 release**, so both are carried, not waited on, and are
filed when the items they need have merged.

1. **Run the four-target matrix.** The suite with `-conformance.record` on
   darwin/arm64, darwin/amd64 under Rosetta 2, linux/arm64 and
   linux/amd64, then `-conformance.compare`.
   - **When:** Wave 6, as a rehearsal of the procedure; again at release.
   - **Who:** Ed.
   - **Notes:**
     - ADR-011 D15 and SC5. darwin/arm64 against linux/amd64 is the
       minimum pair.
     - The rehearsal also runs the operating-system branches and the
       pseudo-terminal proof of §5. Add W5b-7's merge drivers on darwin,
       which have run on linux only.
2. **Run the limit-calibration suite and the benchmarks on each supported
   target.**
   - **When:** Wave 6, after both items merge.
   - **Who:** Ed.
   - **Notes:**
     - ADR-009 D3 item 3 and D18 item 4, and ADR-010 D15 item 2.
     - A default whose worst case breaks a ceiling is lowered only by an
       ADR-009 amendment Ed ratifies (R1-015). That includes ADR-009 A1's
       `schema-subschemas` and `schema-location-length` and their joint
       worst case.
     - Budgets go in ADR-010's implementation log.

For both, the off-tera type is `manual-qa-pass` (`build`, `focus`). The
`focus` names:

- the exact commands, as committed;
- the hosts and their versions to record;
- the evidence to report back: the record files' hashes and the compare
  result for the matrix, the per-target numbers against each ceiling for
  calibration, and anything that failed.

W4-A1R's report is the model.

## Closeout

The wave orchestrator wrote this closeout (Work Order
`01a15670-f61e-7138-bb09-12d40ee762c3`) after the wave drained. It reports
what landed and what carries forward. **It is not a correctness verdict.**

**What I checked, and how:**
- I read every child's report, the verifier's full report, and `main`'s
  log from `1f6d812` to `e6be779`.
- In the merged tree at `e6be779`, I read:
  - `internal/cli/pack.go`: it exists;
  - the wave tier's reach step in AGENTS.md and `gate.sh`;
  - `unwired.txt`: no W6 line remains, and `manifest.any_of` and
    `manifest.one_of.ambiguous` are relisted W7;
  - the silent-ignore rule for `*.metais` in `internal/hostfs/source.go`;
  - grep's default `--max-results` in `internal/cli/browse.go`;
  - the commands in `docs/dev/calibration.md` and `docs/dev/benchmarks.md`.
- I checked every commit in the range for a `Co-authored-by` trailer.
  None of this wave's has one; the brief's squash instruction held.
- I ran no test, no gate and no mutation. Where something is said below
  to be proved, the proof is the producer's or the verifier's, and is
  attributed to them.

**Wave range:** `1f6d812..e6be779` on `main`.

- **The breakdown:** thirteen items were derived and filed as drafts. The
  plan orchestrator approved them with readings (a) to (c). Twelve were
  released as one set. W6-13, the release builder, was held, and later
  cancelled by ruling (below).
- **W6-14:** a corrective item I filed mid-wave (R-W6-2).
- **Verification:** one round, W6-V. It found two blocking defects, both
  in the gate's own scripts. One fix child, W6-V1, merged them; it is
  not re-verified.
- **Non-blocking findings:** two children, plus decisions not to file
  (question 4). **The wave completed without waiting for them, and for
  the two carried acts, deliberately.**
- **Not this wave's commits:** `5e6c9ec` (W5b-NB2, #124), `bee5a4a`
  (W5b-NB1, #125), `73b1758` (#136) and `f995ae5` (#140). The last two
  are the plan orchestrator's records of R-W6-3 and of the release
  builder's move. `5e6c9ec`, from Wave 5b, carries a `Co-authored-by`
  trailer.
- **People consulted:** the breakdown review went to the plan
  orchestrator, as did three escalations:
  - W6-3's reach (R-W6-3);
  - Ed's W6-13 ruling, asked for twice, which ended in the release
    builder's move to Wave 7.

### 1. Did each item land, and does it match the ask?

| Item | Work Order | PR | Merge | Landed |
|---|---|---|---|---|
| W6-1 `pack` | `01a15678-894f-7aba-8025-ccdbe7ef9528` | #128 | `f9d3f50` | `metais pack`: check, then test, then the write. The default name, `--out` a directory, the atomic replace, `--out -`, `--dry-run` (read as unpack's). Pack is in the hostile open set (147 security vectors refuse through it). The round trip `pkg/pack-round-trip`. `TestPackDeterminism` with fault bit 9 (`PackMapOrder`). Statements MTS-AUT-043 to 052 and MTS-PKG-034 to 040 |
| W6-2 the authoring path pinned | `01a1567b-cad4-7309-a066-ca34650107b9` | #138 | `564fc20` | `TestInitScaffoldPinned`. The example-package harness `TestExamplePackages`, which reports that `examples/` is absent. The runner scans every package a pack step writes, with a self-proof. `BenchmarkPack`. Merged over a pre-existing red under R-W6-2 |
| W6-3 the SC7 harness's reach | `01a15678-f4be-775a-9aad-7872f3da6d08` | #133 | `fd59124` | The added variables, decoys, both path modes, check 2, the release build over every vector, and the crash check for D18. The every-command reach runs over every vector in the wave tier, by R-W6-3, and over the canaries' probes in every `go test` |
| W6-4 containers in `test`; the pty proof | `01a15679-a1f9-7330-8abd-628a9d1909a7` | #132 | `7daca16` | Comparison by entries, with the `example.entry_*` codes and `example.expected_unreadable` (MTS-AUT-065 to 071). `TestPresentationUnderTerminal` in `internal/hostfs`, with fault bit 10 (`ColourJSON`). **It ran on no host** (question 2) |
| W6-5 schema compile bounds | `01a15679-460f-7ab2-a4da-8f539bd0ae90` | #126 | `1640d19` | ADR-009 A1: `schema-subschemas` and `schema-location-length` in D3's table, in `internal/limits` and in the pre-check, with vectors |
| W6-6 the calibration suite | `01a1567c-20fa-7259-920a-bf2c8c80d5d8` | #134 | `14f5802` | `TestCalibration` (opt-in, `-calibration.record`) and `TestCalibrationLowered` in every `go test`. The runbook is `docs/dev/calibration.md`. No default changed |
| W6-7 the benchmarks | `01a15679-fcf9-77bc-b3fa-0dcbbacfad0b` | #127 | `3638cf9` | ADR-010 D15 item 1's benchmarks, with baselines labelled as sandbox figures. The runbook is `docs/dev/benchmarks.md` |
| W6-8 corpus: the package bytes | `01a1567a-650b-7167-a970-1d2225a34cd4` | #130 | `4e0c96c` | D17 items 1 to 4 and ADR-001's list, with the package-byte limit pairs. A zip64 extra-field check was fixed red first |
| W6-9 corpus: manifest and YAML | `01a1567a-ba62-767c-a395-d88cd450428e` | #129 | `d3b9b3d` | D17 item 5. Eight of the ten W6 `manifest.*` codes are wired. The YAML-profile cases use validate and check steps (R-W6-9b). `internal/cli`'s corpus walk is concurrent (R-W6-9a) |
| W6-10 corpus: schemas and patterns | `01a1567c-7dba-79c7-ad14-805ed904ba31` | #137 | `05aa706` | D17 item 6 and its limit pairs. **`schema-work` is now enforced** (`internal/schema/work.go`). Two `internal.error` exits were fixed red first |
| W6-11 corpus: templates and outputs | `01a1567b-0f35-7870-baad-6a849380f7ec` | #135 | `7e65ea7` | D17 items 7 and 8, ADR-005's path cases, and their limit pairs. Every case is met at render |
| W6-12 corpus: inputs, Markdown and commands | `01a1567b-6ac7-79a4-a865-5fa5adbe5f98` | #131 | `a0b34a7` | D17 items 9 to 11. The grep, outline and extract cases are `internal/cli` tests; the devices are `internal/hostfs` tests. A lowered `grep-matches` now caps grep's default |
| W6-14 TestHostIsolation fits | `01a1576c-3195-72f1-b3a2-69b217188f03` | #139 | `cbd7960` | Corrective: `internal/conformance` fits go test's 10 minutes on 12-core linux with nothing dropped, by removing a per-process stub cost on linux. Darwin was not run (R-W6-14) |
| W6-V1 blocking fix | `01a15804-3ba2-7bd7-b5a1-72ae7e2d9762` | #141 | `e6be779` | `merge-check.sh` plants statement ids after the tree's highest. `gate-select.sh` sends a `.go` file in no package to every test that names the module root, with new selector tests, red first |

**The plan's §12 acceptance:**
- **SC6 holds on the host:** the verifier's wave tier ran every `security`
  vector through every opening command, `pack` included, green on 12-core
  linux.
- **SC7 holds on the host:** the wave tier ran the full reach, all 25
  groups, green, and `TestHostIsolationCanaries` catches every canary.
- **The round trip:** `pkg/pack-round-trip` for the scaffold and a
  fixture, and `TestPackRoundTripBytes`.
- **The release builder is NOT met in this wave.** §12's criterion "the
  release builder refuses each of its refusals, and produces matching
  `SHA256SUMS` on two runs" is not met. By the plan orchestrator's ruling,
  the builder carries to Wave 7, pending Ed's design session
  `01a1566e-2786-7173-84ff-c1485fcf8cdf` (plan §12, #140). Its draft,
  quoted below, is Wave 7's starting point.
- **The matrix and calibration acts are filed** (question 5).

**Divergences from the ask:**
- **W6-9.4 is not fully met.** `manifest.any_of` and
  `manifest.one_of.ambiguous` cannot be produced by any 1.0 manifest. The
  manifest schema's only `anyOf` sits under `not`, and its `oneOf`s are
  over closed, disjoint branches. W6-9 relisted both W7, for Ed (question 3).
- **W6-3.1** runs the reach over every vector in the wave tier, not in every
  `go test`, by R-W6-3.
- **W6-14.1's darwin half is a declared gap,** by R-W6-14. The matrix act
  carries it.
- **W6-4.5's pseudo-terminal run executed on no host:** every sandbox,
  the verifier's included, denied the slave open, and the `requires`
  probe skipped it. The matrix act carries it.
- **W6-11.2:** no template case is refused at open, so meeting M1 is
  vacuous for W6-11.

### 2. What children flagged, degraded, declined or discovered

None of this was verified by me.

- **R1-015 signals from sandboxes, which are not evidence.** On a
  contended linux sandbox, W6-6's calibration broke nine ceilings at the
  defaults:
  - render-work: 47.9 s of CPU;
  - loop-iterations: 21.9 s (W6-11 saw 10.7 s on darwin);
  - regex-work: 122.5 s;
  - grep-context: 26.0 s and 9,478 MiB;
  - schema-joint-combined: 17.3 s;
  - output-total: 1,817 MiB;
  - schema-work: 98 s and 6,793 MiB, before W6-10 enforced it.

  W6-10 measured about 50 µs of CPU per nested `anyOf` evaluation. W6-7's
  `BenchmarkValidate` took 0.9 to 1.8 s against D15 item 2's 500 ms, about
  80% of it in `hostfs` syscalls. **No default was changed.** The
  calibration act measures these on real hosts, and a breach there becomes
  an ADR-009 amendment for Ed.
- **ADR-011 D16 item 3 conflicts with MTS-AUT-002** for every
  content-class manifest refusal. MTS-AUT-002 makes `check` add
  `report.step_skipped`, so the opening commands cannot all give "the same
  findings". W6-11, then W6-9 under R-W6-9b, prove those cases through
  validate and check steps instead of an open step. This is for Ed.
- **Stricter rules that won, each named in its PR:**
  - pack opens first, then refuses a packed operand (reading (c));
  - pack refuses an output at `metais.yaml`;
  - chapter 2's device names follow ADR-005's fuller list. The text lags,
    which is W6-NB1, member D.
- **Readings, each in its ADR's log:**
  - W6-1: `--dry-run` reads as unpack's dry run.
  - W6-4:
    - `OpenContainer` lifts ADR-001's second-end-record search for expected
      containers;
    - the rendered side is compared from the plan;
    - the pty test lives in `internal/hostfs`, because only `hostfs` may
      import `x/sys`.
  - W6-5: output-validation schemas go through the same pre-check.
- **Seen in several items: the shared host makes the gate slow.**
  `internal/cli` reached 520 to 566 s under `-race`, near the 600 s
  timeout, before R-W6-9a's concurrent walk.

  `internal/conformance` timed out at 609 s on main, which became W6-14.
  The verifier measured 329.9 s and 159.5 s after W6-14. W6-14 names the
  next lever: `TestRealToolInRecord`, about 900 s of CPU.
- **Dispatch hosts default to `GOTOOLCHAIN=local` (go1.26.2).** Every item
  set `auto` by hand, and the gate exports it. `docs/dev/benchmarks.md`
  does not, which is W6-NB2, member D.

### 3. What carries to the next wave

- **For Wave 7's brief:**
  - **The release builder,** pending Ed's ruling in design session
    `01a1566e`. W6-13's draft (`01a1567c-e3db-7d2f-b571-f197f0c9c8a0`,
    cancelled, its task text kept) is the starting point. The part that
    carries:

    > **Scope.** Carries ADR-010 D12 items 1, 2, 4, 5 and 6, ADR-011 D5
    > item 6, and RULING R2-161. `go run ./internal/tools/release -version
    > <semver>` (or where the ruling places it) builds each shipped target
    > (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, with
    > `GOAMD64=v1` and `GOARM64=v8.0` set explicitly) with D12 item 1's
    > static build line. It builds each twice, in two fresh directories with
    > separate `GOCACHE`s, and refuses to publish unless the two binaries'
    > SHA-256 are equal. It refuses a dirty working tree, a `HEAD` that is
    > not the tag, a binary whose `go version -m` lacks `CGO_ENABLED=0`,
    > `-trimpath=true` or the expected GOOS/GOARCH, and an `implements`
    > naming an unreleased specification revision. It writes
    > `metais_<version>_<os>_<arch>.tar.gz` per shipped target, holding
    > `metais`, `LICENSE`, `NOTICE` and `THIRD-PARTY-NOTICES.txt`, written
    > deterministically, and `SHA256SUMS`. Wave 5 found that `archive/tar`
    > imports `os/user`, and `internal/tarw` writes the archive instead;
    > D12 item 5 names `archive/tar`, so decide under the stricter-rule
    > convention and the ruling. Do not add a tripwire allow-list entry or
    > change the tripwire's scope beyond what the ruling prescribes.
    >
    > **Acceptance criteria.**
    > - W6-13.1 The release builder builds each shipped target twice, in
    >   two fresh directories with separate `GOCACHE`s and with `GOAMD64=v1`
    >   and `GOARM64=v8.0` set explicitly, and refuses unless the two
    >   binaries' SHA-256 are equal.
    > - W6-13.2 It refuses a dirty working tree, a `HEAD` that is not the
    >   tag, a binary whose `go version -m` lacks `CGO_ENABLED=0`,
    >   `-trimpath=true` or the expected GOOS/GOARCH, and an `implements`
    >   naming an unreleased specification revision, each refusal proved by
    >   a test.
    > - W6-13.3 It writes `metais_<version>_<os>_<arch>.tar.gz` per shipped
    >   target, holding `metais`, `LICENSE`, `NOTICE` and
    >   `THIRD-PARTY-NOTICES.txt`, written deterministically as ADR-010 D12
    >   item 5 states, and `SHA256SUMS`; two runs produce matching
    >   `SHA256SUMS`.
    > - W6-13.4 `TestNoHostFacts` stays green with no allow-list entry and
    >   no change of scope beyond what Ed's ruling prescribes, and the
    >   release binary is clean under the SC7 harness.
  - **`examples/` is empty,** so `TestExamplePackages` skips. Wave 7's
    example packages make it run.
  - **The matrix and calibration acts** are open (question 5). Their
    results may become an ADR-009 amendment, or fixes that land in Wave 7.
- **For the plan orchestrator, or Ed:**
  - `manifest.any_of` and `manifest.one_of.ambiguous`, which cannot be
    produced (W6-9). The brief says such a code is "a finding for Ed, not
    a permanent line". They now sit on W7 lines.
  - ADR-011 D16 item 3 against MTS-AUT-002 (question 2).
  - `check` accepts a literal manifest output path `../escape.txt`, which
    `test` and `render` refuse. No chapter 13 statement asks `check` to
    judge it: a gap in the specification, not a tool defect (verifier).
  - **To ratify:**
    - R-W6-3, which extends A5: the plan orchestrator recorded it in §12;
    - fault bits 9 to 10 and their names;
    - `internal/hostfs` hosting the pty test;
    - R-W6-9b's form for content-class hostile cases.
- **What I would do differently:**
  - **Check a shared test's run time against the corpus it walks before
    filing a corpus.** Two whole-corpus walks, in `internal/cli` and
    `internal/conformance`, crossed go test's 10-minute timeout as the
    corpus grew. That cost one escalation, one corrective item and two
    wake-ups. A corpus wave should own its harness's timing from the
    start.
  - **Scan the gate's own fixtures for numbers a breakdown is about to
    allot.** I allotted MTS-AUT-043 onward to W6-1 without seeing that
    `merge-check.sh` planted MTS-AUT-043/044. That cost the blocking
    finding.
  - **Ask the off-tera ruling's owner at breakdown whether it will come
    within the wave.** W6-13 was held for the whole wave and then moved.

### 4. What the verification round found, and where every finding went

**Verifier:** W6-V, `01a157e8-626b-7bbf-827d-397a0caed1db`
(`wave-verification`).

- **The gate:** `gate.sh wave` on a fresh clone of `main` at `f995ae5`,
  on 12-core linux/amd64, go1.26.9, `-p 4 -parallel 4`. **GREEN:**
  - 42 packages;
  - all 25 groups of R-W6-3's reach step;
  - `internal/conformance` 329.9 s and `internal/cli` 159.5 s;
  - TestConformance's default run 18.9 s, over 457 vectors and 674 cases.
- **The verdict: blocking**, with two findings.
- **Coverage:**
  - all 13 ids I sent came back;
  - all 13 merges are ancestors of `main`, and the four other commits in
    the range are the ones I named;
  - completeness of my list is trusted, not verified.
- **Depth:** twelve items were **mutated** (about 30 mutations, plus three
  claimed-red rows re-run, all red again). **W6-7 was only inspected:**
  its benchmarks ran and its baselines are present, but nothing was
  broken.
- **What it mutated, in brief:**
  - pack accepting every refused open turned 155 vectors red (M1);
  - removing the scan of pack's output turned the sentinel-pack
    self-proofs red (W6-2.3);
  - dropping the crash check turned `TestSelfProof/panic` and `fatal` red
    (W6-3.7);
  - comparing containers by bytes turned `aut/pack-containers` red (M3);
  - W6-4's `entry_order` reporting;
  - the `$id` half of A1's location length;
  - the zip64 extra-field check;
  - manifest-size and number-length off by one;
  - schema-work one unit short;
  - the ¹²³ device names;
  - grep's default `--max-results`;
  - the findings cap;
  - W6-14's race-built stub.

  M2 held in the wave tier.

**The blocking findings and their fix child.** It is not re-verified:
one round, then a hard stop.

| Finding | Source quoted | Fix child |
|---|---|---|
| `merge-check.sh` red on main: its planted MTS-AUT-043/044 collide with W6-1's pack statements (cross-item) | Brief: "Every earlier guard stays green: […] the merge drivers and regeneration." | W6-V1 `01a15804-3ba2-7bd7-b5a1-72ae7e2d9762`, #141 `e6be779`, member A |
| `gate-select.sh` reaches nothing for a `.go` file in no listed package (a build-ignored `tools/release.go` without a licence header), which `TestLicenceHeaders` fails (cross-item) | Brief: "this wave's verification round mutates the selector too: a change it fails to reach is a blocking finding." | W6-V1, member B |

The two findings share one child: both are in the gate's scripts, one
seam and one gate.

**Non-blocking findings.** Each is listed with where it went. **The wave
completed without waiting for any child below, deliberately.** They run
beside the next wave and are not re-verified.

| Finding | Item | Where it went |
|---|---|---|
| A top-level *directory* named `*.metais` is unpinned: dropping `hostfs/source.go`'s regular-file test leaves every test green (MTS-AUT-047) | W6-1 | W6-NB1 `01a15821-e3c7-7f0f-824c-26cb1e992efa`, member A |
| Pack refuses a package whose stored asset ends with an end-record signature, as R1-014 rules. But its hint ("Pack the package again") and next step (`metais check`) do not help an author. The report that pack *writes* such a package does not reproduce | W6-8 / W6-1 | W6-NB1, member B |
| grep's truncation line credits `--max-results` when a lowered `grep-matches` did the capping. W6-6's report that the limit caps nothing does not reproduce | W6-6 / W6-12 | W6-NB1, member C |
| Chapter 2's entry-name device list omits ¹²³ and the trailing-space trim, which ADR-005 and the tool include. Normative text is wrong, so this is not janitorial | W6-11 | W6-NB1, member D |
| `outline` in the every-command reach is unpinned | W6-3 | W6-NB2 `01a15822-3703-7455-86b5-c54ca3c65732`, member A |
| Calibration's ceiling judgement has no every-`go test` proof | W6-6 | W6-NB2, member B |
| Nothing pins a benchmarked package's baseline | W6-7 | W6-NB2, member C |
| `docs/dev/benchmarks.md` relies on the default `GOTOOLCHAIN`, and hosts here default to `local`. The act reads it as its runbook | W6-7 | W6-NB2, member D |
| W6-9.4: two `manifest.*` codes cannot be produced by any 1.0 manifest | W6-9 | **Not filed:** Ed's call under the brief's rule. Relayed in question 3 |
| `check` accepts the output path `../escape.txt` | W6-11 | **Not filed:** a gap in the specification needing a ruling on chapter 13. Relayed in question 3 |
| The pty run executed on no host | W6-4 | **Not filed as a child:** the matrix act W6-A1 runs it |
| Three `hostfs` device tests skip in sandboxes | W6-12 | **Not filed as a child:** W6-A1 runs them |
| W6-14.1's darwin half is open | W6-14 | **Not filed as a child:** W6-A1 runs it |
| `TestExamplePackages` skips | W6-2 | **Not filed:** by design, Wave 7 fills `examples/` |
| The verifier's shell had `GOTOOLCHAIN=local` | — | **Not filed:** an environment fact, already covered by NB2 member D |
| The `schema-work` report | W6-6 / W6-10 | **No finding:** W6-10 enforces it, and its pair went red under mutation |

**Why the members are grouped:**
- **W6-NB1** groups members that touch `internal/cli`, `internal/hostfs`
  and `spec/`, which the gate selects together.
- **W6-NB2** groups guards that a mutation or a host showed blind, plus
  their runbook.

Both tasks say the members are severable.

**Where the verifier found nothing:** W6-5 and W6-10 have no finding, and
every mutation on them went red. W6-7 was only inspected.

### 5. Out-of-band acts

| Act | Type | Work Order | Placement |
|---|---|---|---|
| W6-A1: rehearse the four-target matrix (ADR-011 D15). It also covers the operating-system branches, the pseudo-terminal proof, the skipped device tests, darwin's `internal/conformance` time, W5b-7's merge drivers on darwin, and the open-format check carried from Wave 5 | `manual-qa-pass` | `01a15822-b803-7c96-9673-42aaf46e69b5` | **Carried, not waited on.** `build` is `e6be779`. The repository holds no matrix runbook, so its `focus` gives ADR-011 D15's commands and asks for the form that worked |
| W6-A2: run the calibration suite and the benchmarks on each supported target | `manual-qa-pass` | `01a15822-7680-746c-b98e-dc38260f857a` | **Carried, not waited on.** `build` is `3638cf9`, which holds W6-6 and W6-7. Its `focus` carries the committed commands, the sandbox breaches to settle, and R1-015 |

Both are open when this closeout merges.

### Rulings made during this wave

- **The breakdown, approved by the plan orchestrator:**
  - (a) the W6 `manifest.*` codes are W6-9's;
  - (b) the pty proof is skipped only through a `requires` probe;
  - (c) pack opens first, then refuses the wrong kind of operand (the
    stricter rule).
- **R-W6-3** (plan orchestrator; plan §12, #136): SC7's every-command reach
  over every vector runs as a wave-tier step, split by area within the
  10-minute timeout. Every `go test` keeps `TestHostIsolation` over every
  vector, with all checks, and the reach over the canaries' probes. No
  check is dropped and no timeout is raised. It extends ADR-010 A5, for
  Ed to ratify.
- **R-W6-9a** (wave orchestrator): no merge over a red. W6-9 made
  `TestReadingRefusalsTriggered`'s corpus walk concurrent.
- **R-W6-9b** (wave orchestrator): content-class manifest and YAML hostile
  cases are `security` vectors with validate and check steps, not an open
  step. This follows W6-11's precedent; the spec tension goes to Ed.
- **R-W6-2** (wave orchestrator): W6-2 merged over a `TestHostIsolation`
  timeout reproduced on unmodified `main`, with its evidence recorded.
  Corrective W6-14 owns the defect.
- **R-W6-14** (wave orchestrator): W6-14 merged on linux evidence. Darwin
  is a declared gap, and the matrix act carries it.
- **The release builder moves to Wave 7** (plan orchestrator; plan §12,
  #140). W6-13's draft was cancelled.
