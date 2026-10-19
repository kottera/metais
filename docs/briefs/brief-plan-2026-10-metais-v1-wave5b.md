# Brief — PLAN-2026-10-metais-v1, Wave 5b: the gate's cost and merge friction

**Plan:** [PLAN-2026-10-metais-v1](../plans/PLAN-2026-10-metais-v1.md), §11b
**Feature:** `metais-v1`. Work Orders are titled `metais-v1 [W5b-<item>]: <action>`;
pull requests and their squash subjects are `<type>(metais-v1): [W5b-<item>] <change>`
(AGENTS.md, Git and pull requests).
**Repository:** `kottera/metais`, branch `main`. Every Work Order of this wave
gets `repo: kottera/metais` and `branch: main`.
**Gated on:** Wave 5, complete. Its verdict is under plan §11. Ed inserted
this wave by guidance to the plan orchestrator on 2026-10-19, and Wave 6
waits on it.

## What this wave is for

Wave 5's items spent longer in the gate than in their work, and most of
their rebase conflicts came from file layout rather than from overlapping
code. This wave fixes both before Wave 6 adds the full hostile corpus and
the benchmarks, which would make both worse.

When it is done:

- **The gate has two tiers,** by an ADR-010 amendment to D15 item 3. Each
  Work Order runs a scoped tier, and the full list runs once per wave on
  the merged tree.
- **Scripts** start the gate in the background, wait on it in bounded
  steps, resume only what is unverified, and re-run one test alone.
- **`go test` is capped on a shared host.**
- **The skills match:** the `verification-gate` and `root-cause` skills,
  already on `main`, match the new gate.
- **The `hostfs` open-file-limit tests** are cheap without proving less,
  or the reason they cannot be is recorded. Whether one of them hangs is
  answered.
- **Adding things no longer collides:** a finding code, a statement, a help
  golden entry or an ADR implementation-log entry does not conflict with a
  sibling doing the same.
- **Text help groups generated refusals by family,** by an ADR-008
  amendment to D15 items 3 and 4.
- **`internal/cli/render.go` is split by phase,** with no change in
  behaviour.

The tool's observable behaviour changes in exactly one place: the layout of
text help.

## What to read first

- **Plan §2**, including the two breakdown rules Ed added on 2026-10-19:
  shared seams are named at breakdown, and statement numbers are allotted
  at breakdown. Read **§4**, **§11b**, and **§11**'s verdict.
- **Wave 5's closeout**, in
  [brief-plan-2026-10-metais-v1-wave5.md](brief-plan-2026-10-metais-v1-wave5.md),
  especially "The gate" and the conformance timing.
- **ADR-010 D14 and D15**, **ADR-008 D15 and D16**, and **ADR-011 D3**
  (statement numbering).
- **AGENTS.md**, whole, and the two skills on `main` (7186c72, #93):
  - `.claude/skills/verification-gate/`: `SKILL.md`, `gate.sh` and
    `gate-wait.sh`;
  - `.claude/skills/root-cause/SKILL.md`.
- **For the model only**, in `kottera/kottera`, which you may read and
  must never write:
  - `scripts/test-touched.sh`, `scripts/gate-start.sh`,
    `scripts/gate-wait.sh`, `scripts/gate-why.sh` and
    `scripts/gate-record.sh`;
  - `.claude/skills/verification-gate/` and `.claude/skills/root-cause/`.

  Do not port kottera's selector, which is about 7,000 lines. metais is one
  module of about 30 packages and needs a small one. The skill text names
  no kottera path, make target or tera internal.
- **The code these fixes touch:**
  - `internal/finding/registry.go` (about 3,900 lines) and its tests,
    especially `TestEmittedCodes` and `spec_test.go`;
  - `internal/specdata`;
  - the help goldens under `internal/cli/testdata/help/` (`help.json` is
    one line);
  - `internal/cli/render.go` (about 1,700 lines), and `renderRefusals`;
  - `internal/hostfs`'s `TestShapesWriteTheirLimitUnderTheOpenFileLimit`
    and `TestExistingWritesItsLimit`;
  - `.gitattributes`;
  - each ADR's "Implementation log" section.

## Ed's rulings — this wave's scope, carried whole

These are Ed's rulings of 2026-10-19, given by guidance. They are decided:
do not reopen them. They are not rulings made by the train, so they are not
for ratification at its close. The amendments that record them are accepted
on merge.

### A. The test-time work

1. **An ADR-010 amendment to D15 item 3, with two tiers.**
   - **The scoped tier, for each Work Order:**
     - build, vet and gofmt for the whole module;
     - `go test -race` for the touched packages and the packages that
       import them;
     - the spec and generated-data tripwires on every run: `internal/finding`,
       `internal/specdata`, `internal/conformance`'s well-formedness tests,
       the help goldens, and the specdata drift.
   - **The wave tier:** the full list in AGENTS.md runs once per wave, at
     wave verification, on the merged tree. It also runs for any change to
     a shared seam the amendment names, for example `internal/fault`,
     `internal/testenv` or `go.mod`.
   - **The six-target cross vet and build** stays in the scoped tier if it
     is cheap, and otherwise moves to the wave tier. The amendment argues
     the choice from measurement.
   - **AGENTS.md's Development section** changes to match.
2. **Gate scripts in metais.** They:
   - start the gate in the background, and wait on it in bounded steps,
     with a record per tier;
   - offer `--resume`, which re-runs only the tiers or packages still
     unverified after a fix;
   - include a "why" helper that re-runs one test alone with
     `-race -count=1`.

   They update the existing `.claude/skills/verification-gate/` scripts
   rather than adding a parallel set.
3. **A shared-host cap.** `go test -p`, and `-parallel` where it matters,
   is derived from the host's cores, with an environment override. It
   follows the reasoning AGENTS.md already applies to fuzzing.
   `METAIS_GATE_P` already exists in `gate.sh`; keep the one override.
4. **The skills.**
   - `verification-gate` covers when to run the gate, running it in the
     background and resuming it, the scoped tier and the wave tier, the rule
     after a rebase, and the ladder for "is this red mine?". That question
     is answered by reading the mechanism, not by attribution against a
     list, and the ladder points to `root-cause`.
   - **The rule after a rebase onto a sibling:** re-run what the rebase
     changed, meaning the generated files, the tripwires and the touched
     packages, not the whole gate.
   - `root-cause` keeps its method: make the failure deterministic, or make
     one occurrence enough to explain it; instrument the seam rather than
     re-run; confirm the fix so that the green means something. It gains
     metais's own instruments as they are learned: `internal/fault`'s bits,
     `internal/testenv`, the detectors, the canaries and CPU-time
     measurement.
   - **No flake register.** There is no `docs/flakes/` and nothing like
     it, and no skill mentions one.
     - A test that times out or fails only under load is a defect with a
       mechanism to find: a timing assumption, an unbounded wait, a test
       that charges wall time, or a gate that oversubscribes the host.
       Fix the mechanism.
     - "It passed alone" closes nothing.
     - No assertion is widened and no timeout raised to hide it.
   - **AGENTS.md names both skills** and says when to read each, so that a
     session that does not load skills still finds them.
5. **The `hostfs` open-file-limit tests.**
   `TestShapesWriteTheirLimitUnderTheOpenFileLimit` and
   `TestExistingWritesItsLimit` take about 237 s under `-race`.
   - Make them cheaper without weakening what they prove, or record why
     they cannot be.
   - Answer whether `TestShapesWriteTheirLimitUnderTheOpenFileLimit/StageNew_abort`,
     which sat in a syscall until the 600 s timeout in W5-1's run on
     `main`, was slow or hung. If it hung, the hang is a defect, and it is
     fixed.

### B. The conflict fixes

1. **Generated files merge themselves.**
   - Pretty-print `internal/cli/testdata/help/help.json`, and any other
     single-line JSON golden, so that git merges them line by line.
   - Add a `.gitattributes` merge driver for every committed generated
     file: `statements.json`, the specdata trees, the help goldens and the
     finding catalog. A rebase then takes one side, and the regenerate step
     plus the existing drift tests fix the rest.
   - **What must be true:** this works in a fresh clone. Either it uses a
     merge driver git provides with no configuration, or a step in the
     gate scripts installs the configuration and the skill says so. A
     driver that silently does nothing without per-clone setup is a
     defect.
   - Document the regenerate step in AGENTS.md and in the
     `verification-gate` skill.
2. **Statement-number blocks at breakdown.** Plan §2 now carries the
   convention. This wave confirms that ADR-011 D3 item 3 permits gaps in
   statement numbers. If it does not, the wave orchestrator says so and
   asks me before anything relies on gaps.
3. **Per-item ADR implementation logs.**
   - Each item writes its log entry as its own file, for example
     `docs/adr/logs/ADR-005/<item>.md`, and does not append to the ADR.
   - Each ADR's "Implementation log" section points to its directory.
   - Move the existing entries or leave them in place, and argue which.
   - Update AGENTS.md's documentation rules.
4. **One edit per new finding code.**
   - Split `internal/finding/registry.go` by area, into one file per code
     family.
   - Make `TestEmittedCodes` derive its expectation from something other
     than a hand-kept copy of every code, or argue what it pins that the
     registry alone cannot.
   - Today one new code changes `spec/12-findings.md`,
     `spec/findings/registry.json`,
     `internal/specdata/findings/registry.json` and `unwired.txt`. Make
     whichever of those can be generated from the registry generated.
   - The aim: adding a code edits one hand-written place, plus
     regeneration.

### C. Help and `render.go`

- **Fix 5: refusals in text help, grouped by family.** This is an
  amendment to ADR-008 D15 items 3 and 4. Today `render --help` is 867
  lines, about 780 of them generated refusal codes, and `check --help` is
  581.
  - `<command> --help` lists each refusal that has a hand-written cause or
    next step on its own, as today.
  - Generated refusals appear as one line per family: the family pattern,
    its count, one cause and one next step.
  - A new topic, `metais help refusals <command>`, lists every code with
    its next step.
  - `help --json` keeps the full declaration.
  - The declaration still names every code, so ADR-008 D16 item 2's proof
    is unchanged.
  - Update the goldens, and add the new topic's golden.
- **Fix 6b: split `internal/cli/render.go` by phase.** This is a
  mechanical move: no new interface and no change in behaviour. One file
  per phase:
  - planning the outputs: `planOutputs`, the planner, `declaration`,
    `container` and `containerOrder`;
  - path and destination checks;
  - parse and render: `renderPlan` and `measure`;
  - writing: `commit`, `writeContainer` and `content`;
  - the command itself: `runRender`, its declaration and refusals, and the
    result.

  The proof is that the full gate and every golden are unchanged.

### Not in this wave

- **Fix 6a, the render pipeline's extension points.** This is a follow-up
  after the train (plan §14), filed at the close as a design item for Ed.
  Do not build it, and do not shape the `render.go` split toward it.
- **The finding catalog in every command's help golden.** Fix 5 settles it.
  Nothing beyond Fix 5 is taken on.

## What Wave 5 learned that changes this one

- **The gate for this wave's own items.** Until the two-tier amendment
  merges, items run Ed's mid-Wave-5 rule: directed checks per item (the
  item's packages and their importers under `-race`, the spec tripwires,
  conformance, the help goldens and the specdata drift), plus the full gate
  at the wave's verification. Once the amendment merges, later items use
  its scoped tier. Every run uses go1.26.9 with `GOTOOLCHAIN=auto`, because
  dispatches default to go1.26.2, and records `go version` and
  GOOS/GOARCH.
- **`TestConformance` is sleep-bound at its 60 s budget.** runCase's
  2-second A/B offset holds a parallel slot. W5-NB3
  (`01a155c2-3d8a-7af0-8fea-3d7e111536e1`) is fixing it now, as a child of
  Wave 5. Do not duplicate it. Rebase onto it if it lands first, and if
  it collides with this wave's gate work, coordinate through the wave
  orchestrator.
- **This wave is mostly shared seams.** Under plan §2's new rule, the
  breakdown names them, and items that touch one seam run in sequence or
  are split by phase. The seams in view:
  - AGENTS.md's Development and documentation sections (A1, A4, B1, B3);
  - the `verification-gate` skill and scripts (A2, A3, A4, B1);
  - the help goldens (B1's pretty-printing, Fix 5's regrouping);
  - `internal/finding` (B4);
  - `internal/cli/render.go` (Fix 6b, and Fix 5 if `renderRefusals` moves
    with it);
  - ADR-010's text (A1) and ADR-008's (Fix 5);
  - every ADR's log section (B3).

  The derivation decides the cut. This list is what I see, not a
  breakdown.
- **Give the verification round a fresh clone,** and have it run the wave
  tier, which is the full list.

## Rules carried whole from the plan (§2 and §4)

- **The specification is written with the code.** This wave changes no
  behaviour of the tool except help's text layout, which is the command
  line's contract (R1-020), not the specification. It therefore adds no
  `spec/` statement unless it finds one is owed, and it adds no finding
  code. Generated spec files must regenerate byte-identical except where a
  fix changes only their production.
- **An item that finds it needs a sibling's unmerged work has found a
  defect in the slicing.** It parks and asks its wave orchestrator.
  Sequencing on a shared seam with `depends_on` is the §2 rule, not that
  defect.
- **Do not widen a refusal or weaken a proof to make anything faster.** No
  assertion widened, no limit raised, no timeout raised, no verification
  step skipped, and no runner skip except a `requires` probe. Making a test
  cheaper must keep what it proves, and the PR shows that it does.
- **Bounded work is measured in CPU time**, never by wall time alone. That
  includes the measurements the ADR-010 amendment argues from.
- **No new dependency.** The gate scripts are shell and Go from the
  standard library. Never copy code from another tool.
- **Every test is isolated** through `internal/testenv`.
- **A Work Order never runs `go test -fuzz`.**
- **Never write to another repository.** `kottera/kottera` is read for its
  model only.

## Harness policy — Claude throughout

| Role | Work Order type |
|---|---|
| Wave derivation and supervision | `wave-orchestrator-derive` |
| Implementation and corrective work | `code-implementation-direct` |
| Whole-wave verification | `wave-verification` |

These are `plugin.dev-pipeline.v1`'s types; name one by its full key wherever
its leaf is ambiguous. Never substitute another harness's type to get past a
capacity wait: park on `awaiting-capacity` and let the seat come back.

## What must not break, and how it is checked

- **Behaviour is unchanged except help's text layout.**
  - Every vector, every `--json` golden and every detector gives the same
    result before and after.
  - `help --json` keeps the full declaration.
  - ADR-008 D16 item 2's per-code test still proves every declared code.
- **Fix 6b is proved by an unchanged full gate and unchanged goldens.**
- **The new gate is not weaker than the old one in what it can catch.** The
  scoped tier runs every package a change can affect (the touched packages
  and their importers), plus every tripwire, and the wave tier runs the
  whole list. The ADR-010 amendment says what the scoped tier cannot catch
  and why the wave tier catches it.
- **The merge driver** is shown working on a fresh clone: a scripted
  conflicting rebase of a generated file resolves, regeneration runs, and
  the drift test passes.
- **The full gate runs green at the wave's verification**, on a fresh clone
  of the merged tree, on go1.26.9.

## Out-of-band acts of this wave

None. The plan orchestrator tracks the train's open acts: the liquid-spec
re-run W5-A1 (`01a155c2-5c85-7762-a3d8-2541e178e29c`) and registering
`metais.org`.

## Closeout

Wave orchestrator: `01a155c8-f9a1-7493-b5c8-764ddeaca7dc`
(`wave-orchestrator-derive`). Base `dcaa698`; the wave ended with `main` at
`20eeeb2`.

**This is not a correctness verdict.** What I checked, and how:

- I read each child's outputs and pull request summary.
- In the merged tree I checked that each mechanism the brief named exists:
  - the four amendments: ADR-010 A5, ADR-008 A1, ADR-006 A2, ADR-011 A5;
  - the per-item log files;
  - the gate and merge scripts and `.gitattributes`;
  - `TestRegistryFamilies`, `TestRegistryFileGenerated` and
    `TestHelpListsNoGeneratedRefusalOnItsOwn`;
  - `check --help`'s family line for `output.declaration.validate_kind`.
- I ran no test, no gate and no mutation. Every result of a run below is a
  child's or the verifier's, and is attributed to them.

The breakdown was approved as drafted on 2026-10-19. The approval also
settled ruling B2 as R-W5b-B2, below.

### 1. Did each item land, and does it match the ask?

All seven items and the one blocking fix are complete and merged.

| Item | Work Order | PR | Merge | What landed (as the producer reports it) |
|---|---|---|---|---|
| W5b-1 two-tier gate | `01a155d9-7eb7-7651-984b-478569461461` | #113 | `278c96a` | ADR-010 A5 defines the scoped tier and the wave tier, what "touched" means, the shared seams (`go.mod`, `go.sum`, `internal/fault`, `internal/testenv`, the gate's scripts) and what the scoped tier cannot catch. `gate.sh` runs either tier (`gate.sh` and `gate.sh wave`) and resumes with `--resume`; `gate-select.sh` is the selector and `gate-why.sh` re-runs one test. The cap is a third of the cores, for both `-p` and `-parallel`, set by `METAIS_GATE_P`. AGENTS.md and the skill are rewritten to match |
| W5b-2 hostfs tests | `01a155d9-c53d-741e-acfd-0875abe8af05` | #111 | `1896cb8` | `StageNew_abort` was **slow, not hung**: one instrumented run timed all 300,100 calls and none took more than 1 s. Redundant test work was removed (about 1 to 1.7 s of CPU). The cost floor is recorded in `docs/adr/logs/ADR-010/W5b-2.md` |
| W5b-3 one edit per code | `01a155da-1e3b-78ed-8591-958ed15fbc36` | #108 | `a4773de` | 16 `registry_<family>.go` files. `spec/findings/registry.json` is now generated from the table (`go generate ./internal/finding`), recorded as ADR-006 A2. `TestEmittedCodes` is replaced by `TestRegistryFamilies`. `unwired.txt` stays hand-written, by argument |
| W5b-4 `render.go` split | `01a155da-50af-7094-bf04-ac93b117ce42` | #109 | `960b567` | `render.go` (the command), `render_plan.go`, `render_check.go`, `render_parse.go` and `render_write.go`; `sampleRefusals` moved to `sample.go`. A go/ast comparison found all 357 declarations identical |
| W5b-5 help refusals | `01a155da-eb27-79fa-b7ef-758928b49bd3` | #114 | `3803a73` | ADR-008 A1. Text help lists generated refusals as one line per family; `metais help refusals <command>` is a new topic with its own goldens; the JSON help goldens are indented. `render --help` went from 1,138 to 305 lines and `check --help` from 584 to 120 |
| W5b-6 per-item logs and root-cause | `01a155da-9d18-7b9c-bdd2-bb14538c688d` | #110 | `e92d2a4` | A pointer after every ADR's `## Implementation log` heading. AGENTS.md's rule for log-entry files. Existing entries left in place, by argument. The `root-cause` skill gains a section on metais's own instruments |
| W5b-7 generated files merge | `01a155db-42da-7904-be5c-ce47870180a7` | #116 | `8834408` | Two drivers: `metais-generated` for whole files, and `metais-generated-region` for the catalog in `12-findings.md`. `merge-setup.sh` installs them, and every gate script runs it. `regenerate.sh`; `merge-check.sh` (five cases on a fresh clone); ADR-011 A5. Ruling B2's reading is in `docs/adr/logs/ADR-011/W5b-7.md` |
| W5b-V1 blocking fix | `01a15660-639e-77c4-901c-a0eb005c3ad0` | #117 | `20eeeb2` | `check` gives the `output` family a subject, so `output.declaration.validate_kind` is on a family line. A separate `wholesale` list keeps check's declared codes unchanged. `TestHelpListsNoGeneratedRefusalOnItsOwn` was red at `8834408` |

**Divergences from the ask:**

- **W5b-1** keeps the six-target cross build in the scoped tier, from
  measurement: 22.5 s of CPU warm, up to 310 s cold, against 1,397 s for the
  whole `-race` suite.
  - Both tiers pass `-count=1`, which the old skill said the gate never
    does. W5b-1 found that the test cache returned stale "(cached)" verdicts
    for the black-box packages after a code change. The amendment records
    this.
- **W5b-2** made the tests barely cheaper. Most of their cost is the
  product's own per-entry work at ADR-009 D3's defaults. The brief's
  alternative, "the reason they cannot be is recorded", is what landed.
- **W5b-3** makes the Go table the source of `registry.json`. ADR-006 A2
  records this under Ed's ruling B4. ADR-011 D3 item 9 stands as written.
- **W5b-7** covers more generated files than the brief listed: the
  scaffold, tar, YAML-writer and vector-package goldens, and
  `THIRD-PARTY-NOTICES.txt`. The verifier confirmed that all 110 files
  under a driver regenerate.

### 2. What children flagged, degraded, declined or discovered

None of this was verified by me unless I say so.

- **W5b-1 found two defects in the old gate and fixed them:**
  - the stale-cache verdict above;
  - its own first `gate.sh` reported PASS over a red package (a `tee`
    exit status). A planted failing test showed it.
- **W5b-1's squash commit `278c96a` carries Co-Authored-By trailers**,
  which AGENTS.md forbids. The producer declared it, and the verifier
  counted four trailers. Removing them would rewrite `main`'s history, so
  no child was filed (question 4).
- **The selector over-selects, and says so.** A path outside every package
  reaches the packages whose tests name it as a path literal, so a docs
  edit reaches `unitext`. A5 item 8 states what this can miss.
- **The hostfs tests ran only on darwin/arm64** (W5b-2); their linux branch
  did not run. W5b-4's and W5b-7's gates ran on linux/amd64. Every other
  item ran on darwin/arm64.
- **W5b-6 and W5b-3 merged in the same minute.** W5b-6 re-ran its checks on
  the merged `main` afterwards. The verifier's wave tier covers both.
- **W5b-7's drivers have not run on Windows or darwin.**
  - A modify/delete conflict never reaches a driver; the skill says to take
    either side and regenerate.
  - In a clone where `merge-setup.sh` has not run, a rebase falls back to
    git's text merge. That merge is loud on a conflict and silent on a
    clean merge, and the drift tests catch a stale result. The next gate
    run installs the drivers and prints a `NOTE`. The verifier judged
    W5b-7.2 met: git gives a repository no way to install a driver on
    clone.
- **W5-NB1, W5-NB2 and W5-NB3 (Wave 5's children) landed during this
  wave** (#107, #115, #112). W5b-1 re-measured its cap after NB3. Before
  NB3, `-parallel 3` ran conformance into its 10-minute timeout. A5
  records the rule that "a test must not hold a parallel slot while it
  waits".

### 3. What carries to the next wave

- **The gate for Wave 6:**
  - items run the scoped tier with `gate.sh`, in the background, resumed
    after a rebase with `gate.sh --resume`;
  - an item that touches a shared seam (`go.mod`, `go.sum`,
    `internal/fault`, `internal/testenv`, the gate's scripts) also runs
    `gate.sh wave` before merging;
  - the wave verifier runs `gate.sh wave` on a fresh clone;
  - every session runs `merge-setup.sh` before its first rebase and
    `regenerate.sh` after every rebase.
- **ADR implementation logs** are files under
  `docs/adr/logs/ADR-<NNN>/<address>.md`, as AGENTS.md now states. A
  breakdown no longer needs to fix the path for its items.
- **Statement numbers:** a breakdown allots each item a block per area, and
  gaps are permitted (R-W5b-B2). This wave allotted none, because it added
  no statement.
- **The help goldens are indented, and the generated refusals are
  grouped.** A Wave 6 item that adds a refusal code changes one family
  line's count, not 80 lines per command.
- **Wave 6 adds the hostile corpus in full.** W5-NB3 (#112)
  made the conformance runner hold no parallel slot through its A/B wait,
  so a case's wait no longer blocks another case. The verifier's wave tier passed the whole suite at
  `-parallel 4`.
- **What I would do differently:**
  - **I fixed the log-file shape at breakdown,** and W5b-6 adopted it
    without a park. That worked; do it for any convention a sibling must
    follow before its owner merges.
  - **I filed W5b-NB2 into the blocking-fixes cohort** by passing the wrong
    id. It had dispatched before I could re-file it. Only its cohort label
    is wrong.

### 4. What the verification round found, and where every finding went

**Verifier:** W5b-V, `01a15653-b323-7747-854b-86d92fdb3f0f`
(`wave-verification`).

- **The gate:** it ran the wave tier,
  `GOTOOLCHAIN=auto .claude/skills/verification-gate/gate.sh wave`, on a
  fresh clone of `main` at `8834408`, with go1.26.9 linux/amd64 at
  `-p 4 -parallel 4`. **GREEN**: 42 of 42 packages under `-race`, and
  build, vet, tidy, gofmt, static, cross and specdata all passed.
- **Other checks:**
  - `merge-check.sh` passed all five cases on a fresh clone;
  - after it corrupted all 110 files under a driver, `regenerate.sh`
    restored every one;
  - it confirmed W5b-4's 357 identical declarations independently.
- **Verdict: blocking**, with one finding.
- **Coverage:** it returned all seven ids I sent, and audited
  `dcaa698..8834408`. The seven items plus the four commits I named as not
  this wave's account for all eleven commits in the range.

**Depth per item:**

| Item | Depth | What the verifier mutated |
|---|---|---|
| W5b-1 | **inspected, not mutated** | No guard of the gate scripts was mutated. The wave tier itself ran through `gate.sh wave` |
| W5b-2 | mutated | A descriptor leak on `Output.MkdirAll`'s existing-directory branch. `StageNew_abort` stayed green (finding NB-1 below); `TestExisting…` alone went red |
| W5b-3 | mutated | Hand edits of both `registry.json` copies and the catalog went red. Dropping a family from the table went red. Swapping two families stayed green, which is an equivalent mutant, since the table is sorted |
| W5b-4 | **inspected, not mutated** | It re-ran the go/ast identity comparison |
| W5b-5 | mutated | Dropping the code-family test or the `p and p.*` branch went red, as did omitting a code from the topic and an extra space in `help.json`. Dropping `f.next == r.Next` stayed green (finding NB-2a) |
| W5b-6 | **inspected, not mutated** | — |
| W5b-7 | mutated | Making `merge-generated.sh` report success on a prose conflict turned `merge-check.sh` red |

W5b-1, W5b-4 and W5b-6 were read, not broken. That is a weaker result than
"verified", and for W5b-1's scripts it means `--resume` and the selector's
guards are untested by anyone but their producer.

**Cross-item findings:** none. The verifier checked that:

- W5b-5's new goldens fall under W5b-7's glob and `regenerate.sh`;
- `gate.sh`'s tripwire names all exist;
- every amendment sits before `## Implementation log`;
- the wave tier is green although W5b-5 and W5b-7 were never gated
  together.

**Findings, one row each:**

| Finding | Item | Disposition | Where it went |
|---|---|---|---|
| `check --help` lists the registry-loop refusal `output.declaration.validate_kind` on its own, with a generated, subject-less cause (quoted W5b-5.2) | W5b-5 | **blocking** | Fix child W5b-V1 `01a15660-639e-77c4-901c-a0eb005c3ad0`, PR #117, merged `20eeeb2`. **Not re-verified**, under the one-round rule |
| NB-1: `StageNew_abort` no longer takes `MkdirAll`'s existing-directory path, which render's `prepare` takes once per file | W5b-2 | non-blocking | Child W5b-NB1 `01a1566a-4184-7367-92dc-49ee5d5dcc56` |
| NB-2a: no test exercises the next-step part of `families()`'s grouping key | W5b-5 | non-blocking | Child W5b-NB2 `01a1566a-70f6-73a5-b93a-d57fae90a1a0` |
| NB-2b: check's generated causes say "a error" / "a info" (predates the wave) | W5b-5 | non-blocking | Child W5b-NB2 `01a1566a-70f6-73a5-b93a-d57fae90a1a0` |
| Three comments cite ADR-011's merge-driver amendment as A1; it is A5 | W5b-7 | non-blocking, prose-only and inert | Janitorial issue [#118](https://github.com/kottera/metais/issues/118) |
| Squash commit `278c96a` carries Co-Authored-By trailers, against AGENTS.md | W5b-1 | non-blocking | **Not filed.** The only repair rewrites `main`'s history, which no Work Order should do. It is for Ed: rewrite `main`, or let it stand |
| The answer to my W5b-7.2 question: without `merge-setup.sh`, a rebase falls back to text merge | W5b-7 | non-blocking | **Not filed.** It is an answer, not a defect. The verifier judged W5b-7.2 met, and the drift tests catch a stale clean merge |

NB-2a and NB-2b share one child, W5b-NB2. Both are in `internal/cli`'s help
code and regenerate the same goldens, and its task says they are
severable. **The wave completed without waiting for W5b-NB1 and W5b-NB2.
That is the intent:** they run beside the next wave and are not
re-verified.

### 5. Out-of-band acts

None. The brief carried none, and no child needed one. The train's open
acts (W5-A1's liquid-spec re-run, registering `metais.org`) are the plan
orchestrator's.

### Rulings made during this wave

- **R-W5b-B2** (Ed, answering the breakdown, 2026-10-19):
  - ADR-011 D3 item 3 permits gaps in statement numbers: a number allotted
    and never used is never assigned.
  - This is not an ADR-011 change, and no contiguity check is added.
  - Plan §11b records it (#106), and so does
    `docs/adr/logs/ADR-011/W5b-7.md`. It governs every later breakdown's
    statement blocks.
- **Kept by Ed at approval:**
  - (a) W5b-7 amends ADR-011 D3 item 7, which became ADR-011 A5;
  - (b) the merge driver covers only `12-findings.md`'s generated region.
- **Fixed at breakdown:**
  - the log-file path `docs/adr/logs/ADR-<NNN>/W5b-<N>.md`;
  - ADR amendments go before `## Implementation log`, and W5b-6's pointers
    after it.

I made no ruling on a child's question; no child asked one.
