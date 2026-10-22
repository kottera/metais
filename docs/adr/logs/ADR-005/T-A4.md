# ADR-005 — T-A4: Amendment A3, the archive is written by `internal/tarw`, and ADR-010 A1 is accepted

**2026-10-22, PLAN-2026-10-metais-v1 T-A4.** Ed Funnekotter ratified the
train's outcome.

- **Amendment A3** writes into D17 item 5 and the needs from ADR-010 that
  `internal/tarw` writes the standard-output archive, the stricter rule
  W5-9 applied because `archive/tar` imports `os/user`. D17's profile is
  unchanged; ADR-010 A9 item 2 records the proof of equivalence.
- **ADR-010 Amendment A1 is accepted**, so D14 item 1.4's cross-reference
  to it no longer calls it proposed.
- **W4-0's ownership boundary for `StageNew` (R-W4-0)** and **W5-V2's
  `Present` status** are ratified as this ADR's logs record them; A1 item
  6 already holds the first.
