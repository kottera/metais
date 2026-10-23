# ADR-004 — W1-NB1: `duration` refuses a week combined with other parts

**2026-10-23, PLAN-2026-10-metais-values-and-rules W1-NB1.** Wave 1's
verification found that no case refused an ISO week followed by another
part, which MTS-FLT-021's ABNF forbids. `flt/duration-refusals` now refuses
`P1WT1H` and `P1W1D`, each with `template.duration_input`, reason `form`.
The `P1WT1H` case is red when a week may be followed by a time part. No
behaviour changes.
