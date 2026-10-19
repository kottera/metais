# Documents

## Requirements

| PRD | Title | Status |
|---|---|---|
| [PRD-001](prd/PRD-001-metais-v1.md) | METAIS v1 — the package format and its reference tool | Accepted |

## Plans

| Plan | Feature | Status |
|---|---|---|
| [PLAN-2026-10-metais-v1](plans/PLAN-2026-10-metais-v1.md) | `metais-v1` | Ratified |

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

## The command line

The JSON documents each command prints are defined by the schemas under
[cli/schemas](cli/schemas/). [cli/sample.md](cli/sample.md) states the rules
of `metais sample` and `render --sample` that only the command line has.
