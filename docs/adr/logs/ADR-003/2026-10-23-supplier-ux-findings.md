# ADR-003 — 2026-10-23-supplier-ux-findings: Amendment A3, an input the constraints refuse is reported alone

**2026-10-23, outside a train.** This item carries out Ed's ruling of
2026-10-23 that the external UX test's findings land before v1.0. In the
test, an agent given only the binary and
`accessibility-conformance-report-1.0.0.metais` supplied `section508.csv`
for the `wcag` edition. It was told the input was not allowed here, then
told six things about its rows. Amendment A3 is proposed with the fix.

- MTS-DAT-010 required every failure of an input's value to be reported,
  which is what the tool did. MTS-DAT-010, MTS-DAT-012 and MTS-FND-004 now
  say that an input whose member the constraints answer with a `false`
  schema is reported by one `input.forbidden` at its root and nothing
  else.
- `internal/inputs`' `mapFailures` maps an input's constraint failures
  before its own schema's. When one refuses the input at its root, it
  reports those refusals and maps nothing else of the input.
  `TestForbiddenInputNotValidated` failed on main, with `input.enum` and
  `input.min_items` beside the refusal. Its controls hold an allowed input
  to every failure, and a `false` schema below the root to refusing only
  its member.
- The new vector `dat/input-constraints-refused` proves the statement:
  one finding for the edition that refuses the table, and three for the
  edition that allows it. The `refused` case fails on main.
