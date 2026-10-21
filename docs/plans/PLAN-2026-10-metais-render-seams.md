# PLAN-2026-10: METAIS render seams — forms, producers, stages and sinks

**Author:** drafted 2026-10-21 by design session
`01a15ac4-2353-7576-a889-515c6ad35285`, under Ed Funnekotter's direction of
that day
**Date:** 2026-10-21
**Feature:** `metais-render-seams`. Every Work Order and pull request of this
train inherits it verbatim: `metais-render-seams [W2-1]: <action>` for a
Work Order, and `<type>(metais-render-seams): [W2-1] <change>` for a pull
request and its squash subject, by `kottera/kottera`'s
`docs/dev/work-order-and-pr-naming.md`, which AGENTS.md adopts.
**Status:** Accepted 2026-10-21 by Ed Funnekotter, with the rulings of §9.
**For ADR:** [ADR-010 Amendment A7](../adr/ADR-010-the-implementation.md#amendments).
**Repository:** `kottera/metais`, branch `main`. The train writes this
repository only.
**Depends on:** PLAN-2026-10-metais-test-economy's Wave 1. Its W1-4
reworks `internal/cli`'s command-line walks and its W1-5 adds the focused
race test of the render's goroutine at `internal/cli/render_write.go`; this
train moves both seams, so it starts after they merge.
**Blocks:** the v1.0 release (PLAN-2026-10-metais-v1 §6, T-A3). Ed ruled on
2026-10-21 that v1.0 ships with this work done, so that the release leaves
no pending work for a 1.1.

## 1. Purpose

ADR-005 D13 states the render as phases, and the code that runs them decides
what a planned file is by flags every phase tests again, so a feature edits
every phase. In PLAN-2026-10-metais-v1 Wave 5 about nine parallel items
edited `render.go`. ADR-010 A7 puts the phases behind four tables: forms
(how a declaration expands), producers (where a file's bytes come from),
stages (post-render transforms, in order) and sinks (where a render goes).
This plan builds it.

**Done means:**

- Every item of ADR-010 A7 is in the tree, each by the item §5 to §7 name.
- Every vector passes with its expected trees unchanged. The one vector
  added is W2-2's, for A7 item 8; no expectation is edited.
- Wave 2's items merged in parallel without editing the same lines outside
  a table entry, which is the measure of whether the seams work.
- The gate is green on `main`.

## 2. Execution model

The items are few and were derived with the design, so this plan carries
them and no plan orchestrator runs it. Each item is filed as a
`code-implementation-direct` Work Order with its `depends_on` edges, and one
`wave-verification` Work Order verifies the train after the last item.
Every Work Order gets `repo: kottera/metais` and `branch: main`. An item may
be split by whoever runs it, never merged with another, and never widened.

**Each item carries these rules whole:**

- **No behaviour changes but A7 item 8's.** No output byte, finding,
  finding order, exit status, help text or result record changes. The
  proof is the vectors with their expected trees untouched; an item that
  needs to edit an expectation has found a defect in itself, not in the
  vector.
- **It moves code, then changes it.** Where an item both relocates and
  rewrites, the relocation is its own commit, so review can read the
  rewrite as a diff.
- **It keeps A7 item 10's guarantees** and says, in its pull request, which
  detector proves each one it touched, and that the detector and its
  control ran.
- **It writes its log entries** to `docs/adr/logs/ADR-010/<address>.md`,
  and to `docs/adr/logs/ADR-005/<address>.md` where it changes how an
  ADR-005 decision is met (AGENTS.md, "Writing documentation").
- **It follows AGENTS.md**, its gate included.

## 3. Overview and sequencing

| Wave | Objective | Depends on |
|---|---|---|
| 0 | ADR-010 A7 and this plan, merged to `main` | — |
| 1 | **The move**: `internal/render` and its one entry point, used by `render` and `test`, laid out so that Wave 2's items own separate files | 0; PLAN-2026-10-metais-test-economy W1-4 and W1-5 |
| 2 | **The tables**: forms and producers; stages, with item 8's limit; sinks | 1 |
| 3 | **The tripwires**, then the train's verification | 2 |

Wave 2's items run in parallel. W1-1 sets the files each one owns (§5), and
the only lines two of them may both touch are table entries.

PLAN-2026-10-metais-ux's Wave 2 may edit `render`'s help text and refusals,
which stay in `internal/cli/render.go`; whichever merges second rebases.
No edge is needed.

## 4. Safety discipline (always on)

- **Every headline guarantee keeps its detector and its fault control**
  (ADR-010 D13), and every fault seam stays where D13 item 2's table puts
  it.
- **Do not widen a refusal or an expectation to make a proof pass**, as
  PLAN-2026-10-metais-v1 §4 says, whole.
- **No new dependency.**
- **Never write to another repository.**

## 5. Wave 1 — the move

### W1-1 — `internal/render`, its entry point, and the files Wave 2 owns

ADR-010 A7 items 2 and 3.

- **Change:**
  - Move ADR-005 D13's phases out of `internal/cli` into `internal/render`:
    `render_plan.go`, `render_parse.go`, `render_check.go`'s path checks and
    destination check, `render_write.go` and `transform.go`. `planned`
    becomes `render.File`, its bytes unexported.
  - One entry point plans, produces and emits to a sink the command gives
    it, with `--only`'s selection between producing and emitting. In this
    item the sinks are the emission code as it is, behind a function per
    mode; W2-3 makes them the interface.
  - `render` and `test` both call it. `test` keeps its own parse of the
    templates its template outputs name, before any example runs (A7 item
    2); its `parseTemplates` stays in `internal/cli`.
  - `internal/cli/render.go` keeps the command: flags, refusals, help, the
    result records and the choice of emission. `outcome.stream` and
    `streamsBytes` are unchanged.
  - Lay the files out so that each Wave 2 item owns its own, and call
    every seam Wave 2 will fill through a function that already has its
    final name and signature — `runStages(c *Staging, f *File)
    []finding.Finding` and `(*File).Open() (io.Reader, func(error) error)`
    among them — so that no Wave 2 item edits another's call site:

    | File in `internal/render` | Holds | Owned in Wave 2 by |
    |---|---|---|
    | `render.go` | the options, the entry point, the depth-first walk | — |
    | `paths.go` | ADR-005 D12's checks and their findings | — |
    | `file.go` | `File`, its record methods, `(*File).Open` | W2-1 |
    | `plan.go` | the planner, assets, mirrors, containers and their order | W2-1 |
    | `produce.go` | phase 4: parsing, the placement map, rendering, measuring | W2-1 |
    | `stages.go` | `runStages`, holding today's CRLF conversion then validation | W2-2 |
    | `emit.go`, `destination.go` | the archive, one file's bytes, the commits, the destination check | W2-3 |

- **What proves it:** every vector, untouched; `go test ./internal/cli
  ./internal/render ./internal/guarantees`, the detectors and their
  controls among them; test-economy W1-5's race test, moved with the
  goroutine.
- **Log:** ADR-010 and ADR-005.

## 6. Wave 2 — the tables

### W2-1 — forms and producers

ADR-010 A7 items 4, 5 and 9.

- **Change:** the `Form` and `Producer` interfaces, each chosen by one
  function with a `switch`; the forms `template`, `asset`, `mirror` and
  `container`; the producers `rendered`, `copied`, `assembled` and
  `literal`, each in a file of its own. The flags `asset != ""`, `tool`
  and `isContainer()` are gone: every phase after planning asks the file's
  producer. The goroutine of ADR-010 A6 item 1 moves into the `assembled`
  producer's file with its race test. The `rendered` producer calls
  `runStages`, whose body it does not touch.
- **What proves it:** every vector, untouched; the determinism detector
  and `MapOrder`'s control; the race test.
- **Log:** ADR-010.

### W2-2 — stages, and their growth charged

ADR-010 A7 items 6, 8 and 9.

- **Change:** the `Stage` type and the ordered slice, CRLF conversion then
  validation, each in a file of its own; `Staging.SetBytes`, which charges
  a change in length against `output-size` and `output-total` on the
  meter `template.Renderer` charged, through a method that
  `internal/template` gains for it. A breach is the file's
  `limit.exceeded.output`, and no later stage runs on that file.
- **Vector:** a file declaring `newline: crlf` whose rendered bytes are
  within `output-size` and whose converted bytes are not, under a lowered
  `--limit`, refused with `limit.exceeded.output`, stating
  [MTS-SEC-010](../../spec/14-security.md#mts-sec-010) (an output's bytes
  are the bytes the render writes). If the item finds the statement does
  not say so plainly, it says so in chapter 14 in the same change. A
  second vector shows the same for `output-total`.
- **Red first:** the vectors fail on `main` before the change: today the
  file is written with exit status 0.
- **What proves it:** the vectors; every other vector, untouched.
- **Log:** ADR-010, ADR-005 (D13 item 4 now holds after D10) and ADR-009
  (D3's output bounds).

### W2-3 — sinks

ADR-010 A7 items 7 and 9.

- **Change:** the `Sink` interface in `internal/render/sink`, with the new
  directory, the existing directory, the archive, one file's bytes and
  `test`'s in-memory comparison, each in a file of its own; the dry run as
  a wrapper that runs `Check` and never `Emit`. `internal/cli` constructs
  the sink; whether standard output carries bytes is still decided from
  the line by `streamsBytes` (A7 item 7). `test`'s comparison of an
  example with its expected tree moves out of `internal/cli/test.go` into
  its sink; the differences it reports are unchanged.
- **What proves it:** every vector, untouched; the confinement detector
  and `SkipConfine`'s control; `TestExistingDestinationDetector` with
  `ReadExisting` and `RemoveForeign`; `test`'s own tests.
- **Log:** ADR-010.

## 7. Wave 3 — the tripwires

### W3-1 — the layering rule and the one goroutine

ADR-010 A7 item 11.

- **Change:** a `single-entry-point` rule in
  `internal/guarantees/layers.json`: `internal/render` imports
  `internal/hostfs` only for `CheckPaths`. A guarantee test that
  `internal/render` and its subpackages hold exactly one `go` statement, in
  the `assembled` producer's file.
- **Red first:** each rule is shown failing on a scratch commit that adds a
  `hostfs` write call to a stage and a second `go` statement to a sink. The
  scratch commit never merges.
- **Log:** ADR-010.

## 8. Verification

One `wave-verification` Work Order, after W3-1, audits every item against
§5 to §7 and §1's "done means", and runs the gate once on the merged tree.
It reports whether Wave 2's three pull requests touched the same lines
anywhere but a table entry, naming any file where they did. That is the
train's measure. Its verdict is the release's edge: T-A3 depends on it.

## 9. Rulings

Ed accepted every lean on 2026-10-21, with two narrowed in the session
before he ruled:

- **R-RS-1.** The extension points are internal seams for the tool's
  developers. R9's closed set, the specification and the exclusion of
  package code are unchanged.
- **R-RS-2.** `internal/render` and `internal/render/sink`, the packages
  ADR-010 D2 names; `internal/cli` keeps each command. `test` shares the
  plan, produce and emit steps, and keeps its own early parse of its
  template outputs' templates, which is `test`'s behaviour.
- **R-RS-3.** Forms and producers are separate tables.
- **R-RS-4.** The tables are explicit: a `switch` or a slice in one file,
  never `init` registration, never a map.
- **R-RS-5.** A stage changes bytes only through `SetBytes`, which charges
  the change against the output limits; a stage reaches no host. This is
  also the fix for the defect found in the session (A7 item 8), which is
  therefore not filed on its own.
- **R-RS-6.** Sinks own phases 3 and 5, the dry run wraps a sink, and
  `test`'s comparison is a sink. Whether standard output carries bytes stays
  decided from the command line.
- **R-RS-7.** The design is ADR-010 Amendment A7, not a new ADR, since
  nothing ADR-005 decides changes.
- **R-RS-8.** This plan's items are filed as Work Orders with `depends_on`
  edges; no plan orchestrator.
- **R-RS-9.** The v1.0 release waits for this train, so that v1.0 leaves no
  pending work for a 1.1.
