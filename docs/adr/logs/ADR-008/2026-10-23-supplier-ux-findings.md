# ADR-008 — 2026-10-23-supplier-ux-findings: Amendment A17, a Next: block names each command line once

**2026-10-23, outside a train.** This item carries out Ed's ruling of
2026-10-23 that the external UX test's findings land before v1.0.
Amendment A17 is proposed with it.

- `inspect`'s `Next:` block over the accessibility conformance report
  named `metais read <package> inputs/tables.md` three times, once for
  each table that shares the guide. `inspectNext` now offers each meaning
  document once, at its first input, naming every input it explains.
- `emit` keeps the first step of each command line in every envelope's
  `next` (`uniqueSteps`), so no command's block repeats one.
- `TestInspectNextReadsEachMeaningOnce` failed on main with three read
  steps for `inputs/tables.md`.
