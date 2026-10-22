# Brief — PLAN-2026-10-metais-ux, Wave 3: measure, fix, and close

**Plan:** [PLAN-2026-10-metais-ux](../plans/PLAN-2026-10-metais-ux.md), §8
**Feature:** `metais-ux`. Work Orders are titled `metais-ux [W3-<item>]: <action>`;
pull requests and their squash subjects are `<type>(metais-ux): [W3-<item>] <change>`
(AGENTS.md, Git and pull requests).
**Repository:** `kottera/metais`, branch `main`. Every Work Order of this wave
gets `repo: kottera/metais` and `branch: main`.
**Gated on:** Wave 2, complete. Its verdict is under plan §7 (#215), and its
closeout is in [brief-plan-2026-10-metais-ux-wave2.md](brief-plan-2026-10-metais-ux-wave2.md).
The gate has cleared.

## Harness policy

Claude throughout, by plan §2. File every item by these types and no other.
If a seat is not available, wait for it rather than substituting another
harness's type.

| Role | Work Order type |
|---|---|
| Wave derivation and supervision | `wave-orchestrator-derive` |
| Implementation and corrective work | `code-implementation-direct` |
| Whole-wave verification | `wave-verification` |

## What this wave is for

This wave produces the evidence that the fixes work, and closes the train.
When it is done:

- **The after scorecard is published.** It covers every scenario × Opus,
  Sonnet and Haiku on post-Wave-2 `main`, with and without the skill
  installed, as `docs/ux/scorecard-<date>-after.md`. Its comparison with
  the baseline is written beside it.
- **R-UX-5's thresholds hold** on the runs that count (below). Every C0
  and C1 run that counts carries a recorded verdict, and the scorecard
  lists each disagreement between a verdict and the phrase check.
- **Every scenario that fell below the thresholds** was root-caused from
  its transcript, fixed in the binary (or the skill), and re-run.
- **The SC1 and SC9 evidence runs exist** on the post-Wave-2 binary,
  ready for Ed to confirm in the rounds that are already filed.
- **The train's documents are closed out:** `docs/INDEX.md`, and each
  amendment's implementation-log entry.

## What to read first

- **The plan:**
  - §1, its "Done means";
  - §2, "A UX run is evidence, not a gate";
  - §4, every ruling, and R-UX-5, R-UX-6, R-UX-8 and R-UX-9 above all;
  - §8, this wave's authoritative scope;
  - the Wave 1 and Wave 2 verdicts, and the rulings recorded beside them.
- **Both earlier closeouts**, and the baseline scorecard
  [scorecard-2026-10-21-baseline.md](../ux/scorecard-2026-10-21-baseline.md),
  whole. Its transcripts are artifact
  `art_boccbn5awud7um657vbfnsbezoefmdmkajnhk4tslubk6l3w27fq`, and its
  "suspected mis-scores" section matters here.
- **`docs/dev/ux-testing.md`**, the method. Read its with-skill mode,
  recorded verdicts, rewording a prompt, and the scorecard's format.
- **`docs/dev/rounds/sc1.md` and `sc9.md`**, the runbooks of the two
  rounds.
- **ADR-008 A2 to A9, ADR-007 A1 and ADR-001 A3**, which define what the
  binary now says.
- **AGENTS.md and the two skills.**

## What the earlier waves learned that changes this one

- **What the comparison measures.** The after binary differs from the
  baseline's by more than Wave 2:
  - PLAN-2026-10-metais-render-seams moved the render pipeline;
  - ADR-001 A3 changed the README in every package, so every packed
    fixture's bytes differ.

  The comparison names both, and attributes a change in a scenario's
  result to Wave 2 only where the transcript shows the session used
  something Wave 2 added.
- **Where the scorer already stands.** It reads `--out` for every
  directory-writing command, and `--to` for the baseline's recordings
  (R-W2-1). It needs no change to score the after binary's flags.
- **Before the scorecard counts toward R-UX-5,** every C0 and C1 run that
  counts carries a recorded verdict, from a judge model or from Ed. The
  scorecard lists every run where the verdict and the phrase check
  disagree (Ed, Wave 1's R-e, through the operator's guidance of
  2026-10-21). The method's "Recorded verdicts" section is where the field
  and the report already live.
  - A judge model's verdict is made outside the gate and recorded in the
    run's record, with the judge's model ID and its rationale.
  - Whether "asking for the facts" passes C0 is exactly the judgement the
    verdict makes. At baseline, every session asked instead of taking a
    reading. The verdict judges against C0's own check in plan §5:
    either reading, said aloud, and no invented facts presented as real.
- **Carried open:**
  - **Candidate 7.** A session with no package in its directory never
    runs `metais`. A1 and A3 on Sonnet and Haiku did this at baseline.
  - **Candidate 10.** `skill --print --harness claude` is refused.
  - **The topics are not well reached.** The repository README and the
    site do not name `help templates` and `help manifest`.
  - **W2-NB1** (`01a1623d-55fc`), three small wording fixes, may still be
    running. One of them — `Next:` against "Read next" — is in the
    skill's text. The with-skill runs should be made on a `main` that
    holds it, or the scorecard must say which commit they ran against.
- **The shared host starves heavy gate packages.**
  - Every Wave 2 item lost gate time to it, and one dispatch hit the
    3-hour tool timeout.
  - Budget the gate per item.
  - A starved timeout is settled by a re-run at the gate's share, never
    by raising the timeout.
  - Clones on this host need `git config core.precomposeunicode false`.
- **Nested runs spend the dispatch's seat quota.** About 54 runs, and then
  re-runs, will need the runner's pacing and resume (R-UX-7). Size the
  items so that no one dispatch has to hold every run.

## Rulings — do not reopen

From the plan:

- **R-UX-5, the thresholds.**
  - Every consuming scenario passes on Opus, Sonnet and Haiku.
  - Every authoring scenario passes on Opus and Sonnet.
  - Haiku's authoring runs are recorded, not gated.
  - The pass rules are as the method states them, with Ed's Wave 1 rule
    changes:
    - `--version` and `help` before `inspect` do not fail inspect-first;
    - `unzip -l` is a note;
    - a write outside the run directory is invalid.
- **R-UX-6.** A `.agents/skills` harness (Codex or OpenCode) is run once,
  only if one can be installed on a host, and its results are recorded,
  not gated.
- **R-UX-8 as amended, and R-UX-9.** A scenario's prompt is reworded only
  to remove ambiguity about what the user wants, never to steer how the
  session gets there. Each rewording is recorded in the next scorecard,
  with the transcript that showed the ambiguity. A session that misreads a
  clear intent is a failure to fix in the tool, not in the prompt. Plan §8
  restates it: a failing scenario is root-caused from its transcript, and
  its prompt is reworded only under R-UX-9.
- **R-UX-2.** Example or fictional data is served on the fly, never copied
  out of a shipped example.
- **Every fix is the binary's** (or the skill's), and it changes the
  specification, the help goldens, the published schemas and the
  vectors in the same pull request, under an amendment where it changes
  what a command does. Amendment numbers are allotted at breakdown, and
  again when a fix item is filed mid-wave. ADR-008 holds A1 to A9,
  ADR-007 holds A1, ADR-001 holds A1 to A3, and ADR-007 A2 was allotted in
  Wave 2 and never used, so it is free.
- **No check is weakened to make a scenario pass.** A check changes only
  because a fixture shows it mis-scores a transcript. The baseline's three
  suspected mis-scores — `names-every-input`, `no-zip-extraction` on an
  `unzip` the sandbox stopped, and C5 sonnet's `script-renders` — are
  settled that way, or by a recorded verdict where the method allows one.
  A check is never settled by changing its threshold. Any check that
  changes is applied to the baseline's transcripts too, and the
  comparison is re-scored, so the two scorecards are read by the same
  scorer.

Rulings made by the plan orchestrator for this wave:

- **Which runs count toward the thresholds.**
  - **Consuming scenarios** count **without the skill installed**. Plan
    §1 says a session with only the binary learns everything from the
    binary, and SC1 says the same.
  - **The with-skill runs** are recorded and compared, and their own
    passes and failures are reported. A consuming scenario that passes
    without the skill and fails with it is a defect in the skill. It is
    fixed, not recorded.
- **Authoring scenarios run both ways too.** The plan names with-skill
  runs only for consuming scenarios, but candidate 7 can only be measured
  with the skill. A1 to A3 run with and without the skill on every model.
  Which mode counts toward the authoring threshold is **not** decided
  here:
  - if the authoring scenarios pass on Opus and Sonnet without the skill,
    the question does not arise;
  - if they fail without it only because the session never ran `metais`
    (candidate 7), and pass with it, park and bring me the runs. That is
    a decision about what the plan's "Done means" requires, and I will
    take it to Ed with the evidence;
  - any other authoring failure is root-caused and fixed like any other.
- **The SC1 and SC9 rounds are not this wave's to close.** They are Work
  Orders `01a15abf-4d58` (SC1) and `01a15abf-6347` (SC9) of
  PLAN-2026-10-metais-v1, already filed as out-of-band acts that Ed takes,
  and T-A3 depends on both. This wave cannot reach them, and must not file
  duplicates of them.
  - **What this wave does instead:** it produces each round's evidence on
    the post-Wave-2 binary, with the harness:
    - for SC1, a C2-shaped run against
      `examples/accessibility-conformance-report`;
    - for SC9, an A1-shaped run whose session works only from `spec/`,
      with no tool source. That may need a scenario whose fixtures hold
      `spec/`, added by the method's "Adding a scenario".
  - **Where the evidence goes:** each round's runbook under
    `docs/dev/rounds/` gains a section naming the commit, the model, the
    transcript's artifact handle and what the transcript shows, so Ed can
    confirm from the runbook when he takes the round.
  - **Where it does not go:** the wave never marks either round done.
  - **If either evidence run fails its round's own bar,** that is a
    finding, fixed like any other.
- **R-UX-6's harness.** Try it only with a harness and a credential that
  are already available to the dispatch, installed into the dispatch's
  scratch. If none is, record "not run, and why" in the scorecard. Nothing
  waits on it, and the runner gains a second harness only if the run
  happens.
- **The close's documents:**
  - `docs/INDEX.md` names the after scorecard;
  - every amendment this train made has its implementation-log entry file
    under `docs/adr/logs/ADR-<NNN>/`. Wave 2's items wrote theirs;
  - plan §8 asks for an entry in ADR-008 and ADR-007, which means checking
    that each amendment's file exists and adding any that is missing.

  The plan's own status and the train closeout are mine. No item edits the
  plan.

## Scope — what belongs elsewhere

- The website and metais.org going live, an MCP server, and gating on
  harnesses other than Claude Code (plan §9).
- The repository README naming the topics is in scope only as a fix for
  a scenario that fails for want of it, shown by a transcript. The site's
  content is not in scope.
- Changes to the specification's normative behaviour, beyond what a
  fix's amendment needs.

## What must not break, and how it is checked

- **Both gate tiers stay green.** The wave tier runs at verification on
  the merged tree, with `internal/conformance` passing on its own merits.
- **The scorer's fixtures stay green**, and every changed check has a
  fixture red before its change.
- **Published scorecards.**
  - Each is published per plan §2: the commit, the model IDs, the harness
    and its version, the host, and the transcripts' artifact handle.
  - Transcripts are kept as artifacts, scrubbed, not committed.
- **The isolation checks hold.** An invalid run is re-run, not scored.
- **The scorecard is a measurement, not a gate.**
  - The wave's acceptance is that R-UX-5's thresholds hold on the
    scorecard after the last re-run.
  - The verification confirms that the published scorecard is what its
    transcripts score to with `main`'s scorer. It need not re-run the
    sessions.

## Out-of-band acts of this wave

| Act | Off-tera type | When | Before | Notes |
|---|---|---|---|---|
| Run a scorecard by hand outside the dispatch sandbox | `manual-qa-pass` (inputs `build`: the commit; `focus`: which scorecard to run and publish) | only if dispatches lose the loopback proxy that R-UX-7 relies on | this wave's verification | Plan §6. Wave 1 found that the runner refuses to start without `ANTHROPIC_BASE_URL`, so this route needs a credential path the runner does not build. If the proxy is lost, park and ask me before filing it. |

Plan §6's other two rows are discharged:
- the skill's ADRs, accepted in #179;
- T-A3's edge, added by Ed on 2026-10-22.

The SC1 and SC9 rounds are PLAN-2026-10-metais-v1's acts, as ruled
above. Filing them again is not this wave's act.

## Closeout

*Written 2026-10-22 by the wave orchestrator (Work Order `01a16240`), from
`main` at `eb03c75`. It records what landed and what carries forward. It is
not a verdict on whether the work is correct. The verification below is the
verifier's, attributed to it. Where I say I checked something, I name how: I
read the children's reports, the merged commits' stats, and the parts of
the diffs and documents named below. I mutated nothing and ran no session.*

**The breakdown.** Six items were derived from this brief and approved by
the plan orchestrator on 2026-10-22, with its readings (a) to (c) below.
Two fix rounds and one blocking fix followed, each filed from what the
measurement found, so the wave ran fourteen items and one verification.

- **Measurement, in parallel:**
  - W3-1, the baseline's suspected mis-scores;
  - W3-2 and W3-3, the after sets without and with the skill, split so
    that no dispatch held every run;
  - W3-5, the SC1 and SC9 evidence.
- **W3-4** waited on W3-1 to W3-3, so that one judge and one scorer read
  every set.
- **W3-6** was a held draft until W3-4 named the failures.
- **W3-7** was filed mid-wave from W3-5's scorer finding.
- **The first fix round** came from W3-4's root causes: W3-8, W3-9, W3-10,
  and W3-11, which the plan orchestrator added. W3-6 then re-ran.
- **The second round** came from W3-6's failures: W3-12, then W3-13's
  re-run.
- **W3-14** is the verification's one blocking fix.

| Item | Work Order | PR | Merge | Amendment |
|---|---|---|---|---|
| W3-1 settle the baseline's suspected mis-scores | `01a16245-9d0b` | #219 | `dae1ef7` | — |
| W3-2 the after set without the skill | `01a16245-ed4e` | #220 | `f65af42` | — |
| W3-3 the after set with the skill | `01a16246-2e89` | #221 | `6d34291` | — |
| W3-4 judge, re-score, compare, root-cause | `01a16247-07bf` | #223 | `8be6331` | — |
| W3-5 the SC1 and SC9 evidence | `01a16246-8c30` | #218 | `332fd2a` | none (A9 allotted, not needed) |
| W3-6 the first re-run | `01a16247-4192` | #229 | `8445628` | — |
| W3-7 a session's own render is not the package | `01a16269-a9a8` | #222 | `8c45e2b` | — |
| W3-8 generated data only when asked; a preview is a render | `01a16286-c70a` | #225 | `ef9b25a` | ADR-008 A10 |
| W3-9 the README: read only with `metais` | `01a16287-0d86` | #226 | `37c0dbe` | ADR-001 A4 |
| W3-10 five check mis-scores | `01a16287-6bc8` | #228 | `bda1501` | — |
| W3-11 §B.6 agrees with MTS-AUT-003 | `01a16287-a239` | #227 | `9653c92` | — |
| W3-12 an example's data is never a real document's | `01a162f6-d7fe` | #230 | `d7ed0f5` | ADR-008 A11 |
| W3-13 the second re-run | `01a162f7-296e` | #231 | `be1c74b` | — |
| W3-14 the blocking fix: name each field as listed | `01a16340-f73c` | #232 | `eb03c75` | none (A12 allotted, not needed) |
| verification | `01a1632a-d8a5` | — | — | — |
| W3-NB1 the scorer's and runner's non-blocking findings (not waited on) | `01a16351-ed1b` | — | — | — |
| W3-NB2 two `internal/cli` non-blocking findings (not waited on) | `01a16352-2b89` | — | — | ADR-008 A12, if needed |

Two commits on `main` since the wave's base `6331562` are not this wave's
items: W2-NB1 (#217), which merged during the wave, and #224, the plan
orchestrator's record of R-W3-1 to R-W3-4.

### 1. What landed, against what was asked

**R-UX-5 after the last re-run.** These are the after scorecard's figures
([scorecard-2026-10-22-after.md](../ux/scorecard-2026-10-22-after.md),
"After the last re-run"), as W3-13 and W3-14 published them and the
verifier re-scored them:

| Threshold | Pass |
|---|---|
| Every consuming scenario on Opus, Sonnet and Haiku, without the skill (counted, R-W3-1) | **18 of 18** |
| Every authoring scenario on Opus and Sonnet, with the skill counting | **6 of 6** |
| Every authoring scenario on Opus and Sonnet, without the skill counting | **4 of 6**: A1 and A3 sonnet fail, candidate 7 |
| Consuming with the skill (recorded, not counted) | 18 of 18, with C4 from W3-14's re-run |

- **The thresholds hold if authoring counts with the skill, 24 of 24, and
  not if it counts without it, 22 of 24.** Which mode counts is Ed's
  (R-W3-2). Plan §8 records no ruling of his as of `eb03c75`, so the
  scorecard states both, as ruled.
- **The baseline** reads 9 of 24 with `main`'s scorer. It was 6 of 24 as
  published, and W3-1, W3-4's verdicts and W3-10 moved it. Its original
  table stays as published, and each change is in a section beside it.

Against the brief's "When it is done":

- **The after scorecard is published:**
  - [scorecard-2026-10-22-after.md](../ux/scorecard-2026-10-22-after.md),
    with the comparison with the baseline and with the skill beside it;
  - [scorecard-2026-10-22-after-with-skill.md](../ux/scorecard-2026-10-22-after-with-skill.md).

  Both carry plan §2's publication facts and every set's artifact handle.
  Every after set ran from `6331562`, which does not hold W2-NB1, and the
  scorecards say so. The re-runs ran from `37c0dbe` (W3-6), `d7ed0f5`
  (W3-13) and `dd9c3db6` (W3-14).
- **Every C0 and C1 run that counts carries a recorded verdict.** One judge
  (`claude-opus-5-5`) with one prompt, published in the after scorecard,
  judged every C0 and C1 run of every set, the baseline's included. The
  verdict disagrees with the phrase check on every C0 run: the judge passes
  a session that names the render reading, asks for the facts and invents
  nothing. The scorecard lists each disagreement.
- **Every scenario below the thresholds was root-caused, fixed and re-run:**
  - C1 on every model: W3-8, then W3-12 for Haiku;
  - C3 on every model: W3-8 and W3-9;
  - C2 haiku and C3 haiku with the skill: W3-8;
  - C4 haiku and C5 sonnet with the skill: W3-12, then W3-14 for C4;
  - five check mis-scores: W3-10.

  No prompt was reworded under R-UX-9.
- **The SC1 and SC9 evidence runs exist.** W3-5 added two scenarios: C6
  `render-acr-fictional`, which is C2's prompt against
  `examples/accessibility-conformance-report`, and A4 `spec-only-package`,
  which is SC9's task with only `spec/` and the binary. It ran each on
  Opus and Sonnet from `29f077f`, whose binary closure is `6331562`'s.
  Each runbook (`docs/dev/rounds/sc1.md`, `sc9.md`) gained an evidence
  section. Neither round is marked done.
  - **SC9:** both runs pass `check`, `test` and `pack` with no finding.
  - **SC1:** both runs pass `validate` and `render`, with `acr.json`
    byte-identical. Both sessions read the package's raw bytes once before
    `inspect`, so neither meets the bar's last item as it stands.
- **The close's documents:**
  - `docs/INDEX.md` names both after scorecards (W3-4, W3-14).
  - Every amendment this train made has its log file. W3-4 found every
    Wave 2 file present. This wave's are
    `docs/adr/logs/ADR-008/W3-8.md`, `W3-12.md` and
    `docs/adr/logs/ADR-001/W3-9.md`.

What I checked on `main` myself, by reading:
- **ADR-001 A4's README sentence** in `spec/02-container.md:337`.
- **ADR-008 A10's `generatedNever`** and the `render --sample` next step
  "Only for a person who asked for an example or a preview" in
  `internal/cli/inspect.go`.
- **ADR-008 A11's `exampleNotFacts`** in the same file.
- **The ADR headings:** ADR-008 A10 and A11, and ADR-001 A4.
- **W3-14's step 3** in `internal/skill/SKILL.md`.
- **The note in `sc9.md`** that the evidence run saw the old §B.6.

### 2. What children flagged, degraded, declined or discovered

- **Mid-wave scorer corrections.** W3-1 and W3-10 changed
  `names-every-input` twice. Each change came with a fixture that was red
  first: a word ending, and a title's number. W3-1 also changed
  `script-renders`, and W3-10 changed the shell reading in three ways:
  `$PWD`, `for` loops, and a `cd` that carries only on exit 0. W3-1 ruled
  `no-zip-extraction` correct as it reads: a sandbox-stopped `unzip` is
  still the session's choice.
- **W3-10 corrected W3-4's reading** of the A3 sonnet mis-score. The real
  mechanism is the harness's `eval … && pwd -P` exiting 1, not the
  here-document. The scorecard section was rewritten.
- **W3-8 removed a test check.** `inspect`'s first-example next step went,
  and `TestInspectExamplesInNameOrder` lost its check of that step. It
  still checks the order of `result.examples`.
- **W3-9 regenerated every packed vector.** Offsets moved by +120.
- **The host.** W3-8 and W3-9 both had starved gate timeouts at 0.22 to
  0.60 of the gate's share, and each passed on `gate.sh --resume`. No
  timeout was raised.
- **W3-11 left two places unqualified on purpose:** the published
  `metais.check.1.json` description, and the site's getting-started page,
  which is out of scope. W3-12 then qualified `README.md` and the skill's
  check step.
- **The with-skill A3 haiku run** (recorded, not gated) still fails,
  because none of its rendered pages survives.
- **R-UX-6 was not run.** No Codex or OpenCode is available to the
  dispatch, and there is no credential for one: the only model route is
  the Anthropic loopback proxy (W3-4).
- **`no-memory-read` marked one W3-14 run invalid** that should not have
  been: the session read back its own spilled output through a glob path.
  It was re-run. W3-NB1 member 5 carries the fix.

### 3. What carries to the next wave and the train's close

- **Ed's ruling on the authoring mode (R-W3-2)** decides whether R-UX-5
  holds: 24 of 24 with the skill counting, or 22 of 24 without it. The
  scorecard states both.
  - Without the skill, A1 and A3 sonnet never run `metais` (candidate 7).
    No binary change reaches a session that never runs it. The fix would
    lie where a newcomer first learns that `metais` is the tool for a
    `.metais`, which is outside this train's scope.
- **The judge's two calls (R-W3-3)** are open to Ed's override, and the
  verdicts depend on them:
  - C0 passes a session that asks for the facts;
  - C1 fails a rendered sample.
- **The SC1 round.** The evidence runs meet every item of SC1's bar except
  the last: both sessions read raw bytes before `inspect`. C6's prompt
  lacks SC1's own "Start with `./metais inspect`". The plan orchestrator
  carries this to the train closeout, and whoever closes the train should
  decide it explicitly. The verifier says the same.
- **The SC9 round.** The evidence was made against the old §B.6, which
  W3-11 corrected afterwards (R-W3-4).
- **Plan-table rows.** The rows for C6 and A4 are in #218's description.
  The plan orchestrator adds them to the plan with the train closeout
  (reading (b)).
- **Unpinned guards in the UX scorer.** The verifier found that each
  scorer change of this wave left live guards no test pins. The filed ones
  are W3-NB1's. A later loosening of a check would cross them silently.
- **What I would do differently:**
  - **Hold the measurement's re-scoring until every check change has
    landed.** My recall of W3-4, to add an edge to W3-7, was refused to
    this worker, so W3-4 scored before W3-7 merged. It changed no cell.
  - **Do not rule against a recorded ruling's plain words.** I ruled that
    W3-13's C4 haiku with-skill miss was not a skill defect. The brief's
    words ("It is fixed, not recorded") said otherwise, and the verifier
    blocked on it rightly. W3-14 cost one dispatch, and was cheaper than
    the argument.
  - **Each fix round re-ran every consuming scenario in both modes,** 36
    runs at about $3 a round. That caught no regression this time, but it
    is what made "after the last re-run" true of the whole set.

**Rulings made during this wave.**

- **At breakdown approval (plan orchestrator, 2026-10-22):**
  - (a) the SC1 and SC9 evidence runs are judged by each round's own bar,
    not by R-UX-5, and are not in the after scorecard;
  - (b) W3-5's plan-table rows go through this closeout;
  - (c) the baseline's C0 and C1 runs get the same judge, and the
    re-judged cells are published beside its table.
- **R-W3-1 to R-W3-4 (plan orchestrator),** recorded in the plan (#224):
  - R-W3-1, the runs that count;
  - R-W3-2, the authoring mode, which is with Ed;
  - R-W3-3, the judge's calls;
  - R-W3-4, §B.6 against MTS-AUT-003.
- **Mine:**
  - **W3-12's README member** (W3-12.6) follows R-W3-4's "anything else
    that repeats the error".
  - **The C4 fix lies in the skill, not the check.** The check had been
    corrected twice, and the brief forbids weakening one to make a
    scenario pass.
  - **No third fix round for C4 haiku. Withdrawn.** The verifier quoted
    the brief against it, and W3-14 fixed it.
- **One escalation.** I parked the candidate-7 runs for the plan
  orchestrator, for Ed, after W3-4, as instructed. Nothing else was
  escalated, and no child parked to ask a question.

### 4. What the verification found, and where each finding went

Verifier `01a1632a-d8a5` returned **blocking**, on one finding.

- **The ids match.** The 13 ids it returned are the 13 I sent.
- **Depth:**
  - `mutated`: W3-1, W3-5, W3-7, W3-8, W3-9, W3-10 and W3-12;
  - `inspected`, as documents and measurements: W3-2, W3-3, W3-4, W3-6,
    W3-11 and W3-13.

  So the prose correction W3-11 had no mutation. The measurement items
  were checked by re-scoring, not by mutation.
- **The wave tier** (`gate.sh wave`) ran on a fresh clone of `be1c74b` and
  was **green**: 54 packages, `go test -race` over every package, and every
  SC7 reach group.
- **The scorecards.** With `main`'s scorer, it re-scored all seven
  published sets from their artifacts. Each came out identical to its
  published scorecard, which is the brief's ask. W3-14's C4 re-run came
  after the verification, which is the round's bound.
- **The audit.** It placed all 15 commits of `6331562..be1c74b`. It states
  that it took the list's completeness on trust.
- **Its 41 mutations:** 23 were red, and 18 stayed green.
  - **W3-1 (7, 2 red):** the green ones were `script-renders`' last-command
    guard, the stem at five letters, the title-begins-with-its-number
    guard, `inRange`'s section guard, and `holdsAll`'s empty guard.
  - **W3-5 (4, all red):** the tree fixture's file arm, its
    outside-the-tree and empty-tree refusals, and the one-kind fixture rule.
  - **W3-7 (8, 2 red):** `ownRender`'s `.metais` and fixture-name
    refusals, singly and together, stayed green. So did the
    render-after-read, added-only and session-script guards.
  - **W3-8 (2, both red):** the skill's "inspect first", and the removed
    example next step re-added.
  - **W3-9 (2, 1 red):** dropping the skill's A4 sentence is caught only by
    the help golden.
  - **W3-10 (14, 8 red):** the green ones were `ends`' five-letter minimum,
    `readBack`'s same-call guard and its refusal check, and three the
    verifier judged equivalent or low-risk.
  - **W3-12 (4, all red):** the Examples line, `ls`'s help, the skill's
    `--fields` route, and its script-test route.

**Where each finding went:**

| Finding | Where it went |
|---|---|
| **Blocking**, cross-item, quoting the brief: "A consuming scenario that passes without the skill and fails with it is a defect in the skill. It is fixed, not recorded." C4 haiku with the skill at W3-13 | **W3-14** (`01a16340-f73c`, #232): the skill names each field as listed; C4 re-run with the skill, 3 of 3 pass. Not re-verified (one round). |
| W3-1: `script-renders`' last-command guard is unpinned | `01a16351-ed1b` (W3-NB1, member 1) |
| W3-7: `ownRender`'s "still counted" refusals are unpinned, and the script branch has no fixture | `01a16351-ed1b` (W3-NB1, member 2) |
| W3-10: three loosened readings' guards are unpinned | `01a16351-ed1b` (W3-NB1, member 3) |
| W3-2's report: the scorecard's Notes print one note twice | `01a16351-ed1b` (W3-NB1, member 4) |
| W3-14's report: `no-memory-read` misses the session's own spilled output read through a glob | `01a16351-ed1b` (W3-NB1, member 5) |
| W3-11: `inspect`'s `metais check` step still says "every defect" with no qualification | `01a16352-2b89` (W3-NB2, member 1) |
| W3-12: `grep`'s and `unpack`'s help lack A11's sentence on examples | `01a16352-2b89` (W3-NB2, member 2) |
| W3-5: the SC1 evidence runs fail the bar's last item | **Not filed:** the plan orchestrator carries it to the train closeout, where the round's set-up is decided |
| The remaining green mutations the verifier recorded as notes, not findings, several of them equivalent (W3-1's stem, title prefix, `inRange`, `holdsAll`; W3-7's three order and change guards; W3-10's three) | **Not filed** as separate findings: the verifier judged them low-risk or equivalent. They stay in the verifier's stored output (`01a1632a-d8a5`) for whoever next changes these functions. W3-NB1 pins only the guards raised as findings. |

**W3-NB1 holds five findings, and W3-NB2 holds two.** Each group is one
package or two the gate selects together:
- W3-NB1's are in `internal/tools/ux` and `internal/tools/ux/score`;
- W3-NB2's are in `internal/cli`, which changes the help goldens together.

Each task says its members are severable. **The wave completed without
waiting for either, by design.** Neither is re-verified, and nothing waits
on them. No finding went to the janitorial backlog: none is prose-only.

### 5. Out-of-band acts

**This wave filed none, and no child needed one.**

- **The scorecard fallback run by hand** (`manual-qa-pass`). The loopback
  proxy held for every run, so it was not filed.
- **The SC1 and SC9 rounds** are PLAN-2026-10-metais-v1's acts
  (`01a15abf-4d58`, `01a15abf-6347`), as the brief rules. This wave filed
  no duplicate, and produced their evidence in the runbooks.
