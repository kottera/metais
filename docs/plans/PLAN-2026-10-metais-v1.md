# PLAN-2026-10: METAIS v1 — the specification and its reference tool, built together, one dependency layer per wave

**Author:** drafted in the 2026-10-18 design session with Ed Funnekotter
**Date:** 2026-10-18
**Feature:** `metais-v1`. Every brief, Work Order and pull request of this
train inherits it verbatim: `metais-v1 [W2-C3]: <action>` for a Work Order,
and `<type>(metais-v1): [W2-C3] <change>` for a pull request and its squash
subject, by `kottera/kottera`'s `docs/dev/work-order-and-pr-naming.md`, which
AGENTS.md adopts.
**Status:** Ratified 2026-10-18 under Ed Funnekotter's direction to merge this
plan and start its train.
**For PRD:** [PRD-001](../prd/PRD-001-metais-v1.md) R1 to R18, and the success
criteria SC1 to SC9 of its §26.
**For ADR:** [ADR-001](../adr/ADR-001-the-container-and-its-layout.md) to
[ADR-011](../adr/ADR-011-the-specification-document-and-conformance.md), and
[RULINGS-v1](../adr/RULINGS-v1.md), whose every lean, Tier 1 and Tier 2, is
ratified.
**Repository:** `kottera/metais`, branch `main`. The train writes this
repository only.
**Depends on:** nothing. The plan orchestrator is filed with no Work Order
dependencies; Wave 1's items wait on §6's credential act, filed as a
person's act ahead of them.

## 1. Purpose

There is no code yet. When the train is done, METAIS is a specification and a
tool that implements it, proved against each other.

**Done means:**

- `metais` builds as one static binary for darwin and linux on amd64 and
  arm64, and compiles and vets for windows on amd64 and arm64 (ADR-010 D12).
  The gate in AGENTS.md, as ADR-010 D15 item 3 extends it, is green on `main`.
- Every command of ADR-008 D1 works end to end, on a packed package and on a
  source directory, through the golden paths of R17: consuming (`inspect`,
  `read`, `schema`, `validate`, `render`) and authoring (`init`, `render
  --sample`, `check`, `test`, `pack`).
- `spec/` holds every chapter of ADR-011 D1, with every normative sentence an
  `MTS-` statement. Every observable requirement has a vector, and the
  reference tool passes every vector in both of ADR-011 D12's environments.
  `spec/statements.json` and the finding catalog are generated and current,
  and every registered finding code is proved.
- The hostile corpus of ADR-009 D17, with ADR-001's container cases and
  ADR-005's path cases, is refused in full with exact findings, and nothing
  is written anywhere (SC6, on the host).
- The SC7 harness runs over every vector and every hostile case, with every
  canary of ADR-009 D16 item 2 caught and the release build clean.
- Every headline guarantee of ADR-010 D13 has its detector and its fault
  control, both running in every `go test`.
- The packages of PRD §4.1 to §4.5 live under `examples/`, each checking,
  testing and packing green, and each previewing with sample data, every
  optional field filled and none, into output that validates (SC4). The
  §4.2 package proves SC2 and the §4.3 package proves SC3.
- The release builder of ADR-010 D12 is ready. The release itself, the
  four-target matrix and the live rounds are §6's acts.

## 2. Execution model

**Waves run one at a time; the items inside a wave run in parallel, and each
item is one pull request.** The waves are cut so that **no item depends on
another item of its own wave**: each wave adds one layer of the dependency
graph, and everything an item builds on has merged in an earlier wave.

The plan orchestrator authors each wave's brief late, after reading the
previous wave's closeout, as
`docs/briefs/brief-plan-2026-10-metais-v1-waveN.md`, where AGENTS.md places
briefs. A brief carries goals, constraints and rulings, never a breakdown.
`wave-orchestrator-derive` derives the items from the brief and the code.
This plan holds no brief and no item list.

AGENTS.md is the repository's conventions file, and every session reads it
first. This train adds none.

**Each brief carries its wave's ADR decisions by number, says which it does
not touch, and carries these rules whole.**

- **The specification is written with the code, never after.** An item that
  makes a behaviour reachable from `metais` adds, in the same pull request:
  - the `spec/` statements that state the behaviour;
  - the vectors under `spec/conformance/vectors/` that cite them;
  - any finding codes it emits, in `spec/findings/registry.json`.

  `TestCoverage` (ADR-011 D14) makes a statement without a vector a red gate,
  so the two cannot be split across pull requests. Where the tool and the
  text disagree, the tool is wrong (AGENTS.md).
- **A library is built ahead of its wiring; its statements land with its
  wiring.** The early waves build internal packages that no command reaches
  yet. Such a package is proved by its unit tests, fuzz targets and
  third-party corpora, and it writes no statement, because nothing a vector
  can drive reaches it. The item that wires it into a command writes the
  statements and vectors for everything that wiring makes reachable.
- **A finding code is registered by the item whose code emits it.** Two
  tests on the registry need settling, and Wave 2's decisions (§8) settle
  them:
  - the drift test, which fails on a code literal missing from the registry
    (ADR-006 D13 item 4);
  - `TestFindingCodesCovered`, which fails on a registered code that no vector
    produces (ADR-011 D14).

  A library that is not yet wired still registers its codes, in the unwired
  list of Wave 2's decisions.
- **Where two items of one wave meet, the one that merges second adds the
  vector for where they meet.** Two examples: an asset declared inside a
  container, and provenance under `test`'s fixed identity. An item rebases
  onto what landed before it, so it can see the meeting point. The wave's
  verification round checks that every such meeting has its vector.
- **An item that finds it needs a sibling's unmerged work has found a defect
  in this plan's slicing.** It does not reach into the sibling's scope. It
  parks and asks its wave orchestrator.
- **Shared seams are named at breakdown** (Ed, 2026-10-19). The wave
  orchestrator names the shared seams its items will touch: a file, a
  generated artefact or a package that several items would edit. Items that
  touch one seam either run in sequence, with `depends_on`, or are split by
  phase rather than by feature.
- **Statement numbers are allotted at breakdown** (Ed, 2026-10-19). The wave
  orchestrator gives each item a block of statement numbers in each area it
  will add to, in the item's task. Numbers an item does not use stay unused.

### Harness policy — Claude throughout

| Role | Work Order type |
|---|---|
| Plan orchestration | `plan-orchestrator-claude` |
| Wave derivation and supervision | `wave-orchestrator-derive` |
| Implementation and corrective work | `code-implementation-direct` |
| Whole-wave verification | `wave-verification` |

These types are the dev pipeline's, registered in the `plugin.dev-pipeline.v1`
vault. Name a type by its full key wherever its leaf is ambiguous. Every
Work Order gets `repo: kottera/metais` and `branch: main`. **Never substitute
another harness's type to get past a capacity wait.** Park on
`awaiting-capacity` and let the seat come back.

## 3. Overview and sequencing

| Wave | Objective | Depends on | Status |
|---|---|---|---|
| 0 | PRD-001, ADR-001 to ADR-011, RULINGS-v1 and this plan, merged to `main` | — | **authored** |
| 1 | **The seams**: the module on Go 1.26.9; `cmd/metais` and the host seams; the finding model and registry; the limits table; the fault, test-environment, version and spec-data packages; the shared interfaces later waves build against; the guarantee tripwires; the `spec/` skeleton | 0 | **complete** — [verdict](#7-wave-1--the-seams) |
| 2 | **The leaf libraries and the proof harness**: the command-line core with `version`, `limits` and `help`; the conformance runner and its self-proof; the document tripwires; zip, Unicode and the portable-path predicate; host files; JSON; YAML; CSV and Markdown; the JSON Schema compiler with the pattern profile; failure mapping, bundling and the static walk; the tar writer; dates; the Liquid front end; the manifest library | 1 | **complete** — [verdict](#8-wave-2--the-leaf-libraries-and-the-proof-harness) |
| 3 | **Opening a package, and the composed libraries**: the package loader and `inspect`, with the container, manifest and version chapters and their vectors; the inputs layer; the Liquid renderer; the standard filter catalog; the sample-value generator; the cross-reference analysis; the OOXML split-run scanner; `init` | 2 | **complete** — [verdict](#9-wave-3--opening-a-package-and-the-composed-libraries) |
| 4 | **Validate, read, the first render, check and unpack**: `validate`; `ls`, `read`, `outline`, `grep`, `schema` and `extract`; `render` of template outputs into a new directory with the template environment; `check`; `unpack`; the Liquid closure | 3 | **complete** — [verdict](#10-wave-4--validate-read-the-first-render-check-and-unpack) |
| 5 | **The render's capabilities, and the rest of authoring**: the METAIS filters and `date`; escaping; `embed` and `provenance`; fan-out, conditions and path templates; assets and the mirror; zip containers and their profiles; CRLF and output validation; existing destinations; dry run, the standard-output archive and `--only`; `sample`; `render --sample`; `test` | 4 | **complete** — [verdict](#11-wave-5--the-renders-capabilities-and-the-rest-of-authoring) |
| 5b | **The gate's cost and merge friction** (inserted by Ed, 2026-10-19): the two-tier gate and its scripts; the shared-host cap; the skills; the `hostfs` limit tests; self-merging generated files; per-item ADR logs; one edit per finding code; help refusals by family; `render.go` split by phase | 5 | **complete** — [verdict](#11b-wave-5b--the-gates-cost-and-merge-friction-inserted-by-ed-2026-10-19) |
| 6 | **Pack, and hardening**: `pack` with the example-package harness; `test`'s container comparison; the hostile corpus in full; the SC7 harness's full reach; limit calibration; benchmark budgets; the release builder | 5b | **complete**, release builder carried to Wave 7 — [verdict](#12-wave-6--pack-and-hardening) |
| 7 | **The example packages and the close-out**: the packages of PRD §4.1 to §4.5; the worked package of `B-worked-package.md`; the specification's 1.0 text complete; the README, documentation and ADR implementation logs | 6 | not started |
| — | **The out-of-band acts**: §6 | — | — |

Two chains set the wave count:

- The chain from the JSON Schema compiler to the manifest to the package
  loader puts the first observable command in Wave 3.
- The chain from `check` and `test` to `pack` puts `pack` in Wave 6.

The Liquid engine runs beside the package chain, from Wave 2 to Wave 4, and
never holds it up.

## 4. Safety discipline (always on)

- **The gate is AGENTS.md's.** There is no CI, and the list in AGENTS.md is
  the gate. Wave 1 extends that list as ADR-010 D15 item 3 says:
  - the static build;
  - `go vet` and `go build` for each of the six targets;
  - the spec-data drift check.

  Every item runs the whole gate before merge. `go test -race ./...` runs the
  conformance suite in both environments, and ADR-011 D11 item 9 bounds that
  run at 60 seconds.
- **Pin the property false first.** Each headline property has a test that is
  red against a tool that breaks it, landed before or with the code that makes
  it true. Where ADR-010 D13 gives the property a fault, both the detector and
  the fault control stay in every `go test`. The properties:
  - **No write outside the output directory.** The confinement detector, with
    `SkipConfine` as its control (Wave 4). On a refusal, nothing is written
    anywhere: ADR-011 D10 item 4's check, from Wave 3.
  - **No host data reaches a template, an output or a finding (SC7).** The
    sentinel environment and A/B differential come in Wave 2. Each ADR-009
    D16 item 2 canary lands, red, with the channel it probes, in Wave 4 and
    Wave 5. The red run is recorded in ADR-009's implementation log before
    the code that makes the release build pass lands (ADR-011 D12 item 6).
  - **Byte-identical outputs within one build.** The determinism detector,
    with `MapOrder` as its control (Wave 4), and the cross-platform record
    (Wave 2, used at release).
  - **Refusal of every hostile package.** The digest detector, with
    `SkipDigest` as its control (Wave 3), and the hostile corpus, refused by
    every package-opening command (Wave 3 onward, in full in Wave 6).
  - **Every finding in one pass.** The SC8 planted-errors test, with
    `FirstFinding` as its control (Wave 4).
  - **No network, no execution.** The import tripwire, with the `netleak`
    fixture as its control (Wave 1).
  - **Read in place.** The browse detector, with `UnpackToTemp` as its control
    (Wave 4).
- **Do not widen a refusal to make a proof pass.** None of these happens to
  get a proof green:
  - no verification step is skipped (R1-013), and no limit is raised (R1-015);
  - no format or keyword is let through (R1-016);
  - the portable-path predicate is not relaxed (R1-023);
  - no line is added to a Golden Liquid exclusion list except a `dialect`
    line naming its ADR-004 item (ADR-010 D4 item 7);
  - no skip is added to the conformance runner except a `requires` probe
    (ADR-011 D9 item 6);
  - no released vector's expectation is changed.

  A proof that cannot pass without one of these is a finding for Ed.
- **The stricter security rule wins.** Where two ADRs, or an ADR and the
  specification text, disagree on a rule that accepts or refuses, the item
  implements the one that refuses more. It names the disagreement in its pull
  request body, and the close-out records it. This is RULINGS-v1's own
  convention.
- **No new dependency.** The admitted modules are ADR-010 D3 item 3's table,
  at its exact versions. Third-party test corpora are vendored only as
  ADR-010 D3 item 6 and D14 item 4 allow. Anything else needs an ADR-010
  amendment, ratified first. Never copy code from another template engine,
  scaffolding tool or schema validator (AGENTS.md).
- **Bounded work is measured in CPU time**, by counters or `getrusage`, never
  by wall time alone (AGENTS.md, ADR-010 D14 item 7).
- **Isolate every test with a temporary `HOME` and temporary directories**,
  through `internal/testenv`. No test reads the developer's configuration.
- **Never write to another repository.** A change `kottera/kottera` needs is
  filed there as its own work. METAIS names, imports and requires nothing of
  tera.

## 5. What this train cannot prove in a sandbox

- **Byte identity across platforms (SC5).** A dispatch runs on one host, one
  operating system and one architecture. The suite proves identity within
  that host's build. Identity across darwin and linux, and across amd64 and
  arm64, needs ADR-011 D15's record taken on a real host of each target and
  compared. darwin/amd64 runs under Rosetta 2 on Apple silicon. This is §6's
  matrix act.
- **The operating-system code the dispatch's host does not run.** The
  no-replace rename has a build-tagged branch per operating system:
  `renameat2(RENAME_NOREPLACE)` on linux, `renamex_np(RENAME_EXCL)` on darwin
  (ADR-010 D11 item 3). Other darwin-only facts are the case-insensitive
  default volume and `ru_maxrss` counted in bytes. The gate compiles and vets
  every target, but runs only the host's branch. The other host's branch is
  run by the matrix act.
- **Shopify's liquid-spec.** It needs Ruby, Bundler and the network
  (ADR-011 D17 item 3). The JSON-RPC server it drives is built and unit-tested
  in Wave 4, and the run itself is a §6 act.
- **Absolute CPU budgets and limit calibration.** ADR-010 D15 item 2's budgets
  are stated on reference hardware, and ADR-009 D18 item 4 calibrates per
  supported target. A contended sandbox gives numbers but not evidence. The
  benchmarks and the calibration suite are written in Wave 6 and run on real
  hosts by §6's act.
- **`govulncheck`.** It needs the network, and ADR-010 D3 item 5 makes it a
  developer step before each release, never a test.
- **The live rounds.** SC1 needs an AI harness, a real product and only the
  binary. SC9 needs an author with no access to the tool's source. Neither
  is a test.
- **The pseudo-terminal proof of ADR-008 D16 item 6**, where the sandbox
  offers no pseudo-terminal device. The item says so in its pull request, and
  the matrix act runs it.

## 6. Out-of-band acts

These are acts no dispatched session can perform. The wave orchestrator files
each as the off-tera Work Order type the deployment declares for a person's
act, places it where the **Before** column says, and gives the items that
need it a `depends_on` edge to it.

| Act | When | Who | Before | Notes |
|---|---|---|---|---|
| Confirm the dev pipeline's GitHub credential reads and pushes `kottera/metais`, a repository created on 2026-10-18, and that a dispatch can build here | Wave 1 | Ed | every Wave 1 item | Probe from a dispatch on the pipeline's worker: clone, push a branch, open and close a pull request. Then fetch `go1.26.9` through `GOTOOLCHAIN=auto` and `go mod download` the modules of ADR-010 D3. Finally run `go test -race` on a cgo package: the race detector needs cgo, and on linux a C compiler (ADR-010 D14 item 8). A failure names what to grant. |
| Register `metais.org` | any time | Ed | the v1.0 release | RULINGS R1-008. The schema `$id`s under `https://metais.org/schemas/` are permanent (ADR-002 D10 item 2, ADR-011 D7). If the domain cannot be had, the `$id`s change by an ADR-002 amendment before anything is published. |
| Run the pinned liquid-spec selection against the reference engine through the Wave 4 JSON-RPC server, and record the result | Wave 4, after the Liquid closure merges | Ed, or a host with Ruby and Bundler | Wave 5 | ADR-011 D17. Each departure must fail as listed, and one that passes fails the run. Run it again at every engine change, at every change to chapters 7 or 8, and at release. |
| Run the four-target matrix: the suite with `-conformance.record` on darwin/arm64, darwin/amd64 under Rosetta 2, linux/arm64 and linux/amd64, then `-conformance.compare` | Wave 6, as a rehearsal of the procedure; again at release | Ed | the v1.0 release | ADR-011 D15 and SC5. darwin/arm64 against linux/amd64 is the minimum pair. The rehearsal also runs the OS branches and the pseudo-terminal proof of §5. |
| Run the limit-calibration suite and the benchmarks on each supported target | Wave 6, after both items merge | Ed | the v1.0 release | ADR-009 D3 item 3 and D18 item 4, and ADR-010 D15 item 2. A default whose worst case breaks a ceiling is lowered only by an ADR-009 amendment Ed ratifies (R1-015). Budgets go in ADR-010's implementation log. |
| Check each example package's source material against its publisher's reuse terms | Wave 7 | Ed | that example's merge | PRD §25. ITI's VPAT is ITI's, so §4.1 is written from W3C's criterion text and GSA's OpenACR catalogue and schema, or in this project's own words. §4.2 uses the EU's DORA texts, Implementing Regulation (EU) 2025/302 Annex II, under the EU's reuse policy. Record each source and its terms in the package's `references/`. |
| SC1 round: an AI harness given only the binary and the §4.1 package produces a valid ACR for a real product | Wave 7, after the §4.1 package merges | Ed | the v1.0 release | Record what the harness read, which must be nothing beyond what `inspect` and the commands it points to show. A finding comes back as a fix item. |
| SC9 round: an author working only from `spec/`, with no access to the tool's source, writes a package that `check`, `test` and `pack` accept cleanly | Wave 7, after the worked package merges | Ed | the v1.0 release | `B-worked-package.md` is where the author starts (ADR-011 D4 item 4). |
| `govulncheck ./...` | before the release | Ed | the v1.0 release | ADR-010 D3 item 5. |
| The v1.0 release: the specification first, then the tool | after the train | Ed | — | Specification: one commit that dates the `1.0` changelog section, sets the front page to `Released`, writes `spec/conformance/released/1.0.sha256` and records the matrix's pass, tagged `spec-1.0` (ADR-011 D5 item 2). Tool: `go run ./internal/tools/release -version 1.0.0` on the tag, built twice and compared (ADR-010 D12), tagged `v1.0.0`. The specification is released no later than the tool that implements it (ADR-011 D5 item 6). Where it is published is outside ADR-010; the repository is private. |
| Ratify the outcome | after the train | Ed | — | Mark the plan `Done`. Ratify any decision this train made under §7 to §13 that is not yet an ADR text: the code exemptions of Wave 2, the placement of `date`, ADR-010 Amendment A1 (`golang.org/x/sys` and `golang.org/x/term` on unix only, W1-F2), and any calibration amendment. A decision the train changed is superseded by a new ADR or an amendment. |

## 7. Wave 1 — the seams

**Objective.** The module exists, and everything every later item stands on is
in place: one way to report a finding, one reader of the environment, one
path to the host's files, one table of limits, and tripwires that fail the
build when code reaches for a host fact.

**Carries:**
- PRD R6 item 3 (the finding model's encodings) and R14 item 4 (structurally);
  R18 item 1 (the targets, as gate checks).
- ADR-006 D1 to D5 and D8 to D13 (the model, codes, registry, report, order,
  cap, encodings, exit table, `internal/finding`, `internal/schemafail`).
- ADR-009 D3, D4 (the table and lowering), D12, D14 and D16 item 3 (the static
  tripwire).
- ADR-010 D1, D2, D11 items 8 to 10, D12 item 3 (the version string), D13
  item 2 (the fault mechanism, with no faults yet) and D15 item 3.
- ADR-011 D1 (the tree), D2 and D5 items 1 and 7 (the front page and
  changelog skeleton).

**Does not touch:** any command beyond the shell; any parser; any statement
beyond chapter 1's front matter; the statement tooling (Wave 2).

**Scope (outline):**

- `go.mod` with ADR-010 D1 item 1's three lines exactly. The Unicode-version
  tripwire comes in Wave 2, with `internal/unitext`.
- `cmd/metais`, exactly ADR-010 D2's one line. `internal/cli` holds `Main` and
  `Run(ctx, Invocation)` with an empty command table, which Wave 2 fills.
- `internal/hostenv.Isolate`: the `NO_COLOR` read, then `os.Clearenv`, then
  `time.Local` set to UTC, and `IsTerminal`.
- `internal/finding`:
  - the model, `New` from registered codes only, and `Report`, with ADR-006
    D8's order, de-duplication, bounded heap and exact totals;
  - `report.truncated`, and the JSON and human encodings of D10 and D11;
  - `Exit` with roles;
  - the `Pos` and `Positions` types of D13 item 2.
- `internal/schemafail`'s failure tree type.
- `spec/findings/registry.json`, with `finding.schema.json` and
  `finding-registry.schema.json` under `spec/schemas/1.0/`. `internal/specdata`
  holds their copies, maintained by `go generate`, with
  `TestSpecDataMatchesSpec`.
- The shared interfaces later waves are written against:
  - the data model's Go representation, which ADR-003 D8, ADR-002 D4 and
    ADR-010 D5 item 4 all assume;
  - `pkgtree.Tree` (`List`, `Open`, `Stat`; ADR-010 D11 item 1), with an
    in-memory tree for tests.
- `internal/limits` (ADR-009 D3's table, exact, with the D4 lowering rules
  and `limit.invalid`); `internal/fault` (the `Set` type only);
  `internal/testenv`; `internal/version`.
- `internal/guarantees`:
  - `layers_test.go`, with ADR-010 D2 item 2's table;
  - the licence-header check;
  - `TestNoHostFacts`, with the import, symbol, directive, dependency-scan and
    canary rules, its exact allow-list, and its fixtures as controls;
  - `notices_test.go`, with `THIRD-PARTY-NOTICES.txt`.
- The `spec/` skeleton:
  - `README.md` with the status block `Version: 1.0`, `Status: Draft`,
    `Latest release: none`;
  - `01-introduction.md` with the RFC 8174 text, the conformance classes, the
    terminology, the notation and the area registry;
  - every other chapter file with its title, status line and summary;
  - `A-references.md`, `B-worked-package.md`, `CHANGELOG.md` (`## 1.0 —
    unreleased`) and `ERRATA.md`;
  - the `.gitattributes` line `spec/conformance/** -text`.
- AGENTS.md's Development section, stating the extended gate (ADR-010 D15
  item 3).

Every later Go item needs `go.mod`, so the wave is narrow by nature. Only
work that touches no Go file can run beside the module.

**Acceptance.**

- The extended gate is green on all six targets.
- `TestNoHostFacts` flags its `os.Getenv` fixture, and the import rule flags
  `internal/guarantees/testdata/netleak/`. Both controls are run by the
  tripwire's own test.
- The report's ordering test was red against a report that keeps insertion
  order, and is green now (ADR-006 D13 item 5). The cap keeps exact totals
  under a lowered `findings` limit.
- `TestSpecDataMatchesSpec` fails on a one-byte change to a spec copy.
- `limits` lowering refuses a raise, an unknown name and a malformed value,
  each as `limit.invalid`.
- A non-test file outside `internal/fault` that names a fault constant fails
  the tripwire. So does a call to `version.Override` from a non-test file.

**Decisions settled before filing:**
- R1-003: Go 1.26.9 and Unicode 15.0.0.
- R1-007: `NO_COLOR` is the only variable read.
- R1-015: limits are exact and never raised.
- R1-019: the exit table.
- R1-022: Apache-2.0 for the specification and the vectors.
- R2-079, R2-080, R2-082, R2-087, R2-088, R2-092: the finding model.
- R2-143: the tripwire's exact allow-list.
- R2-160: the version string.
- R2-162: faults by options injection.
- R2-163: a single-threaded render.
- R2-164: spec files embedded through copies.
- R2-165: `internal/hostfs` and `internal/hostenv.Isolate` are the single host
  seams.
- **The binding constructor ADR-009 D13 names `internal/bindings` lives in
  `internal/template`.** ADR-010's layout owns package names, and RULINGS-v1
  "Resolved" item 51 settles the host seams the same way.

**Rulings made during the wave:**
- **The data model's Go representation lives in `internal/liquid/value`.**
  ADR-010 D2's layout lists that package as the value model of ADR-003 D8
  and ADR-004, and D7 item 1 has `jsonx` and `yamlx` produce the same
  `value` tree. It keeps D2 item 2's rule for `internal/liquid/...`
  (standard library only), keeps each number's literal text (ADR-010 D5
  item 4), and holds no conversion to the schema library's form, which is
  Wave 2's adapter. Wave 1 creates it; Wave 2's Liquid front end builds on
  it.
- **Dependency-scan entries that ADR-009 D16 item 3 and R2-143 prescribe**
  (for `golang.org/x/term` and `golang.org/x/sys`) are the tripwire's
  mechanism, not a widened refusal. An entry neither of them names is a
  finding for Ed.

**Verdict (2026-10-18): complete.** The seams landed as §7 asks: the module
on go1.26.9, the `cmd/metais` shell, `internal/finding` with the registry
and `internal/limits`, the tripwires, the data model in
`internal/liquid/value`, `pkgtree.Tree` with its in-memory tree and contract
test, `internal/version`, and the `spec/` skeleton, in PRs #7 to #13. The
verification round mutated six of the seven code items, about 75 mutations,
and every guard that carries an acceptance criterion went red. It passed
with no blocking finding. Brief and closeout:
[brief-plan-2026-10-metais-v1-wave1.md](../briefs/brief-plan-2026-10-metais-v1-wave1.md)
(brief [#5](https://github.com/kottera/metais/pull/5), rulings
[#6](https://github.com/kottera/metais/pull/6), closeout
[#15](https://github.com/kottera/metais/pull/15)).

What changes later waves:

- **Windows links neither `golang.org/x/sys` nor `golang.org/x/term`**,
  because `x/sys/windows` imports `net`, which ADR-009 D16 item 3 refuses.
  ADR-010 Amendment A1 states this, proposed in
  [#16](https://github.com/kottera/metais/pull/16) and awaiting Ed's
  ratification. Until then the stricter rule governs. Wave 2's windows
  no-replace rename uses the standard library.
- **The MapOrder fault is named by `internal/template`**, which sets the
  engine's own test-only option `Env.UnorderedObjects` from it, so
  `internal/liquid/...` stays standard-library-only. ADR-010 Amendment A2
  records the ruling (design session W1-F3). Wave 4's determinism item
  builds the option, the map-ordered iteration behind it and the tripwire
  that only `internal/template` sets it.
- **The credential act was ended without its probe being run.** The
  evidence came from the items instead: they fetched go1.26.9, ran `-race`
  with cgo, and pushed and merged their PRs. That evidence comes from
  darwin/arm64 only.
- **Not every gate ran on the pinned toolchain.** Some dispatches ran
  go1.26.2 under `GOTOOLCHAIN=local`.
- **Carried and still running when the wave closed:** W1-F1 (four test
  gaps), W1-F3 and the `metais.org` act W1-A2.

## 8. Wave 2 — the leaf libraries and the proof harness

**Objective.** Every package that depends only on the seams exists and is
proved by its own tests. The conformance runner is proved against a
deliberately broken implementation. The command line parses, helps and
answers `version`, `limits` and `help`. No package can be opened yet, and the
suite has no vector.

**Carries:**
- PRD R2 item 3 (the JSON document), R14 items 1 to 3 (the container's and
  the parsers' refusals, as libraries), R15 item 3 (`version`), R16 (the
  runner and tooling), R17 item 3 (help) and R18.
- ADR-001 D1 to D4 (the reader and writer).
- ADR-002 D2 to D10 (the profiles, the model, the gate and the manifest
  schema; the canonical form as JCS).
- ADR-003 D7 (bundling), D8 to D14 (each form's reader, and the static walk
  and coercion).
- ADR-004 D1 to D3 and D16 item 4 (the grammar and parse errors, in the front
  end).
- ADR-005 D12 (the predicate) and D17 (the tar writer).
- ADR-006 D6 and D7 item 4's mapping rules.
- ADR-008 D1, D3, D4, D10 to D13, D15, and D16 items 1, 3 and 7.
- ADR-009 D5 to D7 (the reading checks), D8 (`hostfs`), D9 item 6 and D10.
- ADR-010 D3 to D11, D13 item 3, and D14 items 4 and 6.
- ADR-011 D3, D7, D9 to D15, and D18.

**Does not touch:** the package loader and every command that opens a
package (Wave 3 onward); Liquid rendering and filters (Wave 3); every
statement and vector of chapters 2 to 14.

**Scope (outline).** Each bullet is a leaf. None needs another of this
wave's.

- **The command-line core** (ADR-008, ADR-010 D10 item 3):
  - the argument parser and the declaration each command is built from;
  - help, `help --json` and the topics; the envelope; the streams; colour;
    the exit mapping;
  - `version` (ADR-002 D8's content), `limits` and `help`;
  - the committed schemas under `docs/cli/schemas/`, with every `--json`
    document a test produces validated against its schema;
  - ADR-008 D16 items 1, 3 and 7.

  A command's declaration marks whether the command opens a package.
- **The conformance runner** (ADR-011 D9 to D13, D15, D18):
  - `internal/conformance`, black box, with its sandbox, environments A and
    B, the sentinel scan and the snapshot check;
  - per-step bounds from `getrusage`; the `cli` and `exec` adapters, the
    reference driver and `TestDriverProtocol`; the record and the compare;
  - the self-proof, one fault per check;
  - `spec/schemas/conformance/1/` and `spec/conformance/README.md`;
  - the tripwires `TestVectorsWellFormed`, `TestSuiteTreePortable`,
    `TestNoSentinelInSuite` and `TestHostileVectorsOpenEverywhere`.

  With no vector it reports that it ran none, and passes.
- **The document tripwires**: the statement parser and the generator of
  `spec/statements.json`, and these ADR-011 D14 tests:
  - `TestStatementsWellFormed`, `TestStatementIndex`,
    `TestKeywordsOnlyInStatements`, `TestRecommendationsNotObservable` and
    `TestCoverage`;
  - `TestCitationsResolve`, `TestFindingCatalog` (with the catalog generator
    into `12-findings.md`) and `TestFindingCodesCovered`;
  - `TestExamplesMatchVectors`, `TestReleasedFilesUnchanged`, `TestChangelog`
    and `TestSchemasPublished`.

  They read `vector.json` by the members ADR-011 D9 item 3 defines.
- **Zip, Unicode and paths**:
  - `internal/zipfmt`: the strict reader, with D8 steps 1 to 5 over a
    byte-counting reader, and the `CreateRaw` writer with its deflate canary;
  - `internal/unitext`: NFC, full case folding, categories, and the
    Unicode-version tripwire;
  - the one portable-path predicate of ADR-005 D12, with its collision set;
  - the packed `pkgtree.Tree`;
  - `FuzzZipRead` and `FuzzPortablePath`, and the "every byte read once" test.
- **Host files**: `internal/hostfs` and the source-directory `pkgtree.Tree`.
  - `os.Root` reads of regular files only, and confined writes;
  - fixed staging and temporary names from SHA-256;
  - the no-replace rename, build-tagged per operating system;
  - removal of the render's own creations only;
  - OS errors mapped to ADR-006 D9 item 8's reasons;
  - the device cases as unit tests.
- **JSON**: `internal/jsonx`'s strict reader with positions, `Canonical`
  through `gowebpki/jcs` with RFC 8785's examples, the `json` filter's
  writer, and `FuzzJSONParse`.
- **YAML**: `internal/yamlx` enforcing ADR-002 D2 over `yaml.v3` nodes, the
  `yaml` filter's writer, and `FuzzYAMLProfile`.
- **CSV and Markdown**:
  - the RFC 4180 reader of ADR-010 D8 and the `csv` writer, with the
    round-trip fuzz target;
  - `markdown.SplitFrontmatter` and `markdown.Outline` on `goldmark/v2`, with
    cmark's pathological cases.
- **The JSON Schema compiler**:
  - `schema.NewCompiler`, closed as ADR-010 D5 says, with METAIS's own format
    validators;
  - the pre-checks for unknown keywords, formats and `$schema`;
  - `internal/regexprofile` with ADR-009 D10 item 6's translation table;
  - the adapter to `schemafail`, which never reads message text and never
    relies on order;
  - the JSON-Schema-Test-Suite, vendored with its exclusions;
  - `FuzzRegexProfile`.
- **Failure mapping, bundling and the static walk**:
  - ADR-006 D6, field lookup and the `anyOf`/`oneOf` collapse rule, over
    hand-built `schemafail` trees;
  - ADR-003 D7's bundling, as a value tree;
  - ADR-003 D14's static walk and coercion.
- **The tar writer**: `internal/tarw` and ADR-010 D14 item 5's golden bytes.
- **Dates**: `internal/dates`, with `--now` parsing and ADR-004 D11's `date`,
  input parsing and the closed `strftime` set, tested under several `TZ`
  values.
- **The Liquid front end**:
  - the lexer and the strict `strict2` parser, and `internal/liquid/ast` with
    `Walk`, spans, and binding constructs marked;
  - `internal/liquid/value` with ordered objects;
  - the `Dialect`, `TagDef`, `FilterDef` and meter interfaces that Wave 3's
    renderer and filter catalog implement independently;
  - `ParseValue`, `ParseCondition`, the `include` refusal and
    `template-nesting`;
  - Golden Liquid's invalid-template cases, and `FuzzLiquidParse`.
- **The manifest library** (ADR-002 D4 to D7, D10):
  - the model, built from an already-parsed data model;
  - the field and version-gate rules;
  - `spec/schemas/1.0/manifest.schema.json`, whose `$defs/inputs` follows
    ADR-003 D1 and whose `$defs/outputs` follows ADR-005 D1, D2 and D7 to D9,
    published and embedded.

  The loader composes it in Wave 3. It does not call the validator itself.

**Acceptance.**

- The runner's self-proof fails on each faulty implementation of ADR-011 D18
  item 1 and passes on the clean one. `TestDriverProtocol` passes through
  `exec`.
- The cross-platform record is written and compared on the host.
- Every fuzz target named above has its seed corpus and runs in `go test`.
- Golden Liquid's parse cases pass or carry an exclusion from the closed set.
  The JSON-Schema-Test-Suite passes minus its committed exclusions, each
  naming its ADR-003 item.
- The deflate canary and the Unicode-version tripwire are each red on a
  changed value.
- The echo test of ADR-006 D13 item 5 was red against a mapping that used the
  library's messages, and is green now.
- `metais version --json` validates against `metais.version/1`. `metais
  limits --json` lists ADR-009 D3's table exactly.
- The argument-syntax table tests of ADR-008 D16 item 7 pass.

**Decisions settled before filing:**
- R1-001 and R2-007 to R2-019: the manifest. R1-002 and R1-016: keywords and
  formats.
- R1-004: METAIS's own Liquid engine.
- R1-010: byte identity per build. R1-014: a second end record is refused.
- R1-020 and R2-166: the command line is not the specification, and the
  runner is a black box.
- R1-021 and R2-091: no echo, and neutralisation.
- R1-023: one portable-path rule.
- R2-001 to R2-003 and R2-006: the zip profile.
- R2-024 to R2-028, R2-038 and R2-084: the schema and the mapping.
- R2-039 to R2-043, R2-048, R2-049 and R2-061: the dialect's grammar.
- R2-052 (`date`), R2-072 (pax), R2-077 (fixed names), and R2-114 to
  R2-116, R2-118, R2-119, R2-129 and R2-133 (the command line).
- R2-138, R2-140, R2-141, R2-146, R2-151 to R2-159 and R2-172 to R2-182.
- **`date` lives in `internal/dates` and is registered by `internal/template`
  through `Extend`.** `Core()` holds the rest of ADR-004's standard set, and
  the dialect-list test compares `Core()` with the METAIS extensions against
  ADR-004's catalog. The reasons:
  - `internal/liquid` imports no internal package (ADR-010 D2 item 2);
  - `time.Parse` and `time.Unix` are allowed only in `internal/dates`
    (ADR-009 D16 item 3);
  - ADR-010 D2's layout already gives `internal/dates` the date filter's
    maths.

  This narrows ADR-010 D4 item 2's list of `Core()` by one filter.
- **Codes of class `usage` and `internal` are proved by `internal/cli`
  tests, not by vectors.** `TestFindingCodesCovered` exempts them, and their
  registry entries name no statement. The reasons:
  - the `cli` adapter fails a step on exit 3 and on exit 2 (ADR-011 D11
    item 5), so no vector can produce them;
  - the command line has no chapter (ADR-011 D1 item 2);
  - ADR-008 D16 item 2 already requires a test per declared refusal code.

  ADR-006's *Conformance* asks for a vector per row of the exit table. Rows 2
  and 3 are proved by those tests instead.
- **The unwired list.** A registered code of class `refused`, `content` or
  `destination` that no vector produces yet is listed in
  `internal/finding/testdata/unwired.txt`, with the wave expected to wire it.
  `TestFindingCodesCovered`:
  - accepts a listed code;
  - fails on a listed code that a vector produces, so the list only shrinks;
  - fails on a listed code that is not registered.

  The list must be gone by Wave 7's close. A code that no vector can produce
  on any platform is a finding for Ed, not a permanent line in the list.
- **`TestHostileVectorsOpenEverywhere` reads the opening set from the
  command declarations**, so it holds as commands land over several waves.
  By Wave 4, the declared set is exactly ADR-011 D16 item 3's commands.
- **The runner's own uses of the JSON Schema validator are in test files.**
  ADR-010 D5 item 1's single-call-site tripwire covers non-test code.

**Rulings made during the wave:**
- **The Markdown outline meets ADR-009 D10 item 4 with METAIS's own linear
  CommonMark reading, not goldmark.** W2-9 measured goldmark v2.1.6 as
  roughly quadratic on 10 to 11 of cmark's 25 pathological cases.
  ADR-010 D3's fallback, goldmark v1.8.6, is also superlinear. R2-156 and
  ADR-010 D9 chose goldmark assuming it met the bound. Plan §4's rule, that
  the stricter security rule wins, settles the disagreement:
  - the bound stands whole;
  - `Outline` is METAIS's code, linear by CPU time and faithful to
    CommonMark 0.31.2 on what decides a heading;
  - goldmark leaves the module.

  ADR-010 Amendment A3, proposed, records the change to D3 and D9 item 2,
  superseding R2-156's library choice. Ed ratifies it at the train's close,
  with A1.

**Verdict (2026-10-19): complete.** Every leaf library §8 asks for landed,
in 16 items (#20 to #38). The command line answers `version`, `limits` and
`help`. The conformance runner is proved by a self-proof of 33 faults, and
it reports that it ran no vector. The verification round mutated every item
and found two blocking defects:

- the self-proof's CPU row depended on host load;
- `RemoveCreated` trusted a reusable inode number and removed another
  party's file on linux.

Both were fixed (#39, #40) and the gate is green on go1.26.9. Neither fix
was re-verified in the wave, and the `hostfs` fix has not run on linux.
Brief and closeout:
[brief-plan-2026-10-metais-v1-wave2.md](../briefs/brief-plan-2026-10-metais-v1-wave2.md)
(brief [#18](https://github.com/kottera/metais/pull/18), Markdown ruling
[#35](https://github.com/kottera/metais/pull/35), closeout
[#41](https://github.com/kottera/metais/pull/41)).

A correction to the closeout: ruling R3 (the Markdown reading) was the
plan orchestrator's, under §4, not Ed's. No person was consulted during
this wave. The breakdown review was the plan orchestrator's too.

What changes later waves:

- **For Ed to ratify at the train's close:**
  - ADR-010 A3 (the Markdown reading, superseding R2-156);
  - the dependency-scan entries ruled as R1 and R2;
  - R4 (the notices file follows ADR-010 D12 item 6);
  - ADR-008's envelope `summary`.
- **Ed ruled the MapOrder seam** (W1-F3, ADR-010 A2,
  [#23](https://github.com/kottera/metais/pull/23)). `internal/template`
  sets the engine's test-only `Env.UnorderedObjects`. Wave 4 builds it.
- **`hostfs` holds one open handle per uncommitted entry**, since #40, and
  darwin's default open-file limit is far below ADR-009's outputs limit.
  Design session `01a15284-df6e-7668-b799-62f4c380a84f` must decide this
  before Wave 4's render.
- **Wave 3's wiring names jcs, yaml.v3, jsonschema/v6 and x/text in
  `THIRD-PARTY-NOTICES.txt`.**
- **Wave 3's renderer owes 97 deferred Golden Liquid cases**, listed in
  `render-time.txt`.
- **Open ADR questions:**
  - `limit.exceeded`'s single subject;
  - large integers refused by spelling;
  - `json.duplicate_name`'s `name` parameter;
  - finding construction costing O(pointer length) per finding;
  - two `yaml` writer gaps.

  The Wave 3 brief rules what it can.
- **AGENTS.md now forbids `go test -fuzz` in a Work Order**
  ([#29](https://github.com/kottera/metais/pull/29), by Ed).
- **Carried and still running when the wave closed:** W2-NB1
  `01a15284-bed3-7226-9f3f-495ea5e82436` (two unpinned boundaries), and the
  `hostfs` design session above.

## 9. Wave 3 — opening a package, and the composed libraries

**Objective.** `metais inspect` opens a packed package or a source directory,
verifies all of it, and prints what a supplier reads first. The container,
manifest and version chapters exist with their vectors. Beside it, every
library the next wave wires is complete: the inputs layer, the Liquid
renderer and its standard filters, the sample generator, the cross-reference
and the split-run scanner. `init` writes its scaffold.

**Carries:**
- PRD R1 items 1 to 5, R2 items 1 and 4 (`inspect`), R3 items 1 to 5 (schemas
  compile at open), R4 and R5 (the layer), R13 items 1 and 6 (the
  scaffold, and the generator's core), R14 items 1 to 3, and R15.
- ADR-001 D5 to D9.
- ADR-002 D1, D9 and D11 (composed).
- ADR-003 D1 to D7 at open; D15 to D22 (the layer).
- ADR-004 D2 to D6 and D10's standard catalog, without `date` (the engine);
  ADR-004 D8 item 6.
- ADR-007 D1, D3 (core), D4 (scanner) and D9 (value generation and form
  writers).
- ADR-008 D2, D6, D14 (for `inspect`) and D17.
- ADR-009 D3 (package limits), D5, D6, D9 and D11.
- ADR-010 D4 items 2 to 6 and D13 (`SkipDigest`).
- ADR-011 D16 (the hostile form).

**Does not touch:** `validate`, the reading commands, `render`, `check` and
`unpack` (Wave 4); the METAIS filters and tags (Wave 5); `pack` (Wave 6).

**Scope (outline):**

- **The loader and `inspect`.**
  - Both trees from Wave 2, with names and layout checked uniformly (ADR-001
    D2 and D5).
  - `digests.json` and the package digest (D6). A source directory's digest
    is computed in memory.
  - Step 7's re-derivation of all three compiled files (D7, D8): the canonical
    manifest, the bundled schema, and the references index of ADR-008 D17.
  - The order of ADR-002 D11, every schema compiled (ADR-003 D22 item 1), and
    the package limits.
  - `inspect` with `--guide` and `next` (ADR-008 D6, D14).
  - Spec text: `02-container.md`, `03-manifest.md` and `04-versions.md`;
    chapter 5's rules observable at open (the dialect, formats, references,
    the combined schema); chapter 12's model and package codes; and chapter
    14's package limits and entry confinement.
  - At least one vector per statement, with packed packages committed as
    bytes only where the case is about bytes (ADR-011 D9 item 7).
  - The program that produces committed packages, with its reproduction test
    (ADR-011 D16 item 5).
  - The digest detector, with `SkipDigest` as its control.

  This is the train's widest item. If it cannot be one pull request, the
  breadth of ADR-001's and ADR-002's *Conformance* lists beyond one vector
  per statement moves to Wave 6's corpus items. The statements, and a vector
  for each, may not move.
- **The inputs layer** (ADR-003 D8 to D22):
  - the five forms through Wave 2's readers, coercion by the static walk,
    collections, and the data directory through `hostfs`;
  - `--set`, `--set-json` and `--inputs-json`, with precedence, defaults and
    absence;
  - the input record (value, raw, source and digest);
  - D22's validation order through the compiler, the mapping and the report,
    including D7 item 4's partial validation.

  It is unit-tested, and the `validate` command is Wave 4's.
- **The Liquid renderer**:
  - every tag of ADR-004 D2, `render` through the `Partials` port,
    whitespace and the blank-body rule;
  - truthiness, comparison, strict names, object-output refusal and number
    text;
  - `Eval` and `EvalCondition`, the meter with ADR-009 D11 item 2's charges,
    and panic recovery;
  - `FuzzLiquidRender`, at small limits.
- **The standard filter catalog**: every filter of ADR-010 D4 item 2's list
  but `date`, with ADR-004's semantics and ADR-009's charges, unit-tested from
  ADR-004's text through the Wave 2 interfaces.
- **The sample-value generator** (ADR-007 D9): SplitMix64, the value
  preference, the optional modes, and pattern generation over
  `regexprofile`'s tree. `schema.sample_unsatisfiable` is registered here.
  The form writers come from Wave 2.
- **The cross-reference analysis** (ADR-007 D3), over the engine's AST and
  the schema documents, with its own filter-signature table, its budgets,
  and `template.xref.*` registered.
- **The OOXML split-run scanner** (ADR-007 D4), over the lexer's spans.
- **`init`** (ADR-007 D1, ADR-008 D9): the scaffold exactly as written, the
  name taken from the operand, the empty-directory rule, and `--dry-run`. Its
  pin through `check`, `test` and `pack` is Wave 6's.

**Acceptance.**

- The suite runs its first vectors, in both environments, with nothing
  written anywhere on a refusal. `con/`, `man/` and `ver/` vectors exist for
  every statement of chapters 2 to 4. At least one committed hostile package
  per `package.container.*`, `package.name.*` and `package.digest.*` code is
  refused by `inspect`.
- `SkipDigest` lets a digest-mismatch package through and the detector
  catches it.
- `inspect` on a fixture's source directory and on its committed package
  gives the same result apart from `source`.
- The "every byte read once" budget holds on open (ADR-010 D15 item 2).
- Golden Liquid's render cases that use only core tags pass or carry an
  exclusion from the closed set.
- The inputs layer reports every planted error of an SC8-shaped fixture in
  one call.
- `init`'s output equals ADR-007 D1 item 4 byte for byte.
- The unwired list names the wave for each code registered here.

**Decisions settled before filing:**
- R1-009: the media type is named in chapter 2, and IANA comes later.
- R1-011: no caller path is printed.
- R1-013: the whole package is verified at every open.
- R1-017: strict templates. R1-018: version compatibility. R1-024: empty CSV
  cells.
- R2-004, R2-005, R2-016 and R2-111: the container and the canonical form.
- R2-020 to R2-037: the data specification.
- R2-044 to R2-047, R2-051 and R2-063: values, order and globals.
- R2-094 to R2-098, R2-106 to R2-109 and R2-124: authoring.
- R2-135: the references index.
- R2-150 and R2-178: the hostile packages.

**Rulings made during the wave:**
- **Golden Liquid cases that apply `date` are out of Wave 3's scope, not
  excluded.** `date` is Wave 5's (§8, §11).
  - The Wave 3 runs select a case out if and only if it applies `date`, and
    report the count.
  - The selector fails as soon as `date` is registered in the dialect, so
    Wave 5 cannot leave it behind.
  - Two of the 97 deferred render-time cases are `date` cases, and they stay
    in `render-time.txt` for Wave 5.
- **One item owns the fix that keeps finding construction linear**, with a
  library-level CPU-time proof. The proofs through `inspect` and the inputs
  layer are meetings. This replaces the brief's "whichever item first makes
  that input reachable", which was a race between two parallel items.
- **ADR-003 D8 item 5's large-integer refusal keeps its reach by spelling.**
  ADR-003's ruling 9, ratified, makes that reach deliberate. ADR-002 D4
  item 3 governs the manifest only, so the two do not disagree. Every
  binary64 above 2^53 is integral, so refusing by value would also refuse
  ordinary large floats in a user's data.
  - The brief's rule that a file the tool writes must be readable by the
    tool holds for numbers within ±(2^53−1).
  - Beyond that range, RFC 8785's integer spelling of a canonical number is
    a stated property: the specification says such text is not accepted as
    input, and a vector shows that `maximum: 11e17` opens. Compiled files
    are compared as bytes, so no package becomes unopenable.
  - The sample generator's JSON writer spells such a number with an
    exponent, so a supplied one can be saved and read back. A CSV integer
    column still refuses it.
- **Chapter 2's statement area is `PKG`** (`MTS-PKG-nnn`, `vectors/pkg/`),
  not `CON`, because `con` is a Windows device name that ADR-011 D9 item 8
  refuses. ADR-011's amendment, proposed, is ratified at the train's close.

**Verdict (2026-10-19): complete.** `metais inspect` opens packed packages
and source directories, verifies all of each, and refuses the committed
hostile packages. Chapters 2 to 4 have a vector for every statement, and the
suite runs its first vectors in both environments. Every library Wave 4
wires is in place: the inputs layer, the renderer and `Core()`, the sample
generator, the cross-reference, the OOXML scanner, and `init`.

That took nine items, three corrective items and one blocking fix, #46 to
#59. The verification round made about 150 mutations across all twelve, and
its gate was green on go1.26.9 linux/amd64. Its one blocking finding, a
missing proof note on MTS-VER-007, was fixed in #59 and not re-verified.
Wave 2's `hostfs` fix #40 has now passed on linux.

Brief and closeout:
[brief-plan-2026-10-metais-v1-wave3.md](../briefs/brief-plan-2026-10-metais-v1-wave3.md)
(brief [#43](https://github.com/kottera/metais/pull/43), rulings
[#45](https://github.com/kottera/metais/pull/45) and
[#54](https://github.com/kottera/metais/pull/54), closeout
[#61](https://github.com/kottera/metais/pull/61)). No person was consulted
during the wave.

What changes later waves:

- **Two findings for Ed are now on his board as design sessions, and
  neither blocks Wave 4:**
  - `01a15348-4ab5-7743-9802-86072e41b5bf`: 16 registered codes no vector
    can produce (§8). Needed by Wave 7.
  - `01a15348-66d1-73ef-bed1-8d62aa3dff0e`: jsonschema v6's `Compile` is
    quadratic on a hostile schema, and fixing it at the cause needs a
    dependency change. Needed by Wave 6's calibration.
- **The `hostfs` open-handle design session**
  (`01a15284-df6e-7668-b799-62f4c380a84f`) is still open. Wave 4's render
  writes only into a new directory, and its brief states the constraint
  that render must meet from ADR-009's exact limits. The general mechanism
  must be settled before Wave 5's existing destinations.
- **The gate flaked under load on linux** in two producers' runs: a CPU-ratio
  test, and self-proof wall bounds under parallel `-race`. The merged-tree
  gate was green. Plan §4 and AGENTS.md apply: the mechanism is debugged,
  and no assertion is widened.
- **For Ed to ratify at the train's close:**
  - ADR-011 A1 (`PKG`);
  - the rulings recorded under this section;
  - the readings Wave 3 took where ADRs disagree on reporting rather than
    on acceptance. ADR-006 D8 de-duplicates repeats of one name. ADR-007
    D3's suggestion distance is followed.
- **Still running when the wave closed:** W3-NB2
  (`01a15339-e49f-757a-b579-8991efd5d775`) and janitorial issue #60.

## 10. Wave 4 — validate, read, the first render, check and unpack

**Objective.** A supplier validates inputs and renders template outputs into
a new directory. A harness reads any part of a package in place. An author
checks a package without data and unpacks one. The template environment of
chapter 7 is proved through `render`. The headline detectors for confinement,
determinism, read-in-place and one pass are in every `go test`. The first SC7
canaries are caught.

**Carries:**
- PRD R2 items 1 and 2, R3, R4, R5, R6 items 1 to 4, R7 items 1 to 3, R8
  item 1, R10 items 1 to 3, R12 items 1 and 2 (from the manifest), R13 items
  2 and 5, R14 item 4, and SC8.
- ADR-003 (observable); ADR-004 D1 to D8 and D16; ADR-005 D2, D4 item 4
  (defaults), D12 (manifest paths), D13, D14 item 1 and D19.
- ADR-006 D6 to D11 (observable).
- ADR-007 D2 to D4 and D8.
- ADR-008 D5, D7, D8 items 1 to 3, and D16 items 2, 4 and 8.
- ADR-009 D7, D8 items 1 to 3, D11, D13, D15, and D16 items 1 and 2 (the
  `env`, `cwd` and `path` canaries).
- ADR-010 D4 item 7 and D13 (`SkipConfine`, `MapOrder`, `UnpackToTemp`,
  `FirstFinding`).
- ADR-011 D17.

**Does not touch:** the METAIS filters, `date`, `embed`, `asset`,
`provenance`, `escape: html`, `for_each`, `when`, path templates, the mirror,
assets, containers, transforms, existing destinations, dry run and standard
output (Wave 5); `sample`, `render --sample` and `test` (Wave 5); `pack`
(Wave 6).

**Scope (outline):**

- **`validate`**, with every supply flag and `--resolved`.
  - Spec text: chapter 5's supplied-data rules, chapter 6, and chapter 12's
    keyword mapping, positions, order, cap, echo and neutralisation.
  - Vectors for every row of ADR-003's and ADR-006's *Conformance* lists,
    including §4.1's and §4.2's constraint shapes (SC2's mechanism).
  - The SC8 test, with `FirstFinding` as its control; the data limits; the
    hostile-input cases; and the `path` canary.
- **The reading commands**: `ls`, `read`, `outline`, `grep`, `schema` and
  `extract` (ADR-008 D5), with the glob grammar. Reading in place is not an
  operation (ADR-011 D8 item 3), so these are proved by:
  - ADR-008 D16's tests, and the packed-and-source agreement test;
  - the "read --lines decompresses once more" budget;
  - the read-in-place detector, with `UnpackToTemp` as its control;
  - each command joining the opening set of every `security` vector.
- **The first render.** `render --out <new dir>` of template outputs, with
  each path given or defaulted.
  - `internal/template` binds `inputs`, `package`, `now`, `sources` and
    `output`, materialises declared-absent members, applies strict names and
    loads partials confined to `templates/`.
  - Phases 1 to 4 run with every output attempted; the predicate applies to
    manifest paths; staging and the atomic rename follow ADR-005 D14 item 1.
  - Spec text: chapter 7 in full. ADR-011 D17 item 4 asks for every
    statement there to be covered by METAIS vectors, besides the delegated
    run.
  - The template work limits.
  - The confinement detector, with `SkipConfine` as its control, and the
    determinism detector, with `MapOrder` as its control (ADR-010 D13
    item 4).
  - The `env` and `cwd` canaries.
- **`check`**, every step of ADR-007 D2 in order, with `report.step_skipped`.
  - The cross-reference and the split-run scanner are wired.
  - Static checks: schema defaults and examples, form and shape agreement,
    output declarations (profiles, `validate` kinds, the path template's
    restricted mode), the lowest minor, and the guide's links.
  - Spec text: chapter 13's `AUT` statements for `check`.
- **`unpack`** (ADR-007 D8), with staging or an empty target, and the
  round-trip half that needs no `pack`.
- **The Liquid closure**:
  - Golden Liquid passes in full or carries exclusions from the closed set;
  - the liquid-spec JSON-RPC server, as a test-only program;
  - `spec/conformance/delegated/liquid-spec.json` with the pin, the feature
    gates and the departures, each naming its statement.

**Acceptance.**

- The four detectors above fail on their faults and pass on the tool.
- The `env`, `cwd` and `path` canaries are each caught by the harness. Their
  red run is recorded in ADR-009's implementation log before the release
  build passes them.
- Every `security` vector refuses identically through `inspect`, `ls
  --recursive`, `schema`, `validate`, `render --out out` and `unpack --to
  unpacked`.
- `TestCoverage` is green over chapters 2 to 7, 12, and 13 so far.
- The unwired list holds nothing this wave wires.
- The liquid-spec act of §6 is filed and waits on the closure item.

**Decisions settled before filing:**
- R1-012: existing files are never read, held from the first render on.
- R1-017: strict templates.
- R2-037 and R2-086: partial validation, and constraints skipped when an
  input fails to parse.
- R2-060 to R2-063: rendering errors and globals.
- R2-081 and R2-134: `--fail-on`. R2-093: the `at` line.
- R2-099: `check`'s severities.
- R2-105 and R2-136: `unpack`.
- R2-117, R2-120 to R2-122, R2-126, R2-128, R2-130 to R2-132: the commands.
- R2-144 and R2-145: truncation and neutralisation.
- R2-176 and R2-182: the liquid-spec pin.

**Rulings made during the wave:**
- **Four intra-wave edges are sanctioned in Wave 4, and no others.**
  - One item changes `hostfs`'s write side for the wave's writers, and
    render, unpack and the reading commands wait on it. Wave 2's #40, which
    §2's slicing did not foresee, made that change shared, and parallel
    edits of security-critical code would be worse than one short serial
    step.
  - The Liquid closure waits on the render item, because each delegated
    departure names a chapter 7 statement that the render item writes.
- **`extract --overwrite` moves to Wave 5.** Replacing files in an existing
  directory is an existing destination (§10 "Does not touch"), and it rests
  on `hostfs`'s general ownership mechanism, which awaits Ed's design
  session. Wave 4's `extract` writes only where the target does not exist.

**Verdict (2026-10-19): complete.** What Wave 4 delivered:

- `validate`, the reading commands (`ls`, `read`, `outline`, `grep`,
  `schema` and `extract`), `render --out <new dir>` with chapter 7's
  template environment, `check`, and `unpack` all landed;
- so did the Liquid closure: Golden Liquid in full, the liquid-spec JSON-RPC
  server, and the delegated-run file;
- the confinement, determinism, read-in-place and one-pass detectors run in
  every `go test` with their faults, and the `env`, `cwd` and `path`
  canaries are caught.

That took seven items (#67 to #75). The verification round mutated all
seven and found one blocking defect: a fixture that `.gitignore` dropped
was never committed. It was fixed in #77 and not re-verified.

Two results are weaker than they read:

- **The canaries' red runs** are recorded in ADR-009's log, but within each
  squashed pull request rather than as earlier commits on `main`.
- **The liquid-spec departures are a prediction,** derived by emulation
  without Ruby: 490 predicted departures. The §6 act
  `01a15414-bf58-7436-8dc0-e1df81410d34` measures them, and Wave 5 is not
  filed until it has ended.

Brief and closeout:
[brief-plan-2026-10-metais-v1-wave4.md](../briefs/brief-plan-2026-10-metais-v1-wave4.md)
(brief [#64](https://github.com/kottera/metais/pull/64), rulings
[#66](https://github.com/kottera/metais/pull/66), closeout
[#79](https://github.com/kottera/metais/pull/79)). No person was consulted
during the wave.

What changes later waves:

- **Ed ruled on three design sessions:**
  - **Existing destinations stage inside the output directory** (ADR-005
    A1, [#74](https://github.com/kottera/metais/pull/74)). Wave 5's existing
    destinations, `extract --overwrite` and `sample --to` implement it.
  - **Schema compile cost is bounded by two new package limits** (ADR-009
    A1, [#76](https://github.com/kottera/metais/pull/76)). Wave 6 implements
    and calibrates them.
  - **The unreachable manifest codes leave the registry, and the package
    operand codes get vectors** (ADR-006 A1 and ADR-011 A4,
    [#78](https://github.com/kottera/metais/pull/78)). Wave 7 carries the
    work. `collection.item_name` and `input.not_readable`, added in this
    wave, join the same treatment.
- **Wave 5 carries:**
  - `extract --overwrite`;
  - `unpack --dry-run`;
  - one answer for an existing empty destination across `render` and
    `extract`;
  - `input.raw_unavailable` through `embed`;
  - the two xref codes listed W5;
  - the chapter 7 and `check` remainders the closeout names.
- **ADR-011 D11 item 9's 60 seconds binds the default conformance run**
  (`TestConformance`), as its text says, not the whole package with the
  tests that re-run the suite.
- **For Ed to ratify at the train's close:**
  - ADR-011 A2 (`unwritable-dirs`);
  - ruling R-W4-0;
  - the codes and readings in the closeout's question 2.
- **Still running when the wave closed:** W4-NB1
  `01a15414-2558-70cb-9443-095630a625eb` and W4-NB2
  `01a15414-8ee4-71c1-b5ec-15de3abbb36b`.

**The liquid-spec act (§6), its first run.** W4-A1
(`01a15414-bf58-7436-8dc0-e1df81410d34`), run by Ed on darwin/arm64 at
`d4ba3ba`, failed: the committed adapter never loaded the liquid gem, so no
case ran. Ed's diagnostic run, with that line added, matched the prediction
exactly. The plan orchestrator filed the fix, W4-A1F
(`01a15429-00fb-76e9-8cdb-561d4975c696`,
[#83](https://github.com/kottera/metais/pull/83)). The fix loads the gem,
sets UTF-8, and adds `TestAdapterSetup` as the guard a dispatch can run.
The re-run, W4-A1R (`01a15429-2531-762a-aebc-1507b28a76a0`), passed at
`a88b034` in both the C and a UTF-8 locale: "1801 pass, 490 departures fail
as listed, 103 held out for applying date, 0 fail otherwise, 0 departures
pass". The 490 departures are now measured, not predicted. The row's notes
require another run once Wave 5 registers `date` and writes chapter 8.

## 11. Wave 5 — the render's capabilities, and the rest of authoring

**Objective.** Everything `render` can do, it does, each capability with its
statements and vectors. An author previews with sample data, keeps it as an
example, and tests the examples.

**Carries:**
- PRD R7 items 4 to 6, R8 items 2 to 5, R9, R10, R11, R12 items 2 to 5, R13
  items 3 and 6, and R14 item 4 (the `home`, `tz`, `clock` and `pid`
  canaries).
- ADR-004 D7 items 4 to 6 and 9, D9 to D15.
- ADR-005 D1 to D11, D14 item 2, and D15 to D18.
- ADR-007 D5 (without containers), D6 and D9.
- ADR-008 D8 items 4 to 9, D9 (`sample`, `test`) and D16 item 5.
- ADR-009 D8 item 4 and D17 item 8.

**Does not touch:** `pack` and `test`'s container comparison (Wave 6); the
corpus in full, the full SC7 reach, calibration and the release builder
(Wave 6).

**Scope (outline).** Each bullet is independent. The second of two to merge
adds the vector where they meet (§2).

- **The METAIS filters** `json`, `yaml`, `csv`, `markdown_table` and `slug`,
  and `date` through `internal/dates`, with the `tz` and `clock` canaries.
  Chapter 8's filter statements.
- **Escaping**: `escape: html`, `safe` and its position rule, and
  `json_script` with `escape` and `escape_once` under it (ADR-004 D9).
- **`embed` and `provenance`**, with the raw form, `input.raw_unavailable`,
  every style, the closed fields, and the `pid` canary.
- **Fan-out, conditions and paths**: `for_each` and `when` as expressions,
  path templates in their restricted mode, `item`, `fan_out` with `previous`
  and `next`, `sources` for naming outputs from a collection, and paths from
  data refused by the predicate.
- **Assets and the mirror** (ADR-005 D5, D6), and the `asset` tag (ADR-004
  D13) with its placement map and the `home` canary.
- **Zip containers and profiles**: entries as declarations, order, `stored`,
  nesting to `container-depth`, `ooxml`, `odf` and `epub`, and the
  tool-written `mimetype`. Vectors compare by entries (ADR-011 D10 item 3).
- **Transforms**: `newline: crlf`, and output validation for `json` and
  `yaml` outputs (ADR-005 D10, D11).
- **Existing destinations**: `--overwrite`, the per-file checks (links,
  junctions, FIFOs, case variants, and ADR-005 A1's device and staging-name
  checks), the staging directory inside the output directory and the commit
  by unit (ADR-005 A1). Nothing is read and nothing is deleted.
- **Dry run, the archive and `--only`**: the listing with `status`, the pax
  stream compared by entries, the selected file raw, and ADR-008 D16 item 5's
  standard-output purity, pinned red first.
- **`sample`**: `--to` and `--example`, fixed points, the form writers, and
  repeatable vectors followed by a `validate` step (ADR-011 D10 item 7).
  Chapter 13's `SMP` statements.
- **`render --sample`**: the preview, `--optional`, `--seed` and `--items`,
  and `--data` refused.
- **`test`**:
  - the example layout and `example.yaml`;
  - ADR-004 D15's fixed identity;
  - byte comparison of regular files and difference reporting;
  - `--update` deleting inside `expected/` only (R1-006).

  Comparison of container outputs is Wave 6's.

**Acceptance.**

- Every statement of chapters 8 to 11 has a vector.
- Every `pth/` vector runs on every platform.
- The `home`, `tz`, `clock` and `pid` canaries are caught, each recorded red
  first in ADR-009's implementation log.
- The standard-output purity test was red against a build that writes a
  notice to standard output.
- Every meeting of two capabilities this wave added has its vector, and the
  verification round lists them.
- `sample` vectors are repeatable across two fresh sandboxes, and the data
  they write validates.

**Decisions settled before filing:**
- R1-005: the sample commands. R1-006: `test --update` may delete inside
  `expected/`. R1-012: existing outputs are never read.
- R2-050 and R2-052 to R2-059: escaping, filters and tags.
- R2-064 to R2-078: outputs.
- R2-100, R2-101 and R2-110: examples.
- R2-113, R2-125 and R2-127: the command line.
- R2-054: no `sha256` filter in 1.0.

**The integration gate (W5-I1).** W5-1, W5-2, W5-5 and W5-7 to W5-12
merged without the full gate, by Ed's ruling of 2026-10-19. The full gate
ran once on `main` at `8278575` with `gate.sh`, on darwin/arm64 under
`go1.26.9`, with `-p 8` on a 16-core host. It was green, and nothing came
from the test cache:

```
=== PASS build (41s)
=== PASS vet (18s)
=== PASS test (893s)
=== PASS tidy (2s)
=== PASS gofmt (1s)
=== PASS static (9s)
=== PASS cross (201s)
=== PASS specdata (3s)
GATE DONE: PASS
```

The wall-clock-bound tests that timed out on the items passed, so no
timeout was left to settle.

**`TestConformance` is at ADR-011 D11 item 9's 60 seconds, and over it on
a busy host.** It ran 301 vectors and 413 cases, with none skipped. The
default run, not race-built and run alone, took 56.5 s, 65.6 s and 84.1 s
of wall time on that host, against 20 to 62 s of CPU time. The
mechanism is the A/B start offset. `runCase` sleeps until two seconds
after A ended before it starts B, and the sleep holds the case's parallel
slot. So 413 cases at the default `-test.parallel` of 16 spend about 52 s
asleep, whatever the host's speed. At `-test.parallel 64` the same binary
took 21.4 s with the same CPU time. The budget needs a runner that does not
hold a slot while it waits, and no ADR decides that, so the integration
gate leaves it to the wave orchestrator.

**Verdict (2026-10-19): complete.** Every capability §11 asks of `render`
landed, each with its statements and vectors:

- the METAIS filters and `date`;
- escaping;
- `embed` and `provenance`;
- fan-out, conditions and path templates;
- assets and the mirror;
- zip containers with their profiles;
- CRLF and output validation;
- existing destinations under ADR-005 A1;
- dry run, the standard-output archive and `--only`.

So did `sample`, `render --sample` and `test`, and Wave 4's carried items.
No declaration reaches `internal.error` except through a panic. `unwired.txt`
holds no W5 entry, and the `home`, `tz`, `clock` and `pid` canaries are
caught.

That took thirteen items (#86 to #99). Ed changed the gate mid-wave, by
guidance: each item ran directed checks, and W5-I1
([#100](https://github.com/kottera/metais/pull/100)) ran the full gate once
on the merged tree, green. The verification round ran the full gate green on
a fresh clone and found two blocking composition defects. Both were fixed
(#101, #102) and not re-verified. Two items, W5-10 and W5-I1, were inspected
by the verifier but not mutated. Restoring `date` changed the delegated
counts to "1845 pass, 549 departures fail as listed".

Brief and closeout:
[brief-plan-2026-10-metais-v1-wave5.md](../briefs/brief-plan-2026-10-metais-v1-wave5.md)
(brief [#85](https://github.com/kottera/metais/pull/85), closeout
[#103](https://github.com/kottera/metais/pull/103)).

What changes later waves:

- **Ed ruled, by guidance to the plan orchestrator, that the gate's cost and
  the merge friction are fixed before Wave 6.** That is Wave 5b, below.
- **`TestConformance` sits at its 60 s budget, because it is sleep-bound.**
  W5-NB3 (`01a155c2-3d8a-7af0-8fea-3d7e111536e1`) must land before Wave 6
  grows the hostile corpus.
- **The liquid-spec run is due again after this wave's chapter 8.** The
  carried act W5-A1 (`01a155c2-5c85-7762-a3d8-2541e178e29c`) asks for it.
  W5-1's own run passed, but not cleanly.
- **For Ed to ratify at the train's close:**
  - the wave's breakdown readings (a) to (d);
  - R-W5-11 and R-W5-12;
  - `hostfs.RemoveEmptyDir`;
  - W5-V2's `Present` status;
  - ADR-009 D8 item 3's `output.link`, which contradicts its own table's
    `output.destination.link`.
- **Still running when the wave closed:** W5-NB1, W5-NB2 and W5-NB3.

## 11b. Wave 5b — the gate's cost and merge friction (inserted by Ed, 2026-10-19)

**Why it exists.** Wave 5's items spent longer in the gate than in their
work. Seven full gates ran at once on two laptops, packages timed out under
load in items that never touched them, and every sibling merge forced a
rebase and another full gate. Most rebase conflicts came from file layout,
not from overlapping code. Ed ruled the fixes by guidance on 2026-10-19,
and they land before Wave 6, which waits on this wave.

**Objective.** When this wave is done:

- the gate has two tiers, by an ADR-010 amendment to D15 item 3, with
  scripts that run it in the background, resume it, and re-run one test
  alone;
- `go test` is capped on a shared host;
- the `verification-gate` and `root-cause` skills match;
- the `hostfs` open-file-limit tests are cheap, or the reason they cannot
  be is recorded;
- adding a finding code, a statement, a help golden entry or an ADR log
  entry no longer collides with a sibling;
- text help groups generated refusals by family (an ADR-008 D15
  amendment);
- `internal/cli/render.go` is split by phase.

**Scope.** Ed's rulings, carried whole in the wave's brief:

- the test-time item;
- the four conflict fixes;
- Fix 5 (help refusals);
- Fix 6b (the `render.go` split).

**Does not touch:** any behaviour of the tool beyond help's text layout; no
new command and no new finding code; render-pipeline extension points
(after the train, §14).

**Decisions settled before filing** (Ed, 2026-10-19):

- the two-tier model;
- no flake register, and every red is root-caused;
- the shared-host `-p` cap;
- the four conflict fixes;
- Fix 5;
- Fix 6b;
- §2's shared-seams and statement-number rules.

These are Ed's rulings, not the train's, and need no ratification at the
close.

**Rulings made during the wave:**
- **ADR-011 D3 item 3 permits gaps in statement numbers**, so §2's
  statement-number blocks need no ADR change. The item protects the
  stability of an identifier: it is never assigned by position, never
  reused, and never renumbered. A number allotted in a block and never used
  is never assigned, so none of that is broken. Within each item, numbers
  are still taken in order of addition. No contiguity check is added.
  Ed's guidance reserved the question to him only if the item forbade gaps.

**Verdict (2026-10-19): complete.** Every one of Ed's rulings for this wave
landed, in seven items (#108 to #116):

- the two-tier gate, by ADR-010 Amendment A5, with `gate.sh` and its scoped
  tier, `--resume`, the selector, and the `go test -p` and `-parallel` cap;
- the skills, matching the gate, with no flake register;
- the `hostfs` limit tests: `StageNew_abort` was slow, not hung, and their
  cost floor is recorded;
- one hand-edited place per finding code;
- `render.go` split by phase;
- help refusals grouped by family, by ADR-008 Amendment A1, with the JSON
  help goldens pretty-printed;
- per-item ADR log files;
- generated files that merge themselves, with ADR-011 D3 item 7 amended to
  match.

W5-NB3 (#112) brought `TestConformance` back well within its 60 s budget.
The verification round ran the wave tier on a fresh clone, green on all 42
packages. It found one blocking defect in help's family lines, fixed in #117
and not re-verified.

Brief and closeout:
[brief-plan-2026-10-metais-v1-wave5b.md](../briefs/brief-plan-2026-10-metais-v1-wave5b.md)
(brief [#105](https://github.com/kottera/metais/pull/105), ruling
[#106](https://github.com/kottera/metais/pull/106), closeout
[#119](https://github.com/kottera/metais/pull/119)).

Weaker than it reads:

- **W5b-1's `--resume` and selector were inspected, not mutated**, and so
  were W5b-4 and W5b-6. The scoped tier now decides what each item tests,
  so the selector's guards are tested by their producer alone.
- **The merge drivers have run on linux only.**

What changes later waves:

- **Items run `gate.sh`'s scoped tier**, and the wave tier when they touch a
  shared seam. Sessions run `merge-setup.sh` before their first rebase and
  `regenerate.sh` after every rebase.
- **The liquid-spec run after Wave 5 passed cleanly.** W5-A1
  (`01a155c2-5c85-7762-a3d8-2541e178e29c`) ran at `cdce4d9` in both locales:
  liquid-spec exited 0, with "1845 pass, 549 departures fail as listed, 0
  fail otherwise, 0 departures pass".
- **For Ed: 36 commits on `main` carry `Co-authored-by` trailers**, against
  AGENTS.md. Among them are `278c96a` (W5b-1), `30929d5` (W5-I1) and every
  squash merge of the plan orchestrator's own briefs and verdicts, #5 to
  #121. GitHub adds the trailer to a default squash body when a commit's
  local author identity, a machine-local address, differs from the account
  that merges. Removing them means rewriting `main`'s history, which is
  Ed's decision. From #122 on, the plan orchestrator commits under the
  account's noreply address and merges with an explicit body, and the
  Wave 6 brief tells every item to do the same.
- **Still running when the wave closed:** W5b-NB1
  (`01a1566a-4184-7367-92dc-49ee5d5dcc56`) and W5b-NB2
  (`01a1566a-70f6-73a5-b93a-d57fae90a1a0`).

## 12. Wave 6 — pack, and hardening

**Objective.** `pack` closes the authoring path. The hostile corpus and the
SC7 harness reach everything ADR-009 lists. Limits and budgets are measured.
A release can be built.

**Carries:**
- PRD R1 item 4 and R13 item 4 (pack), R9 item 5 and R13 item 3 (containers
  in `test`), R14 in full, R18, SC4's harness, SC6 and SC7.
- ADR-001 D4 (through `pack`).
- ADR-007 D1 item 7, D5 item 3 (containers) and D7.
- ADR-008 D9 (`pack`).
- ADR-009 D3 item 3, D16 and D17 in full, and D18.
- ADR-010 D12, D14 and D15 items 1 and 2.
- ADR-011 D15 (the matrix procedure) and D16 in full.

**Does not touch:** the example packages and the close-out documents
(Wave 7).

**Scope (outline):**

- **`pack`**:
  - check, then test, then the deterministic write (ADR-007 D7), with the
    default name, the atomic replace, and `--out -`;
  - the pack → unpack → pack round trip;
  - chapter 13's `pack` statements, and chapter 2's writer statements,
    observable through the `pack` operation (ADR-001 *Conformance*, "Writer");
  - the SC7 scan of the bytes and entries of every package `pack` writes;
  - `init` → `check` → `test` → `pack` pinned (ADR-007 D1 item 7);
  - the example-package harness: a test that runs `check`, `test`, `pack` and
    `render --sample` previews, with `--optional all` and `none`, over every
    directory under `examples/`, and validates what they write. It reports
    when there is none.
- **`test`'s container comparison**: by entries, `stored` methods only, the
  `example.entry_*` codes, and unreadable expected containers.
- **The hostile corpus in full**, split by ADR-009 D17's areas: every case of
  items 1 to 12, ADR-001's *Conformance* list, and ADR-005's path cases.
  - Each case has exactly one defect, its committed bytes, its program, and
    bounds lowered to what the defence needs.
  - Device cases are `hostfs` unit tests.
  - No case exits through a panic or a runtime fatal error (ADR-009 D18).
- **The SC7 harness's full reach** (ADR-009 D16 item 1):
  - every command, in human and JSON form;
  - the added variables, the decoy trees, both path modes, and check 2's real
    host facts;
  - `TestHostIsolationCanaries` over every canary, and the release build
    clean.
- **Limit calibration**: the worst case of each work and output limit at its
  default, recorded per target, with ADR-009 D18 item 2's ceilings as the
  acceptance. The suite is written here and run on real hosts by §6.
- **Schema compile bounds** (ADR-009 Amendment A1): `schema-subschemas` and
  `schema-location-length` in D3's table, `internal/limits` and the
  pre-check, with vectors for both refusals and the combined schema's
  count; their joint worst case joins the calibration suite.
- **Benchmarks** of ADR-010 D15 item 1, with committed baselines.
- **The release builder** (ADR-010 D12):
  - `internal/tools/release`, which builds twice and compares;
  - the per-target archives and `SHA256SUMS`;
  - the refusals: a dirty tree, a `HEAD` that is not the tag, a missing
    build setting, and an `implements` naming an unreleased revision
    (ADR-011 D5 item 6).

**Acceptance.**

- SC6 holds on the host: every `security` vector refuses through every
  opening command with exact findings, and nothing changes anywhere.
- SC7 holds on the host: every canary is caught and the release build is
  clean.
- The packed scaffold and a fixture survive the round trip with equal package
  digests and, within this build, equal bytes.
- The release builder refuses each of its refusals, and produces matching
  `SHA256SUMS` on two runs.
- The matrix and calibration acts of §6 are filed.

**Decisions settled before filing:**
- R1-015: a calibration result that breaks a ceiling is escalated as an
  ADR-009 amendment, never applied silently.
- R2-102, R2-103, R2-104 and R2-123: `pack` and containers in `test`.
- R2-137, R2-139, R2-142 and R2-147 to R2-150: limits, links, canaries and
  the corpus.
- R2-161: the release is built twice. R2-175: the matrix. R2-178: one defect
  per hostile package.

**Rulings made during the wave:**
- **The SC7 harness's every-command reach runs over every vector at the wave
  tier, not in every `go test`** (plan orchestrator, 2026-10-20; for Ed to
  ratify, because it extends ADR-010 A5).
  - Over every vector, the reach is 28,908 processes. At the gate's share of
    the host it timed out once Wave 6's corpus landed.
  - Every `go test` still runs TestHostIsolation over every vector with the
    release build (variables, decoys, both path modes, checks 1 to 4, and
    the crash check), and runs the reach over the canaries' probes and its
    self-test. So a harness that stops catching a leak still goes red on
    every change.
  - The full reach over every vector is a named wave-tier step in
    `gate.sh wave`, split by area within the existing timeout. It runs at
    every wave's verification and before the release.
  - No check is dropped, no timeout raised, and no host share enlarged.
- **The release builder moves to Wave 7** (plan orchestrator, 2026-10-20).
  - Ed's design session `01a1566e-2786-7173-84ff-c1485fcf8cdf`, on how
    `internal/tools/release` is allowed past ADR-009 D16 item 3's symbol
    rule, was still unclaimed when every other Wave 6 item had merged.
  - Rather than stop the train, Wave 6 verifies and closes without W6-13.
    §12's release-builder acceptance is unmet in Wave 6, and it carries to
    Wave 7.
  - Wave 7 builds the release builder only once Ed has ruled. His ruling is
    recorded here when it comes.

**Verdict (2026-10-20): complete, with the release builder carried to
Wave 7.**

What landed:

- `pack`, round-tripping through `unpack` and pinned from `init` through
  `check`, `test` and `pack`, with the example-package harness;
- `test`'s comparison of containers by entries;
- the hostile corpus in full, by ADR-009 D17's areas;
- the SC7 harness's full reach, as a wave-tier step under R-W6-3;
- ADR-009 A1's schema compile bounds;
- the limit-calibration suite;
- the benchmarks, with sandbox baselines.

That took thirteen items: W6-1 to W6-12 and the corrective W6-14 (#128 to
the merge before #141). The verification round ran the wave tier green on a
fresh clone, all 25 reach groups included, and mutated twelve of the
thirteen items. It found two blocking defects in the gate's own scripts,
fixed in #141 and not re-verified.

Brief and closeout:
[brief-plan-2026-10-metais-v1-wave6.md](../briefs/brief-plan-2026-10-metais-v1-wave6.md)
(brief [#121](https://github.com/kottera/metais/pull/121) and
[#122](https://github.com/kottera/metais/pull/122), rulings
[#136](https://github.com/kottera/metais/pull/136) and
[#140](https://github.com/kottera/metais/pull/140), closeout
[#142](https://github.com/kottera/metais/pull/142)). No person was consulted
during the wave.

Not met, or weaker than it reads:

- **§12's release-builder acceptance is not met.** Ed's design session
  `01a1566e-2786-7173-84ff-c1485fcf8cdf` was still unclaimed, so the
  builder moved to Wave 7, which builds it once Ed rules. W6-13's drafted
  task is the starting point.
- **Some proofs have run on no host:**
  - the pseudo-terminal proof has run on no host;
  - W6-14's darwin half is unproven;
  - three `hostfs` device tests skip in sandboxes.

  The matrix rehearsal W6-A1 (`01a15822-b803-7c96-9673-42aaf46e69b5`)
  carries all three.
- **The sandbox calibration breached nine ceilings.** No default was
  changed. W6-A2 (`01a15822-7680-746c-b98e-dc38260f857a`) measures on real
  hosts, and a real breach becomes an ADR-009 amendment for Ed (R1-015).
- **W6-7 (benchmarks) was inspected, not mutated.**

What changes the last wave:

- **`check` accepts a literal output path `../escape.txt`**, which
  `render` and `test` refuse. It is not a confinement breach, but `check`
  exists to say whether a package will render. Wave 7's brief rules that
  `check` refuses it, under the stricter-rule convention, with a chapter 13
  statement and a vector.
- **Two `manifest.*` codes no 1.0 manifest can produce**
  (`manifest.any_of`, `manifest.one_of.ambiguous`) join the W7 work that
  ADR-006 A1 rules.
- **For Ed to ratify at the train's close:**
  - R-W6-3;
  - fault bits 9 and 10;
  - `internal/hostfs` hosting the pty test;
  - R-W6-9b, which proves content-class hostile cases through validate and
    check steps. ADR-011 D16 item 3's "the same findings" collides with
    MTS-AUT-002's `report.step_skipped` in `check`.
- **Still running when the wave closed:** W6-NB1
  (`01a15821-e3c7-7f0f-824c-26cb1e992efa`) and W6-NB2
  (`01a15822-3703-7455-86b5-c54ca3c65732`).

## 13. Wave 7 — the example packages and the close-out

**Objective.** The use cases of PRD §4.1 to §4.5 ship as packages, and the
specification's 1.0 text is complete and ready for the release act.

**Carries:**
- PRD §4.1 to §4.5, SC2, SC3 and SC4; R16 in full; and R17 items 1 and 2
  (documented golden paths).
- ADR-011 D4 (examples and the worked package).
- Every ADR's implementation log.

**Does not touch:** the release itself, the matrix and the live rounds (§6).

**Scope (outline):**

- **One package per use case**, each a source directory
  `examples/<package name>/` passing the Wave 6 harness:
  - **§4.1, the conformance report.** The edition decides the tables through
    `input_constraints`, and criteria are supplied as CSV. It renders a
    `.docx` through the `ooxml` profile from templates of its XML parts with
    no split-run finding, and a machine-readable copy validated by
    `validate`. Its references hold the criterion text.
  - **§4.2, the staged incident report.** The glossary is the schemas, and
    the stage decides required fields. It carries SC2's test: each stage
    accepted with exactly its fields, and every omitted mandatory field
    reported.
  - **§4.3, the status widget.** A minimal package, with SC3's test: one
    `validate` and one `render` from JSON, no harness.
  - **§4.4, the static site.** A Markdown collection with frontmatter, one
    page per article, an index and a feed over the same collection, and
    assets.
  - **§4.5, the repository scaffold.** Conditional files, names derived from
    inputs, `executable: true`, and dot-directories under `templates/`.

  Each package's sources and their reuse terms are recorded in its
  `references/`. An example does not merge before §6's terms check for it
  ends.
- **The worked package**: `B-worked-package.md` built from `gen/` vectors on
  the authoring golden path, with `TestExamplesMatchVectors` green.
- **The codes no vector could produce** (design session W3-A1):
  - ADR-006 Amendment A1: the 14 `manifest.*` keyword codes the 1.0
    manifest schema cannot produce leave the registry, and a tripwire
    holds every keyword the manifest schema uses to a registered code;
  - ADR-011 Amendment A4: the setup's `absent` and `unreadable`, the
    `unreadable-files` requirement, and `open` vectors for
    `package.not_found` and `package.not_readable`, which stay class
    `refused`.
- **The close-out**:
  - `A-references.md` complete; the front page's contents; `CHANGELOG.md`'s
    `1.0` section saying "First revision" and still `unreleased`;
  - `README.md` with the status and both golden paths;
  - a documentation index under `docs/cli/` pointing at `metais help`, which
    stays authoritative;
  - each ADR's implementation log, from the plan's wave verdicts;
  - `docs/INDEX.md` with a row for this plan;
  - the unwired list removed, because it is empty.

  The close-out describes `examples/` generically, so it holds whichever
  packages merge before it.

**Acceptance.**

- The harness passes over every package under `examples/`.
- SC2 and SC3 have their tests. SC4 holds.
- Every chapter is complete: `TestCoverage`, `TestChangelog`,
  `TestFindingCodesCovered` with no unwired list, and every ADR-011 D14
  tripwire, all green.
- The front page reads `Status: Draft`, ready for the release commit.
- The SC1 and SC9 acts are filed.

**Decisions settled before filing:**
- R1-008: the `$id` domain, registered by §6's act before publication.
- R1-009: the media type is named, and IANA comes after publication.
- R1-022: the licence.
- R2-169: one living text, with `spec-1.0` tags.
- R2-170: released vectors are the tie-breaker.
- R2-177: specification examples are vectors.
- **Example packages live under `examples/` at the repository root**, as
  source directories. They are not conformance vectors (ADR-011's *What this
  does not decide*), and ADR-001 D5 item 5 forbids a `README.md` inside a
  source directory's top level, so each package's own documentation is its
  guide and its meaning documents.

**Rulings made during the wave:**
- **A code whose input name cannot live in the committed suite tree gets a
  run-time setup** (plan orchestrator, 2026-10-20; for Ed to ratify).
  - `collection.item_name` needs an item name holding `:`, `\` or invalid
    UTF-8, which ADR-011 D9 item 8 forbids in a vector's tree.
  - An ADR-011 amendment in the shape of Ed's A4 adds a setup member that
    creates such a name at run time, behind a `requires` probe for a
    filesystem that allows it. The portable-name rule for the committed
    tree is untouched.
  - The same applies to `output.destination.write_failed`, if no existing
    setup reaches it.
  - The unwired list ends empty, with no exemption list.
- **The release builder remains held** for Ed's design session
  `01a1566e-2786-7173-84ff-c1485fcf8cdf`. His ruling is recorded here when
  it comes.

## 14. What this plan does not cover

These are ruled out by PRD-001 §3, or deferred:

- Running code from a package: hooks, scripts, custom filters, post-render
  tasks or external converters.
- Template languages other than Liquid.
- Prescribing where data comes from: source kinds, question plans or an
  interactive gathering loop.
- Persisted answers or state.
- Re-rendering onto earlier output and merging.
- Live or client-refreshed output.
- Templated binary output.
- Compression or encoding transforms beyond zip containers.
- Fetching packages or inputs over the network, or a registry.
- A public Go API.
- Signing packages.
- An MCP server mode, and integration with any harness or workflow system,
  tera included. Such integrations are built on the command line, outside
  this repository.
- Shipping a Windows binary. Windows is compiled and vetted, not shipped
  (ADR-010 D12 item 2).
- zip64, which needs a new major (ADR-001 D3).
- The move to `go.yaml.in/yaml/v4` or to Go 1.27 and Unicode 17.0.0 (ADR-010
  D1 item 4, D6 item 6).
- IANA registration of `application/vnd.metais+zip`, which comes after the
  specification is public (R1-009).
- Hosting anything at `https://metais.org/` (ADR-011 D7 item 7).
- Publishing the specification or the tool outside this private repository.
- Continuous integration.
- A second implementation.
- A register of third-party conformance claims.

**Follow-up after the train** (Ed, 2026-10-19): design extension points for
the render pipeline. Output forms become handlers, post-render transforms an
ordered table of stages, and destinations sinks, so that a new feature is
one new file and one table entry. It is filed at the train's close as a
design item for Ed, and it is not built in this train.

## Appendix A — where each requirement and success criterion is proved

| PRD | Waves |
|---|---|
| R1 one file | 2 (zip), 3 (open, digests, compiled files), 6 (`pack`) |
| R2 read in place | 2 (the JSON document), 3 (`inspect`), 4 (the reading commands) |
| R3 data specification | 2 (the compiler), 3 (at open), 4 (`validate`) |
| R4 input forms | 2 (readers), 3 (the layer), 4 (vectors), 5 (`embed`) |
| R5 supplying inputs | 3 (the layer), 4 (vectors) |
| R6 validation | 1 (the model), 2 (the mapping), 4 (vectors, SC8) |
| R7 template language | 2 to 3 (the engine), 4 (chapter 7), 5 (chapter 8) |
| R8 outputs | 4 (template outputs), 5 (the rest) |
| R9 transforms | 5, 6 (containers in `test`) |
| R10 purity and determinism | 1 (tripwires), 2 (the runner's environment B), 4 (the determinism detector), 6 (matrix rehearsal) |
| R11 provenance | 5 |
| R12 safe rendering | 4 (new directory, confinement), 5 (the rest) |
| R13 authoring | 3 (`init`, the generator), 4 (`check`, `unpack`), 5 (`test`, `sample`), 6 (`pack`) |
| R14 untrusted input | 1, 2, 3, 4, 6 |
| R15 versions | 2 (`version`, the gate), 3 (vectors) |
| R16 specification | every wave, completed in 7 |
| R17 golden paths | 2 (help), 3 (`inspect`'s next), 7 (README) |
| R18 portable | 1 (the gate's targets), 6 (the release builder), §6 (the matrix) |
| SC1 | §6 |
| SC2, SC3, SC4 | 7 |
| SC5 | 2 (the record), §6 (the matrix) |
| SC6 | 3 onward, in full in 6 |
| SC7 | 2 (the environments), 4 and 5 (the canaries), 6 (full reach) |
| SC8 | 4 |
| SC9 | 7 (the worked package), §6 (the round) |
