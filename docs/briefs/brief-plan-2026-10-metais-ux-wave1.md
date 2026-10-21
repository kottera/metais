# Brief — PLAN-2026-10-metais-ux, Wave 1: the harness and the baseline

**Plan:** [PLAN-2026-10-metais-ux](../plans/PLAN-2026-10-metais-ux.md), §5
**Feature:** `metais-ux`. Work Orders are titled `metais-ux [W1-<item>]: <action>`;
pull requests and their squash subjects are `<type>(metais-ux): [W1-<item>] <change>`
(AGENTS.md, Git and pull requests).
**Repository:** `kottera/metais`, branch `main`. Every Work Order of this wave
gets `repo: kottera/metais` and `branch: main`.
**Gated on:** Wave 0, the plan itself, merged to `main` in
[#183](https://github.com/kottera/metais/pull/183) and ratified 2026-10-21.
The gate has cleared.

## Harness policy

Claude throughout, by plan §2 (which adopts PLAN-2026-10-metais-v1 §2's
policy whole). File every item by these types and no other; if a seat is
not available, wait for it rather than substituting another harness's type.

| Role | Work Order type |
|---|---|
| Wave derivation and supervision | `wave-orchestrator-derive` |
| Implementation and corrective work | `code-implementation-direct` |
| Whole-wave verification | `wave-verification` |

## What this wave is for

When this wave is done, anyone can watch a fresh agent use `metais` the
same way twice, and today's binary has a number:

- **`docs/dev/ux-testing.md`** states the method: the scenarios, the
  isolation rules as checks the runner makes, the models, what each run
  records, the scorecard's format, and how a scenario is added.
- **The scenario set** of plan §5 exists, one directory per scenario with
  its prompt, its fixtures and its outcome check.
- **The runner** runs scenario × model in isolated nested `claude -p`
  sessions, writes each run's record, paces itself, resumes a partial
  scorecard, and prints the scorecard. The binary can never reach it, and
  `internal/guarantees` proves that.
- **The scorer** reads a run's stream-json transcript and directory and
  computes plan §4 R-UX-5's pass rules and each scenario's outcome check.
  It is unit-tested in the gate against committed transcript fixtures, each
  failing fixture seen red.
- **The baseline scorecard** — every scenario × Opus, Sonnet and Haiku on
  the binary as it stands before any Wave 2 change — is published as
  `docs/ux/scorecard-<date>-baseline.md`, with its transcripts kept as Work
  Order artifacts.

## What to read first

- **The plan:** §1 (what the 2026-10-20 runs found: this is what the
  scorer must be able to see), §2 (its two added rules and "a UX run is
  evidence, not a gate"), §4 (every ruling, and R-UX-5, R-UX-7 and R-UX-8
  above all) and §5, which is this wave's authoritative scope. This brief
  binds §5 and does not restate its scenario table; build from the table.
- **ADR-009 A2**, the release builder's allowance and the `never-depends`
  rule that holds `cmd/metais` away from `internal/tools/...`. The runner
  is held the same way (below).
- **ADR-009 D16 item 3**, the symbol rule the runner must get past, and
  `internal/guarantees/layers.json` with its tests.
- **ADR-012 and AGENTS.md's testing rules** — prove each property once, at
  the smallest size that can fail it; pin a property false first.
- **`.claude/skills/verification-gate` and `root-cause`.** Every item uses
  the two-tier gate, explicit squash messages, and commits under the GitHub
  account's noreply identity.
- **[PLAN-2026-10-metais-test-economy](../plans/PLAN-2026-10-metais-test-economy.md)**
  §3 and §7, for the seams below.

## What has changed since the plan was written

- **Both items the plan calls "in flight" have merged.** The generated
  vectors (`01a15ad6`) landed in #181 and `metais skill` (`01a15ad7`, with
  ADR-008 A2 and ADR-009 A5 accepted in #179) landed in #185. So the
  binary this wave's baseline measures already has `metais skill`. The
  baseline runs **without the skill installed** — Wave 3 runs each
  consuming scenario with and without it, against this baseline. Record in
  the scorecard that the skill existed but was not installed.
- **Plan §6's first out-of-band act — accepting ADR-008 A2 and ADR-009 A5
  — is discharged** (#179). Nothing of this wave files it.
- **The test-economy train runs beside this one.** Its §3 table has no
  status column, so read where it is from `git log`, its plan's wave
  verdicts, and the open pull requests (`gh pr list -R kottera/metais`).
  Its W1-3 shares and reworks the guarantee package's builds, listings and
  gate-script tests, which is where the runner's guarantee lands. Under
  plan §2, an item of this wave that edits a file an in-flight
  test-economy item edits takes a `depends_on` edge to it and merges after
  it; if that item is not yours to edge (another train's), sequence by
  rebasing onto it once it merges, and say so in the breakdown.

## Rulings — do not reopen

From the plan, ratified 2026-10-21:

- R-UX-5: the pass rules — first `metais` command of a consuming scenario
  is `inspect`; no zip extraction; no unknown-command error; the package
  untouched by a consuming scenario; the scenario's own outcome check. For
  a real document (C1), the check is asking rather than inventing.
- R-UX-6: the runner takes the harness as a parameter; Claude Code is the
  only one implemented and gated on in this train.
- R-UX-7: the runs happen from dispatches, through the loopback proxy, with
  the placeholder `ANTHROPIC_AUTH_TOKEN`, per-session directory and
  `$TMPDIR`, pacing and resume, and `metais` built with `-buildvcs=false`.
- R-UX-8: in a consuming scenario, extracting the package (`unzip`,
  `unpack`, `extract`), changing it or repacking it fails the run.
- "A UX run is evidence, not a gate": nested runs never execute in
  `gate.sh`. The scorer's unit tests over committed fixtures do.

Rulings made by the plan orchestrator for this wave:

- **The runner's host access is an ADR-009 amendment in A2's shape.** It
  needs what D16 item 3 refuses elsewhere: `os/exec` (to build `metais` and
  start `claude`), the environment, file writes, and — unlike the release
  builder — the clock, because §5 records each run's time and the runner
  paces. That is within the plan's ratified design ("held so by
  `internal/guarantees`, as ADR-009 A2 holds the release builder"), so the
  amendment is accepted under the plan's ratification and its status line
  says so. What must be true, and what the amendment states:
  - `cmd/metais` can never reach the runner, proved by a guarantee test
    (placing it under `internal/tools/` puts it under A2's existing
    `never-depends` rule; elsewhere, the amendment adds the rule);
  - nothing the runner reads reaches a template, an output or a finding of
    `metais`, because the runner never runs inside it;
  - the allowance is the narrowest list of symbols the runner uses, and
    the scorer — which runs in the gate — gets none of it unless it needs
    it.
  The amendment number is allotted at breakdown, after checking the
  test-economy train's in-flight ADR-009 drafts (ADR-009 already holds
  A7). If the runner needs anything the plan does not name — a network
  call of its own, say — park and ask me.
- **The probe's transcripts.** Plan §5 names Work Order `01a15b3b`'s
  artifacts among the scorer's fixtures, and asks for 2026-10-20's runs
  "re-recorded". This train was handed no artifact handles. Re-record the
  failing shapes with the runner — the unzip, `metais cat`, the copied
  shipped example, and the probe's unpack-rewrite-repack of R-UX-8 — and
  commit those as fixtures, each seen red. A recorded fixture must show
  the behaviour it is named for; where a model will not reproduce one on
  demand, a fixture edited to show it is acceptable only if its file says
  so and names the real run it was derived from.
- **Fixtures are scrubbed.** A committed transcript carries no credential
  and no path outside the run's own directory that would identify the
  host beyond what the scorecard already publishes. The placeholder token
  is not a secret, but the scrub removes it anyway, so a later reader never
  has to decide.
- **The baseline's binary** is `main` before any Wave 2 change. This wave
  changes nothing `cmd/metais` reaches, so every commit of this wave builds
  the same binary; the scorecard names the commit it built and confirms
  that. It is published per plan §2: the commit, the model IDs, the
  harness and its version, the host, and where the transcripts are.
- **The baseline is a measurement, not a target.** It is expected to fail
  many scenarios — that is what Wave 2 is for. No item of this wave
  changes `metais`, its help, its specification or its scaffolds to make a
  scenario pass, and no scenario's prompt or check is reworded to make one
  pass. A check is changed only because it mis-scores a transcript, shown
  by a fixture.

## Scope — what belongs to a later wave

- **Wave 2** owns every change to the binary, its help, `schema`'s
  listing, `init`, the specification and the help goldens. Findings the
  baseline makes go into its scorecard, as candidates for Wave 2 — not into
  fix items here.
- **Wave 3** owns the post-Wave-2 scorecard, the with-skill runs, the
  comparison, any `.agents/skills` harness run (R-UX-6), and the SC1 and
  SC9 rounds.
- The method document describes the with-skill mode as a runner option, so
  Wave 3 can use it, but the baseline does not run it.

## What must not break, and how it is checked

- Both gate tiers stay green. The runner's and the scorer's packages are
  in the scoped tier's reach like any other; the nested runs are not.
- `internal/guarantees` proves the runner unreachable from `cmd/metais`,
  with the property pinned false first (a fixture package that does reach
  it is refused).
- The runner's isolation is proved, not assumed: the checks of §5's
  isolation list (fresh directory outside any repository, `metais` from
  the commit under test first on PATH, auto-memory off both ways, project
  settings only, `--strict-mcp-config`, `--no-session-persistence`, and
  each transcript checked for a memory-file read) are made by the runner,
  and a run that fails one is recorded as invalid, not scored. Confirm the
  `claude` flags against the version the dispatch actually runs and record
  that version.
- The scorer's every rule has a fixture that passes it and one that fails
  it.
- `docs/INDEX.md` names the new documents.

## Out-of-band acts of this wave

From plan §6:

| Act | Off-tera type | When | Before | Notes |
|---|---|---|---|---|
| Run a scorecard on Ed's host | `manual-qa-pass` (inputs `build`: the commit; `focus`: "run the UX runner's baseline scorecard on this host and publish it as `docs/ux/scorecard-<date>-baseline.md`, with the transcripts kept") | only if dispatches lose the loopback proxy that R-UX-7 relies on | this wave's verification | The runner and the scenarios are this wave's. Ed runs one command and publishes the scorecard. File it only if the baseline item finds the proxy gone; then the verification waits on it. |

The other two rows of §6 are not this wave's: the skill's acceptance is
discharged, and T-A3's edge comes with Wave 2.

## Closeout

*Not yet written. The wave orchestrator replaces this line with the wave's
closeout.*
