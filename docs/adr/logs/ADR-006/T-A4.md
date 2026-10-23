# ADR-006 — T-A4: Amendment A3, the per-position keyword reading and the train's reporting readings

**2026-10-22, PLAN-2026-10-metais-v1 T-A4.** Ed Funnekotter ratified the
train's outcome. Amendment A3 writes into this ADR what its logs recorded:

- **A1 item 4 read where each keyword stands** (W7-1, tightened by W7-NB2):
  `manifest.any_of` and `manifest.one_of.ambiguous` are not registered for
  1.0, and two `oneOf` branches are exclusive only when one allows only an
  object. A1 items 1 and 4 point at it.
- **D4 item 1:** the inputs object as a whole is the invocation
  (`input.constraints_failed`, `input.json_arg_invalid`; W3-2).
- **D9 item 7:** `check`'s cross-reference suggestions follow ADR-007 D3's
  edit distance 2 (W3-6).
- ***Conformance*:** exit statuses 2 and 3, and the classes `usage` and
  `internal`, are proved by `internal/cli` tests (plan §8).

ADR-003 A1 takes the de-duplication reading into ADR-003 D9 item 2, so the
three reporting disagreements of Wave 3 are settled in the text. No test
changes: each reading is what the tool already does, and
`TestManifestKeywordCodes` and its control already hold item 1.
