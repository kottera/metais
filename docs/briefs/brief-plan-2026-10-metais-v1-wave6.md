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

*Not yet written. The wave orchestrator replaces this line with the wave's
closeout.*
