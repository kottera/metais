# Documents

## Requirements

| PRD | Title | Status |
|---|---|---|
| [PRD-001](prd/PRD-001-metais-v1.md) | METAIS v1 — the package format and its reference tool | Accepted |

## Plans

| Plan | Feature | Status |
|---|---|---|
| [PLAN-2026-10-metais-v1](plans/PLAN-2026-10-metais-v1.md) | `metais-v1` | Ratified; §3 gives each wave's status |

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

`dev/rounds/` holds the runbooks of the live rounds of plan §6:
each note says what its round establishes and what it reports back.
