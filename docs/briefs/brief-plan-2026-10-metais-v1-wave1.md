# Brief — PLAN-2026-10-metais-v1, Wave 1: the seams

**Plan:** [PLAN-2026-10-metais-v1](../plans/PLAN-2026-10-metais-v1.md), §7
**Feature:** `metais-v1`. Work Orders are titled `metais-v1 [W1-<item>]: <action>`;
pull requests and their squash subjects are `<type>(metais-v1): [W1-<item>] <change>`
(AGENTS.md, Git and pull requests).
**Repository:** `kottera/metais`, branch `main`. Every Work Order of this wave
gets `repo: kottera/metais` and `branch: main`.
**Gated on:** Wave 0 (PRD-001, ADR-001 to ADR-011, RULINGS-v1 and the plan),
merged to `main` as of `49dcbea`. The gate has cleared.

## What this wave is for

When Wave 1 is done the module `github.com/kottera/metais` exists on Go
1.26.9, and everything every later item stands on is in place: one way to
report a finding, one reader of the environment, one table of limits, one
fault mechanism, one test environment, one version, the embedded copies of
the spec files the binary needs, the shared interfaces later waves are
written against, and tripwires that fail the build when code reaches for a
host fact or crosses a layer. `spec/` exists as a skeleton, with chapter 1's
front matter written. AGENTS.md states the extended gate, and that gate is
green on all six targets. No command beyond the empty shell does anything
yet.

## What to read first

- **The plan:** §2 (execution model, the rules carried below), §4 (safety
  discipline), §5 (what a sandbox cannot prove), §6 (out-of-band acts) and
  §7 (this wave: objective, carries, does-not-touch, scope outline,
  acceptance, decisions). §7 is the authoritative scope; this brief does not
  restate its outline, it binds it.
- **ADR-006** D1 to D5 and D8 to D13: the finding model, codes, registry,
  report, order, cap, encodings, the exit table, `internal/finding`,
  `internal/schemafail`, and D13 item 2's `Pos` and `Positions`.
- **ADR-009** D3 and D4 (the limits table and its lowering), D12, D14, and
  D16 item 3 (the static tripwire).
- **ADR-010** D1 (toolchain and module), D2 (layout and layering), D11 items
  8 to 10, D12 item 3 (the version string), D13 item 2 (the fault mechanism,
  with no faults yet) and D15 item 3 (the gate). D3 item 3 is the admitted
  dependency table — read it before adding any `require`.
- **ADR-011** D1 (the `spec/` tree), D2, and D5 items 1 and 7 (the front
  page and the changelog skeleton).
- **ADR-003** D8, **ADR-002** D4 and **ADR-010** D5 item 4, for what the
  data model's Go representation must be; **ADR-010** D11 item 1 for
  `pkgtree.Tree`.
- **RULINGS-v1**: the rulings listed under "Rulings already made" below, and
  "Resolved" item 51.
- **AGENTS.md**, whole. It is the gate and the conventions.

## What the previous wave learned that changes this one

Wave 0 was documents only and has no closeout. What is true of the
repository now, and differs from nothing the plan predicted:

- There is no Go code, no `go.mod`, no `spec/`, no `docs/briefs/` beyond this
  file, and no CI. AGENTS.md's Development section still says "There is no
  code yet" and lists the five-line gate; the item that creates `go.mod`
  updates it to ADR-010 D15 item 3's extended gate (plan §7, last scope
  bullet).
- The plan orchestrator's own dispatch on this worker fetched this
  repository, pushed a branch and opened and merged this brief's pull
  request with the workspace's credential and `gh`. That is evidence for,
  not a discharge of, §6's credential act below: the act also covers the
  Go toolchain fetch, `go mod download` and a cgo `go test -race`, which no
  one has yet run here.

## Rulings already made — do not reopen

From plan §7 "Decisions settled before filing", ratified with RULINGS-v1:

- R1-003: Go 1.26.9 and Unicode 15.0.0. `go.mod` is ADR-010 D1 item 1's
  three lines exactly.
- R1-007: `NO_COLOR` is the only environment variable read.
- R1-015: limits are exact and are never raised.
- R1-019: the exit table.
- R1-022: Apache-2.0 for the specification and the vectors.
- R2-079, R2-080, R2-082, R2-087, R2-088, R2-092: the finding model.
- R2-143: the tripwire's exact allow-list.
- R2-160: the version string.
- R2-162: faults by options injection.
- R2-163: a single-threaded render.
- R2-164: spec files embedded through copies.
- R2-165: `internal/hostfs` and `internal/hostenv.Isolate` are the single
  host seams.
- The binding constructor ADR-009 D13 calls `internal/bindings` lives in
  `internal/template`. ADR-010's layout owns package names.

Rulings made by the plan orchestrator for this wave:

- **The intra-wave edge on the module is sanctioned, and it is the only
  one.** §2 cuts waves so no item depends on a sibling, and §7 says the wave
  is narrow because every Go item needs `go.mod`. The wave orchestrator may
  therefore give the items that write Go a `depends_on` edge to the one item
  that creates the module (`go.mod`, `cmd/metais`'s shell, the licence-header
  check and the AGENTS.md gate text, or however it scopes that bootstrap).
  It may equally keep the Go work in fewer, larger items; prefer the split
  where each half is verifiable on its own. No other edge between Wave 1
  items is sanctioned; one that seems needed is §2's slicing defect, and the
  item parks and asks.
- **`internal/hostfs` itself is Wave 2's** (plan §8, "Host files"). Wave 1's
  "one path to the host's files" is held by rule, not by code:
  `TestNoHostFacts`'s allow-list names `internal/hostfs` and
  `internal/hostenv` as the only places host access may appear, so Wave 2's
  package lands into a seam that already exists. Wave 1 creates no
  file-access code outside what its own tests need through
  `internal/testenv`.

## Rules carried whole from the plan (§2 and §4)

- **The specification is written with the code, never after.** An item that
  makes a behaviour reachable from `metais` adds, in the same pull request,
  the `spec/` statements that state it, the vectors under
  `spec/conformance/vectors/` that cite them, and any finding codes it emits
  in `spec/findings/registry.json`. In this wave nothing beyond the empty
  shell is reachable, so no statement beyond chapter 1's front matter is
  written; the statement tooling and `TestCoverage` are Wave 2's.
- **A library is built ahead of its wiring; its statements land with its
  wiring.** The packages of this wave are proved by unit tests and write no
  statement.
- **A finding code is registered by the item whose code emits it.**
  `limit.invalid` is emitted by `internal/limits` in this wave and is
  registered here. The drift test and `TestFindingCodesCovered`, and the
  unwired-codes list, are settled in Wave 2.
- **Where two items of one wave meet, the one that merges second adds the
  test for where they meet**, after rebasing onto what landed first. The
  verification round checks every such meeting has one.
- **An item that finds it needs a sibling's unmerged work has found a
  defect in the plan's slicing.** It does not reach into the sibling's
  scope; it parks and asks.
- **Pin the property false first.** Each headline guarantee touched here —
  the import tripwire (`netleak` control), `TestNoHostFacts` (its
  `os.Getenv` fixture), the report's ordering (red against insertion order,
  ADR-006 D13 item 5) — has a test red against a tool that breaks it, and
  both detector and control stay in every `go test`.
- **Do not widen a refusal to make a proof pass**: no limit raised, no
  allow-list entry added to get a tripwire green. A proof that cannot pass
  without one is a finding for Ed, through the wave orchestrator.
- **The stricter security rule wins** where two ADRs disagree on what is
  accepted or refused; the item names the disagreement in its pull request
  body and the closeout records it.
- **No new dependency.** Only ADR-010 D3 item 3's modules, at their exact
  versions. Never copy code from another template engine, scaffolding tool
  or schema validator.
- **Bounded work is measured in CPU time**, never by wall time alone.
- **Every test is isolated** with a temporary `HOME` and temporary
  directories, through `internal/testenv` once it exists.
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

In scope: everything in plan §7's scope outline and nothing else.

Belongs to later waves and must not be absorbed here (plan §7 "Does not
touch", §8):

- every command beyond the shell — `internal/cli` holds `Main` and
  `Run(ctx, Invocation)` with an empty command table; `version`, `limits`,
  `help`, the argument parser and the envelope are Wave 2's;
- every parser (zip, JSON, YAML, CSV, Markdown, Liquid, JSON Schema) and
  `internal/unitext` with its Unicode-version tripwire (Wave 2);
- `internal/hostfs` and the source-directory or packed `pkgtree.Tree`
  (Wave 2) — Wave 1 lands the interface and an in-memory tree for tests;
- the statement parser, `spec/statements.json`, the conformance runner and
  every D14 document tripwire (Wave 2);
- every statement beyond chapter 1's front matter, and every vector;
- any fault constant beyond the `Set` type — faults arrive with their
  detectors from Wave 3.

## What must not break, and how it is checked

- **The gate**, run whole by every item before merge: AGENTS.md's list as
  ADR-010 D15 item 3 extends it — the five base checks, the static build of
  `./cmd/metais`, `go vet` and `go build` for each of the six targets, and
  the spec-data drift check. The item that creates `go.mod` writes this
  list into AGENTS.md; until it merges, the five base checks are the gate.
- **Plan §7's acceptance**, each criterion proved by a test in the tree:
  the six-target gate green; `TestNoHostFacts` flagging its `os.Getenv`
  fixture and the import rule flagging `internal/guarantees/testdata/netleak/`,
  both run by the tripwire's own test; the report's ordering test red
  against insertion order and green now, and the cap keeping exact totals
  under a lowered `findings` limit; `TestSpecDataMatchesSpec` failing on a
  one-byte change to a spec copy; `limits` lowering refusing a raise, an
  unknown name and a malformed value, each as `limit.invalid`; the tripwire
  failing on a non-test file outside `internal/fault` that names a fault
  constant, and on a non-test call to `version.Override`.
- **What a sandbox cannot prove** (plan §5): the darwin/linux operating
  system branches are compiled and vetted by the gate but run only on the
  host's own. Say so in the pull request where it applies; do not claim it.

## Out-of-band acts of this wave

From plan §6. An out-of-band act is work no dispatched session can do; the
wave orchestrator files each as an ordinary item of the off-tera type that
fits, and decides where it sits. This deployment declares the off-tera types
`ratify-plan`, `dogfood-round`, `live-deploy`, `darwin-measurement`,
`manual-qa-pass` and `design-session`; none is named for a generic person's
act, so pick the closest by its inputs — `manual-qa-pass` (`build`, `focus`)
fits both acts below, with `build` the current `main` commit and `focus` the
act's whole text and what to report back.

1. **Confirm the dev pipeline's GitHub credential reads and pushes
   `kottera/metais`, and that a dispatch can build here.**
   - **When:** Wave 1. **Who:** Ed. **Before:** every Wave 1 item.
   - **Notes:** Probe from a dispatch on the pipeline's worker: clone, push
     a branch, open and close a pull request. Then fetch `go1.26.9` through
     `GOTOOLCHAIN=auto` and `go mod download` the modules of ADR-010 D3.
     Finally run `go test -race` on a cgo package: the race detector needs
     cgo, and on linux a C compiler (ADR-010 D14 item 8). A failure names
     what to grant.
   - The plan's **Before** asks that every Wave 1 item wait on it, so each
     item takes a `depends_on` edge to it.
2. **Register `metais.org`.**
   - **When:** any time. **Who:** Ed. **Before:** the v1.0 release, which
     is after the train.
   - **Notes:** RULINGS R1-008. The schema `$id`s under
     `https://metais.org/schemas/` are permanent (ADR-002 D10 item 2,
     ADR-011 D7). If the domain cannot be had, the `$id`s change by an
     ADR-002 amendment before anything is published.
   - Filed in this wave so Ed is reminded early: this wave lands the first
     schemas carrying those `$id`s. Nothing in the train waits on it; it is
     carried, not waited on, and the plan orchestrator waits on it at the
     train's close.

## Closeout

Written by the wave orchestrator (Work Order
`01a150fe-fa51-7c77-9964-1d8149f5b5bc`) after the wave drained. It
reports what landed and what carries forward. **It is not a correctness
verdict.** I read the children's reports, the merge commits and selected
files (`go.mod`, `cmd/metais/main.go`, `spec/findings/registry.json`,
ADR-010's text and implementation log). I mutated nothing. Where a
guarantee is said to be proved below, the proof is the verification
round's, attributed to it, not mine.

**Wave range:** `87c9247..2f8eb42` on `main`. Eight children were approved
and released as one set: one out-of-band act and seven code items, all
complete. One verification round passed with no blocking finding.
Non-blocking findings went to three children, one act and one janitorial
issue, named under question 4. **The wave completed without waiting for
them, deliberately.** One person was consulted, at the breakdown review.

### 1. Did each item land, and does it match the ask?

| Item | Work Order | PR | Merge | Landed |
|---|---|---|---|---|
| W1-A1 credential and build probe | `01a15104-30cd-75c0-8b87-a70de3c65ea2` | — | — | Ended complete by Ed **without the probe being run** (see question 5) |
| W1-1 module, shell, extended gate | `01a15104-9f35-7a8f-ad6d-8cb4afcdd064` | #8 | `1474387` | `go.mod` is ADR-010 D1 item 1's three lines; `cmd/metais/main.go` is D2's one line (I read both). Also landed: `internal/cli`'s empty shell, `internal/hostenv`, `internal/fault`'s `Set`, `internal/testenv` (built on `testing.TB` methods only), `internal/specdata`'s copier and the licence-header check. AGENTS.md now states the extended gate |
| W1-2 spec skeleton | `01a15104-ed67-7abe-8459-6a7d071b62f4` | #7 | `eecac12` | `spec/` per ADR-011 D1: the status block, chapter 1's front matter with RFC 8174's text, chapters 02–15 as title, status and summary, the A/B/CHANGELOG/ERRATA files, and `.gitattributes`. It merged before `go.mod` existed, so no gate ran, and the PR says so |
| W1-3 finding model, registry, limits | `01a15105-4e64-7a19-9f2c-40be16d4241c` | #13 | `2f8eb42` | `internal/finding`, `internal/schemafail`, `internal/limits`; `registry.json`, which registers exactly `limit.invalid` and `report.truncated` (I read it); the two schemas under `spec/schemas/1.0/` and their specdata copies |
| W1-4 tripwires and notices | `01a15105-b618-7aa2-ab5c-3b7fd3810ea6` | #12 | `fdc218c` | `layers_test.go` over `layers.json`; `TestNoHostFacts` with R2-143's four-entry module allow-list and a 12-entry exact dependency allow-list for `x/sys/unix` and `x/term`; the fault and `version.Override` rules; `notices_test.go` with `THIRD-PARTY-NOTICES.txt` |
| W1-5 data model | `01a15106-0de2-74ed-bf00-34850a290ed3` | #10 | `ab7eebe` | A sealed six-kind `Value` in `internal/liquid/value`, per the review ruling below. Numbers keep their literal; objects keep unique names in UTF-16 code-unit order; the package is standard library only |
| W1-6 pkgtree | `01a15106-568d-7255-b024-b43eda6db5f3` | #11 | `97f8cd3` | `pkgtree.Tree{List, Stat, Open}` with its contract in doc comments; `pkgtreetest.New` (the in-memory tree) and `pkgtreetest.Check` (the reusable contract test) |
| W1-7 version | `01a15106-874b-740b-85ba-3e43a991baa4` | #9 | `4f20abe` | `version.String()` per ADR-010 D12 item 3, `Implements()` = `["1.0"]`, and the test-only `Override` |

**Divergences from the ask:**

- **W1-4 changed W1-1's `internal/hostenv` on windows.** The import rule
  found that `golang.org/x/sys/windows` imports `net`. W1-4 took the
  stricter rule, so windows `IsTerminal` now calls
  `syscall.GetConsoleMode` and `x/term` is linked on unix only. This
  departs from ADR-010 D3 item 3's table as written. The amendment is
  owed (W1-F2 below).
- **W1-3 added a test-only `require` of
  `santhosh-tekuri/jsonschema/v6` v6.0.3**, with `x/text` v0.43.0
  indirect. Both are admitted modules (ADR-010 D3 item 3), used to
  validate the registry against its schema. Neither is linked into
  the binary.
- **The empty shell exits 2 on every invocation** (W1-1's choice, within
  the limits its task set). Status 2 is ADR-006 D12's internal-error
  status. Wave 2's command table replaces this.

### 2. What children flagged, degraded, declined or discovered

- **Two ADRs disagree on windows dependencies** (W1-4). ADR-009 D16
  item 3 and ADR-010 D11 item 10 keep `net` out on every target.
  ADR-010 D3 item 3 and D11 item 3 put `x/term` and `x/sys` on windows,
  including `MoveFileExW` from `x/sys`. W1-4 recorded this in ADR-010's
  implementation log, and named `syscall.MoveFile` as the candidate for
  Wave 2's windows no-replace rename.
- **The MapOrder seam conflicts with the layering rule** (W1-4). ADR-010
  D13 item 2 puts the seam in `internal/liquid/value`, which D2 item 2
  keeps standard library only. Needs a ruling before Wave 4 (W1-F3).
- **TestNoHostFacts reads more strictly than the ADR lists**:
  - `time.LoadLocation` is refused everywhere;
  - the environment setters, `os.StartProcess`, the `syscall` process
    starters, `io/ioutil`, `filepath.Walk`/`WalkDir`/`Glob` and
    `runtime.GOARCH` are added to the lists;
  - the browse rule confines every `hostfs` reference to `Extract`.
- **`internal/tools/release` will need an allow-list ruling** when it
  lands in Wave 6 (W1-4).
- **The finding model's readings** (W1-3, each named in PR #13):
  - beyond the bounded heap, the report keeps a spill of findings past
    the cap that tie with the last listed one, so that ADR-006 D8
    items 3 and 6 stay exact;
  - the registry's span enum adds `none`;
  - limit units and classes are named for Wave 2's `limits` command;
  - ADR-006 D10 item 4 puts `summary` in the envelope, and ADR-008
    D10's envelope does not list it. Left for Wave 2;
  - `Report.Add` panics on duplicates positioned at different lines or
    columns.
- **The specdata copier cannot make the first copy of a new spec file**
  (W1-3). The package embeds the copy and does not build without it,
  so the first copy is made by hand.
- **W1-6's pkgtree has its own error types.** It avoids `io/fs` and
  `fmt`, which pull in `os`, `time` and `syscall`, and defines its own
  sentinel and `PathError`. Lookups do not check that a path is well
  formed; the portable-path predicate stays Wave 2's.
- **Not run in the sandbox (plan §5):**
  - the darwin, linux and windows branches were compiled and vetted
    for every target, and run only on darwin/arm64;
  - `IsTerminal`'s terminal-true branch is untested, because there was
    no pseudo-terminal.
- **Not every gate ran on the pinned toolchain.** W1-4's gate and the
  verifier's gate ran on go1.26.2 under the deployment's
  `GOTOOLCHAIN=local`. W1-5 and W1-6 report go1.26.9 through
  `GOTOOLCHAIN=auto`. Before merging this closeout I ran AGENTS.md's gate
  once over `2f8eb42` plus this document, on go1.26.9 darwin/arm64 with
  cgo for `-race`. It was green.

### 3. What carries to the next wave

- **Wave 2's `internal/hostfs` must not use `golang.org/x/sys/windows`**:
  `TestNoHostFacts` will refuse it. The windows no-replace rename needs
  a standard-library call that meets ADR-005 D14. W1-F2 writes the
  ADR-010 amendment; until Ed ratifies it, the implementation log entry
  and this paragraph are the record.
- **Wave 2's Liquid front end builds on `internal/liquid/value`**, which
  already holds the data model. It extends the package and does not
  re-create it (ruling (a) below).
- **Wave 2 settles the open points on findings and the command line**:
  - the envelope's `summary` member (ADR-006 D10 item 4 against
    ADR-008 D10);
  - the drift test, `TestFindingCodesCovered` and the unwired-codes
    list (`limit.invalid` and `report.truncated` are registered and
    have no vector);
  - the empty shell's exit 2.
- **Adding a spec copy:** a new embedded spec file is listed in
  `internal/specdata`'s `specFiles` and copied by hand the first time.
- **Wave 4's determinism item needs the MapOrder ruling** (W1-F3).
- **What I would do differently:**
  - **Name meeting-point tests in the items' criteria.** Two meeting
    points were left to "whichever merges second", and the second
    merger did neither: W1-2/W1-3's front-page links and W1-6/W1-4's
    `pkgtreetest` rule. A criterion makes the obligation checkable.
  - **Don't file an act as `manual-qa-pass` when the plan says
    "probe from a dispatch".** W1-A1's taker said so: the act would
    better have been an item a dispatch runs, so its evidence exists.

### 4. What the verification round found, and where every finding went

**Verifier:** `01a15161-1747-7320-a9e4-0d8fc939b83a` (wave-verification).
Its verdict is **pass**, and nothing blocks. It ran AGENTS.md's gate once
on the merged tree, green (go1.26.2, see above), and audited all seven
merge commits as ancestors of `main` against `base_ref`. It placed the one
unlisted commit, `7eea8b8`, as the plan orchestrator's documentation.
Every id I sent came back:

- **Mutated** (about 75 mutations, made in module code rather than in the
  producers' fixtures): W1-1, W1-3, W1-4, W1-5, W1-6 and W1-7. Every guard
  that carries a criterion went red, including plan §7's acceptance
  proofs:
  - the `os.Getenv` and `netleak` controls;
  - order red against insertion order;
  - exact totals under a lowered `findings` limit;
  - the one-byte specdata drift;
  - the three `limit.invalid` refusals;
  - the fault-constant and `version.Override` rules.
- **Inspected only:** W1-2 (documents), by script: the status block, the
  BCP 14 text, each chapter's opening and statement count, and a
  capitals-keyword scan of all of `spec/`.
- **Nothing found:** W1-6 and W1-7.

**Blocking findings:** none, so no fix child was filed.

**Non-blocking findings.** Each row is one finding. Four findings are
grouped in one child (W1-F1): each is a missing test in a different
package, one gate run verifies them all, and each is marked severable.

| Finding | From | Where it went |
|---|---|---|
| `cmd/metais/main.go`'s shape is not pinned (`os.Args = os.Args[:1]` before `Isolate` stays green) | W1-1 | W1-F1 `01a1516d-186d-7cbd-b0c3-4efd7170c565` |
| Exit's package-subject set is pinned only for `manifest` and `template` | W1-3 | W1-F1 `01a1516d-186d-7cbd-b0c3-4efd7170c565` |
| No layer rule keeps non-test code from importing `internal/pkgtree/pkgtreetest` (W1-6/W1-4 meeting point) | W1-4, cross-item | W1-F1 `01a1516d-186d-7cbd-b0c3-4efd7170c565` |
| `WrittenAsInteger`'s uppercase-`E` branch is untested | W1-5 | W1-F1 `01a1516d-186d-7cbd-b0c3-4efd7170c565` |
| Windows `IsTerminal` uses `syscall.GetConsoleMode`; ADR-010 D2/D3/D11 still say `x/term`/`x/sys` | W1-4 | W1-F2 `01a1516d-9896-7e8e-8329-608de55fce9a` (proposes the ADR-010 amendment; Ed ratifies it) |
| The MapOrder seam is in `internal/liquid/value`, which must stay standard library only | W1-4 | W1-F3 `01a1516d-c089-7157-8ab8-6ef30c2b325c` (`design-session`: a ruling for Ed, needed before Wave 4) |
| `spec/README.md` does not link `spec/findings/` or `spec/schemas/` (W1-2/W1-3 meeting point) | W1-2, cross-item | janitorial issue #14: prose only, informative, read by no gate |

**Not filed, with reasons:**

- **`version.String`'s no-build-info branch** cannot be reached from a
  test, because `debug.ReadBuildInfo` always succeeds in a module-built
  test binary. The verifier noted it as not a finding, since the code
  follows ADR-010 D12 item 3. I agree and filed nothing.
- **The import rule's scope.** One mutation stayed green because the
  import rule scans only `./cmd/metais`'s closure, which is what
  ADR-009 D16 item 3 specifies. Re-tested inside the closure, it went
  red. This is not a defect.
- **The "only importer" claims** in W1-4's allow-list reasons were
  re-checked on the merged tree and still hold. This is a
  confirmation, not a finding.

**No re-verification.** The W1-F children are not re-verified in this
wave. If any needs checking, that belongs in a later wave's brief.

### 5. Out-of-band acts

| Act | Type | Work Order | Placement | State |
|---|---|---|---|---|
| Confirm the pipeline's credential, the go1.26.9 fetch, the module download and cgo `-race` | `manual-qa-pass` | W1-A1 `01a15104-30cd-75c0-8b87-a70de3c65ea2` | Waited on by every item | Complete. Ed ended it on his own assurance, **with none of the three probe steps run**: no GOOS/GOARCH or `go version` was recorded. The evidence came afterwards, from the items: W1-5 and W1-6 fetched go1.26.9 through `GOTOOLCHAIN=auto` and ran `-race` with cgo; every item pushed and merged a PR |
| Register `metais.org` (R1-008) | `manual-qa-pass` | W1-A2 `01a1516d-d1ff-7124-a73a-1df0ca782993` | Carried, not waited on; filed at the wave's close as agreed in review | Open. The plan orchestrator waits on it at the train's close |
| Rule on the MapOrder seam | `design-session` | W1-F3 `01a1516d-c089-7157-8ab8-6ef30c2b325c` | Carried; filed for a verifier finding | Open. Needed before Wave 4 |

### Rulings made during this wave

Given or agreed at the breakdown review:

- **(a) The data model's home is `internal/liquid/value`.** ADR-010 D2's
  layout lists it as "the value model — ADR-003 D8, ADR-004", and D7
  item 1 says the readers produce "the same `value` tree". This governs
  W1-5.
- **(b) The dependency-scan allow-list.** Entries that ADR-009 D16
  item 3 and R2-143 themselves prescribe, such as the `x/term` and
  `x/sys` ones, are the mechanism and not a widening. Any entry neither
  names is forbidden. This governs W1-4.
- **(c) No allow-list entries for `internal/testenv` or the specdata
  copier.** They pass `TestNoHostFacts` without new entries. This
  governs W1-1.
- **(d) The finding model and the limits table stay one item.** This
  governs W1-3.

No child parked with a question during the wave.
