# Brief — PLAN-2026-10-metais-v1, Wave 4: validate, read, the first render, check and unpack

**Plan:** [PLAN-2026-10-metais-v1](../plans/PLAN-2026-10-metais-v1.md), §10
**Feature:** `metais-v1`. Work Orders are titled `metais-v1 [W4-<item>]: <action>`;
pull requests and their squash subjects are `<type>(metais-v1): [W4-<item>] <change>`
(AGENTS.md, Git and pull requests).
**Repository:** `kottera/metais`, branch `main`. Every Work Order of this wave
gets `repo: kottera/metais` and `branch: main`.
**Gated on:** Wave 3, complete. Its verdict is under plan §9, and its
closeout is in
[brief-plan-2026-10-metais-v1-wave3.md](brief-plan-2026-10-metais-v1-wave3.md).
The gate has cleared.

## What this wave is for

When Wave 4 is done:

- A supplier validates inputs with every supply flag and `--resolved`, and
  renders template outputs into a new directory.
- A harness reads any part of a package in place with `ls`, `read`,
  `outline`, `grep`, `schema` and `extract`.
- An author checks a package without data, and unpacks one.
- The template environment of chapter 7 is proved through `render`.
- The headline detectors for confinement, determinism, read-in-place and
  one pass run in every `go test`, each with its fault as control.
- The first SC7 canaries (`env`, `cwd`, `path`) are caught.
- The Liquid engine is closed: Golden Liquid in full, the liquid-spec
  JSON-RPC server built, and the delegated-run file committed.

## What to read first

- **The plan:** §2 (the execution model and the rules carried below), §4
  (safety discipline), §5, §6 (the liquid-spec act), and §10 (this wave:
  objective, carries, does-not-touch, scope outline, acceptance, settled
  decisions). §10 is the authoritative scope. This brief binds it and does
  not restate its outline. Read the rulings and verdicts under §7 to §9 too.
- **Wave 3's closeout**, all of it, especially questions 2 and 3.
- **The ADR decisions §10 carries:**
  - ADR-003, as observable;
  - ADR-004 D1 to D8 and D16;
  - ADR-005 D2, D4 item 4, D12, D13, D14 item 1 and D19;
  - ADR-006 D6 to D11;
  - ADR-007 D2 to D4 and D8;
  - ADR-008 D5, D7, D8 items 1 to 3, and D16 items 2, 4 and 8;
  - ADR-009 D7, D8 items 1 to 3, D11, D13, D15, and D16 items 1 and 2;
  - ADR-010 D4 item 7 and D13;
  - ADR-011 D17.
- **ADR-010's Amendments:**
  - **A2 is Ed's ruling and binds this wave.** `internal/template` names
    `fault.MapOrder` and sets the engine's test-only `Env.UnorderedObjects`
    from it. `internal/liquid/...` stays standard library only. A new
    tripwire rule forbids any non-test file outside `internal/template`
    from setting the option.
  - A1 and A3 are proposed.
- **ADR-011 A1** (the `PKG` area), proposed.
- **RULINGS-v1**: the rulings §10 lists.
- **AGENTS.md**, whole.

Read the code you wire. Wave 3 left these entry points:

- `inputs.New` with the `Supply` methods;
- `liquid.Render` with `Env{Dialect, Globals, Partials, Escape, Meter,
  Limits}`, and `Core()`;
- `xref`, and `ooxml.Scan`;
- the package loader behind `inspect`;
- `hostfs`'s write side;
- `internal/conformance` with its adapters, and `unwired.txt`, which lists
  106 codes for W4.

## What Wave 3 learned that changes this one

- **The runner needs an unwritable-destination pre-state**, behind a
  `requires` probe (ADR-011 D9 item 6). Without it,
  `output.destination.write_failed` (from `init`) and render's
  `commit_failed` have no vector, and they become findings for Ed under
  plan §8. Building that pre-state is in this wave's scope, as runner work
  feeding the render's and `init`'s vectors.
- **The renderer reports a wrong-type loop limit or offset as engine code
  `argument`.** `internal/template` maps it to ADR-006's finding.
- **`input.raw_unavailable` was left to the render** (W3-2).
- **`{field}` messages exceed ADR-006 D9 item 1's 240 code points on long
  pointers.** This is the wave that makes such messages observable through
  `validate`, so it holds the bound. The message is bounded, and the full
  pointer stays in the finding's location.
- **The gate flaked under load on linux** in producers' runs: a CPU-ratio
  test in `internal/author/ooxml`, and self-proof wall bounds under
  parallel `-race`. A flake an item meets is a finding. Its mechanism is
  debugged and made deterministic, never closed by widening an assertion
  or a bound (AGENTS.md). Report every flake seen, with the host.
- **Dispatches default to `GOTOOLCHAIN=local` (go1.26.2).** Run the gate
  with `GOTOOLCHAIN=auto`, and record `go version` and GOOS/GOARCH in the
  PR.
- **The `date` filter and its 10 Golden Liquid cases are Wave 5's.** "Golden
  Liquid passes in full" in §10 means everything but the cases the Wave 3
  selector holds out for `date`. The selector fails by design when Wave 5
  registers `date`.
- **Meeting points are named in item criteria**, with the merging-second
  item owning the test. Where a ruling assigns a fix to an item, name the
  owner at breakdown, not "whichever first".
- **Check criteria against the guards' rules and the corpora's counts
  before filing.** Vector area codes obey ADR-011 D9 item 8's portable-name
  rule; chapter 2's area is `PKG`.

## Rulings already made — do not reopen

From plan §10, "Decisions settled before filing", ratified with RULINGS-v1:

- R1-012: existing files are never read, held from the first render on.
- R1-017: strict templates.
- R2-037 and R2-086: partial validation, and constraints skipped when an
  input fails to parse.
- R2-060 to R2-063: rendering errors and globals.
- R2-081 and R2-134: `--fail-on`.
- R2-093: the `at` line.
- R2-099: `check`'s severities.
- R2-105 and R2-136: `unpack`.
- R2-117, R2-120 to R2-122, R2-126, R2-128 and R2-130 to R2-132: the
  commands.
- R2-144 and R2-145: truncation and neutralisation.
- R2-176 and R2-182: the liquid-spec pin.

Earlier waves' rulings stand, recorded under plan §7 to §9:

- the data model's home;
- the dependency-scan reading;
- the envelope's `summary`;
- the Markdown reading;
- the package-home constraint;
- the `date` selector;
- the large-integer reach;
- `PKG`.

Rulings made by the plan orchestrator for this wave:

- **`hostfs` must not make a render's limits unreachable.** Design session
  `01a15284-df6e-7668-b799-62f4c380a84f` is still open with Ed. It asks
  whether `hostfs` may hold one open handle per created, uncommitted entry
  (#40), or which ownership mechanism replaces that. This wave renders and
  unpacks only into a new directory (ADR-005 D14 item 1), so it does not
  wait on the general answer. These must be true of what this wave ships:
  - A render or an unpack within ADR-009's limits does not fail because of
    a host's default open-file limit. Darwin's default soft limit is far
    below the outputs limit, and limits are exact (R1-015), so a mechanism
    that silently lowers one is a defect.
  - Whatever a failed render or unpack removes is provably its own (ADR-005
    D14 item 5), never established by re-identifying a reusable name or
    inode.
  - Nothing depends on the process umask granting the owner read.

  If the wave cannot meet these without choosing `hostfs`'s general
  mechanism, the item parks and asks. It does not choose that mechanism on
  its own. If Ed's session ends during the wave, I relay its ruling, and it
  governs.
- **`limit.exceeded`'s subject.** This wave is the first to emit it for a
  render's work and for outputs (ADR-009 D19), and the registry gives it
  one subject, `input`. What must be true: each emission's subject is
  truthful under ADR-006 D4. Settle it inside ADR-006's registry model,
  for example by codes or subjects that registry already allows, and name
  the reading in the PR. If ADR-006's model cannot express it without an
  amendment, park.
- **Reporting readings from Wave 3 stand.** ADR-006 D8's de-duplication
  key governs repeats of one name, because ADR-006 owns the report.
  ADR-007 D3's suggestion distance governs `check`'s suggestions, because
  it is the specific rule. The invocation is the subject for
  `input.constraints_failed` and `input.json_arg_invalid`. These are not
  accept/refuse disagreements, and they go to Ed for ratification at the
  close.

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
- **A library is built ahead of its wiring, and its statements land with
  its wiring.** This wave is mostly wiring. A code it makes producible
  leaves `unwired.txt` in the same pull request.
- **A finding code is registered by the item whose code emits it.**
- **Where two items of one wave meet, the one that merges second adds the
  vector for where they meet**, after rebasing. Name each meeting in the
  criteria. The verification round checks that every meeting has one. This
  wave's widest meeting: every `security` vector refuses identically
  through `inspect`, `ls --recursive`, `schema`, `validate`, `render --out
  out` and `unpack --to unpacked`. Each command joins the opening set as it
  lands, which `TestHostileVectorsOpenEverywhere` reads from the
  declarations.
- **An item that finds it needs a sibling's unmerged work has found a
  defect in the plan's slicing.** It does not reach into the sibling's
  scope. It parks and asks. There are no `depends_on` edges between this
  wave's code items. The one edge in this wave is the liquid-spec act's
  edge onto the closure item (below).
- **Pin the property false first.** Each of these is red against a tool
  that breaks it, lands before or with the code that makes it true, and
  stays in every `go test`:
  - **no write outside the output directory**: the confinement detector,
    with `SkipConfine`;
  - **byte-identical outputs within one build**: the determinism detector,
    with `MapOrder` through A2's `Env.UnorderedObjects`;
  - **read in place**: the browse detector, with `UnpackToTemp`;
  - **every finding in one pass**: the SC8 planted-errors test, with
    `FirstFinding`;
  - **the `env`, `cwd` and `path` canaries**, each landing red with the
    channel it probes. Their red run is recorded in ADR-009's
    implementation log before the code that makes the release build pass
    them lands (ADR-011 D12 item 6).
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
  `THIRD-PARTY-NOTICES.txt` by the item that links it (R4). Never copy code
  from another template engine, scaffolding tool or schema validator.
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

In scope: everything in plan §10's scope outline, the runner's
unwritable-destination pre-state, and the carried fixes named above.

These belong to Wave 5 or later and must not be absorbed here (plan §10
"Does not touch", §11, §12):

- the METAIS filters, `date`, `embed`, `asset` and `provenance`;
- `escape: html`, `for_each`, `when`, path templates, the mirror, assets,
  containers and transforms;
- existing destinations (ADR-005 D14 item 2), dry run, standard output and
  `--only`;
- `sample`, `render --sample` and `test` (Wave 5);
- `pack` (Wave 6);
- `hostfs`'s general ownership mechanism, pending Ed's design session;
- the jsonschema `Compile` cost, pending design session
  `01a15348-66d1-73ef-bed1-8d62aa3dff0e` and Wave 6's calibration;
- the 16 codes no vector can produce, pending design session
  `01a15348-4ab5-7743-9802-86072e41b5bf`. Leave them as listed W7.

## What must not break, and how it is checked

- **The gate**, AGENTS.md's Development list, run whole on go1.26.9 by
  every item before merge, and by the verification round on the merged
  tree.
- **Plan §10's acceptance**, each criterion proved in the tree:
  - the four detectors fail on their faults and pass on the tool;
  - the `env`, `cwd` and `path` canaries are each caught, with their red
    run recorded first;
  - every `security` vector refuses identically through `inspect`, `ls
    --recursive`, `schema`, `validate`, `render --out out` and `unpack --to
    unpacked`;
  - `TestCoverage` is green over chapters 2 to 7, 12, and 13 so far;
  - the unwired list holds nothing this wave wires;
  - the liquid-spec act of §6 is filed and waits on the closure item.
- **The conformance suite stays within ADR-011 D11 item 9's 60 seconds**
  under `go test -race`. The wave adds many vectors, so watch it.
- **Every earlier guard stays green:**
  - `TestNoHostFacts` and the layering table, with A2's new rule;
  - the licence header, notices and spec-data drift;
  - the document tripwires and the runner's self-proof;
  - the digest detector and the read-once budget;
  - the hostile corpus through every opening command.
- **What a sandbox cannot prove** (plan §5): operating-system branches run
  only on the host's own; the pseudo-terminal proof and SC5 are §6's
  matrix act; the liquid-spec run itself is §6's act below. Each PR claims
  nothing it did not run. An item or verifier that lands on linux runs
  `go test -race ./internal/hostfs/...` and reports it.

## Out-of-band acts of this wave

From plan §6, carried whole:

- **Run the pinned liquid-spec selection against the reference engine**
  through the Wave 4 JSON-RPC server, and record the result.
  - **When:** Wave 4, after the Liquid closure merges.
  - **Who:** Ed, or a host with Ruby and Bundler.
  - **Before:** Wave 5.
  - **Notes:** ADR-011 D17. Each departure must fail as listed, and one
    that passes fails the run. Run it again at every engine change, at
    every change to chapters 7 or 8, and at release.

Placement is the wave orchestrator's.

- **Plan §10's acceptance asks that it be filed and wait on the closure
  item**, so it takes a `depends_on` edge to that item.
- **Its Before is Wave 5, not this wave's close.** It may be carried rather
  than waited on, and the plan orchestrator will not file Wave 5 until it
  has ended.
- **The type:** this deployment's off-tera types are `ratify-plan`,
  `dogfood-round`, `live-deploy`, `darwin-measurement`, `manual-qa-pass`
  and `design-session`. `manual-qa-pass` (`build`, `focus`) fits, with
  `build` the closure item's merge commit.
- **What to ask for, learned from Wave 1's credential act, which was ended
  with nothing run:** the `focus` names the exact command to run, the host
  and its Ruby and Bundler versions to record, and the result to report
  back. That result is the pass/fail count, and every departure's outcome.

## Closeout

The wave orchestrator wrote this closeout (Work Order
`01a1534a-2fe2-7c9a-9ea7-bcc2fea20cad`) after the wave drained. It reports
what landed and what carries forward. **It is not a correctness verdict.**

**What I checked, and how:**
- I read every child's report, the verifier's full report, and the merge
  commits on `main`.
- In the merged tree at `d4ba3ba`, I read:
  - `internal/fault`: the five bits are as assigned at breakdown;
  - `unwired.txt`: no `W4` line remains;
  - `spec/07-template-environment.md`;
  - `spec/conformance/delegated/`;
  - ADR-009's implementation log: the three canary red runs are recorded;
  - W4-V1's diff.
- I confirmed the missing `hidden-only` fixture myself before filing its fix.
- I ran no test, no gate and no mutation. Where a guarantee is said below to
  be proved, the proof is the producer's or the verifier's, and it is
  attributed to them.

**Wave range:** `c9fa2d5..d4ba3ba` on `main`.
- **The breakdown:** seven implementation items. The plan orchestrator
  approved it with one edit (Decision 2) and sanctioned four `depends_on`
  edges (Decision 1). It was released as one set.
- **Verification:** one round, which found one blocking defect. It got one
  fix child, W4-V1, which merged and is not re-verified.
- **Non-blocking findings:** they went to two children, and some to the plan
  orchestrator, with reasons (question 4). **The wave completed without
  waiting for them, deliberately.**
- **The liquid-spec act** is filed and carried (question 5).
- **Commits in the range that are not items:**
  - #66, the plan orchestrator's record of this wave's rulings;
  - #65 and #73 (W3-NB2);
  - #74 (W2-A1);
  - #76 (W3-A2);
  - #78 (W3-A1, design session `01a15348-4ab5`'s ruling).
- **No person was consulted.** The breakdown review went to the plan
  orchestrator. W4-0's question was decided here, as ruling R-W4-0 below.

### 1. Did each item land, and does it match the ask?

| Item | Work Order | PR | Merge | Landed |
|---|---|---|---|---|
| W4-0 `hostfs` write shapes | `01a15353-d737-7e8e-85a6-e44c3ea43abc` | #67 | `7afb1a9` | `StageNew` (a new directory through staging), `WriteInPlace` (an empty target) and `CreateNew` (`extract`). Each holds bounded handles: 100,000 files under an open-file soft limit of 128. Each makes a umask grant. `fault.SkipConfine` sits at its `hostfs` seam. The ownership boundary is ruling R-W4-0 |
| W4-1 `validate` | `01a15354-8220-791c-94f4-4bfd86717d7a` | #69 | `d8b621d` | `validate` with every supply flag and `--resolved`. 44 statements across chapters 5, 6, 12 and 14, with 27 vectors. `FirstFinding` and `TestOnePassDetector` (SC8). The 240-code-point bound. `--fail-on`, once, for every command that reports findings. The `path` canary. 84 codes off the unwired list |
| W4-2 reading commands | `01a15354-f1c0-7d90-a712-ac7ded2a72b5` | #70 | `c8c717c` | `ls`, `read`, `outline`, `grep`, `schema` and `extract`, in `internal/browse`. `UnpackToTemp` and the read-in-place detector. The `read --lines` budget, counted in decodes. All six commands join the opening set. `extract` refuses an existing target and has no `--overwrite` (Decision 2) |
| W4-3 render | `01a15355-804e-736e-817f-11db46f96057` | #72 | `f341681` | `internal/template` and `render --out <new dir>` through `StageNew`. Chapter 7 (MTS-TPL-001 to 052), plus render's statements in chapters 9, 11 and 14, with 33 vectors. The confinement and determinism detectors, and A2's tripwire. `TestHostIsolation` and `TestHostIsolationCanaries`, with the `env`, `cwd` and `path` canaries. Every meeting it was second to |
| W4-4 `check` | `01a15355-e7a5-70e2-8357-d8c0cc2cfc5e` | #68 | `c7cd167` | Every ADR-007 D2 step, with `report.step_skipped`. xref and the OOXML scan are wired. MTS-AUT-001 to 023, with 18 vectors. The `template.xref.*` and `template.ooxml.*` codes are off the list, except two moved to W5 |
| W4-5 `unpack` and the unwritable pre-state | `01a15356-4a5e-7989-aea3-f6acf17091a4` | #71 | `84d3885` | `unpack`, staged or in place, and the round trip without `pack`. MTS-AUT-024 to 029. `setup.unwritable` with `requires: unwritable-dirs` (ADR-011 A2, proposed). `aut/unpack-write-failed`. The self-proof flake's two mechanisms, fixed |
| W4-6 Liquid closure | `01a15356-b4cf-734b-a531-f1fe881b47a7` | #75 | `aee03df` | Golden Liquid in full: 1,098 cases, 10 selected out for `date`, 1,088 run, 908 pass and 180 excluded, all on `dialect` lines. `internal/liquidspec`, the JSON-RPC server. `spec/conformance/delegated/` with the pin, the gates, 490 departures each tied to a statement, `liquid-spec.rb` and the run's README. Engine fixes against gem 5.14.0 `strict2`, with MTS-TPL-053 to 055 |
| W4-V1 blocking fix | `01a153fb-d8a6-7284-8be0-0f0bc4fada17` | #77 | `d4ba3ba` | Commits `rnd/destination-not-empty`'s `setup/hidden-only/.DS_Store`, and adds `!/spec/conformance/**` to `.gitignore` so that no conformance fixture can be dropped again |

**The gate.** Every item reports AGENTS.md's whole gate green on go1.26.9
before merge:
- W4-0, W4-1, W4-2 and W4-6 ran on linux/amd64;
- W4-3, W4-4, W4-5 and W4-V1 ran on darwin/arm64.

There are two exceptions:
- **W4-6 merged over a red it did not cause.** It reported the red, and it is
  question 4's blocking finding.
- **W4-3, W4-4 and W4-5 did not run `go test -race ./internal/hostfs/...` on
  linux,** because their hosts were darwin. W4-0, W4-2 and W4-6 did.

**Divergences from the ask:**
- **W4-0.2 is narrowed by ruling R-W4-0.** For `StageNew`, a foreign file
  swapped in at a name the write created, inside the staging directory, is
  removed. This is a declared gap.
- **W4-1.1:** `--resolved`'s document is proved by a Go test, not a vector.
  The vector format compares only findings, trees, streams and listings.
- **W4-1.4:** `collection.item_name` and `input.not_readable` moved from W4
  to W7, because no vector could produce them (question 4, finding b).
- **W4-3.5 is partly unmet:** `input.raw_unavailable` is listed W5 with no
  vector, because only `embed`, which is Wave 5's, raises it.
- **W4-3.5:** `argument` maps to a new code, `template.tag_argument`.
- **W4-3.4:** `limit.exceeded`'s subject is settled with two new codes,
  `limit.exceeded.template` and `limit.exceeded.output`, inside ADR-006's
  registry model.
- **W4-4.8 is unmet:** the OOXML CPU-ratio flake did not reproduce on
  darwin/arm64, for the producer or the verifier.
- **W4-5.8 is fixed but not observed:** two mechanisms were found, pinned by
  tests and fixed, with the 5 s bound unchanged. Neither host reproduced the
  flake.
- **W4-6.2:** the server is the `internal/liquidspec` test binary run with
  `-serve`, not a `main` package, because R2-143's allow-list keeps
  `os.Stdin` and `os.Exit` to `cmd/metais`.
- **W4-6 touched outside its item where engine fixes forced it:**
  MTS-AUT-014's text and the `dynamic_reference` hint, and xref's dead
  branch.

### 2. What children flagged, degraded, declined or discovered

None of this was verified by me.

- **Wave 5's share of chapter 7 (W4-3):**
  - the `fan_out` and `item` members;
  - D7 item 10's table of contexts;
  - D9's escaping;
  - render-work's per-filter charge table.

  Declarations this wave does not render end with `internal.error`, by
  design, until Wave 5:
  - assets, mirrors and containers;
  - `when`, `for_each`, `escape: html` and `newline: crlf`;
  - `validate`;
  - Liquid paths;
  - `--out -`.
- **`check`'s Wave 5 share (W4-4):**
  - output validation schemas are not compiled;
  - container profiles are not checked;
  - the lowest minor is not computed, because 1.0 is the only minor.
- **`--fail-on warning` on render** is implemented but unobservable, because
  render emits no warning this wave.
- **The canaries' red runs** are recorded in ADR-009's log, but they land in
  the same squash commit as the release code that passes them. "Recorded
  first" holds within the PR's history only (W4-3).
- **Stricter rules that won, each named in its PR:**
  - W4-5 opens the package before refusing a directory operand
    (`usage.not_a_packed_package`), so that security vectors refuse
    identically through `unpack`. ADR-008 D9 item 1 alone would refuse
    earlier.
  - W4-2 adds `read`, `outline`, `grep` and `extract` to the opening set,
    beyond ADR-011 D16 item 3's list.
- **Readings taken where the ADRs are silent**, each recorded in the
  ADRs' implementation logs by its producer:
  - W4-1: refusals the ADRs leave to the command line get
    `usage.data_dir_unreadable` and `usage.conflicting_flags`.
  - W4-2: a pattern or glob is an operand, so it gets `usage.bad_glob` and
    `usage.bad_pattern`; `grep`'s regex-work exhaustion is
    `usage.bad_pattern` with reason `regex_work`.
  - W4-3: an existing empty `--out` is `output.destination.write_failed`,
    reason `exists`.
  - W4-4:
    - a package refused on read gets no `report.step_skipped`;
    - a step is skipped only when the manifest's model is absent;
    - it adds the codes `manifest.input.form_shape`,
      `manifest.input.collection_shape` and
      `output.declaration.validate_kind`;
    - `template.syntax` gains an always-present `cause`.
- **Conformance timing.** Measured on darwin/arm64:
  - `TestConformance` alone, under `-race`, took 31.4 s for the verifier, and
    20 to 47 s for the producers;
  - the whole `internal/conformance` package takes about 100 s, because
    `TestRealToolInRecord` and `TestDriverProtocol` re-run the suite.

  W4-1 and W4-2 read ADR-011 D11 item 9's 60 s as applying to the package.
  The verifier read it as the default run, as D11 item 9's text says, and
  found it within the bound.
- **Flakes:** none met in the wave apart from the two named above, which did
  not reproduce.
- **The liquid-spec departures are a prediction.** No Ruby was available,
  and W4-6 derived them by emulation: 2,394 run, 1,801 pass, 490 departures,
  103 held out for `date`. Until Wave 5 registers `date`, the verdict is
  `TestDelegatedRun`, not liquid-spec's exit status.

### 3. What carries to the next wave

- **For Wave 5's brief:**
  - `extract --overwrite` (Decision 2);
  - `unpack --dry-run` (cross-item finding X2);
  - `input.raw_unavailable`'s vector through `embed`;
  - `template.xref.unknown_input` and `template.xref.missing_asset`, now
    listed W5;
  - the chapter 7 and `check` remainders in question 2.

  Wave 5's existing destinations should settle one answer for an existing
  empty destination across `render` and `extract` (X1). Today `render` says
  `write_failed`/`exists` and `extract` says `not_empty`. Both are interim
  refusals that Wave 5 replaces.
- **Before Wave 5 is filed:** the liquid-spec act (question 5) must end. It
  will turn the 490 predicted departures into a measured result.
- **For the plan orchestrator, or Ed:**
  - **(b)** The W7 list is 18, not 16. #78 resolved the original 16, but
    `collection.item_name` and `input.not_readable` joined it in W4-1, and
    they need the same treatment. ADR-011 A4's `unreadable` setup, from
    #78, may make `input.not_readable` producible.
  - **X4** `init` has no conformance operation, so `init`'s
    `output.destination.write_failed` is proved only by Go tests. The code
    itself is produced by `aut/unpack-write-failed`.
  - **X5** Settle which run ADR-011 D11 item 9's 60 s binds. Producers read
    it two ways.
- **For Ed to ratify:**
  - ADR-011 A2, proposed (`unwritable-dirs`);
  - ruling R-W4-0;
  - Decisions 1 and 2, which are already recorded under plan §10;
  - the new codes and readings in question 2.
- **What I would do differently:**
  - **Look for shared infrastructure when deriving.** Three writers shared
    `hostfs`, and that cost one sanctioned edge set. The SC7 harness shared
    by two items worked as a meeting only because the owner was named.
  - **Check that each criterion is achievable inside the brief's scope**
    before filing. W4-3.5 asked for a vector that only Wave 5's `embed` can
    produce.
  - **Have the verifier run the gate on a fresh clone.** The red came from
    an ignored file, which a dirty working tree hides. W4-V1's `.gitignore`
    negation now prevents the case.

### 4. What the verification round found, and where every finding went

**Verifier:** `01a153e4-09bc-7799-9e9f-e904d0cb882d` (`wave-verification`),
on a clean worktree of `main` at `aee03df`, go1.26.9 darwin/arm64. Verdict:
**blocking**, with one blocking finding.
- **Every id I sent came back,** all 7 at depth **mutated**, with 31
  mutations in total.
- **It audited the range against `base_ref`.** All 7 merges are ancestors of
  `main`, and the four non-item commits then in the range are placed.
  Completeness of the item list is trusted, not verified: the verifier
  cannot see its siblings.
- **The guards that went RED under mutation, as they should:**
  - the five faults' controls (SC8, read-in-place, determinism,
    confinement) and A2's `UnorderedObjects` tripwire;
  - `hostfs`'s kind check, its `Remove`-not-`RemoveAll`, and its umask
    restore;
  - `--fail-on` on `check`;
  - supply order;
  - the `{field}` bound;
  - `read --lines` edges and the binary-to-terminal refusal;
  - `unpack`'s `.compiled/` exclusion, its Abort and its in-place report;
  - `render`'s outputs limit;
  - `bindings`' materialisation;
  - the departure-statement check;
  - the `date` hold-out.
- **Every meeting exists:**
  - the `path` canary's row, with its red run in ADR-009's log;
  - `rnd/destination-unwritable`;
  - `template.ParseEntry`, with `TestCheckAndRenderParseAlike`;
  - the opening set: 11 commands, green.
- **The gate:** RED in four tests, all on `rnd/destination-not-empty/hidden-only`.
  Everything else was green, including the six-target vet and build, and
  the spec-data drift check.

**The blocking finding.** It quotes the brief: "The gate, AGENTS.md's
Development list, run whole on go1.26.9 by every item before merge, and by
the verification round on the merged tree."
- The fixture `setup/hidden-only/.DS_Store` was ignored by `.gitignore`
  line 15, and never committed by W4-3.
- **Fix child:** W4-V1 `01a153fb-d8a6-7284-8be0-0f0bc4fada17`, #77
  (`d4ba3ba`). It reports the whole gate green on a fresh clone at its head.
  It is **not re-verified**: one round, hard stop.

**Non-blocking findings.** Each is listed with where it went. **The wave
completed without waiting for any child below, deliberately.**

| Finding (verifier's title, shortened) | Item | Where it went |
|---|---|---|
| (a) `CreateNew`'s exclusive Mkdir is unpinned in its race window (MkdirAll mutation GREEN) | W4-0 | child W4-NB1 `01a15414-2558-70cb-9443-095630a625eb`, member A |
| (b) Two W4 codes reclassified to W7, growing the design session's list | W4-1 | not filed as a child: it is a notification, not a change. The plan orchestrator owns the relation with design session `01a15348-4ab5`, which #78 has since ruled on for the original 16; question 3 relays it |
| (c) `browse`'s neutralisation of U+2066–U+2069 is untested | W4-2 | W4-NB1, member B |
| (d) W4-3.5's `input.raw_unavailable` vector is unmeetable before `embed` | W4-3 | not filed: the vector needs Wave 5's `embed`. The code is listed W5 in `unwired.txt`, so `TestFindingCodesCovered` makes Wave 5 produce it. Carried in question 3 |
| (e) MTS-TPL-028's `$dynamicRef` clause has no vector | W4-3 | W4-NB1, member C |
| (f) render's Abort after a failed create is unpinned | W4-3 | W4-NB1, member D |
| (g) `check`'s statics beneath `allOf`/`anyOf`/`oneOf`/`prefixItems` are untested | W4-4 | W4-NB1, member E |
| (h) The OOXML CPU-ratio flake did not reproduce | W4-4 | not filed: there is nothing to fix without a reproduction, and the producer recorded an observation on the test. It stays open as an observation, and whoever meets it next debugs it under AGENTS.md |
| (i) No self-proof wall-bound flake was seen | W4-5 | not filed: the fix landed, and the verifier's run neither confirms nor refutes it |
| (j) The 490 departures are a prediction until the run | W4-6 | the liquid-spec act, question 5 |
| X1 Three answers for an existing empty destination | W4-2/3/5 | not filed: both differing refusals are interim until Wave 5's existing destinations replace them. Carried to Wave 5's brief, question 3 |
| X2 `extract` ships `--dry-run`, while `unpack` deferred its own under the same plan line | W4-2/5 | not filed: a reading of plan §10's "dry run" that the plan orchestrator owns. Carried, question 3 |
| X3 `check`'s xref and render disagree on `xs["first"]` after W4-6 | W4-4/6 | child W4-NB2 `01a15414-8ee4-71c1-b5ec-15de3abbb36b` |
| X4 `init` has no vector operation for its `write_failed` | W4-5 | not filed: the code is produced by `aut/unpack-write-failed`, and an `init` op would be a chapter 15 change nothing asks for. Relayed, question 3 |
| X5 Conformance timing against ADR-011 D11 item 9 | all | not filed: within the bound under D11 item 9's text (31.4 s). Its reading is relayed, question 3 |

**Why five findings share W4-NB1.** Members A to E are one kind of work: pin
a guard that a mutation showed blind. They touch different packages
(`hostfs`, `browse`, a `tpl/` vector, `internal/cli`, `internal/author`), so
they cannot conflict, and one gate run verifies all of them. Its task marks
them severable. X3 is a behaviour change in `internal/xref`, so it has its
own child.

**Where the verifier found nothing:** W4-5 and W4-6 have no defect beyond
the unreproduced flake (i) and the predicted departures (j). On W4-1, every
mutation went red; its one finding (b) is bookkeeping.

### 5. Out-of-band acts

| Act | Type | Work Order | Placement |
|---|---|---|---|
| Run the pinned liquid-spec selection against the reference engine through the JSON-RPC server, and record the result (plan §6, ADR-011 D17) | `manual-qa-pass` | `01a15414-bf58-7436-8dc0-e1df81410d34` | **Carried, not waited on.** Its Before is Wave 5. It has `depends_on` W4-6 (`01a15356-b4cf-734b-a531-f1fe881b47a7`), as plan §10's acceptance asks. Its `build` is `d4ba3ba`. Its `focus` carries the exact commands from `spec/conformance/delegated/README.md`, the versions to record, and what to report |

It is open when this closeout merges. The plan orchestrator files Wave 5
only after it has ended.

### Rulings made during this wave

- **Decision 1 (the plan orchestrator, on the breakdown).** Exactly four
  intra-wave edges are sanctioned: W4-0 → W4-2, W4-3 and W4-5, and W4-3 →
  W4-6. A new need for one is still plan §2's slicing defect.
- **Decision 2 (the plan orchestrator).** `extract --overwrite` moves to
  Wave 5. This wave's `extract` writes only where the target does not exist,
  and does not declare `--overwrite`.
- **Chapters 9 and 11 (the plan orchestrator).** W4-3 writes, with vectors,
  whatever its render makes observable.
- **R-W4-0 (the wave orchestrator, answering W4-0).** For the `StageNew`
  shape, the ownership boundary is the staging directory: created
  exclusively under its fixed name, and held by the write's root handle.
  - It follows ADR-005 D14 item 1.5: "remove the staging directory and the
    ancestors created in step 1 (everything removed is the render's own)".
  - It is not `hostfs`'s general mechanism, which stays with design session
    `01a15284-df6e-7668-b799-62f4c380a84f`.
  - The declared residual gap: a foreign file swapped in at a name the write
    created, inside the staging directory, is removed.
  - In-place shapes remove nothing on failure.

  For the plan orchestrator's ratification.
