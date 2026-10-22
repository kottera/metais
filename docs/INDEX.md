# Documents

## Requirements

| PRD | Title | Status |
|---|---|---|
| [PRD-001](prd/PRD-001-metais-v1.md) | METAIS v1 — the package format and its reference tool | Accepted |

## Plans

| Plan | Feature | Status |
|---|---|---|
| [PLAN-2026-10-metais-v1](plans/PLAN-2026-10-metais-v1.md) | `metais-v1` | Ratified; §3 gives each wave's status |
| [PLAN-2026-10-metais-test-economy](plans/PLAN-2026-10-metais-test-economy.md) | `metais-test-economy` | Accepted |
| [PLAN-2026-10-metais-ux](plans/PLAN-2026-10-metais-ux.md) | `metais-ux` | Ratified; §3 gives each wave's status |
| [PLAN-2026-10-metais-render-seams](plans/PLAN-2026-10-metais-render-seams.md) | `metais-render-seams` | Accepted; §3 gives each wave's status |

Each wave's brief, ending in the closeout its wave orchestrator writes, is under [briefs/](briefs/), named
`brief-plan-2026-10-metais-v1-wave<N>.md`:

| Brief | Wave |
|---|---|
| [wave1](briefs/brief-plan-2026-10-metais-v1-wave1.md) | 1 — the seams |
| [wave2](briefs/brief-plan-2026-10-metais-v1-wave2.md) | 2 — the leaf libraries and the proof harness |
| [wave3](briefs/brief-plan-2026-10-metais-v1-wave3.md) | 3 — opening a package, and the composed libraries |
| [wave4](briefs/brief-plan-2026-10-metais-v1-wave4.md) | 4 — validate, read, the first render, check and unpack |
| [wave5](briefs/brief-plan-2026-10-metais-v1-wave5.md) | 5 — the render's capabilities, and the rest of authoring |
| [wave5b](briefs/brief-plan-2026-10-metais-v1-wave5b.md) | 5b — the gate's cost and merge friction |
| [wave6](briefs/brief-plan-2026-10-metais-v1-wave6.md) | 6 — pack, and hardening |
| [wave7](briefs/brief-plan-2026-10-metais-v1-wave7.md) | 7 — the example packages and the close-out |

## Decisions

| ADR | Title | Status |
|---|---|---|
| [ADR-001](adr/ADR-001-the-container-and-its-layout.md) | The container and its layout | Accepted |
| [ADR-002](adr/ADR-002-the-manifest-and-versioning.md) | The manifest and versioning | Accepted |
| [ADR-003](adr/ADR-003-the-data-specification.md) | The data specification | Accepted |
| [ADR-004](adr/ADR-004-the-template-environment.md) | The template environment | Accepted |
| [ADR-005](adr/ADR-005-outputs-transforms-and-rendering.md) | Outputs, transforms and rendering | Accepted |
| [ADR-006](adr/ADR-006-validation-and-findings.md) | Validation and findings | Accepted |
| [ADR-007](adr/ADR-007-authoring.md) | Authoring | Accepted |
| [ADR-008](adr/ADR-008-the-command-line-and-its-machine-output.md) | The command line and its machine output | Accepted |
| [ADR-009](adr/ADR-009-the-security-model.md) | The security model | Accepted |
| [ADR-010](adr/ADR-010-the-implementation.md) | The implementation | Accepted |
| [ADR-011](adr/ADR-011-the-specification-document-and-conformance.md) | The specification document and conformance | Accepted |
| [ADR-012](adr/ADR-012-test-economy.md) | Test economy | Accepted |

PRD-001 §27 maps each requirement to the ADR that designs it. [RULINGS-v1](adr/RULINGS-v1.md) records the rulings the v1 ADRs proposed, ratified 2026-10-18.

Each ADR's implementation log is its "Implementation log" section together
with the entry files under [adr/logs/](adr/logs/)`ADR-<NNN>/`, one file per
work item (AGENTS.md, "Writing documentation").

## The command line

`metais help` is the command line's authoritative reference.
[cli/README.md](cli/README.md) is the index of what `docs/cli/` adds to it:
the schemas under [cli/schemas](cli/schemas/) that define the JSON document
each command prints, and the pages that state the rules of `pack`, `sample`
and `render --sample`, and `test` that only the command line has.

## Development

[dev/calibration.md](dev/calibration.md) is how the limit-calibration suite
of ADR-009 D18 item 4 is run on a target, the record it writes, and what
plan §6's act reports back.

[dev/benchmarks.md](dev/benchmarks.md) is how the benchmarks of ADR-010 D15
are run and compared with their committed baselines, and which benchmark
measures which budget.

[dev/release.md](dev/release.md) is how a release's archives are built
with the release builder of ADR-010 D12, and what it refuses, and how
the metais.org site, built from `site/`, goes live at the release.

[dev/ux-testing.md](dev/ux-testing.md) is how fresh agent sessions are run
over the UX scenario set of PLAN-2026-10-metais-ux: the scenarios, the
isolation checks the runner makes, what each run records, how a set is
run, paced and resumed, how each run is scored, how a set is scrubbed, and
the scorecard's format.

[dev/rounds/](dev/rounds/) holds the runbooks of the live rounds of plan §6:
each note says what its round establishes and what it reports back.

## UX scorecards

Each scorecard of PLAN-2026-10-metais-ux's fresh agent sessions is under
[ux/](ux/), named `scorecard-<date>-<name>.md`, in the runner's format of
[dev/ux-testing.md](dev/ux-testing.md):

| Scorecard | What it measures |
|---|---|
| [2026-10-21 baseline](ux/scorecard-2026-10-21-baseline.md) | Wave 1's baseline: every scenario on Opus, Sonnet and Haiku, on the binary before any Wave 2 change, without the skill installed, with the failures as candidates for Wave 2 |
| [2026-10-22 after](ux/scorecard-2026-10-22-after.md) | Wave 3's after scorecard, the one that counts toward R-UX-5: every scenario on Opus, Sonnet and Haiku, on post-Wave-2 `main`, without the skill installed, with the C0 and C1 runs' recorded verdicts, the judge's prompt, the comparison with the baseline and with the skill, and the root cause of each failure; and its standing result after W3-13's re-run of every consuming scenario in both modes on `main` with W3-12, the last, with W3-6's earlier re-run kept on record and R-UX-5's verdict stated both ways for the authoring threshold |
| [2026-10-22 after, with the skill](ux/scorecard-2026-10-22-after-with-skill.md) | The same set with `metais skill` installed in each session: recorded and compared, its consuming runs not counted toward R-UX-5 |
