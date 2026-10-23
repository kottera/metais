# ADR-008 — T-A4: Amendment A15, the envelope's `summary`, and `outline`'s reader

**2026-10-22, PLAN-2026-10-metais-v1 T-A4.** Ed Funnekotter ratified the
train's outcome.

- **Amendment A15** writes the W2-1 reconciliation, which this ADR's
  implementation log records for ratification, into D10 item 1: the
  envelope's members are `format`, `ok`, `findings`, `summary`, `result`
  and `next`. `TestEnvelopeMemberOrder` already holds the type and
  `metais.envelope.1.json` to that order.
- ***Interfaces* (ADR-010)** names METAIS's own CommonMark reading for
  `outline`, under ADR-010 Amendment A3, which is accepted, and no longer
  calls ADR-010 Amendment A1 proposed.
