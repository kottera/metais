# ADR-006 — W7-NB3: the nested-findings bound pinned in every go test

**2026-10-20, PLAN-2026-10-metais-v1 W7-NB3.** W7-C6 built a choice's
finding listing at most the command's findings limit
(`schemafail.Target.Listed`, `finding.WithListedAtMost`), so that no one
`input.any_of` or `input.one_of` holds more than a report could list
(MTS-FND-020, MTS-FND-021). W7-V found only the opt-in calibration case
`findings-nested` measured it: since the report caps what it lists when it
is read, removing the bound changes no document, only the memory held.

`internal/inputs`' `TestNestedFindingsBounded` measures it by a work
counter, the findings a choice's finding holds, itself and every finding
within its alternatives at any depth, as `Validate` returns it before any
report. An `anyOf`, and a `oneOf`, of two branches that each fail eight
properties holds 17 at the default limit, and exactly the limit at
`findings` 1, 2, 5 and 16. Dropping `WithListedAtMost` from the mapping,
or setting the inputs target's `Listed` to 0, turns it red. No behaviour
changes. The output validation's and the manifest's targets set `Listed`
the same way and are not pinned by this test.
