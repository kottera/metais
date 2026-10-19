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

*Not yet written. The wave orchestrator replaces this line with the wave's
closeout.*
