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
  a real document (C1 render-real, "render this file", no data given), the
  check is asking rather than inventing.
- R-UX-6: the runner takes the harness as a parameter; Claude Code is the
  only one implemented and gated on in this train.
- R-UX-7: the runs happen from dispatches, through the loopback proxy, with
  the placeholder `ANTHROPIC_AUTH_TOKEN`, per-session directory and
  `$TMPDIR`, pacing and resume, and `metais` built with `-buildvcs=false`.
- R-UX-8 (amended 2026-10-21): in a consuming scenario whose prompt asks
  for output ("render…", "produce the report from…"), extracting the
  package (`unzip`, `unpack`, `extract`), changing it or repacking it fails
  the run. It does not apply to a prompt that leaves the user's intent
  open, such as C0's "fill this file".
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
  shipped example, and an R-UX-8 failure from a prompt that asks for
  output — and commit those as fixtures, each seen red. The probe's
  unpack-rewrite-repack, made under "fill this file … with new fictional
  data", is not a failure fixture: it is C0's pass path once the session
  states the reading it took. A recorded fixture must show
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
  pass. A prompt is reworded only under R-UX-9: to remove ambiguity about
  what the user wants, recorded in the next scorecard with the transcript
  that showed it. A check is changed only because it mis-scores a
  transcript, shown by a fixture.

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
| Run a scorecard by hand outside the dispatch sandbox | `manual-qa-pass` (inputs `build`: the commit; `focus`: "run the UX runner's baseline scorecard on this host and publish it as `docs/ux/scorecard-<date>-baseline.md`, with the transcripts kept") | only if dispatches lose the loopback proxy that R-UX-7 relies on | this wave's verification | The runner and the scenarios are this wave's. Ed runs one command and publishes the scorecard. File it only if the baseline item finds the proxy gone; then the verification waits on it. |

The other two rows of §6 are not this wave's: the skill's acceptance is
discharged, and T-A3's edge comes with Wave 2.

## Closeout

*Written 2026-10-22 by the wave orchestrator (Work Order `01a15e89`), from
`main` at `51b2c9e`. This records what landed and what carries forward. It
is not a verdict on whether the work is correct: the verification below is
the verifier's, attributed to it, and nothing here is my own test of the
code.*

**The breakdown.** Three items, run in sequence, each on a real dependency:

- the scorer's fixtures had to be recorded with the runner;
- the baseline needed both the runner and the scorer.

Ed approved the breakdown on 2026-10-21. Ed amended the plan in #192
(`929c974`) after this brief merged: C0 was added, C1 and C2 became
"render" prompts, R-UX-8 was scoped and R-UX-9 added. The items were
derived from the plan as amended, and W1-1 corrected this brief to match.

| Item | Work Order | PR | Merge |
|---|---|---|---|
| W1-1 the runner, its isolation checks, the scenario set, ADR-009 A8 | `01a15e8e-b007` | #194 | `c268fb9` |
| W1-2 the scorer, each scenario's check, the fixtures, the scorecard | `01a15e8f-0e54` | #196 | `5e64a4c` |
| W1-3 the baseline scorecard | `01a15f30-58a0` | #199 | `51b2c9e` |
| verification | `01a16049-f594` | — | — |
| W1-NB1 five non-blocking findings (not waited on) | `01a160f8-151a` | — | — |

Work Order `01a15e8f-5ef3` was the first filing of W1-3. I cancelled it
before it ever ran: Ed's answer to W1-2 held W1-3 until a new isolation
check had merged, and `recall` is not open to this worker. It was re-filed
as `01a15f30` with Ed's rulings in its task.

### 1. What landed, against what was asked

- **W1-1 (#194).**
  - **The runner** is `internal/tools/ux`, a main package run with
    `go run`. It runs scenario × model as nested `claude -p` sessions. Each
    session gets its own directory, `HOME` and `TMPDIR`, and the placeholder
    `ANTHROPIC_AUTH_TOKEN`.
  - It builds metais with `-trimpath -buildvcs=false`, paces its runs, and
    resumes a partial set.
  - It takes `-harness`, with Claude Code the only harness implemented, and
    `-with-skill` for Wave 3.
  - **The isolation checks**: §5's list became 13 checks per run, plus
    W1-2's `writes-inside-run`. A run failing a check is recorded invalid,
    or `not-run` when the check comes before the session.
  - **ADR-009 A8** allows `os/exec`, the environment, the file functions,
    `time.Now` and a main's process symbols. It covers `internal/tools/ux`
    alone, not the packages beneath it. The existing `never-depends` rule
    holds `cmd/metais` away from it.
  - **The scenarios**: nine directories under
    `internal/tools/ux/scenarios/`. Each declares `asks_for_output`. A2's
    invoice source lives with its scenario.
  - **Docs**: `docs/dev/ux-testing.md`, named in `docs/INDEX.md`. The flags
    were confirmed against `claude` 2.1.299.
  - This matches the ask.
- **W1-2 (#196).**
  - **Where it lives**: the scorer is `internal/tools/ux/score`. The
    transcript reader is `streamjson`, used by both the runner and the
    scorer, and the scrub is `scrub`. None of them imports `os` or `time`.
  - **The fixtures**: 23 committed runs. 14 are real; the other 9,
    `edited-*`, each name the real run they came from.
  - **The scorecard**: `-scorecard` prints it.
  - It applied Ed's rulings, listed below, and added the `verdict` field
    for C0 and C1. This matches the ask, as amended by those rulings.
- **W1-3 (#199).**
  - **The scorecard** is `docs/ux/scorecard-2026-10-21-baseline.md`, built
    from `5e64a4c`. No file in `cmd/metais`'s closure differs from
    `6827a6b`.
  - **The runs**: all 27 were made (C0–C5 and A1–A3, on
    `claude-opus-5-5`, `claude-sonnet-5-5` and `claude-haiku-5-5`), without
    the skill installed. All 27 are valid.
  - **The result**: **6 of 24 gated runs pass**, 2 of 9 on each model, so
    R-UX-5's thresholds do not hold, as expected of a baseline.
  - **The transcripts** are artifact
    `art_boccbn5awud7um657vbfnsbezoefmdmkajnhk4tslubk6l3w27fq`.
  - **A check correction**: it corrected `no-memory-read`, which had
    recorded C4 opus invalid for reading back Claude Code's own spilled tool
    output. The fixture `TestSpilledToolOutput` was seen red against the old
    check. This matches the ask.

**The gate.** #199 merged over a scoped gate whose only reds were two
starved timeouts, by ruling R-g below. The wave tier on the merged tree is
**red on one step only**: `internal/conformance` timed out at 0.52, and
then 0.39, of 4 cores. It passed alone in 807 s under a raised timeout.
Every other step and package of `gate.sh wave` passed, including
`internal/guarantees` and every `internal/tools/ux` package.

So the brief's "Both gate tiers stay green" is **not met literally**. The
one red is a host that cannot give conformance its share inside go test's
ten minutes. No change of this wave reaches conformance's code: it builds
`cmd/metais`, whose closure this wave left untouched. I filed no child for
it, for these reasons:

- The defect is the package's runtime against the bound on a shared host.
- That runtime is PLAN-2026-10-metais-test-economy's Wave 2 (§8, the
  conformance runner), another train's work.
- The verifier's run is the evidence.

The plan orchestrator should decide whether it wants a full-share
conformance run of `51b2c9e` on another host.

### 2. What children flagged, declined or discovered

- **Fallback outside the dispatch sandbox.** The runner refuses to start
  without `ANTHROPIC_BASE_URL` (W1-1). So plan §6's fallback, a scorecard
  run by hand outside the dispatch sandbox, needs a credential route the
  runner does not build. `docs/dev/ux-testing.md` says so.
- **C0 and C1 checks.** These are scored by phrase checks whose limits the
  scorecard prints (Ed's ruling). The C0 and C1 rows carry no recorded
  verdicts.
- **Suspected mis-scores.** W1-3 found four. One, the scorer's reading of a
  render ignoring `--out -`, is mechanical and is carried by W1-NB1. The
  other three carry to Wave 3:
  - **`names-every-input`.** It fails C4 opus and C4 sonnet, whose answers
    name every required field in the template's own terms. Sonnet's run
    would otherwise pass. Whether a field named in another wording counts is
    a judgement for Wave 3's recorded verdicts.
  - **`no-zip-extraction`.** It counts an `unzip` the sandbox stopped
    before it ran. W1-3 reads this as the rule's limit, not a mis-score: the
    rule measures what the session chose.
  - **`script-renders`, C5 sonnet.** The script did render. A later `diff`
    in the same tool call failed on the sandbox, and the harness then
    reported the whole call as an error. The run fails three other rules
    regardless.
- **A sibling train moved the binary.** PLAN-2026-10-metais-render-seams
  merged changes to the render pipeline in `cmd/metais`'s closure (W1-1,
  W2-1 and W2-2: `598443a`, `84c2f83`, `c202db5`). They landed after the
  baseline's build at `5e64a4c`. The baseline still measures this wave's
  starting binary. But **Wave 3's "after" binary will differ from it by
  more than Wave 2's fixes.** The verifier re-scored the baseline
  transcripts with `main`'s scorer and got the same tables.

### 3. What carries to the next wave

- **The candidates for Wave 2** are the scorecard's "What the failures
  show" section, ten items. The largest:
  - nothing a session sees first sends it to `inspect`, so it reaches for
    `unzip` (nearly every failing run);
  - `inspect`'s next steps offer no preview (`render --sample`) and no
    `sample` (C3 failed on every model);
  - `--out` against `--to` for `unpack` and `extract`;
  - no help topic on templates, filters or escaping (A3);
  - nothing marks an example's data as fictional (C1 haiku).
- **Not reachable from the binary.** A session with no package in its
  directory never runs `metais` (A1 and A3, sonnet and haiku). The binary
  cannot fix that. It is the skill's, Wave 3's, or the first place a
  newcomer is told about the tool.
- **C0 failed on every model**, because every session asked for the facts
  instead of taking a reading. Whether asking should pass C0 is for the
  recorded verdicts Wave 3 needs. The prompt was not reworded (R-UX-9), and
  no R-UX-9 rewording was made in this wave.
- **Before Wave 3's scorecard counts**: a recorded verdict on every C0 and
  C1 run, by a judge model or Ed, with a disagreement report (Ed's ruling).
- **The host.** The shared host starves the gate's heavy packages often
  enough that two of this wave's gate runs went red on starvation. Plan the
  next wave's gate time with that in mind.
- **What I would do differently.** I would capture the scorer's
  run-time-verdict question in the brief, so that W1-2 did not have to park
  on it. I would also have checked before filing that `recall` is open to
  this worker: holding W1-3 cost a cancel and a re-file.

**Rulings made during this wave.** R-a to R-d and R-g are the
orchestrator's; R-e is Ed's; R-f is W1-2's reconciliation.

- **R-a.** Each scenario declares whether its prompt asks for output.
  R-UX-8 applies to C1, C2, C3 and C5, not to C4 or C0. This follows from
  the amended R-UX-8's own wording.
- **R-b.** In C0, raw zip extraction fails R-UX-5. `metais unpack` on C0's
  edit path does not.
- **R-c.** A2's invoice source lives with its scenario, because no example
  holds an invoice.
- **R-d.** The runner's amendment is ADR-009 A8.
- **R-e (Ed, 2026-10-21, answering W1-2):**
  - C0 and C1 use a narrower mechanical check whose limits are stated. A
    recorded verdict and a disagreement report are required before Wave 3's
    scorecard counts.
  - `--version` and `help` before `inspect` do not break inspect-first; a
    read of the package's contents does.
  - `unzip -l` is a note, not a failure.
  - A run that writes outside its own run directory is invalid. This check
    was required before W1-3 and landed in #196.
  - Usernames in `ls -l` output are scrubbed.
- **R-f.** `metais unpack` and `extract` count as extraction in consuming
  scenarios other than C0. Authoring scenarios unpack to edit.
- **R-g.** #199 may merge over a scoped gate red only on starved timeouts,
  because each red package passed alone with the change. This covers #199
  only.

### 4. What the verification found, and where each finding went

Verifier `01a16049` returned **pass, with nothing blocking**.

- **Every item it was sent came back, each at depth `mutated`**: the ids it
  returned are the three I sent.
- **The audit.** It checked that each merge commit is an ancestor of `main`
  and matches its item. It compared each commit's diff with
  `go list -deps ./cmd/metais`: no non-test file changed.
- **The transcripts.** It fetched the transcript artifact: 27 valid runs,
  holding no token and no host path. Re-scored with `main`'s scorer, they
  reproduce the published tables and the 6 of 24.
- **Its 30 mutations**, all reverted:
  - **W1-1**: each isolation check disabled in turn (strict MCP,
    credential, metais first on PATH, project settings only, session as
    started, outside a repository), and `TMPDIR` and `CLAUDE_CODE_TMPDIR`
    set to `/tmp`. Every one went red.
  - **The guarantee on the real tree**: a `cmd/metais` file importing
    `internal/tools/ux/scenario` was refused, red.
  - **W1-1's claimed reds**, spot-checked: the `never-depends` rule
    narrowed, and the allow-list given `beneath: true`. Both red.
  - **W1-2**: R-UX-8 applied to every consuming scenario, the C0 `unpack`
    exception removed, unpack counted in authoring scenarios, and the edit
    reading passed without a stated reading. All red. The scrub's token
    regex dropped went red in the scrub's own test.
  - **The scorer under A8**: a scorer file calling `time.Now` was refused
    by `TestNoHostFacts`, red.
  - **W1-3**: three ways of widening the spilled-output regex, and the
    producer's claimed red. All red.

Its findings, each with where it went:

| Finding | Where it went |
|---|---|
| W1-1: `-buildvcs=false` and `-trimpath` in the runner's build are pinned by no test | `01a160f8-151a` (W1-NB1, member 1) |
| W1-2: inspect-first's half about metais commands before `inspect`, and its never-inspected branch, have no fixture | `01a160f8-151a` (W1-NB1, member 2) |
| W1-2: C0's "invents no facts presented as real" clause and its render reading are unpinned | `01a160f8-151a` (W1-NB1, member 3) |
| W1-2: the placeholder token is declared twice, with nothing tying the two together | `01a160f8-151a` (W1-NB1, member 4) |
| W1-3's own suspected mis-score: the render reading ignores `--out -` | `01a160f8-151a` (W1-NB1, member 5) |
| Cross-item: the wave tier is red only on `internal/conformance`'s starved timeout | **Not filed**: another train's scope; reasons in §1 |
| Cross-item: the runner's and the scorer's interfaces held across the three items | **Not filed**: a confirmation, not a defect |
| W1-3: W1-3.4 verified by fetching the artifact | **Not filed**: a confirmation, not a defect |

**One child carries all five members.** Each repair is a test or fixture
pin under `internal/tools/ux/...`, the scoped gate selects those packages
together, and no member's decision changes another's. Its task says the
members are severable. **The wave completed without waiting for W1-NB1, by
design**: it is not re-verified in this wave, and nothing waits on it. No
finding went to the janitorial backlog, because none was prose-only.

### 5. Out-of-band acts

The brief's one act was a `manual-qa-pass` run by hand outside the dispatch sandbox, to be filed only
if dispatches lost the loopback proxy. **It was not filed**: every nested
session of this wave, including all 27 baseline runs, ran through the
dispatch's proxy. No child needed an act. No act is open from this wave.
