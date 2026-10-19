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

*Not yet written. The wave orchestrator replaces this line with the wave's
closeout.*
