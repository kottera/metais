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

*Not yet written. The wave orchestrator replaces this line with the wave's
closeout.*
