# ADR-008 — W3-NB2: inspect's check step says what MTS-AUT-003 does, and grep's and unpack's help say an example's inputs are never a real document's data (Amendment A12)

**2026-10-22, PLAN-2026-10-metais-ux W3-NB2.** Amendment A12 is
implemented. `internal/cli`'s `inspectNext` gives the step
`metais check <package>` of a source directory the reason held by the new
constant `checkStep`, which says that check reports every defect without
data, or only the refusal of a source it cannot read, as MTS-AUT-003 and
`metais help check` say. The help of `grep` and `unpack` ends with
`examplesEntries`, the paragraph the help of `ls`, `read` and `extract`
carries under A11 item 1.

- `supplier_test.go` holds the amendment: `TestInspectNextOnTheFly` checks
  the check step's words, and `TestExampleInputsAreNeverFacts` checks the
  help of `grep` and `unpack`. Each was red against the code before the
  change.
- The help goldens were rewritten by `regenerate.sh`; `skill-print.md`
  holds neither text and did not change. No chapter of the specification,
  published schema or vector changes.
