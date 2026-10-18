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

*Not yet written. The wave orchestrator replaces this line with the wave's
closeout.*
