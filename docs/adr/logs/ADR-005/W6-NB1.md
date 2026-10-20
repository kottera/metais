# ADR-005 — W6-NB1: chapter 2 states D12 item 4's device names for entry names

**2026-10-20, PLAN-2026-10-metais-v1 W6-NB1.** Chapter 2's entry-name
rule listed the Windows device names without D12 item 4's superscript-digit
forms (`com¹` to `lpt³`) and without the trim of trailing U+0020 before a
name's first `.`, which `internal/unitext` applies to entry names and
chapter 11's MTS-PTH-010 states for output paths. The ADR's rule is the
stricter, and it is the one the tool enforces: MTS-PKG-070 states it for
entry names. The vectors `pkg/name-reserved-superscript` (`assets/COM¹.txt`)
and `pkg/name-reserved-spaced` (`assets/Con .md`) cite it, their packages
written by `internal/vectorgen`.
