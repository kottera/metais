# Brief — PLAN-2026-10-metais-v1, Wave 3: opening a package, and the composed libraries

**Plan:** [PLAN-2026-10-metais-v1](../plans/PLAN-2026-10-metais-v1.md), §9
**Feature:** `metais-v1`. Work Orders are titled `metais-v1 [W3-<item>]: <action>`;
pull requests and their squash subjects are `<type>(metais-v1): [W3-<item>] <change>`
(AGENTS.md, Git and pull requests).
**Repository:** `kottera/metais`, branch `main`. Every Work Order of this wave
gets `repo: kottera/metais` and `branch: main`.
**Gated on:** Wave 2, complete. Its verdict is under plan §8, and its
closeout is in
[brief-plan-2026-10-metais-v1-wave2.md](brief-plan-2026-10-metais-v1-wave2.md).
The gate has cleared.

## What this wave is for

When Wave 3 is done, `metais inspect` opens a packed package or a source
directory, verifies all of it, and prints what a supplier reads first. It is
the first command a vector can drive, so the suite runs real vectors for the
first time, in both environments, and nothing is written anywhere on a
refusal. The container, manifest and version chapters (2, 3 and 4) exist in
full, with a vector for every statement. So do the parts of chapters 5, 12
and 14 that opening makes observable.

Beside it, every library the next wave wires is complete: the inputs layer,
the Liquid renderer and its standard filter catalog, the sample-value
generator, the cross-reference analysis and the OOXML split-run scanner.
`init` writes its scaffold.

## What to read first

- **The plan:** §2 (the execution model and the rules carried below), §4
  (safety discipline), §5, and §9 (this wave: objective, carries,
  does-not-touch, scope outline, acceptance, settled decisions). §9 is the
  authoritative scope. This brief binds it and does not restate its
  outline. Read the rulings and verdicts under §7 and §8 too.
- **Wave 2's closeout**, all of it. Questions 2 and 3 say what this wave
  inherits.
- **The ADR decisions §9 carries:**
  - ADR-001 D5 to D9;
  - ADR-002 D1, D9 and D11;
  - ADR-003 D1 to D7 at open, and D15 to D22;
  - ADR-004 D2 to D6, D8 item 6, and D10's standard catalog without
    `date`;
  - ADR-007 D1, D3, D4 and D9;
  - ADR-008 D2, D6, D14 and D17;
  - ADR-009 D3, D5, D6, D9 and D11;
  - ADR-010 D4 items 2 to 6, and D13 (`SkipDigest`);
  - ADR-011 D16, and D9 item 7 on committed bytes.
- **ADR-010's Amendments A1 to A3.** A2 (the MapOrder seam, which Ed ruled)
  binds the renderer's shape even though Wave 4 builds the fault. A1 and A3
  are proposed.
- **RULINGS-v1**: the rulings §9 lists.
- **AGENTS.md**, whole. Its Development section is the gate, and it now
  says a Work Order never runs `go test -fuzz`.

Read the code you compose. That is everything Wave 2 left in `internal/`,
especially:

- `hostfs.OpenSource` and its `SourceOptions.Names`;
- `zipfmt` and the packed tree;
- `jsonx`, `yamlx`, `inputs/csv` and `markdown`;
- `schema`, with its bundling, walk and coercion;
- `schemafail.Map` and `manifest`;
- `liquid`, with its dialect interfaces;
- `internal/conformance` and its README;
- the document tripwires in `internal/guarantees`;
- `unwired.txt`.

## What Wave 2 learned that changes this one

- **The loader composes what Wave 2 left as parameters:**
  - W2-4's portable-path predicate is passed as
    `hostfs.SourceOptions.Names`, so both trees check names under the one
    rule;
  - ADR-001 D8 steps 6 and 7;
  - ADR-009 D6 item 3;
  - whether a manifest path names an entry;
  - the Liquid parse of `path`, `when` and `for_each`;
  - the `template.*` and `input.*` findings the front end and the dates
    library left for wiring.
- **The notices file is now the wiring wave's job** (Wave 2 ruling R4).
  `THIRD-PARTY-NOTICES.txt` lists exactly what `./cmd/metais` links
  (ADR-010 D12 item 6). The first item whose wiring links jcs, yaml.v3,
  jsonschema/v6 or x/text into the binary names that module there, or
  `TestThirdPartyNotices` goes red.
- **The renderer owes 97 deferred Golden Liquid cases.**
  `internal/liquid/testdata/golden-liquid/render-time.txt` lists invalid
  cases that ADR-004 refuses at render, each naming its ADR-004 item. They
  pass in this wave or carry a closed-set exclusion. The arity lines are
  removed when the catalog's `Check` refuses literal arity at parse; the
  test forces it.
- **`unwired.txt` lists every refused, content and destination code Wave 2
  registered.** A code this wave makes producible by a vector leaves the
  list in the same pull request, because the test fails on a listed code a
  vector produces. Codes registered here name the wave expected to wire
  them.
- **The `hostfs` fix #40 has not run on linux**, and Wave 2's one linux
  defect was found only because two producers happened to land there. No
  brief can choose a dispatch's host. So any item, or the verification
  round, that lands on linux runs `go test -race ./internal/hostfs/...` and
  reports the result. If none does, the closeout says so, and the check
  carries to Wave 6's matrix rehearsal.
- **`hostfs`'s open-handle cost** (one held handle per uncommitted entry,
  since #40) is awaiting a design session for Ed before Wave 4.
  `01a15284-df6e-7668-b799-62f4c380a84f` carries it. This wave's loader
  only reads. It must not build on, or change, `hostfs`'s write side.
- **Acceptance criteria are checked against the guards' own rules before
  filing.** Copy the brief's and the ADRs' wording rather than
  paraphrasing it: two of Wave 2's criteria contradicted an ADR, and one
  paraphrase cost two parks.
- **The `date` filter is Wave 5's, not this wave's.** It is registered by
  `internal/template` through `Extend` (plan §8). §9's standard catalog is
  everything but `date`. (Wave 2's brief said Wave 3 wires the
  registration; §11 is right, and that line was wrong.)

## Rulings already made — do not reopen

From plan §9, "Decisions settled before filing", ratified with RULINGS-v1:

- R1-009: the media type is named in chapter 2; IANA registration comes
  later.
- R1-011: no caller path is printed.
- R1-013: the whole package is verified at every open.
- R1-017: strict templates.
- R1-018: version compatibility.
- R1-024: empty CSV cells.
- R2-004, R2-005, R2-016 and R2-111: the container and the canonical form.
- R2-020 to R2-037: the data specification.
- R2-044 to R2-047, R2-051 and R2-063: values, order and globals.
- R2-094 to R2-098, R2-106 to R2-109 and R2-124: authoring.
- R2-135: the references index.
- R2-150 and R2-178: the hostile packages.

Earlier waves' rulings stand. They are recorded under plan §7 and §8, and
in the Wave 1 and Wave 2 closeouts:

- the data model in `internal/liquid/value`;
- the dependency-scan reading;
- the envelope's `summary`;
- the Markdown reading (A3);
- the package-home constraint;
- R1, R2 and R4.

Rulings made by the plan orchestrator for this wave, on the questions Wave 2
left open:

- **`json.duplicate_name` keeps its `name` parameter.** ADR-006 D9 item 4
  forbids a string *value* from the inputs in any message or parameter. A
  member name is not a value. ADR-006 already repeats member names in
  `params.property`, and every location pointer carries them. The
  neutralisation of R2-091 applies to the name as to any parameter. No
  registry change.
- **Building findings must not be superlinear in hostile input.** Wave 2
  found that a finding costs O(pointer length), so many duplicates under a
  long name make building them quadratic. This wave makes hostile input
  reachable through `inspect` and the inputs layer. So the path from
  parsing to the report must stay bounded in CPU time on such an input,
  proved by a CPU-time test, without raising or bypassing a limit. Whichever
  item first makes that input reachable owns the proof. Where the cost
  lives in `internal/finding` or `internal/jsonx`, that item fixes it there.
- **A file the tool writes must be readable by the tool.** The canonical
  form (`jsonx.Canonical`, and the compiled files that the open re-derives
  and compares) and the `yaml` and `csv` writers the sample generator uses
  emit only text their own readers accept.
  - Wave 2 found two `yaml` writer gaps that break this: U+FFFE and U+FFFF
    are not escaped, and an integral number between 2^53−1 and 1e21 is
    written as an integer literal the reader refuses. The item that first
    uses the writer fixes both.
  - ADR-003 D8 item 5 refuses large integers by spelling, so `11e17` is
    accepted while its canonical text would be refused. If that makes a
    package the tool produces, or one whose compiled files it re-derives,
    unopenable, the item parks and asks with options. Do not choose how
    widely the refusal reaches on your own: whether it extends past the
    ADR's wording is an accept/refuse change to a data rule.
- **`limit.exceeded`'s single subject is not this wave's to settle.** This
  wave emits it only for inputs, which is the subject it is registered
  with. Wave 4's render and outputs are where ADR-009 D19's other uses
  first appear.
- **The breadth of the loader item.** If the loader and `inspect` cannot be
  one pull request, §9 already rules what may move: the breadth of ADR-001's
  and ADR-002's *Conformance* lists, beyond one vector per statement, moves
  to Wave 6's corpus items. The statements, and a vector for each, may not
  move. The wave orchestrator may cut the loader into several items, but
  §2's rule holds: no item depends on another item of this wave. A cut that
  needs an edge is the slicing defect, and the wave orchestrator parks and
  asks me before filing it.

## Rules carried whole from the plan (§2 and §4)

- **The specification is written with the code, never after.** An item
  that makes a behaviour reachable from `metais` adds, in the same pull
  request:
  - the `spec/` statements that state it;
  - the vectors under `spec/conformance/vectors/` that cite them;
  - any finding codes it emits, in `spec/findings/registry.json`.

  `TestCoverage` makes a statement without a vector a red gate. Where the
  tool and the text disagree, the tool is wrong. This is the first wave
  where that rule bites, and the suite runs each vector in both of ADR-011
  D12's environments.
- **A library is built ahead of its wiring, and its statements land with
  its wiring.** The inputs layer, renderer, filter catalog, sample
  generator, cross-reference and scanner are proved by unit tests, fuzz
  seed corpora and third-party corpora, and write no statement. Wave 4
  wires them.
- **A finding code is registered by the item whose code emits it.** A code
  not yet producible by a vector goes on the unwired list with the wave
  expected to wire it.
- **Where two items of one wave meet, the one that merges second adds the
  vector (or test) for where they meet**, after rebasing onto what landed
  first. Name each meeting in the criteria of the item expected to merge
  second, or of both. The verification round checks that every meeting has
  one.
- **An item that finds it needs a sibling's unmerged work has found a
  defect in the plan's slicing.** It does not reach into the sibling's
  scope. It parks and asks.
- **Pin the property false first.** These must be red against a tool that
  breaks them, and stay in every `go test`:
  - **on a refusal, nothing is written anywhere** (ADR-011 D10 item 4's
    check, from this wave);
  - **the digest detector, with `SkipDigest` as its control**: the control
    lets a digest-mismatch package through and the detector catches it;
  - **the hostile corpus, refused by every package-opening command**,
    starting with `inspect`;
  - **the "every byte read once" budget on open** (ADR-010 D15 item 2).
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
  names the disagreement in its pull request body, and the closeout records
  it.
- **No new dependency.** Only ADR-010 D3 item 3's modules at their exact
  versions, as A1 and A3 amend that table. goldmark is gone. Third-party
  corpora are vendored only from their real sources, with licence and
  commit (ADR-010 D3 item 6, D14 item 4). Never copy code from another
  template engine, scaffolding tool or schema validator.
- **Bounded work is measured in CPU time**, never by wall time alone.
- **Every test is isolated** with a temporary `HOME` and temporary
  directories, through `internal/testenv`.
- **A Work Order never runs `go test -fuzz`** (AGENTS.md). A fuzz target is
  proved by its seed corpus in plain `go test`.
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

In scope: everything in plan §9's scope outline and nothing else.

These belong to later waves and must not be absorbed here (plan §9 "Does
not touch", §10, §11):

- `validate`, `ls`, `read`, `outline`, `grep`, `schema`, `extract`,
  `render`, `check` and `unpack` (Wave 4). The inputs layer, renderer,
  catalog, generator, cross-reference and scanner are built as libraries
  here and wired there;
- the template environment of a render, the confinement detector
  (`SkipConfine`), the determinism detector (`MapOrder` and
  `Env.UnorderedObjects`, per A2), the browse detector, the SC8
  planted-errors test with `FirstFinding`, and the Liquid closure with its
  JSON-RPC server (Wave 4);
- the METAIS filters and tags, `date`, `embed`, `asset` and `provenance`
  (Wave 5);
- `pack`, `init`'s pin through `check`, `test` and `pack`, the hostile
  corpus in full, and the release builder (Wave 6);
- any change to `hostfs`'s write side, pending design session
  `01a15284-df6e-7668-b799-62f4c380a84f`.

## What must not break, and how it is checked

- **The gate**, AGENTS.md's Development list, run whole on go1.26.9 by
  every item before merge, and by the verification round on the merged
  tree. Each PR records the `go version` and the host's GOOS/GOARCH it ran
  on.
- **Plan §9's acceptance**, each criterion proved by a test or vector in
  the tree:
  - the suite runs its first vectors in both environments, with nothing
    written anywhere on a refusal;
  - `con/`, `man/` and `ver/` vectors exist for every statement of
    chapters 2 to 4;
  - at least one committed hostile package per `package.container.*`,
    `package.name.*` and `package.digest.*` code is refused by `inspect`;
  - `SkipDigest` lets a digest-mismatch package through and the detector
    catches it;
  - `inspect` on a fixture's source directory and on its committed package
    gives the same result apart from `source`;
  - the "every byte read once" budget holds on open;
  - Golden Liquid's core-tag render cases pass or carry a closed-set
    exclusion, and so do the 97 deferred render-time cases;
  - the inputs layer reports every planted error of an SC8-shaped fixture
    in one call;
  - `init`'s output equals ADR-007 D1 item 4 byte for byte;
  - the unwired list names the wave for each code registered here.
- **The conformance suite stays within ADR-011 D11 item 9's 60 seconds**
  under `go test -race`, now that it has vectors.
- **Every earlier guard stays green:**
  - `TestNoHostFacts`, with every new package in `layers.json` and no
    dependency-scan entry beyond what ADR-009 D16 item 3 and R2-143
    prescribe (an entry that is needed and not prescribed is a park);
  - the layering table, licence header and notices check;
  - the spec-data drift check, with new spec copies made by hand the first
    time;
  - the document tripwires;
  - the runner's self-proof.
- **What a sandbox cannot prove** (plan §5): operating-system branches run
  only on the host's own; SC5 is §6's matrix act. Each PR claims nothing
  it did not run.

## Out-of-band acts of this wave

None. Plan §6 names no act whose **When** is Wave 3. The plan orchestrator
tracks the open acts the train carries: W1-A2, register `metais.org`; and
the `hostfs` design session `01a15284-df6e-7668-b799-62f4c380a84f`.

## Closeout

The wave orchestrator wrote this closeout (Work Order
`01a15288-f803-7a12-b629-faf7943dbdcc`) after the wave drained. It reports
what landed and what carries forward. **It is not a correctness verdict.**

What I checked, and how:
- I read every child's report, its PR, and the merge commits on `main`.
- I read the verifier's full report.
- I read these parts of the merged tree:
  - the chapter files and the vector directories;
  - `unwired.txt`;
  - `THIRD-PARTY-NOTICES.txt`;
  - MTS-VER-007 and its new note;
  - plan §9's recorded rulings.
- I ran no test, no gate and no mutation. Where a guarantee is said below to
  be proved, the proof is the producer's or the verifier's, and it is
  attributed to them, not to me.

**Wave range:** `bfdb183..ff8debe` on `main`.
- **The breakdown:** nine implementation items, approved by the plan
  orchestrator with two edits and released as one set with no
  `depends_on` edges.
- **Three corrective items**, filed on rulings made during the wave. Two of
  them depend on the items they follow.
- **One verification round**, which found one blocking defect. It got one fix
  child, which merged and is not re-verified.
- **Non-blocking findings** went to two children and one janitorial issue
  (question 4). **The wave completed without waiting for them,
  deliberately.**
- The range also holds three commits that are not items: #44 (W2-NB1,
  Wave 2's carried child) and #45 and #54 (the plan orchestrator's
  recorded rulings).
- **No person was consulted.** The breakdown review and both escalations went
  to the plan orchestrator.

### 1. Did each item land, and does it match the ask?

| Item | Work Order | PR | Merge | Landed |
|---|---|---|---|---|
| W3-1 loader and `inspect` | `01a1528f-747c-7ef0-841c-9b1622e3015f` | #53 | `53f25ce` | `internal/loader`, `inspect` with `--guide` and `next`, digests, step 7's re-derivation of the three compiled files, `SkipDigest` and its detector, the committed-package program and its reproduction test. Spec chapters 02–04 are written, with the parts of 05, 12 and 14 that opening makes observable. The first vectors run: `pkg/`, `man/`, `ver/`, `dat/` and `sec/`. `THIRD-PARTY-NOTICES.txt` now names jcs, yaml.v3, jsonschema/v6 and x/text |
| W3-2 inputs layer | `01a1528f-ea22-7d75-866c-3137634174df` | #50 | `70cbb4f` | `internal/inputs`, a library: the five forms, collections, the data directory (read side only), the supply arguments, the input record and D22's order. 23 codes are registered as W4, and the W3 input-subject codes are relabelled W4 |
| W3-3 renderer | `01a15290-5cb2-7480-953b-fb7e120f0829` | #48 | `594e96b` | `Render`, `Eval` and `EvalCondition`, every ADR-004 D2 tag, the meter, panic recovery, A2's single iteration seam and `FuzzLiquidRender`. It merged second against W3-4, so it ran the full Golden Liquid set |
| W3-4 filter catalog | `01a15291-21fd-7da0-a351-c58df9dc1292` | #46 | `e157688` | `liquid.Core()`: every standard filter but `date`, plus `escape`'s alias `h`. `Check` refuses literal arity at parse, which forced 73 lines out of `render-time.txt`. It also adds the mechanical `date` selector |
| W3-5 sample generator | `01a15291-b3f9-7528-9a10-35fbba74618d` | #49 | `341b883` | `internal/sample`: SplitMix64, the value preference, the optional modes and pattern generation. `schema.sample_unsatisfiable` is registered. Both `yaml` writer gaps are fixed |
| W3-6 cross-reference | `01a15292-16d0-701b-89f3-097041e09a4f` | #47 | `ac6fbb2` | `internal/xref`: ADR-007 D3 items 1–13, its own filter-signature table, the budgets, and 14 `template.xref.*` codes listed W4 |
| W3-7 OOXML scanner | `01a15292-75b2-700f-b3a2-95dc8fd05148` | #52 | `341b63b` | `internal/author/ooxml`: a hand-written lenient tokenizer over `liquid.Constructs`' spans, and both `template.ooxml.*` codes listed W4 |
| W3-8 `init` | `01a15293-0dd0-71dc-a9a0-5583300703c4` | #51 | `57dffee` | `metais init` in `internal/author`: the scaffold of ADR-007 D1 item 4 as golden bytes, the empty-directory rule and `--dry-run`. It uses `hostfs`'s write side unchanged |
| W3-9 finding cost | `01a15293-7fdf-7451-851e-ccf93a367cdc` | #55 | `e135f00` | `finding.Path`, a token-structured pointer, so building, ordering and placing findings is linear on duplicate-heavy hostile input. Reports are byte-identical. `Report.Add`'s panic on three or more repeats is fixed |
| W3-5F large-integer JSON | `01a152da-c9d0-779d-972c-8f584025a810` | #56 | `220e723` | `jsonx.EncodeInput`: sample JSON spells an integral number beyond ±(2^53−1) with an exponent. `Encode` and `Canonical` are unchanged |
| W3-1F stated rule, PKG | `01a152db-087e-76ae-9a9e-c1e2affb2c86` | #57 | `9f928ce` | MTS-MAN-023 with three `man/` vectors (`maximum: 11e17` opens), and ADR-011 Amendment A1, proposed (area `PKG`) |
| W3-9F validation path | `01a152e0-c4e8-78b6-a974-06dfcd7b3138` | #58 | `5ba29e6` | `schemafail.Target.Locate` takes a `*finding.Path`, so the schema-validation path to the report is linear through the inputs layer and the loader |
| W3-1V proof note | `01a15332-3140-7c31-b7cb-b0dc296924ad` | #59 | `ff8debe` | The blocking fix: MTS-VER-007's proof note, and the tripwire `TestProofNotes`, which is red on the chapter as it stood |

Each item ran AGENTS.md's whole gate on go1.26.9, on darwin/arm64 or
linux/amd64, and each PR names which. Every item reports it green, with two
exceptions:
- W3-1's final run had one red step that the change did not cause: the
  `internal/author/ooxml` CPU-ratio test, which passed alone (question 2).
- W3-8 ran the gate on `82d4bb8`, not on its merged head, which added only
  a plan-document commit.

The verifier ran the gate once on the merged tree at `5ba29e6`, on go1.26.9
linux/amd64: green.

**Divergences from the ask:**

- **W3-1.2 reads `pkg/`, not `con/`.** `con` is a Windows device name that
  ADR-011 D9 item 8 refuses, so the stricter rule renamed chapter 2's area
  `PKG`. The plan orchestrator accepted it. W3-1F wrote ADR-011 A1, proposed.
- **W3-1.14 is narrowed.** By the plan orchestrator's ruling (b), the
  canonical form must be readable for numbers within ±(2^53−1). Beyond that
  range, MTS-MAN-023 states that RFC 8785's integer spelling is not accepted
  as input. W3-1 had merged with the case declared as a gap, and W3-1F
  replaced the gap with that stated rule.
- **The brief's "97 deferred render-time cases" reads as 95** (ruling Q1).
  The 2 `date` cases stay in `render-time.txt` for Wave 5. The verifier
  checked the full count and it matches W3-3's report:
  - 1,098 cases: 10 selected out, 1,088 run, 913 pass and 175 excluded;
  - of the 95 deferred cases, 22 are refused at render and 73 at parse.
- **The 175 exclusions are all `dialect` lines.** 29 date from Wave 2, 87
  are W3-4's filter lines and 59 are W3-3's. The verifier checked that every
  new line names an ADR-004 item. That is many lines, though each is within
  the brief's one permitted kind. Each W3-4 line names its ADR-004 item: 80
  under D6 item 1, 5 under D4 item 4, 1 under D4 item 5, 1 under D6 item 6.
- **W3-9.2 was met in two steps.** #55 fixed the duplicate-name path and
  declared the schema-validation path open, because fixing it needed
  W3-1's then-unmerged `Target.Locate`. W3-9F (#58) closed it after W3-1
  merged.
- **W3-8.2's "suggested valid name" is given only for `--name`.** A
  suggestion derived from `<dir>` would print part of the caller's path, and
  R1-011 forbids that. The PR names the disagreement, and the verifier calls
  the reading legitimate.

### 2. What children flagged, degraded, declined or discovered

None of this was verified by me. Where the verifier confirmed something, it
says so.

- **Not run where the sandbox could not reach:**
  - `go test -race ./internal/hostfs/...` passed on linux in W3-7, W3-5F,
    W3-1F, W3-9F and W3-1V, and in the verifier's run. One skip each time:
    `TestDeviceInADirRefusedUnopened` cannot open `/dev`. **Wave 2's #40
    has now run on linux.**
  - The windows branches were compiled and vetted only.
- **The gate flakes under load on linux**, outside the merged-tree run:
  - in W3-1's final run, `internal/author/ooxml`'s CPU-ratio test failed in
    the full `-race` run and passed alone;
  - in W3-5F's run, three `internal/conformance` self-proof rows hit their
    5 s wall bounds under the full parallel `-race`;
  - the verifier's merged-tree gate was green.
- **Toolchain:** dispatches default to `GOTOOLCHAIN=local` (go1.26.2). Every
  item and the verifier ran the gate with `GOTOOLCHAIN=auto` to reach
  go1.26.9.
- **ADR disagreements, each named in its PR:**
  - W3-2: ADR-003 D11 item 9 and ADR-006 D7 item 4 on partial CSV
    validation. D7 item 4 applies to an input's own file, D11 item 9 to a
    collection item.
  - W3-6: ADR-007 D3's edit distance 2 and ADR-006 D9 item 7's
    max(1, ⌊n/3⌋) for suggestions. ADR-007 was followed.
  - W3-8: ADR-008 D9's suggested name and R1-011, as above.
  - W3-9: ADR-003 D9 item 2 ("every occurrence reported") and ADR-006 D8
    item 3's de-duplication key, which holds no line or column, so repeats
    of one name are one finding. It is a reporting disagreement, not an
    accept/refuse one, so plan §4 does not settle it. D8 was implemented
    as written.
- **Readings taken where the ADRs are silent:**
  - W3-2: ADR-006 D4 has no subject for the inputs object as a whole, so
    `input.constraints_failed` and `input.json_arg_invalid` are about the
    invocation.
  - W3-3: a wrong-type loop limit/offset reports engine code `argument`,
    for `internal/template` to map.
  - W3-3: the renderer charges units ADR-009's table omits (range elements,
    array writes, string size), and counts partial entry as `WorkPartials`.
  - W3-4 is stricter than the Ruby gem in four places: radix and
    leading-zero integer strings, integer bit-indexing, mixed `sort` keys,
    and `size` of a number.
  - W3-7 scans a partial rendered under several bindings once.
- **Residual costs, out of reach of this wave:**
  - jsonschema v6.0.3's own `Compile` is quadratic on a schema with many
    subschemas under one long member name: 15 s at 256 KiB (W3-9F). Fixing
    it means changing the dependency. It is recorded as an observation.
  - A CSV input with one very long header name hashes it once per record
    (W3-9F).
  - OOXML prefix propagation can be quadratic in the template count, not in
    one template's size (W3-7).
- **Messages over the bound:** `{field}` messages hold the whole pointer, so
  on a long pointer they exceed ADR-006 D9 item 1's 240 code points (W3-9).
- **One latent panic:** one `Report.Add` panic remains, in a combination the
  producer says no reader produces. The verifier probed for it on the merged
  tree, after W3-9F, and found it unreachable.
- **Codes no vector can produce:** 14 `manifest.*` keyword codes the 1.0
  manifest schema cannot produce, plus `package.not_found` and
  `package.not_readable`. They are listed W7 in `unwired.txt`. Plan §8
  makes such a code a finding for Ed.
- **A runner measurement problem:** on linux, `ru_maxrss` appears to include
  the parent's peak, so plain non-race `go test ./internal/conformance` trips
  the 32 MiB bound. Filed (question 4).
- **The unwired list now stands at:** W4 106, W5 33, W6 10, W7 16. No W3
  label remains.

### 3. What carries to the next wave

- **Wave 4 wires what this wave built as libraries:** the inputs layer
  (`validate`, `render`), the renderer and `Core()` (through
  `internal/template`), the cross-reference and OOXML scanner (`check`). Its
  brief should name the libraries' entry points as landed:
  - `inputs.New` with the `Supply` methods;
  - `liquid.Render` with `Env{Dialect, Globals, Partials, Escape, Meter,
    Limits}`;
  - `xref`, and `ooxml.Scan`.
- **For Wave 4's brief:**
  - **The runner's unwritable-destination pre-state**, behind a `requires`
    probe. The plan orchestrator undertook to put it in the brief. Without
    it, `output.destination.write_failed` (W3-8) and `commit_failed` become
    findings for Ed.
  - **The `argument` engine code** (W3-3) needs its ADR-006 mapping in
    `internal/template`.
  - **`input.raw_unavailable`** was left to the render (W3-2).
  - **`internal/template` registers `date` through `Extend` in Wave 5.** The
    Golden Liquid selector then fails by design, and the 10 `date` cases come
    back.
- **For the plan orchestrator, a ruling or Ed's:**
  - ADR-003 D9 item 2 against ADR-006 D8 item 3 on repeated names;
  - ADR-006 D4's missing subject for the inputs object;
  - the 16 W7 codes no vector can produce;
  - `{field}` messages over 240 code points;
  - ADR-007 D3 against ADR-006 D9 item 7 on suggestion distance;
  - jsonschema's quadratic `Compile`, which needs a dependency change.
- **For Ed to ratify at the train's close:**
  - ADR-011 A1 (`PKG`);
  - plan §9's recorded rulings: Q1 `date`, the finding-cost owner, and the
    large-integer rule;
  - this wave's rulings below.
- **Still running when the wave closed:** W3-NB1, W3-NB2 and issue #60
  (question 4), and design session `01a15284-df6e-7668-b799-62f4c380a84f`,
  which must decide `hostfs`'s handle cost before Wave 4's render.
- **What I would do differently:**
  - **Check a breakdown's numbers against the corpus before filing.** I found
    the 2 `date` cases among the 97 only by counting, after drafting. A brief
    criterion with a count deserves a count of its own before review.
  - **Check vector area codes against D9 item 8's portable-name rule when
    deriving.** `con/` was in my own W3-1.2, copied faithfully from a brief
    that took it from ADR-011.
  - **Where a ruling hands a fix to "whichever item first", name the owner
    at breakdown.** That worked here, and it would have worked better if the
    large-integer question had been put the same way. W3-1 and W3-5 each
    met it separately, from opposite sides, two waits apart.

### 4. What the verification round found, and where every finding went

**Verifier:** `01a1530b-2692-7cec-9006-98b254b51edb` (`wave-verification`),
on `main` at `5ba29e6`, a full clone, on linux/amd64. Verdict: **blocking**,
with one blocking finding.
- **Every id I sent came back**, all 12 at depth **mutated**: about 150
  mutations, plus spot-checks of the producers' claimed-red rows.
- **It audited the range against `base_ref`.** All 12 merges are on `main`.
  The other three commits are placed (#44, #45, #54). `internal/hostfs` is
  unchanged across the wave.
- **Every meeting test exists and goes red when its guard is broken:**
  - `TestInputsOverTheLoader`;
  - `TestInspectOpensTheScaffold`;
  - `man/expression-filters`. It has no `when` case, because
    `ParseCondition` refuses a filter in a condition;
  - `TestGoldenLiquidRender`, checked by full count;
  - `TestFilterTableMeetsCore`;
  - `TestSampleDataValidates`;
  - #55's CPU tests through `inspect` and the inputs layer.
- **Its gate run:** the whole gate was green on go1.26.9, and the
  conformance suite took 25 s alone under `-race`, within 60 s.

**What it mutated, per item.** Each mutation went red unless the row says
otherwise.
- **W3-1:**
  - dropped each compiled-file compare;
  - made `VerifyDigests` always skip, and made it ignore `SkipDigest`;
  - disabled the package-digest compare;
  - made `inspect` write a file on a refusal (76 subtests red);
  - re-read an entry before step 7 (17 entries decoded 18 times);
  - disabled the NFC check;
  - replaced `Core()` in the loader;
  - wrote via `io/ioutil` on a refusal: `TestRefusalWritesNothing` went red,
    `TestOnlyReads` stayed **green** (finding (a));
  - appended a byte to a committed package (the reproduction test).
- **W3-2:**
  - reordered precedence;
  - dropped partial validation;
  - kept only the first finding;
  - referenced `hostfs`'s writer;
  - disabled D22 item 4's skip: stayed **green** (finding (b)).
- **W3-3:**
  - the date selector by name prefix;
  - re-added an arity line;
  - the `and`/`or` charge;
  - the meter's limit edge;
  - partial-depth off by one;
  - a Go map range;
  - a non-standard import;
  - disabled panic recovery;
  - object output;
  - the float exponent edge;
  - `time.Now()` in the meter: stayed **green** (finding (c)).
- **W3-4:**
  - the keyword checks;
  - `Core()` registering `date`;
  - dropping `h`;
  - the literal checks;
  - `join` building before charging;
  - charging after the work, and dropping the string-key `sort` charge:
    both stayed **green** (finding (d)).
- **W3-5:**
  - each `yaml` escape;
  - the large-number thresholds;
  - SplitMix64's shift;
  - a pattern draw;
  - a reason code;
  - a stray byte in sample JSON;
  - `safeMax` = 2^53: stayed **green** (finding (f)).
- **W3-6:**
  - the budget edges;
  - the edit distance;
  - property-name arguments;
  - dropping `h`;
  - a parser import;
  - `first` given `last`'s signature, and the case fold disabled: both
    stayed **green** (finding (g)).
- **W3-7:**
  - each namespace;
  - two causes;
  - markup-only constructs;
  - entity decoding;
  - two quadratic reintroductions;
  - a manifest import;
  - dropping `&gt;`, and ignoring root shadowing: both stayed **green**
    (finding (h)).
- **W3-8:**
  - the quiet list;
  - the listing cap;
  - `--dry-run` writing;
  - file modes;
  - `os.Getwd`;
  - `..` as a name;
  - removal on a failed write;
  - a golden byte;
  - file order;
  - a suggestion that leaks a path.
- **W3-9:**
  - the old production code restored (every CPU test red);
  - O(pointer) reintroduced in `duplicateKey`;
  - the path digest dropped;
  - `ChildToken`'s parent ignored.
- **W3-5F:**
  - `jsonx.Encode` restored for sample JSON;
  - a non-shortest exponent;
  - the CSV guard disabled.
- **W3-1F:**
  - the stored schema re-parsed and refused on a finding;
  - `jsonx`'s unsafe-integer refusal disabled;
  - the source vector hidden.
- **W3-9F:**
  - the old production code restored (both CPU tests red);
  - `PathCache` never sharing;
  - `SpanParent` misplaced.

**The items where the verifier found nothing:** W3-5F and W3-1F.

**The blocking finding, and its fix child:**

| Finding | Quoted commitment | Fix |
|---|---|---|
| MTS-VER-007 (`spec/04-versions.md:55`, from W3-1) is a `(not observable)` MUST on a processor with no proof note after it. No tripwire checked the rule | ADR-011 D3 item 5, a design document: "A `not observable` requirement on a processor is followed by a note naming how it is proved for the reference tool" | W3-1V `01a15332-3140-7c31-b7cb-b0dc296924ad`, #59 `ff8debe`. The note names `metais version --json`'s `specification.implements`, `TestVersion` and `TestImplements`. The new tripwire `TestProofNotes` is red against the chapter as it stood. **Not re-verified.** |

**Non-blocking findings.** Each row is one finding. NB1 groups eight: each is
a test that pins a guard, or one ADR line, in a different package, and one
gate run verifies them all. NB2 groups two that are both in
`internal/conformance`. Both children are told their members are severable.
**The wave completed without waiting for them, deliberately.** They start
beside the next wave rather than in front of it.

| Finding | From | Where it went |
|---|---|---|
| (a) `TestOnlyReads` refuses only an `os` import; an `io/ioutil` write stays green | W3-1 | W3-NB1 `01a15339-99c9-7ad5-8cc6-ed15d3ab3b71` |
| (b) D22 item 4's constraints-not-evaluated skip has no behavioural test | W3-2 | W3-NB1 `01a15339-99c9-7ad5-8cc6-ed15d3ab3b71` |
| (c) No clock guard on `meter.go`, `render.go` and `eval.go` | W3-3 | W3-NB1 `01a15339-99c9-7ad5-8cc6-ed15d3ab3b71` |
| (d) Apply's charge order, and `sort`'s string-byte charge, are unpinned | W3-4 | W3-NB1 `01a15339-99c9-7ad5-8cc6-ed15d3ab3b71` |
| (e) `Core()` holds `h`, which ADR-010 D4 item 2's list omits. It is an ADR stating something false about what shipped, so it gets a child, not the backlog | W3-4 | W3-NB1 `01a15339-99c9-7ad5-8cc6-ed15d3ab3b71` |
| (f) The generator's ±(2^53−1) clamp is unpinned at its boundary | W3-5 | W3-NB1 `01a15339-99c9-7ad5-8cc6-ed15d3ab3b71` |
| (g) `first` and `last` are never told apart; nearest's case fold is untested | W3-6 | W3-NB1 `01a15339-99c9-7ad5-8cc6-ed15d3ab3b71` |
| (h) The `&gt;` entity and transitive root shadowing are untested | W3-7 | W3-NB1 `01a15339-99c9-7ad5-8cc6-ed15d3ab3b71` |
| (i) The runner's nothing-written check cannot see the host temporary directory, because `Isolate` clears `TMPDIR` | W3-1, the runner's | W3-NB2 `01a15339-e49f-757a-b579-8991efd5d775` |
| (j) Plain non-race `go test ./internal/conformance` trips the 32 MiB RSS bound on linux | cross-item | W3-NB2 `01a15339-e49f-757a-b579-8991efd5d775` |
| (k) A stale comment in `internal/cli/opening_test.go` says the opening set is empty | W3-1 | Janitorial issue [#60](https://github.com/kottera/metais/issues/60): prose-only, in a test comment no gate reads |

A first NB2 draft, `01a15339-cbc8-7c6b-9d0d-8c1ff06efbb3`, was cancelled
before release because I filed it into the wrong cohort. It never ran.

**Findings not filed, with the reason:**
- **`unwired.txt` cannot reach empty by Wave 7** (W3-1). This is the 16 W7
  codes of question 2. It is a ruling for Ed under plan §8, not work, and it
  is carried in question 3.
- **No suggested name for a name taken from `<dir>`** (W3-8). The verifier
  calls it legitimate: it is R1-011, the stricter rule. Nothing changes.
- **The remaining `Report.Add` panic is not reachable from any reader**
  (W3-9). The verifier confirmed the producer's claim on the merged tree. It
  is recorded in question 2.
- **W3-9F.2's test drives `loader.Open`, not the command** (W3-9F). That is
  the call `inspect` makes, and the verifier accepts it.
- **"Composition checks found no drift"** is the verifier's cross-item
  all-clear. There is nothing to file.

### 5. Out-of-band acts

The brief named none, and none was filed. The open acts the train carries,
W1-A2 (`metais.org`) and design session
`01a15284-df6e-7668-b799-62f4c380a84f` (the `hostfs` handle cost), are the
plan orchestrator's.

### Rulings made during this wave

**The plan orchestrator's, recorded under plan §9:**
- **Q1, `date`:** Golden Liquid cases that apply `date` are out of scope,
  selected out mechanically by a selector that fails once `date` is
  registered. 95 of the 97 deferred cases run.
- **Q2:** W3-3 and W3-4 stay split. Whoever merges second runs the full set,
  checked by full count.
- **One owner for the finding-cost fix** (W3-9). The path proofs are
  meetings.
- **Large integers:** (b) plus B′.
  - ADR-003 D8 item 5 is unchanged.
  - The canonical-form rule holds within ±(2^53−1).
  - MTS-MAN-023 states the rest (W3-1F), and sample JSON uses an exponent
    (W3-5F).
- **`PKG` for chapter 2's area**, with ADR-011 A1, proposed.

**Mine, each given to the child it governs and named in its PR:**
- **R-W3-5 (option A).** The generator never makes an integral number beyond
  ±(2^53−1), and it refused to save a supplied one.
  - **Grounds:** the brief's "A file the tool writes must be readable by the
    tool", and its "Do not choose how widely the refusal reaches on your
    own".
  - **Governs:** W3-5. Its JSON half was superseded by B′ in W3-5F.
- **R-W3-8 (A).** `output.destination.write_failed` stays registered and
  listed W4.
  - **Grounds:** plan §8 makes the list temporary. A runner pre-state behind
    a `requires` probe, ADR-011 D9 item 6's permitted skip, can produce the
    code.
  - **Governs:** W3-8.
- **R-W3-1.** #53 merges with the large-integer case declared as a gap, while
  the reach of the rule went up for a ruling.
  - **Grounds:** the brief reserves that choice.
  - **Governs:** W3-1. Superseded by W3-1F.
- **R-W3-9 (A).** #55 merges with the schema-validation path as a declared
  gap, and W3-9F owns it after W3-1. `Report.Add`'s panic is fixed under
  ADR-006 D8 item 3 as written.
  - **Grounds:** the brief's rule on needing a sibling's unmerged work.
    A crash is wrong under any reading.
  - **Governs:** W3-9 and W3-9F.
