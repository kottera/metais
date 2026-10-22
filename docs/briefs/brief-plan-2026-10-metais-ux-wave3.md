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
| Run a scorecard on Ed's host | `manual-qa-pass` (inputs `build`: the commit; `focus`: which scorecard to run and publish) | only if dispatches lose the loopback proxy that R-UX-7 relies on | this wave's verification | Plan §6. Wave 1 found that the runner refuses to start without `ANTHROPIC_BASE_URL`, so this route needs a credential path the runner does not build. If the proxy is lost, park and ask me before filing it. |

Plan §6's other two rows are discharged:
- the skill's ADRs, accepted in #179;
- T-A3's edge, added by Ed on 2026-10-22.

The SC1 and SC9 rounds are PLAN-2026-10-metais-v1's acts, as ruled
above. Filing them again is not this wave's act.

## Closeout

*Not yet written. The wave orchestrator replaces this line with the wave's
closeout.*
